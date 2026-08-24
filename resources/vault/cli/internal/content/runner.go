package content

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/envkit-go"
	"time"
)

// Runner executes the supported Vault client contract. Consumers use this
// interface instead of direct HTTP calls, and receive only an explicitly
// supplied scoped token.
type Runner interface {
	Run(ctx context.Context, vaultArgs []string, stdin []byte) (stdout []byte, stderr []byte, err error)
}

type nativeRunner struct {
	binary   string
	addr     string
	token    string
	provider string
	timeout  time.Duration
}

var _ Runner = (*nativeRunner)(nil)

// NewDefaultRunner always uses the signed managed-service client path. It
// never discovers a Docker container, reads bootstrap material, initializes,
// or unseals Vault.
func NewDefaultRunner() Runner { return NewNativeRunner() }

func NewNativeRunner() Runner {
	binary := strings.TrimSpace(os.Getenv("VROOLI_VAULT_BINARY"))
	if binary == "" {
		binary = strings.TrimSpace(os.Getenv("VROOLI_MANAGED_SERVICE_ARTIFACT"))
	}
	if binary == "" {
		binary = "vault"
	}
	return &nativeRunner{
		binary:   binary,
		addr:     envOr("VAULT_ADDR", "http://127.0.0.1:8200"),
		token:    strings.TrimSpace(os.Getenv("VAULT_TOKEN")),
		provider: strings.TrimSpace(os.Getenv("VROOLI_MANAGED_PROVIDER")),
		timeout:  30 * time.Second,
	}
}

func (n *nativeRunner) Run(ctx context.Context, vaultArgs []string, stdin []byte) ([]byte, []byte, error) {
	if strings.EqualFold(n.provider, "remote-vrooli") {
		return nil, nil, fmt.Errorf("remote-vrooli provider must use the scenario API; direct Vault client access is forbidden")
	}
	if strings.TrimSpace(n.token) == "" {
		return nil, nil, fmt.Errorf("VAULT_TOKEN is required for managed Vault content access")
	}
	if n.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, n.timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, n.binary, vaultArgs...)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkit.Env{"VAULT_ADDR=" + n.addr, "VAULT_TOKEN=" + n.token})
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
