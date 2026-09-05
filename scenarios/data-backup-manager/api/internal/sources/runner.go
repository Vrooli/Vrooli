package sources

import (
	"context"
	"fmt"
	"os/exec"
)

// CommandRunner is the process-exec boundary for source resource CLIs.
// Production wires ExecRunner; tests wire mocks.FakeCommandRunner to record
// calls and assert argv shape (notably: no secret ever appears in argv).
//
// seam: CommandRunner is the process-exec boundary for source resource CLIs.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner is the production CommandRunner. It shells out to the named
// binary via os/exec and returns stdout. A non-zero exit code surfaces as
// an error that wraps stderr so callers can record a failed run without
// losing context.
type ExecRunner struct{}

// Compile-time guarantee.
var _ CommandRunner = ExecRunner{}

// Run executes name with args and returns stdout. On a non-zero exit the
// stderr is appended to the error message for diagnostics.
func (r ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if ee, ok := err.(*exec.ExitError); ok {
			exitErr = ee
			return nil, fmt.Errorf("%s %v: %w: %s", name, args, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}
