// Package buildinfo exposes the node-agent's build fingerprint so the
// control plane can correlate a node's reported behaviour with a specific
// agent build (Phase 0 deliverable: build-fingerprint). The three vars are
// overridden at link time via -ldflags "-X ...="; when they are left at their
// zero value (e.g. a `go run` invocation) Fingerprint falls back to the VCS
// metadata Go embeds in the binary via runtime/debug.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// These are set at build time by the Makefile's -ldflags. Keep the variable
// names and the package path in lockstep with the Makefile's LDFLAGS.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Fingerprint returns a single, stable, human-and-log-friendly identifier for
// this agent build, e.g. "dev (a1b2c3d, 2026-06-18)". It is what the agent
// reports as Handshake.agent_version.
func Fingerprint() string {
	version := Version
	commit := Commit

	// Fall back to embedded VCS info when ldflags were not supplied (the
	// common case for `go run`/`go test`). This keeps the fingerprint useful
	// in development without forcing every invocation through the Makefile.
	if commit == "unknown" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				if s.Key == "vcs.revision" && s.Value != "" {
					commit = shortCommit(s.Value)
				}
			}
		}
	}

	var b strings.Builder
	b.WriteString(version)
	b.WriteString(" (")
	b.WriteString(commit)
	if Date != "unknown" {
		b.WriteString(", ")
		b.WriteString(Date)
	}
	b.WriteString(")")
	return b.String()
}

// shortCommit truncates a full git SHA to the conventional 7 chars, leaving
// shorter values untouched.
func shortCommit(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
