package resolver

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Credentials struct {
	Username string
	Password string
}

type SecretResolver interface {
	ResolveAdGuardCredentials(ctx context.Context, cfg BackendConfig) (Credentials, error)
}

var ErrSecretMissing = errors.New("adguard home credential secret is missing")

type VaultSecretResolver struct {
	Command  string
	Run      func(ctx context.Context, name string, args ...string) (string, error)
	LookPath func(string) (string, error)
}

func NewVaultSecretResolver() *VaultSecretResolver {
	return &VaultSecretResolver{
		Command:  "resource-vault",
		Run:      runVaultCommand,
		LookPath: exec.LookPath,
	}
}

func (r *VaultSecretResolver) ResolveAdGuardCredentials(ctx context.Context, cfg BackendConfig) (Credentials, error) {
	path, err := normalizeSecretRef(cfg.TokenRef)
	if err != nil {
		return Credentials{}, err
	}

	password, found, err := r.getSecret(ctx, path, "password")
	if err != nil {
		return Credentials{}, fmt.Errorf("read AdGuard Home password from secret reference: %w", err)
	}
	if !found {
		return Credentials{}, fmt.Errorf("%w: %s/password", ErrSecretMissing, path)
	}

	username := strings.TrimSpace(cfg.Username)
	if username == "" {
		if value, found, err := r.getSecret(ctx, path, "username"); err != nil {
			return Credentials{}, fmt.Errorf("read AdGuard Home username from secret reference: %w", err)
		} else if found {
			username = value
		}
	}
	if username == "" {
		username = "admin"
	}

	return Credentials{Username: username, Password: password}, nil
}

func (r *VaultSecretResolver) getSecret(ctx context.Context, path, key string) (string, bool, error) {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = "resource-vault"
	}
	if r.LookPath != nil {
		if _, err := r.LookPath(command); err != nil {
			return "", false, fmt.Errorf("vault CLI %q unavailable: %w", command, err)
		}
	}
	run := r.Run
	if run == nil {
		run = runVaultCommand
	}
	value, err := run(ctx, command, "content", "get", "--path", path, "--key", key, "--format", "raw")
	if err != nil {
		if isMissingSecretError(err) {
			return "", false, nil
		}
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

type vaultCLIError struct {
	Command string
	Args    []string
	Stderr  string
	Err     error
}

func (e *vaultCLIError) Error() string {
	detail := strings.TrimSpace(e.Stderr)
	if detail == "" {
		return fmt.Sprintf("%s %s: %v", e.Command, strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("%s %s: %v: %s", e.Command, strings.Join(e.Args, " "), e.Err, detail)
}

func (e *vaultCLIError) Unwrap() error {
	return e.Err
}

func runVaultCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		cliErr := &vaultCLIError{Command: name, Args: append([]string(nil), args...), Err: err}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			cliErr.Stderr = string(exitErr.Stderr)
		}
		return "", cliErr
	}
	return strings.TrimSpace(string(out)), nil
}

func normalizeSecretRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("token_ref is required")
	}
	ref = strings.TrimPrefix(ref, "secret://")
	if !strings.HasPrefix(ref, "secret/") {
		return "", fmt.Errorf("unsupported secret reference %q; expected secret/resources/...", redactedRef(ref))
	}
	return ref, nil
}

func isMissingSecretError(err error) bool {
	var cliErr *vaultCLIError
	if !errors.As(err, &cliErr) {
		return false
	}
	msg := strings.ToLower(cliErr.Stderr)
	return strings.Contains(msg, "no value found") ||
		strings.Contains(msg, "no secret exists") ||
		strings.Contains(msg, "not found")
}

func redactedRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if len(ref) <= 12 {
		return "[redacted]"
	}
	return ref[:6] + "..." + ref[len(ref)-4:]
}
