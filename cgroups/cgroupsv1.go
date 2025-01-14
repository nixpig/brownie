package cgroups

import (
	"fmt"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/opencontainers/runtime-spec/specs-go"
)

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
