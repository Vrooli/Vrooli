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
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"
)

// Call describes a single kopia invocation: the argv passed to the kopia
// binary (without the binary name) and an env overlay merged over the process
// environment. Secrets travel in Env, never in Args.
type Call struct {
	Args []string
	Env  map[string]string
	// CaptureStderr includes stderr in successful output for commands whose
	// reviewable result is emitted by Kopia's logger rather than stdout.
	CaptureStderr bool
	// Stdin, when non-nil, is wired to the kopia process stdin.
	Stdin []byte
}

const (
	SFTPPrivateKeyEnv      = "VROOLI_KOPIA_SFTP_PRIVATE_KEY"
	SFTPKeyFilePlaceholder = "__VROOLI_KOPIA_SFTP_KEYFILE__"
)

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
	callEnv := make(map[string]string, len(c.Env))
	for key, value := range c.Env {
		callEnv[key] = value
	}
	args := append([]string(nil), c.Args...)
	if privateKey, ok := callEnv[SFTPPrivateKeyEnv]; ok {
		delete(callEnv, SFTPPrivateKeyEnv)
		needsKeyFile := false
		for _, arg := range args {
			if arg == SFTPKeyFilePlaceholder {
				needsKeyFile = true
				break
			}
		}
		if !needsKeyFile {
			privateKey = ""
		} // Saved Kopia config already embeds the key.
		if strings.TrimSpace(privateKey) == "" && needsKeyFile {
			return nil, fmt.Errorf("SFTP private key is empty")
		}
		if needsKeyFile {
			keyFile, err := os.CreateTemp("", "vrooli-kopia-sftp-key-*")
			if err != nil {
				return nil, fmt.Errorf("create private SFTP runtime key file: %w", err)
			}
			keyPath := keyFile.Name()
			defer os.Remove(keyPath)
			if err := keyFile.Chmod(0o600); err != nil {
				keyFile.Close()
				return nil, fmt.Errorf("protect private SFTP runtime key file: %w", err)
			}
			if _, err := keyFile.WriteString(privateKey); err != nil {
				keyFile.Close()
				return nil, fmt.Errorf("write private SFTP runtime key file: %w", err)
			}
			if err := keyFile.Close(); err != nil {
				return nil, fmt.Errorf("close private SFTP runtime key file: %w", err)
			}
			for i := range args {
				if args[i] == SFTPKeyFilePlaceholder {
					args[i] = filepath.Clean(keyPath)
				}
			}
		}
	} else {
		for _, arg := range args {
			if arg == SFTPKeyFilePlaceholder {
				return nil, fmt.Errorf("SFTP repository requires a credential-authority private key")
			}
		}
	}
	binary := strings.TrimSpace(r.BinaryPath)
	if binary == "" {
		binary = "kopia"
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, mapEnv(callEnv))
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
			return stdout.Bytes(), fmt.Errorf("kopia %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return stdout.Bytes(), fmt.Errorf("kopia %s: %w", strings.Join(args, " "), err)
	}
	if c.CaptureStderr {
		return append(stdout.Bytes(), stderr.Bytes()...), nil
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
