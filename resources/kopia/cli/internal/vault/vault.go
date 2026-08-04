// Package vault is the seam through which the resource sources every secret —
// repository encryption passphrases and S3 credentials. It NEVER touches
// Vault's HTTP API and NEVER reads a plaintext credential file: the production
// implementation shells the resource-vault CLI (wrap-not-use). Per-repo dynamic
// secrets (passphrase, S3 keys keyed by repo name) are read and written through
// the resource-vault content surface.
package vault

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Vault reads and writes secrets.
//
// seam: Vault sources secrets via the resource-vault CLI. Production wires
// *CLIVault from this package; unit tests wire mocks.FakeVault from vault/mocks.
type Vault interface {
	// GetSecret returns the value at (path, key). found is false when the
	// secret is absent (distinct from an error talking to vault).
	GetSecret(ctx context.Context, path, key string) (value string, found bool, err error)
	// PutSecret stores value at (path, key).
	PutSecret(ctx context.Context, path, key, value string) error
	// DeleteSecret removes the secret at path. Missing paths are ignored by the
	// vault resource CLI.
	DeleteSecret(ctx context.Context, path string) error
}

const (
	accessKeyKey = "access_key"
	secretKeyKey = "secret_key"
)

// S3Path is the vault path holding a repository's S3 credentials.
func S3Path(repo string) string {
	return "secret/resources/kopia/s3/" + repo
}

// S3Credentials holds an S3/MinIO access/secret key pair.
type S3Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// Valid reports whether both halves of the credential are present.
func (c S3Credentials) Valid() bool {
	return strings.TrimSpace(c.AccessKeyID) != "" && strings.TrimSpace(c.SecretAccessKey) != ""
}

// PutS3Credentials stores an S3 access/secret key pair for a repository.
func PutS3Credentials(ctx context.Context, v Vault, repo string, creds S3Credentials) error {
	path := S3Path(repo)
	if err := v.PutSecret(ctx, path, accessKeyKey, creds.AccessKeyID); err != nil {
		return fmt.Errorf("store s3 access key for repo %q: %w", repo, err)
	}
	if err := v.PutSecret(ctx, path, secretKeyKey, creds.SecretAccessKey); err != nil {
		return fmt.Errorf("store s3 secret key for repo %q: %w", repo, err)
	}
	return nil
}

// DeleteS3Credentials removes the Vault path owned by an S3 repository.
func DeleteS3Credentials(ctx context.Context, v Vault, repo string) error {
	if err := v.DeleteSecret(ctx, S3Path(repo)); err != nil {
		return fmt.Errorf("delete s3 credentials for repo %q: %w", repo, err)
	}
	return nil
}

// S3CredentialsFor returns the stored S3 credentials for a repository. found is
// false when no S3 credentials have been stored (a filesystem repo, say).
func S3CredentialsFor(ctx context.Context, v Vault, repo string) (S3Credentials, bool, error) {
	path := S3Path(repo)
	access, accessFound, err := v.GetSecret(ctx, path, accessKeyKey)
	if err != nil {
		return S3Credentials{}, false, fmt.Errorf("read s3 access key for repo %q: %w", repo, err)
	}
	secret, secretFound, err := v.GetSecret(ctx, path, secretKeyKey)
	if err != nil {
		return S3Credentials{}, false, fmt.Errorf("read s3 secret key for repo %q: %w", repo, err)
	}
	if !accessFound || !secretFound {
		return S3Credentials{}, false, nil
	}
	return S3Credentials{AccessKeyID: access, SecretAccessKey: secret}, true, nil
}

// CLIVault is the production Vault. It shells the resource-vault CLI for both
// reads and writes, satisfying wrap-not-use: the only external surface touched
// is resource-vault, never Vault's HTTP API.
type CLIVault struct {
	// Command is the vault resource CLI command (default "resource-vault").
	Command string
	// Run executes a command and returns trimmed stdout. Overridable in tests.
	Run func(ctx context.Context, name string, args ...string) (string, error)
	// LookPath verifies the vault CLI is present. Overridable in tests.
	LookPath func(string) (string, error)
}

var _ Vault = (*CLIVault)(nil)

// CLIError preserves resource-vault stderr so callers can distinguish a
// missing secret from an unavailable Vault/container.
type CLIError struct {
	Command string
	Args    []string
	Stderr  string
	Err     error
}

func (e *CLIError) Error() string {
	if strings.TrimSpace(e.Stderr) == "" {
		return fmt.Sprintf("%s %s: %v", e.Command, strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("%s %s: %v: %s", e.Command, strings.Join(e.Args, " "), e.Err, strings.TrimSpace(e.Stderr))
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// NewCLIVault returns a CLIVault wired to the standard resource-vault CLI.
func NewCLIVault() *CLIVault {
	return &CLIVault{
		Command:  "resource-vault",
		Run:      runCommand,
		LookPath: exec.LookPath,
	}
}

func (c *CLIVault) command() string {
	if strings.TrimSpace(c.Command) == "" {
		return "resource-vault"
	}
	return c.Command
}

// GetSecret reads (path, key) via `resource-vault content get`. Only the
// resource-vault missing-secret shape is treated as not found; transport,
// Docker, and Vault outages are hard errors.
func (c *CLIVault) GetSecret(ctx context.Context, path, key string) (string, bool, error) {
	if c.LookPath != nil {
		if _, err := c.LookPath(c.command()); err != nil {
			return "", false, fmt.Errorf("vault CLI %q unavailable: %w", c.command(), err)
		}
	}
	run := c.Run
	if run == nil {
		run = runCommand
	}
	value, err := run(ctx, c.command(), "content", "get", "--path", path, "--key", key, "--format", "raw")
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

// PutSecret writes (path, key)=value via `resource-vault content set`.
func (c *CLIVault) PutSecret(ctx context.Context, path, key, value string) error {
	if c.LookPath != nil {
		if _, err := c.LookPath(c.command()); err != nil {
			return fmt.Errorf("vault CLI %q unavailable: %w", c.command(), err)
		}
	}
	run := c.Run
	if run == nil {
		run = runCommand
	}
	if _, err := run(ctx, c.command(), "content", "set", "--path", path, "--key", key, "--value", value); err != nil {
		return fmt.Errorf("vault content set %s/%s: %w", path, key, err)
	}
	return nil
}

// DeleteSecret removes a secret path via `resource-vault content delete`.
func (c *CLIVault) DeleteSecret(ctx context.Context, path string) error {
	if c.LookPath != nil {
		if _, err := c.LookPath(c.command()); err != nil {
			return fmt.Errorf("vault CLI %q unavailable: %w", c.command(), err)
		}
	}
	run := c.Run
	if run == nil {
		run = runCommand
	}
	if _, err := run(ctx, c.command(), "content", "delete", "--path", path); err != nil {
		return fmt.Errorf("vault content delete %s: %w", path, err)
	}
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		cliErr := &CLIError{Command: name, Args: append([]string(nil), args...), Err: err}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			cliErr.Stderr = string(exitErr.Stderr)
		}
		return "", cliErr
	}
	return strings.TrimSpace(string(out)), nil
}

func isMissingSecretError(err error) bool {
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		return false
	}
	msg := strings.ToLower(cliErr.Stderr)
	return strings.Contains(msg, "no value found") ||
		strings.Contains(msg, "no secret exists") ||
		strings.Contains(msg, "not found")
}
