package runtime

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config is the typed, local-only Vault server configuration used by the
// managed-service path. It intentionally does not contain bootstrap tokens or
// unseal material; those belong in the provider's secure secret store.
type Config struct {
	StoragePath  string
	ListenAddr   string
	APIAddr      string
	DisableMlock bool
	UI           bool
}

// DefaultConfig creates the normal local managed-service configuration. Vault
// is bound to loopback by default: exposing an unauthenticated local resource
// to the LAN is never an implicit deployment choice.
func DefaultConfig(dataDir string, port int) (Config, error) {
	if port <= 0 || port > 65535 {
		return Config{}, fmt.Errorf("vault port must be between 1 and 65535")
	}
	return Config{
		// Reusing the canonical resource data directory makes the native managed
		// server read the existing supported file-storage state instead of
		// silently creating a second empty Vault tree during migration.
		StoragePath:  filepath.Clean(dataDir),
		ListenAddr:   net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		APIAddr:      "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		DisableMlock: true,
		UI:           false,
	}, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.StoragePath) == "" || !filepath.IsAbs(c.StoragePath) {
		return fmt.Errorf("vault storage path must be absolute")
	}
	host, port, err := net.SplitHostPort(c.ListenAddr)
	if err != nil || host == "" || port == "" {
		return fmt.Errorf("vault listen address must be host:port")
	}
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		return fmt.Errorf("managed Vault must bind loopback; got %q", host)
	}
	if _, err := net.ResolveTCPAddr("tcp", c.ListenAddr); err != nil {
		return fmt.Errorf("vault listen address is invalid: %w", err)
	}
	if strings.TrimSpace(c.APIAddr) == "" || !strings.HasPrefix(c.APIAddr, "http://") {
		return fmt.Errorf("vault api address must be an http URL")
	}
	return nil
}

// RenderHCL renders the narrow file-storage configuration accepted by the
// local Vault server. Values are always quoted with Go's string encoder, which
// is compatible with HCL string syntax and prevents interpolation injection.
func (c Config) RenderHCL() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	return fmt.Sprintf(`storage "file" {
  path = %s
}

listener "tcp" {
  address = %s
  tls_disable = true
}

api_addr = %s
disable_mlock = %t
ui = %t
`, strconv.Quote(c.StoragePath), strconv.Quote(c.ListenAddr), strconv.Quote(c.APIAddr), c.DisableMlock, c.UI), nil
}

// Write writes a durable server configuration without granting group or world
// read access. The file has no secrets, but its path/storage topology is still
// local control-plane metadata.
func (c Config) Write(path string) error {
	body, err := c.RenderHCL()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create Vault configuration directory: %w", err)
	}
	return os.WriteFile(path, []byte(body), 0o600)
}
