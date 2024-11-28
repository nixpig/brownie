package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/container/filesystem"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

func (c *Container) Reexec1(log *zerolog.Logger) error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(filepath.Join("/var/lib/brownie/containers", c.ID(), initSockFilename))
	if err != nil {
		return fmt.Errorf("create init sock sender: %w", err)
	}
	defer c.initIPC.closer()

	for _, ns := range c.Spec.Linux.Namespaces {
		if ns.Path == "" {
			continue
		}

		// FIXME: setns on "mount" and "pid" namespaces fails with EINVAL
		// See issue in go - https://github.com/golang/go/issues/8676
		// Solution to this used by runc - https://github.com/opencontainers/runc/tree/main/libcontainer/nsenter
		if ns.Type == specs.PIDNamespace || ns.Type == specs.MountNamespace {
			continue
		}

		fd, err := syscall.Open(ns.Path, syscall.O_RDONLY, 0666)
		if err != nil {
			return fmt.Errorf("open ns path: %w", err)
		}
		defer syscall.Close(fd)

		syscall.RawSyscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	}

	// if opts.ConsoleSocketFD != 0 {
	// 	log.Info().Msg("creating new terminal pty")
	// 	pty, err := terminal.NewPty()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer pty.Close()
	//
	// 	log.Info().Msg("connecting to terminal pty")
	// 	if err := pty.Connect(); err != nil {
	// 		return err
	// 	}
	//
	// 	log.Info().Msg("opening terminal pty socket")
	// 	consoleSocketPty := terminal.OpenPtySocket(
	// 		opts.ConsoleSocketFD,
	// 		opts.ConsoleSocketPath,
	// 	)
	// 	defer consoleSocketPty.Close()
	//
	// 	// FIXME: how do we pass ptysocket struct between fork?
	// 	log.Info().Msg("send message over terminal pty socket")
	// 	if err := consoleSocketPty.SendMsg(pty); err != nil {
	// 		return err
	// 	}
	// }

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join("/var/lib/brownie/containers", c.ID(), containerSockFilename),
	); err != nil {
		return fmt.Errorf("remove socket before creating: %w", err)
	}

	listCh, listCloser, err := ipc.NewReceiver(filepath.Join("/var/lib/brownie/containers", c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("create new socket receiver channel: %w", err)
	}
	defer listCloser()

	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			return fmt.Errorf("create oom score adj file: %w", err)
		}
	}

	cmd := exec.Command(
		"/proc/self/exe",
		[]string{"reexec", "--stage", "2", c.ID()}...,
	)

	c.initIPC.ch <- []byte("ready")

	if err := ipc.WaitForMsg(listCh, "start", func() error {
		if err := cmd.Start(); err != nil {
			log.Error().Err(err).Msg("🔷 failed to start container")
			c.SetStatus(specs.StateStopped)
			if err := c.Save(); err != nil {
				return fmt.Errorf("(start 1) write state file: %w", err)
			}

			return err
		}

		c.SetStatus(specs.StateRunning)
		if err := c.Save(); err != nil {
			// do something with err??
			log.Error().Err(err).Msg("⁉️ host save state running")
			fmt.Println(err)
			return fmt.Errorf("save host container state: %w", err)
		}

		// FIXME: do these need to move up before the cmd.Wait call??
		if err := c.ExecHooks("poststart", log); err != nil {
			// TODO: how to handle this (log a warning) from start command??
			// FIXME: needs to 'log a warning'
			fmt.Println("WARNING: ", err)
		}

		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("ERROR IN WAITING IN REEXEC1")
			return fmt.Errorf("waiting for cmd wait in reexec: %w", err)
		}

		return nil
	}); err != nil {
		log.Error().Err(err).Msg("error in waitformsg")
		return err
	}

	return nil
}