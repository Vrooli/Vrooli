package validation

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Commander is the process-execution seam every Scanner runs through. It is an
// interface so tests can stub scanner output without installing the real
// binaries (gitleaks, gosec, …). The production implementation is execCommander.
//
// Run deliberately separates a *failure to start* (binary missing at runtime,
// context cancelled — returned as err) from a *non-zero exit* (many scanners
// exit 1 when they find issues — reported via exitCode with err==nil). Scanners
// parse stdout regardless of exitCode.
type Commander interface {
	// LookPath resolves a binary on PATH, mirroring exec.LookPath.
	LookPath(name string) (string, error)
	// Run executes name with args in dir, returning captured stdout/stderr and
	// the process exit code. err is non-nil only when the process could not be
	// started or was cancelled — never for a clean non-zero exit.
	Run(ctx context.Context, dir, name string, args ...string) (stdout, stderr []byte, exitCode int, err error)
}

// execCommander is the real os/exec-backed Commander.
type execCommander struct{}

// NewExecCommander returns the production Commander.
func NewExecCommander() Commander { return execCommander{} }

func (execCommander) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (execCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// A non-zero exit is expected (scanners signal findings that way) and
		// is NOT a runner failure — surface it via exitCode with err==nil.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout.Bytes(), stderr.Bytes(), exitErr.ExitCode(), nil
		}
		// Genuine start/cancel failure.
		return stdout.Bytes(), stderr.Bytes(), -1, err
	}
	return stdout.Bytes(), stderr.Bytes(), 0, nil
}
