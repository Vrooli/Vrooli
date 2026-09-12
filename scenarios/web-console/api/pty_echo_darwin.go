//go:build darwin

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
	"web-console/session"
)

func (p *realPTY) TerminalEchoState() (session.EchoState, error) {
	return readPTYEchoFD(int(p.ptmx.Fd()))
}

func readPTYEchoFD(fd int) (session.EchoState, error) {
	attrs, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return session.EchoState{}, fmt.Errorf("read PTY echo state: %w", err)
	}
	return session.EchoState{Known: true, EchoEnabled: attrs.Lflag&unix.ECHO != 0}, nil
}

func readPTYEchoPath(path string) (session.EchoState, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOCTTY|unix.O_NONBLOCK, 0)
	if err != nil {
		return session.EchoState{}, fmt.Errorf("open pane tty: %w", err)
	}
	defer unix.Close(fd)
	return readPTYEchoFD(fd)
}
