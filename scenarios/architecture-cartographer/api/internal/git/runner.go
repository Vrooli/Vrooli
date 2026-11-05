// Package git is the seam between cartographer and the local `git`
// binary. Used by the git-co-edit signal (and later apply, when it
// commits changes). Production wires RealRunner; tests pass mocks.FakeRunner.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner is the seam interface. The narrow surface is intentional —
// only the subcommands cartographer actually invokes appear here.
type Runner interface {
	// IsAvailable returns true when `git` is on PATH and a repository
	// is reachable from the cwd.
	IsAvailable(ctx context.Context) bool

	// Log invokes `git log` with the given args and returns its
	// stdout. Used by git-co-edit to read co-edit pairs.
	Log(ctx context.Context, args ...string) (string, error)
}

// RealRunner is the production Runner. Shells out to `git` via os/exec.
type RealRunner struct{}

// NewRealRunner returns the production Runner.
func NewRealRunner() *RealRunner { return &RealRunner{} }

func (RealRunner) IsAvailable(ctx context.Context) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (RealRunner) Log(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"log"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	return string(out), nil
}

var _ Runner = (*RealRunner)(nil)
