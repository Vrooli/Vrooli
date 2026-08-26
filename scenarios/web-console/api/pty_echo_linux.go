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
	attrs, err := unix.IoctlGetTermios(int(p.ptmx.Fd()), unix.TCGETS)
	if err != nil {
		return session.EchoState{}, fmt.Errorf("read PTY echo state: %w", err)
	}
	return session.EchoState{Known: true, EchoEnabled: attrs.Lflag&unix.ECHO != 0}, nil
}
