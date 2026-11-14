package ssh

import (
	"context"
	"os/exec"
	"strings"
)

// CommandRunner abstracts local command execution (e.g. ssh-keygen).
// The default implementation uses os/exec; tests can substitute a fake.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// ExecCommandRunner implements CommandRunner using os/exec.
type ExecCommandRunner struct{}

// Run executes the named command with the given arguments.
func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.Output()
	var stderr []byte
	if ee, ok := err.(*exec.ExitError); ok {
		stderr = ee.Stderr
	}
	return stdout, stderr, err
}

// quoteSingle quotes a string for safe use in single-quoted shell contexts.
// It is the standard way to quote strings passed as remote SSH command
// arguments. Duplicated from scenario-to-cloud's shellutil (kept local per the
// duplicate-before-extract rule).
func quoteSingle(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
