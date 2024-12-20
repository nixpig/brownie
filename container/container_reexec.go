package container

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"syscall"

	"github.com/nixpig/brownie/capabilities"
	"github.com/nixpig/brownie/cgroups"
	"github.com/nixpig/brownie/filesystem"
	"github.com/nixpig/brownie/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Reexec() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if c.State.ConsoleSocket != nil {
		pty, err := terminal.NewPty()
		if err != nil {
			return fmt.Errorf("new pty: %w", err)
		}

		if err := terminal.SendPty(
			*c.State.ConsoleSocket,
			pty,
		); err != nil {
			return fmt.Errorf("connect pty and socket: %w", err)
		}

		if err := pty.Connect(); err != nil {
			return fmt.Errorf("connect pty: %w", err)
		}
	}

	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}

	// send "ready"
	initConn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.ID(), initSockFilename),
	)
	if err != nil {
		return err
	}

	initConn.Write([]byte("ready"))
	// close asap so it doesn't leak into the container
	defer initConn.Close()

	// wait for "start"
	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.ID(), containerSockFilename),
	); err != nil {
		return fmt.Errorf("remove any existing container socket: %w", err)
	}

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.ID(), containerSockFilename),
	)
	if err != nil {
		return err
	}

	containerConn, err := listener.Accept()
	if err != nil {
		return err
	}

	b := make([]byte, 128)
	n, err := containerConn.Read(b)
	if err != nil {
		return fmt.Errorf("read from container socket: %w", err)
	}

	msg := string(b[:n])
	if msg != "start" {
		return fmt.Errorf("expecting 'start', received '%s'", msg)
	}

	// close as soon as we're done so they don't leak into the container
	defer containerConn.Close()
	defer listener.Close()

	// after receiving "start"
	if c.Spec.Process == nil {
		return errors.New("process is required")
	}

	if err := filesystem.PivotRoot(c.Rootfs()); err != nil {
		return err
	}

	if err := filesystem.MountMaskedPaths(
		c.Spec.Linux.MaskedPaths,
	); err != nil {
		return err
	}

	if err := filesystem.MountReadonlyPaths(
		c.Spec.Linux.ReadonlyPaths,
	); err != nil {
		return err
	}

	if err := filesystem.SetRootfsMountPropagation(
		c.Spec.Linux.RootfsPropagation,
	); err != nil {
		return err
	}

	if err := filesystem.MountRootReadonly(
		c.Spec.Root.Readonly,
	); err != nil {
		return err
	}

	if slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname([]byte(c.Spec.Hostname)); err != nil {
			return err
		}

		if err := syscall.Setdomainname([]byte(c.Spec.Domainname)); err != nil {
			return err
		}
	}

	if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		return err
	}

	if err := capabilities.SetCapabilities(
		c.Spec.Process.Capabilities,
	); err != nil {
		return err
	}

	if err := syscall.Setuid(int(c.Spec.Process.User.UID)); err != nil {
		return fmt.Errorf("set UID: %w", err)
	}

	if err := syscall.Setgid(int(c.Spec.Process.User.GID)); err != nil {
		return fmt.Errorf("set GID: %w", err)
	}

	additionalGids := make([]int, len(c.Spec.Process.User.AdditionalGids))
	for i, gid := range c.Spec.Process.User.AdditionalGids {
		additionalGids[i] = int(gid)
	}

	if err := syscall.Setgroups(additionalGids); err != nil {
		return fmt.Errorf("set additional GIDs: %w", err)
	}

	if err := c.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	if err := os.Chdir(c.Spec.Process.Cwd); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}

	binary, err := exec.LookPath(c.Spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find process binary: %w", err)
	}

	args := c.Spec.Process.Args
	env := os.Environ()

	if err := syscall.Exec(binary, args, env); err != nil {
		return fmt.Errorf("execve (%s, %v, %v): %w", binary, args, env, err)
	}

	panic("if you're here, you done fucked up!")
}
