// Package langrecover provides per-language recovery strategies that heal
// the most common "scenario fails to (re)start because of dependency drift"
// failure classes. Strategies are signature-gated: they only run when the
// prior failure output matches a known healable pattern.
//
// [REQ:HEAL-ACTION-001]
package langrecover

import (
	"context"
	"errors"
)

// ErrRunnerRequired is returned by a strategy invoked without a Runner. The
// production Runner is NewRunner over the caller's toolchain table; there is
// no implicit default because only the caller can resolve the runtime-home
// bin directory the table needs.
var ErrRunnerRequired = errors.New("langrecover: a Runner is required")

// Kind identifies which language-level recovery strategy applies.
type Kind string

const (
	KindGo       Kind = "go"
	KindPnpm     Kind = "pnpm"
	KindRepoRoot Kind = "repo-root"
)

// Result is the outcome of a single recovery strategy invocation.
type Result struct {
	// Kind is the strategy that ran.
	Kind Kind
	// Command is a human-readable description of what was executed
	// (e.g. "go mod tidy" or "pnpm install --ignore-workspace --no-frozen-lockfile").
	Command string
	// WorkingDir is the directory the recovery command ran in
	// (relative to repo root).
	WorkingDir string
	// Output is the combined stdout/stderr of the recovery command, capped.
	Output string
	// ModifiedTrackedFiles is true when the recovery action changed at least
	// one file that git already tracks (e.g. go.mod, go.sum, pnpm-lock.yaml).
	// Surfaced to the caller so the autoheal incident log makes the silent
	// mutation visible.
	ModifiedTrackedFiles bool
	// ModifiedPaths lists the tracked paths that changed, relative to the
	// scenario directory.
	ModifiedPaths []string
	// VersionDeltas records modules whose selected version changed as a side
	// effect of the recovery. Populated for Go strategies, where `go mod tidy`
	// re-runs minimal version selection and can bump a direct dependency the
	// operator never asked to move. Empty for strategies that cannot do this.
	VersionDeltas []VersionDelta
	// Err is the recovery command's error, if any.
	Err error
}

// Runner runs a command in a specific working directory and returns the
// combined stdout/stderr plus any error. Strategies depend on this seam so
// tests can inject a fake without spawning real subprocesses.
type Runner func(ctx context.Context, dir, name string, args ...string) ([]byte, error)
