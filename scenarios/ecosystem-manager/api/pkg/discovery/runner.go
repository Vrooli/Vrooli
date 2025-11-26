package discovery

import (
	"context"
	"os/exec"
	"time"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type defaultCommandRunner struct{}

func (defaultCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

var execRunner commandRunner = defaultCommandRunner{}

var commandTimeout = 8 * time.Second
