//go:build linux

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
	"web-console/session"
)

// TerminalEchoState reads the terminal's current local echo bit from the
// master PTY. Failure is fail-closed; callers must not predict unknown input.
func (p *realPTY) TerminalEchoState() (session.EchoState, error) {
	return readPTYEchoFD(int(p.ptmx.Fd()))
}

func readPTYEchoFD(fd int) (session.EchoState, error) {
	attrs, err := unix.IoctlGetTermios(fd, unix.TCGETS)
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
