package ssh

import (
	"fmt"
	"strings"
	"time"
)

// Connection defaults applied when a request omits them.
const (
	DefaultPort = 22
	DefaultUser = "root"
)

// normalizePort applies the protocol default and rejects values that cannot be
// represented by a TCP port. Request boundaries call it before local or remote
// work so invalid input is reported as such rather than as a connection error.
func normalizePort(port int) (int, error) {
	if port == 0 {
		port = DefaultPort
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("SSH port must be between 1 and 65535, got %d", port)
	}
	return port, nil
}

// ConnectionConfig holds SSH connection parameters. KnownHostsFile is the bridge-owned
// known_hosts path threaded into the system ssh/scp invocations so their TOFU
// state lives in the same file the x/crypto key-copy path writes — one coherent
// host-key store under the bridge state dir, never the operator's ~/.ssh.
type ConnectionConfig struct {
	Host           string
	Port           int
	User           string
	KeyPath        string
	KnownHostsFile string
}

// NewConfig creates a ConnectionConfig with defaults applied for missing values.
func NewConfig(host string, port int, user, keyPath, knownHostsFile string) ConnectionConfig {
	if port == 0 {
		port = DefaultPort
	}
	if user == "" {
		user = DefaultUser
	}
	return ConnectionConfig{
		Host:           host,
		Port:           port,
		User:           user,
		KeyPath:        strings.TrimSpace(keyPath),
		KnownHostsFile: strings.TrimSpace(knownHostsFile),
	}
}

// nowTimestamp returns the current UTC time in RFC3339 format.
func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Result holds the output of an SSH command execution.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}
