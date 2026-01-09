package ssh

import (
	"strings"

	"scenario-to-cloud/domain"
)

// Config holds SSH connection parameters.
type Config struct {
	Host    string
	Port    int
	User    string
	KeyPath string
}

// ConfigDefaults provides common default values for SSH connection parameters.
const (
	DefaultPort = 22
	DefaultUser = "root"
)

// NewConfig creates a Config with defaults applied for missing values.
func NewConfig(host string, port int, user string, keyPath string) Config {
	if port == 0 {
		port = DefaultPort
	}
	if user == "" {
		user = DefaultUser
	}
	return Config{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: strings.TrimSpace(keyPath),
	}
}

// ConfigFromManifest creates an SSH Config from a CloudManifest's VPS target.
func ConfigFromManifest(manifest domain.CloudManifest) Config {
	vps := manifest.Target.VPS
	return Config{
		Host:    vps.Host,
		Port:    vps.Port,
		User:    vps.User,
		KeyPath: strings.TrimSpace(vps.KeyPath),
	}
}

// Result holds the output of an SSH command execution.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}
