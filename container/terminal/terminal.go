package terminal

import (
	"fmt"
	"os"
	"syscall"

	"github.com/google/goterm/term"
	"golang.org/x/sys/unix"
)

type Pty struct {
	master *os.File
	slave  *os.File
}

func (p *Pty) New() (*Pty, error) {
	pty, err := term.OpenPTY()
	if err != nil {
		return nil, fmt.Errorf("open pty: %w", err)
	}

	master, slave := pty.Master, pty.Slave

	return &Pty{
		master: master,
		slave:  slave,
	}, nil
}

func (p *Pty) Connect() error {
	if _, err := unix.Setsid(); err != nil {
		return fmt.Errorf("setsid: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 0); err != nil {
		return fmt.Errorf("dup2 stdin: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 1); err != nil {
		return fmt.Errorf("dup2 stdout: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 2); err != nil {
		return fmt.Errorf("dup2 stderr: %w", err)
	}

	return nil
}

type PtySocket struct {
	SocketFd int
}

func (ps *PtySocket) New(consoleSocketPath string) (*PtySocket, error) {
	fd, err := syscall.Socket(
		syscall.AF_UNIX,
		syscall.SOCK_STREAM,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("create console socket: %w", err)
	}

	if err := syscall.Connect(
		fd,
		&syscall.SockaddrUnix{
			Name: consoleSocketPath,
		},
	); err != nil {
		return nil, fmt.Errorf("connect to console socket: %w", err)
	}

	return &PtySocket{
		SocketFd: fd,
	}, nil
}

func (ps *PtySocket) Close() error {
	if err := syscall.Close(ps.SocketFd); err != nil {
		return fmt.Errorf("close console socket: %w", err)
	}

	return nil
}

func (ps *PtySocket) SendPty(pty *Pty) error {
	masterFds := []int{int(pty.master.Fd())}
	cmsg := syscall.UnixRights(masterFds...)

	if err := syscall.Sendmsg(
		ps.SocketFd,
		[]byte("/dev/pts/ptmx"),
		cmsg,
		nil,
		0,
	); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	return nil
}
