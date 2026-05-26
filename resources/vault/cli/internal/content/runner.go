// Package content implements the resource-vault `content` command group: the
// CLI surface through which other resources and scenarios read and write
// secrets in the Vault KV store (wrap-not-use). Consumers MUST use this CLI;
// they never talk to Vault's HTTP API directly. The command shells `vault kv`
// inside the resource's own container, exactly as the postgres resource's
// `content` command shells psql inside its container.
package content

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultContainer = "vault"
	// defaultAddr is the in-container address the vault server listens on; we
	// exec inside the container so loopback is correct.
	defaultAddr = "http://127.0.0.1:8200"
	// devTokenEnv is the env var the dev-mode server records its root token in
	// (set via -dev-root-token-id in the manifest runtime.command).
	devTokenEnv = "VAULT_DEV_ROOT_TOKEN_ID"
)

// Runner executes a `vault <args...>` invocation inside the vault container,
// with VAULT_ADDR and VAULT_TOKEN supplied. It abstracts `docker exec` so unit
// tests can assert the vault argv without a live container.
//
// seam: production wires *dockerRunner; tests wire a fake recording Runner.
type Runner interface {
	Run(ctx context.Context, vaultArgs []string, stdin []byte) (stdout []byte, stderr []byte, err error)
}

// dockerRunner is the production Runner: `docker exec -i -e VAULT_ADDR -e
// VAULT_TOKEN <container> vault <vaultArgs...>`.
type dockerRunner struct {
	container string
	addr      string
	token     string // empty => resolve from the container on first use
	timeout   time.Duration
}

var _ Runner = (*dockerRunner)(nil)

// NewDockerRunner returns a Runner bound to the vault container. Container,
// address and token may be overridden via VROOLI_VAULT_CONTAINER, VAULT_ADDR
// and VAULT_TOKEN; otherwise they default to the manifest values and the live
// dev-mode root token discovered from the container.
func NewDockerRunner() Runner {
	return &dockerRunner{
		container: envOr("VROOLI_VAULT_CONTAINER", defaultContainer),
		addr:      envOr("VAULT_ADDR", defaultAddr),
		token:     strings.TrimSpace(os.Getenv("VAULT_TOKEN")),
		timeout:   30 * time.Second,
	}
}

func (d *dockerRunner) Run(ctx context.Context, vaultArgs []string, stdin []byte) ([]byte, []byte, error) {
	token, err := d.resolveToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	args := []string{"exec", "-i", "-e", "VAULT_ADDR=" + d.addr, "-e", "VAULT_TOKEN=" + token, d.container, "vault"}
	args = append(args, vaultArgs...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// resolveToken returns the configured token or discovers the live dev-mode root
// token from the container. It never falls back to a guessed/empty token.
func (d *dockerRunner) resolveToken(ctx context.Context) (string, error) {
	if d.token != "" {
		return d.token, nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(probeCtx, "docker", "exec", d.container, "printenv", devTokenEnv).Output()
	if err != nil {
		return "", fmt.Errorf("resolve vault token from container %q: %w (set VAULT_TOKEN to override)", d.container, err)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("vault container %q reported an empty %s; set VAULT_TOKEN explicitly", d.container, devTokenEnv)
	}
	d.token = token
	return token, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
