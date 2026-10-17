//go:build windows

package main

import (
	"context"
	"errors"
	"os/exec"

	"web-console/internal/pty"
	"web-console/session"
)

var errPTYClosed = errors.New("pty is closed")

const (
	tmuxSessionPrefix     = "wc-"
	maxKeystrokeArgvBytes = 8 * 1024
)

type tmuxPTY struct{ sessionName string }

func (p *tmuxPTY) Read([]byte) (int, error)                   { return 0, errPTYClosed }
func (p *tmuxPTY) WriteInput([]byte, pty.InputKind) error     { return pty.ErrUnsupported }
func (p *tmuxPTY) SetSize(uint16, uint16) error               { return pty.ErrUnsupported }
func (p *tmuxPTY) Close() error                               { return nil }
func (p *tmuxPTY) Kill() error                                { return nil }
func (p *tmuxPTY) ExitCode() int                              { return -1 }
func (p *tmuxPTY) ProbeReady(context.Context) error           { return pty.ErrUnsupported }
func (p *tmuxPTY) CurrentDir(context.Context) (string, error) { return "", pty.ErrUnsupported }
func (p *tmuxPTY) TerminalEchoState() (session.EchoState, error) {
	return session.EchoState{}, session.ErrEchoStateUnsupported
}
func (p *tmuxPTY) SetMouseMode(bool) error        { return pty.ErrUnsupported }
func (p *tmuxPTY) MouseMode() (bool, error)       { return false, pty.ErrUnsupported }
func (p *tmuxPTY) Scroll(int) error               { return pty.ErrUnsupported }
func (p *tmuxPTY) PaneInAltScreen() (bool, error) { return false, pty.ErrUnsupported }

func tmuxPTYFactory(pty.LaunchSpec) (pty.PTY, error) { return nil, pty.ErrUnsupported }
func tmuxAttach(string) (*tmuxPTY, error)            { return nil, pty.ErrUnsupported }
func tmuxAttachAsPTY(string) (pty.PTY, error)        { return nil, pty.ErrUnsupported }
func DiscoverTmuxSessions() ([]string, error)        { return nil, nil }
func refreshTmuxOptions(string)                      {}
func tmuxCmd(args ...string) *exec.Cmd               { return exec.Command("tmux", args...) }
func applyTmuxOptions(string, bool)                  {}
func tmuxMouseModeValue(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func buildTmuxNewSessionArgs(sessionName, workingDir string, spec pty.LaunchSpec) []string {
	return []string{"new-session", "-d", "-s", sessionName, "-c", workingDir, "-x", "0", "-y", "0", spec.Shell}
}
