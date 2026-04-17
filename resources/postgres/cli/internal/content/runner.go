// Package content implements the `resource-postgres content` subcommand group.
package content

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"
)

// Runner executes a command in a PostgreSQL container. It abstracts `docker exec`
// so handlers can be unit-tested with an injected fake.
type Runner interface {
	Run(ctx context.Context, container string, args []string, stdin io.Reader, env []string) ([]byte, []byte, error)
}

// dockerRunner is the production Runner: it shells out to `docker exec -i`.
type dockerRunner struct {
	timeout time.Duration
}

// NewDockerRunner returns the default production runner with a sane timeout.
func NewDockerRunner() Runner {
	return &dockerRunner{timeout: 60 * time.Second}
}

func (d *dockerRunner) Run(ctx context.Context, container string, args []string, stdin io.Reader, env []string) ([]byte, []byte, error) {
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	cmdArgs := append([]string{"exec", "-i"}, dockerEnvFlags(env)...)
	cmdArgs = append(cmdArgs, container)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func dockerEnvFlags(env []string) []string {
	out := make([]string, 0, len(env)*2)
	for _, e := range env {
		out = append(out, "-e", e)
	}
	return out
}
