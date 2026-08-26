//go:build !linux

package main

import "web-console/session"

func (p *realPTY) TerminalEchoState() (session.EchoState, error) {
	return session.EchoState{}, session.ErrEchoStateUnsupported
}
