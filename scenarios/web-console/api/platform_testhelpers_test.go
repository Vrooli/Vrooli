package main

import (
	"runtime"
	"testing"

	"web-console/internal/backend"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// requireLocalPTY declares that a test exercises the host PTY implementation.
// The declaration is deliberately centralized so platform skips remain typed,
// consistently recorded, and easy for the greenfield contract test to audit.
func requireLocalPTY(t *testing.T) {
	t.Helper()
	if !localPTYAvailable() {
		repocontracttest.SkipPlatform(t, "local PTY is unavailable on this platform")
	}
}

// requireTmux declares that a test talks to a real tmux server. Use the
// per-test socket after the capability check so integration tests cannot
// touch an operator's default tmux server.
func requireTmux(t *testing.T) {
	t.Helper()
	available, reason := backend.CheckTmuxAvailable()
	if !available {
		repocontracttest.SkipPlatform(t, reason)
	}
	requireLocalPTY(t)
	requireIsolatedTmux(t)
}

// requireUnixTools declares tests that invoke POSIX-only host tools directly.
// Keep this separate from the PTY and tmux declarations so a test's reason for
// skipping remains precise when the platform seam is audited.
func requireUnixTools(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "POSIX host tools are unavailable on Windows")
	}
}
