package execution

import (
	"bytes"
	"context"
	"os/exec"
)

// Command describes one lint tool invocation.
type Command struct {
	Dir  string
	Env  []string
	Name string
	Args []string
}

// Result captures process output and exit status.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner executes lint-tool commands.
type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

// ProductionRunner executes commands through os/exec.
type ProductionRunner struct{}

// Run executes a command and preserves non-zero exit codes in Result.
func (ProductionRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	argv := append([]string(nil), cmd.Args...)
	execCmd := exec.CommandContext(ctx, cmd.Name, argv...)
	execCmd.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		execCmd.Env = append([]string(nil), cmd.Env...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return Result{
				Stdout:   stdout.Bytes(),
				Stderr:   stderr.Bytes(),
				ExitCode: exitErr.ExitCode(),
			}, nil
		}
		return Result{}, err
	}

	return Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}, nil
}
