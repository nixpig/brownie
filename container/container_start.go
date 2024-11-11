package container

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Start() error {
	if c.Spec.Process == nil {
		c.SetStatus(specs.StateStopped)
		if err := c.HSave(); err != nil {
			return fmt.Errorf("(start 1) write state file: %w", err)
		}
		return nil
	}

	if !c.CanBeStarted() {
		return errors.New("container cannot be started in current state")
	}

	if err := c.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", filepath.Join(c.Bundle(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	if err := c.ExecHooks("prestart"); err != nil {
		c.SetStatus(specs.StateStopped)
		if err := c.HSave(); err != nil {
			return fmt.Errorf("(start 2) write state file: %w", err)
		}

		// TODO: run DELETE tasks here, then...

		if err := c.ExecHooks("poststop"); err != nil {
			fmt.Println("WARNING: failed to execute poststop hooks")
		}

		return errors.New("failed to run prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		c.SetStatus(specs.StateStopped)
		if err := c.HSave(); err != nil {
			return fmt.Errorf("(start 1) write state file: %w", err)
		}
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	return nil
}
