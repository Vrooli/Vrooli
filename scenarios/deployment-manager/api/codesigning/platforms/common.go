package platforms

import (
	"bytes"
	"context"
	"os/exec"

	"deployment-manager/codesigning"
)

// realCommandRunner implements codesigning.CommandRunner using os/exec.
type realCommandRunner struct{}

func newRealCommandRunner() codesigning.CommandRunner {
	return &realCommandRunner{}
}

func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func (r *realCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
