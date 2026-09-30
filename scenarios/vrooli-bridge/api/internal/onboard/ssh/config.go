package ssh

import (
	"strings"
	"time"
)

// Connection defaults applied when a request omits them.
const (
	DefaultPort = 22
	DefaultUser = "root"
)

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
