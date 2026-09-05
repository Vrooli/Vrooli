package validation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/api-core/metrics"
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

// ProcessCommander is the optional process-aware extension. Commander stays
// unchanged so existing scanner doubles remain source-compatible.
type ProcessCommander interface {
	Commander
	RunProcess(ctx context.Context, dir, name string, args ...string) (stdout, stderr []byte, exitCode int, state *os.ProcessState, err error)
}

// EnvironmentCommander is the optional extension used when a scanner needs a
// bounded child environment. Commander remains source-compatible with existing
// test doubles and callers that do not need overlays.
type EnvironmentCommander interface {
	ProcessCommander
	RunProcessWithEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) (stdout, stderr []byte, exitCode int, state *os.ProcessState, err error)
}

func runCommand(ctx context.Context, cmd Commander, dir, name string, args ...string) ([]byte, []byte, int, *os.ProcessState, error) {
	return runCommandWithEnv(ctx, cmd, dir, nil, name, args...)
}

func runCommandWithEnv(ctx context.Context, cmd Commander, dir string, env map[string]string, name string, args ...string) ([]byte, []byte, int, *os.ProcessState, error) {
	if len(env) > 0 {
		if environmentCmd, ok := cmd.(EnvironmentCommander); ok {
			stdout, stderr, exitCode, state, err := environmentCmd.RunProcessWithEnv(ctx, dir, env, name, args...)
			metrics.ObserveProcess(ctx, state)
			return stdout, stderr, exitCode, state, err
		}
	}
	if processCmd, ok := cmd.(ProcessCommander); ok {
		stdout, stderr, exitCode, state, err := processCmd.RunProcess(ctx, dir, name, args...)
		metrics.ObserveProcess(ctx, state)
		return stdout, stderr, exitCode, state, err
	}
	stdout, stderr, exitCode, err := cmd.Run(ctx, dir, name, args...)
	return stdout, stderr, exitCode, nil, err
}

// execCommander is the real os/exec-backed Commander.
type execCommander struct{}

// NewExecCommander returns the production Commander.
func NewExecCommander() Commander { return execCommander{} }

func (execCommander) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (execCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, error) {
	stdout, stderr, exitCode, _, err := execCommander{}.RunProcess(ctx, dir, name, args...)
	return stdout, stderr, exitCode, err
}

func (execCommander) RunProcess(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, *os.ProcessState, error) {
	return execCommander{}.RunProcessWithEnv(ctx, dir, nil, name, args...)
}

func (execCommander) RunProcessWithEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) ([]byte, []byte, int, *os.ProcessState, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range env {
			prefix := key + "="
			replaced := false
			for i, entry := range cmd.Env {
				if strings.HasPrefix(entry, prefix) {
					cmd.Env[i] = prefix + value
					replaced = true
					break
				}
			}
			if !replaced {
				cmd.Env = append(cmd.Env, prefix+value)
			}
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// A non-zero exit is expected (scanners signal findings that way) and
		// is NOT a runner failure — surface it via exitCode with err==nil.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout.Bytes(), stderr.Bytes(), exitErr.ExitCode(), cmd.ProcessState, nil
		}
		// Genuine start/cancel failure.
		return stdout.Bytes(), stderr.Bytes(), -1, cmd.ProcessState, err
	}
	return stdout.Bytes(), stderr.Bytes(), 0, cmd.ProcessState, nil
}
