package main

import (
	"web-console/internal/backend"
)

// InitDefaultRegistry creates a backend registry with the standard and
// persistent (tmux) backends registered.
//
// This wiring lives in package main because the factories (defaultPTYFactory,
// tmuxPTYFactory) operate on PTY types that the backend package intentionally
// does not depend on.
func InitDefaultRegistry() *backend.Registry {
	reg := backend.New()

	reg.Register(backend.Descriptor{
		ID:              backend.Standard,
		DisplayName:     "Standard",
		Description:     "In-memory session. Fast and lightweight, but lost on restart.",
		SurvivesRestart: false,
		Available:       true,
	}, defaultPTYFactory)

	tmuxAvail, tmuxReason := backend.CheckTmuxAvailable()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		DisplayName:     "Persistent",
		Description:     "Backed by tmux. Survives web console restarts.",
		SurvivesRestart: true,
		Available:       tmuxAvail,
		Reason:          tmuxReason,
	}, tmuxPTYFactory)

	return reg
}
