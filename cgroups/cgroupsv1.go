package cgroups

import (
	"fmt"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func IsUnified() bool {
	return cgroups.Mode() == cgroups.Unified
}

func AddV2(
	containerID string,
	devices []specs.LinuxDeviceCgroup,
	pid int,
) error {
	cg, err := cgroup2.NewSystemd(
		"/",
		fmt.Sprintf("%s.slice", containerID),
		-1,
		&cgroup2.Resources{
			Devices: devices,
		},
	)
	if err != nil {
		return fmt.Errorf("create cgroups (id: %s): %w", containerID, err)
	}

	if err := cg.AddProc(uint64(pid)); err != nil {
		return fmt.Errorf("add pid to cgroup2: %w", err)
	}

	return nil
}

func AddV1(
	path string,
	devices []specs.LinuxDeviceCgroup,
	pid int,
) error {
	staticPath := cgroup1.StaticPath(path)

	cg, err := cgroup1.New(
		staticPath,
		&specs.LinuxResources{
			Devices: devices,
		},
	)
	if err != nil {
		return fmt.Errorf("create cgroups (path: %s): %w", path, err)
	}

	if err := cg.Add(cgroup1.Process{Pid: pid}); err != nil {
		return fmt.Errorf("add cgroups (path: %s, pid: %d): %w", path, pid, err)
	}

	return nil
}

func DeleteV2(containerID string) error {
	cg, err := cgroup2.LoadSystemd("/", fmt.Sprintf("%s.slice", containerID))
	if err != nil {
		return fmt.Errorf("load cgroups (id: %s): ", containerID, err)
	}

	if err := cg.DeleteSystemd(); err != nil {
		return fmt.Errorf("delete cgroups (id: %s): %w", err)
	}

	return nil
}

func DeleteV1(path string) error {
	staticPath := cgroup1.StaticPath(path)

	cg, err := cgroup1.Load(staticPath)
	if err != nil {
		return fmt.Errorf("load cgroups (path: %s): %w", path, err)
	}

	if err := cg.Delete(); err != nil {
		return fmt.Errorf("delete cgroups (path: %s): %w", path, err)
	}

	return nil
}
