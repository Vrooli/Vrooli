package services

import (
	"bytes"
	"context"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, dir string) (stdout, stderr string, exitCode int, err error)
}

// ExecCommandRunner runs real OS commands using os/exec.
type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args []string, dir string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}
