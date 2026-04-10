package ssh

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts local command execution (e.g., ssh-keygen).
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
