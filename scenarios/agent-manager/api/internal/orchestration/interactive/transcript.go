// This file exposes transcript access for interactive sessions.
package interactive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// DiscoverParams is the input to findTranscript.
type DiscoverParams struct {
	RunnerType domain.RunnerType
	WorkingDir string
	// RunDir is the agent-manager-owned per-run dir holding the relocated
	// CODEX_HOME/GROK_HOME (ignored for claude).
	RunDir string
	// LaunchedAt bounds the claude discovery to files created at/after launch,
	// so a stale transcript from a previous session in the same shared cwd slug
	// is not picked up (design §3 / risk R3).
	LaunchedAt time.Time
	// HomeDir overrides the user's home directory for claude projects discovery
	// (tests point it at a fixture tree). Empty uses os.UserHomeDir.
	HomeDir string
}

// findTranscript returns the discovered agent-owned transcript path, or "" when
// no matching transcript exists yet (the CLI may not have written it — callers
// poll), or an error only on an unexpected filesystem failure or unsupported
// runner. Discovery rules are per design §3.
func findTranscript(p DiscoverParams) (string, error) {
	spec, ok := specFor(p.RunnerType)
	if !ok {
		return "", fmt.Errorf("interactive transcript discovery unsupported for runner %q", p.RunnerType)
	}

	switch p.RunnerType {
	case domain.RunnerTypeCodex:
		// $CODEX_HOME/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl — run-scoped
		// home means at most one rollout, so newest under the fixed depth wins.
		home := homeDirFor(spec, p.RunDir)
		return newestGlob(filepath.Join(home, "sessions", "*", "*", "*", "rollout-*.jsonl"))
	case domain.RunnerTypeGrok:
		// $GROK_HOME/sessions/<url-encoded-cwd>/<session>/updates.jsonl —
		// run-scoped home, so newest under the fixed depth wins.
		home := homeDirFor(spec, p.RunDir)
		return newestGlob(filepath.Join(home, "sessions", "*", "*", "updates.jsonl"))
	case domain.RunnerTypeClaudeCode:
		// ~/.claude/projects/<cwd-slug>/<session>.jsonl on the SHARED projects
		// tree: bound to files modified at/after launch, newest wins (design §3).
		home := p.HomeDir
		if home == "" {
			h, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home dir for claude transcript discovery: %w", err)
			}
			home = h
		}
		slug := SlugifyCwd(p.WorkingDir)
		pattern := filepath.Join(home, ".claude", "projects", slug, "*.jsonl")
		return newestGlobAfter(pattern, p.LaunchedAt)
	default:
		return "", fmt.Errorf("interactive transcript discovery unsupported for runner %q", p.RunnerType)
	}
}

// SlugifyCwd converts an absolute cwd to claude's projects-dir slug: every
// non-alphanumeric character becomes '-'. Verified: /home/matthalloran8/Vrooli
// → -home-matthalloran8-Vrooli (design §3).
func SlugifyCwd(cwd string) string {
	var b strings.Builder
	b.Grow(len(cwd))
	for _, r := range cwd {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// newestGlob returns the glob match with the newest mtime, or "" when none
// match. A glob syntax error is returned as an error.
func newestGlob(pattern string) (string, error) {
	return newestGlobAfter(pattern, time.Time{})
}

// newestGlobAfter returns the newest-mtime glob match whose mtime is at/after
// notBefore, or "" when none qualify. A zero notBefore accepts any file.
func newestGlobAfter(pattern string, notBefore time.Time) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob %q: %w", pattern, err)
	}
	// Small negative skew so a transcript created a moment before the recorded
	// launch timestamp (clock granularity) is not excluded.
	cutoff := notBefore
	if !cutoff.IsZero() {
		cutoff = cutoff.Add(-2 * time.Second)
	}
	best := ""
	var bestMod time.Time
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best = m
			bestMod = info.ModTime()
		}
	}
	return best, nil
}
