package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

type Device struct {
	Source string
	Target string
	Fstype string
	Flags  uintptr
	Data   string
}

var (
	defaultFileMode        = os.FileMode(0666)
	defaultUID      uint32 = 0
	defaultGID      uint32 = 0
)

var (
	AllDevices           = "a"
	BlockDevice          = "b"
	CharDevice           = "c"
	UnbufferedCharDevice = "u"
	FifoDevice           = "p"
)

var defaultDevices = []specs.LinuxDevice{
	{
		Type:     CharDevice,
		Path:     "/dev/null",
		Major:    1,
		Minor:    3,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/zero",
		Major:    1,
		Minor:    5,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/full",
		Major:    1,
		Minor:    7,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/random",
		Major:    1,
		Minor:    8,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/urandom",
		Major:    1,
		Minor:    9,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/tty",
		Major:    5,
		Minor:    0,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
}

func (d *Device) Mount() error {
	if _, err := os.Stat(d.Target); os.IsNotExist(err) {
		f, err := os.Create(d.Target)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("create device target if not exists: %w", err)
		}
		if f != nil {
			f.Close()
		}
	}

	// added to satisfy 'docker run' issue
	// TODO: figure out _why_
	if d.Fstype == "cgroup" {
		return nil
	}

	if err := syscall.Mount(
		d.Source,
		d.Target,
		d.Fstype,
		d.Flags,
		d.Data,
	); err != nil {
		return fmt.Errorf("mounting device: %w", err)
	}

	return nil
}

func mountDefaultDevices(rootfs string) error {
	return mountDevices(defaultDevices, rootfs)
}

func mountSpecDevices(devices []specs.LinuxDevice, rootfs string) error {
	for _, dev := range devices {
		absPath := filepath.Join(rootfs, strings.TrimPrefix(dev.Path, "/"))

		dt := map[string]uint32{
			"b": unix.S_IFBLK,
			"c": unix.S_IFCHR,
			"s": unix.S_IFSOCK,
			"p": unix.S_IFIFO,
		}

		if err := unix.Mknod(
			absPath,
			dt[dev.Type],
			int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
		); err != nil {
			return err
		}

		if err := syscall.Chmod(absPath, uint32(*dev.FileMode)); err != nil {
			return err
		}

		if dev.UID != nil && dev.GID != nil {
			if err := os.Chown(
				absPath,
				int(*dev.UID),
				int(*dev.GID),
			); err != nil {
				return err
			}
		}
	}

	return nil
}
