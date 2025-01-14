package container

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func (c *Container) Kill(sig unix.Signal) error {
	if !c.CanBeKilled() {
		return fmt.Errorf("container cannot be killed in current state (%s)", c.Status())
	}

	if err := syscall.Kill(c.PID(), sig); err != nil {
		return fmt.Errorf("failed to execute kill syscall (process: %d): %w", c.PID(), err)
	}

	c.SetStatus(specs.StateStopped)
	if err := c.Save(); err != nil {
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	if err := c.ExecHooks("poststop"); err != nil {
		fmt.Println("Warning: failed to execute poststop hooks")
	}

	return nil
}
