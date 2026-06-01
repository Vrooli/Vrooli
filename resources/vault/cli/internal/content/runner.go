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
	"encoding/json"
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
	devTokenEnv   = "VAULT_DEV_ROOT_TOKEN_ID"
	bootstrapPath = "/vault/file/.vrooli-bootstrap.json"
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
	container    string
	addr         string
	token        string // empty => resolve from the container on first use
	tokenFromEnv bool
	timeout      time.Duration
}

var _ Runner = (*dockerRunner)(nil)

// NewDockerRunner returns a Runner bound to the vault container. Container,
// address and token may be overridden via VROOLI_VAULT_CONTAINER, VAULT_ADDR
// and VAULT_TOKEN; otherwise they default to the manifest values and the live
// dev-mode root token discovered from the container.
func NewDockerRunner() Runner {
	token := strings.TrimSpace(os.Getenv("VAULT_TOKEN"))
	return &dockerRunner{
		container:    envOr("VROOLI_VAULT_CONTAINER", defaultContainer),
		addr:         envOr("VAULT_ADDR", defaultAddr),
		token:        token,
		tokenFromEnv: token != "",
		timeout:      30 * time.Second,
	}
}

func (d *dockerRunner) Run(ctx context.Context, vaultArgs []string, stdin []byte) ([]byte, []byte, error) {
	if err := d.ensureInitializedAndUnsealed(ctx); err != nil {
		return nil, nil, err
	}
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
		if !d.tokenFromEnv || d.tokenUsable(ctx, d.token) {
			return d.token, nil
		}
		d.token = ""
		d.tokenFromEnv = false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(probeCtx, "docker", "exec", d.container, "sh", "-c", "test -f "+bootstrapPath+" && sed -n 's/.*\"root_token\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p' "+bootstrapPath).Output(); err == nil {
		if token := strings.TrimSpace(string(out)); token != "" {
			d.token = token
			d.tokenFromEnv = false
			return token, nil
		}
	}
	out, err := exec.CommandContext(probeCtx, "docker", "exec", d.container, "printenv", devTokenEnv).Output()
	if err != nil {
		return "", fmt.Errorf("resolve vault token from container %q: %w (set VAULT_TOKEN to override)", d.container, err)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("vault container %q reported an empty %s; set VAULT_TOKEN explicitly", d.container, devTokenEnv)
	}
	d.token = token
	d.tokenFromEnv = false
	return token, nil
}

func (d *dockerRunner) tokenUsable(ctx context.Context, token string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-e", "VAULT_ADDR="+d.addr, "-e", "VAULT_TOKEN="+token, d.container, "vault", "token", "lookup", "-format=json")
	return cmd.Run() == nil
}

type vaultStatus struct {
	Initialized bool   `json:"initialized"`
	Sealed      bool   `json:"sealed"`
	StorageType string `json:"storage_type"`
}

type operatorInitOutput struct {
	UnsealKeysB64 []string `json:"unseal_keys_b64"`
	RootToken     string   `json:"root_token"`
}

func (d *dockerRunner) ensureInitializedAndUnsealed(ctx context.Context) error {
	status, err := d.status(ctx)
	if err != nil {
		return err
	}
	if !status.Initialized {
		initOut, err := d.operatorInit(ctx)
		if err != nil {
			return err
		}
		if initOut.RootToken == "" || len(initOut.UnsealKeysB64) == 0 || strings.TrimSpace(initOut.UnsealKeysB64[0]) == "" {
			return fmt.Errorf("vault operator init returned incomplete bootstrap material")
		}
		if err := d.writeBootstrap(ctx, initOut); err != nil {
			return err
		}
		d.token = initOut.RootToken
		status, err = d.status(ctx)
		if err != nil {
			return err
		}
	}
	if status.Sealed {
		key, err := d.bootstrapUnsealKey(ctx)
		if err != nil {
			return err
		}
		if err := d.operatorUnseal(ctx, key); err != nil {
			return err
		}
	}
	if err := d.ensureKVv2(ctx); err != nil {
		return err
	}
	return nil
}

func (d *dockerRunner) status(ctx context.Context) (vaultStatus, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-e", "VAULT_ADDR="+d.addr, d.container, "vault", "status", "-format=json")
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok && len(out) > 0 {
			// Vault uses non-zero status for sealed/uninitialized states even
			// when -format=json returns a useful body on stdout.
		} else {
			return vaultStatus{}, fmt.Errorf("vault status in container %q: %w", d.container, err)
		}
	}
	var status vaultStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return vaultStatus{}, fmt.Errorf("parse vault status: %w", err)
	}
	return status, nil
}

func (d *dockerRunner) operatorInit(ctx context.Context) (operatorInitOutput, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-e", "VAULT_ADDR="+d.addr, d.container, "vault", "operator", "init", "-key-shares=1", "-key-threshold=1", "-format=json")
	out, err := cmd.Output()
	if err != nil {
		return operatorInitOutput{}, fmt.Errorf("vault operator init: %w", err)
	}
	var initOut operatorInitOutput
	if err := json.Unmarshal(out, &initOut); err != nil {
		return operatorInitOutput{}, fmt.Errorf("parse vault operator init output: %w", err)
	}
	return initOut, nil
}

func (d *dockerRunner) writeBootstrap(ctx context.Context, initOut operatorInitOutput) error {
	data, err := json.Marshal(initOut)
	if err != nil {
		return err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-i", d.container, "sh", "-c", "umask 077 && cat > "+bootstrapPath)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write vault bootstrap file: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *dockerRunner) bootstrapUnsealKey(ctx context.Context) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(probeCtx, "docker", "exec", d.container, "sh", "-c", "test -f "+bootstrapPath+" && sed -n 's/.*\"unseal_keys_b64\"[[:space:]]*:[[:space:]]*\\[\"\\([^\"]*\\)\"\\].*/\\1/p' "+bootstrapPath).Output()
	if err != nil {
		return "", fmt.Errorf("read vault bootstrap unseal key: %w", err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", fmt.Errorf("vault bootstrap file has no unseal key")
	}
	return key, nil
}

func (d *dockerRunner) operatorUnseal(ctx context.Context, key string) error {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-e", "VAULT_ADDR="+d.addr, d.container, "vault", "operator", "unseal", key)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vault operator unseal: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *dockerRunner) ensureKVv2(ctx context.Context) error {
	token, err := d.resolveToken(ctx)
	if err != nil {
		return err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "exec", "-e", "VAULT_ADDR="+d.addr, "-e", "VAULT_TOKEN="+token, d.container, "vault", "secrets", "enable", "-path=secret", "kv-v2")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.ToLower(string(out))
	if strings.Contains(msg, "path is already in use") || strings.Contains(msg, "existing mount") {
		return nil
	}
	return fmt.Errorf("enable vault kv-v2 secret mount: %w: %s", err, strings.TrimSpace(string(out)))
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
