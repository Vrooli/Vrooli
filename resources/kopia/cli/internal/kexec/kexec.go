// Package kexec is the single seam through which the resource shells out to the
// kopia binary. Every command handler routes through Runner; no other package
// is permitted to exec kopia directly. This is the one place a fake is swapped
// in for unit tests (mocks.FakeRunner), and the one place that injects the
// secret environment (KOPIA_PASSWORD, AWS_*) sourced from the credential authority.
package kexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/envkit-go"
)

// Call describes a single kopia invocation: the argv passed to the kopia
// binary (without the binary name) and an env overlay merged over the process
// environment. Secrets travel in Env, never in Args.
type Call struct {
	Args []string
	Env  map[string]string
	// Stdin, when non-nil, is wired to the kopia process stdin.
	Stdin []byte
}

// Runner executes kopia invocations.
//
// seam: Runner executes the kopia binary. Production wires *BinaryRunner from
// this package; unit tests wire mocks.FakeRunner from kexec/mocks.
type Runner interface {
	Run(ctx context.Context, c Call) ([]byte, error)
}

// BinaryRunner is the production Runner. It resolves the kopia binary path once
// and execs it for each call.
type BinaryRunner struct {
	// BinaryPath is the resolved path to the kopia binary (e.g. /usr/local/bin/kopia).
	BinaryPath string
}

var _ Runner = (*BinaryRunner)(nil)

// NewBinaryRunner returns a BinaryRunner bound to the given kopia binary path.
func NewBinaryRunner(binaryPath string) *BinaryRunner {
	return &BinaryRunner{BinaryPath: binaryPath}
}

// Run executes `kopia <args...>` with the env overlay applied. On a non-zero
// exit, stderr is folded into the returned error so callers never have to read
// the binary's stderr stream directly.
func (r *BinaryRunner) Run(ctx context.Context, c Call) ([]byte, error) {
	binary := strings.TrimSpace(r.BinaryPath)
	if binary == "" {
		binary = "kopia"
	}
	cmd := exec.CommandContext(ctx, binary, c.Args...)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, mapEnv(c.Env))
	if len(c.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(c.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return stdout.Bytes(), fmt.Errorf("kopia %s: %w: %s", strings.Join(c.Args, " "), err, msg)
		}
		return stdout.Bytes(), fmt.Errorf("kopia %s: %w", strings.Join(c.Args, " "), err)
	}
	return stdout.Bytes(), nil
}

func mapEnv(values map[string]string) envkit.Env {
	entries := make(envkit.Env, 0, len(values))
	for key, value := range values {
		entries = append(entries, key+"="+value)
	}
	return entries
}
