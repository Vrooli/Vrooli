// Package pty is the web-console compatibility surface for the shared session
// contract. Concrete local and tmux implementations remain in package main,
// while every consumer now shares one transport-neutral interface.
package pty

import sessioncore "github.com/vrooli/vrooli/packages/session-core"

type InputKind = sessioncore.InputKind

const (
	KindKeystroke = sessioncore.KindKeystroke
	KindPaste     = sessioncore.KindPaste
)

type (
	PTY        = sessioncore.PTY
	LaunchSpec = sessioncore.LaunchSpec
	Factory    = sessioncore.Factory
)
