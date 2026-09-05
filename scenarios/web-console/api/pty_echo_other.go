//go:build !linux && !darwin && !windows

package main

import "web-console/session"

func (p *realPTY) TerminalEchoState() (session.EchoState, error) {
	return session.EchoState{}, session.ErrEchoStateUnsupported
}

func readPTYEchoPath(string) (session.EchoState, error) {
	return session.EchoState{}, session.ErrEchoStateUnsupported
}
