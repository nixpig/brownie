package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts) error {
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

	if stat, _ := os.Stat(containerPath); stat != nil {
		return pkg.ErrContainerExists
	}

	if err := os.MkdirAll(containerPath, os.ModeDir); err != nil {
		return fmt.Errorf("make brownie container directory: %w", err)
	}

	absBundlePath, err := filepath.Abs(opts.Bundle)
	if err != nil {
		return fmt.Errorf("get absolute path to bundle: %w", err)
	}

	configJSON, err := os.ReadFile(filepath.Join(absBundlePath, "config.json"))
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	state := &specs.State{
		Version:     spec.Version,
		ID:          opts.ID,
		Status:      specs.StateCreating,
		Bundle:      absBundlePath,
		Annotations: spec.Annotations,
	}

	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save state (creating): %w", err)
	}

	bundleRootfs := filepath.Join(absBundlePath, spec.Root.Path)
	containerRootfs := filepath.Join(containerPath, spec.Root.Path)
	containerConfigPath := filepath.Join(containerPath, "config.json")

	if err := cp.Copy(bundleRootfs, containerRootfs); err != nil {
		return fmt.Errorf("copy bundle rootfs to container rootfs: %w", err)
	}

	if err := cp.Copy(filepath.Join(absBundlePath, "config.json"), containerConfigPath); err != nil {
		return fmt.Errorf("copy container spec: %w", err)
	}

	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(spec.Hooks.CreateRuntime); err != nil {
		return fmt.Errorf("run createRuntime hooks: %w", err)
	}

	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(spec.Hooks.CreateContainer); err != nil {
		return fmt.Errorf("run createContainer hooks: %w", err)
	}

	initSockAddr := filepath.Join(containerPath, "init.sock")
	containerSockAddr := filepath.Join(containerPath, "container.sock")

	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			"some stupid id",
			initSockAddr,
			containerSockAddr,
		}...)

	if spec.Linux == nil {
		return errors.New("not a linux container")
	}
	var cloneFlags uintptr
	for _, ns := range spec.Linux.Namespaces {
		ns := internal.LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			return err
		}

		cloneFlags = cloneFlags | flag
	}

	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   cloneFlags,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(spec.Process.User.UID),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(spec.Process.User.GID),
				Size:        1,
			},
		},
	}

	forkCmd.Start()
	pid := forkCmd.Process.Pid
	if err := forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach from process: %w", err)
	}

	// wait for container to be ready
	if err := os.RemoveAll(initSockAddr); err != nil {
		return err
	}
	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on init socket: %w", err)
	}
	defer listener.Close()

	initConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer initConn.Close()

	b := make([]byte, 128)

	for {
		n, err := initConn.Read(b)
		if err != nil || n == 0 {
			continue
		}

		if string(b[:n]) == "ready" {
			break
		}
	}

	state.Status = specs.StateCreated
	state.Pid = pid
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
