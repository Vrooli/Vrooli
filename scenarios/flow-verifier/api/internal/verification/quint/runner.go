package quint

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	Args []string
	Dir  string
}

type Result struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, command Command) (Result, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command Command) (Result, error) {
	if len(command.Args) == 0 {
		return Result{}, fmt.Errorf("command args are required")
	}
	cmd := exec.CommandContext(ctx, command.Args[0], command.Args[1:]...)
	cmd.Dir = command.Dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		return result, fmt.Errorf("command failed: %s\n%s", strings.Join(command.Args, " "), strings.TrimSpace(result.Stdout+"\n"+result.Stderr))
	}
	return result, nil
}
