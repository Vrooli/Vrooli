package capture

import (
	"github.com/vrooli/browser-automation-studio/services/readiness"
)

// The readiness contract is shared with the workflow executor, which settles a
// page between steps for the same reason capture settles one before a snapshot.
// The implementation lives in services/readiness so both callers resolve the
// same profile the same way; this file is the capture-side adapter.

// NewReadinessProfileResolver constructs the production resolver for the
// Experience Manager-owned profile RPC.
func NewReadinessProfileResolver() ReadinessProfileResolver {
	return readiness.NewProfileResolver()
}

func containsRoute(routes []string, route string) bool {
	return readiness.ContainsRoute(routes, route)
}

func terminalReadinessSelector(binding, kind string, states []string) string {
	return readiness.TerminalSelector(binding, kind, states)
}
