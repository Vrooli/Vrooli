//go:build linux

package securestore

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// nativeDefault uses libsecret's Secret Service command client. If no desktop
// keyring is available it fails closed, so a headless Linux target stays
// conditional instead of leaving Vault recovery material on disk.
func nativeDefault() Store {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return Absent("secret-tool (libsecret) is not installed")
	}
	return secretToolStore{}
}

type secretToolStore struct{}

func (secretToolStore) AdapterName() string { return "libsecret" }

// secretToolTimeout bounds every Secret Service call.
//
// Without it, a host whose keyring will not load has no timeout at all:
// gnome-keyring answers a write by raising an unlock prompt, and on a machine
// being reached over SSH or driven by a supervisor there is nobody to dismiss
// it. The call then blocks forever. That is not a hypothetical — it is how the
// keyring corruption this package now repairs presented itself: not as a failed
// credential write, but as a `vrooli` process that never returned.
//
// A failure here is recoverable and an operator can see it. A hang is neither,
// and it takes autoheal down with it, since a supervisor that blocks on a
// credential read cannot repair the thing that is blocking it.
const secretToolTimeout = 15 * time.Second

// secretToolCommand builds every Secret Service invocation, so all three
// operations reach the same bus and share one timeout. sessionEnviron repairs a
// session that names another user's bus; it returns nil when nothing needs
// repair, and a nil Env already means "inherit" to exec.
//
// The caller must invoke the returned cancel func once the command has
// finished.
func secretToolCommand(args ...string) (*exec.Cmd, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), secretToolTimeout)
	cmd := exec.CommandContext(ctx, "secret-tool", args...)
	cmd.Env = sessionEnviron()
	return cmd, ctx, cancel
}

// classifySecretToolTimeout turns the generic "signal: killed" that a context
// deadline produces into the one condition an operator can act on.
func classifySecretToolTimeout(stage string) error {
	return fmt.Errorf("%w: %s: the Secret Service did not answer within %s, which usually means it is waiting on an unlock prompt nobody can answer; run `vrooli credentials keyring inspect`",
		ErrUnavailable, stage, secretToolTimeout)
}

func (secretToolStore) Put(service, key, value string) error {
	cmd, ctx, cancel := secretToolCommand("store", "--label=Vrooli managed resource", "service", service, "key", key)
	defer cancel()
	// The value goes over stdin so it never appears in argv, /proc, or shell
	// history. Every adapter on every platform owes the same guarantee.
	cmd.Stdin = strings.NewReader(value)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return classifySecretToolTimeout("store secure resource material")
	}
	return classifySecretToolError("store secure resource material", err, string(output))
}

func (secretToolStore) Get(service, key string) (string, error) {
	cmd, ctx, cancel := secretToolCommand("lookup", "service", service, "key", key)
	defer cancel()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err == nil {
		return strings.TrimSuffix(string(stdout), "\n"), nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", classifySecretToolTimeout("read secure resource material")
	}
	if isSecretToolNotFound(err, len(stdout), stderr.String()) {
		return "", fmt.Errorf("%w: %s/%s", ErrNotFound, service, key)
	}
	return "", classifySecretToolError("read secure resource material", err, stderr.String())
}

func (secretToolStore) Delete(service, key string) error {
	cmd, ctx, cancel := secretToolCommand("clear", "service", service, "key", key)
	defer cancel()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return classifySecretToolTimeout("delete secure resource material")
	}
	// Removing what is already gone is the state the caller asked for, and
	// every adapter must agree on that. The shared conformance suite caught
	// this adapter disagreeing.
	if isSecretToolNotFound(err, len(stdout), stderr.String()) {
		return nil
	}
	return classifySecretToolError("delete secure resource material", err, stderr.String())
}

// isSecretToolNotFound recognizes the one way libsecret says "the Secret
// Service answered and holds no such item": exit status 1 with no output on
// either stream. A connection or protocol failure always writes to stderr, so
// the two cannot be confused — and conflating them is what made an unset API
// key abort a scenario start.
func isSecretToolNotFound(err error, stdoutLen int, stderr string) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) &&
		exitErr.ExitCode() == 1 &&
		stdoutLen == 0 &&
		strings.TrimSpace(stderr) == "" &&
		len(exitErr.Stderr) == 0
}

// classifySecretToolError keeps Put, Get, and Delete answering identically for
// identical host conditions. Before this, an uninstalled secret-tool produced
// ErrUnavailable from Delete and a bare exec error from Put and Get.
func classifySecretToolError(stage string, err error, output string) error {
	detail := strings.TrimSpace(output)
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: secret-tool (libsecret) is not installed", ErrAbsent)
	}
	message := stage + ": " + err.Error()
	if detail != "" {
		message += ": " + detail
	}
	if diagnosis := sessionDiagnosis(); diagnosis != "" {
		message += " (" + diagnosis + ")"
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, message)
}
