package ssh

// Host-key trust is implemented once in packages/ssh-core. This adapter keeps
// the scenario-to-cloud API stable while it migrates onto that shared package.

import (
	"fmt"
	"os"
	"path/filepath"

	sshcore "github.com/vrooli/ssh-core"
	gossh "golang.org/x/crypto/ssh"
)

func NewTOFUHostKeyCallback(host string, port int) (gossh.HostKeyCallback, error) {
	path, err := ensureKnownHostsFile()
	if err != nil {
		return nil, err
	}
	return NewTOFUHostKeyCallbackForPath(host, port, path)
}

func NewTOFUHostKeyCallbackForPath(host string, port int, path string) (gossh.HostKeyCallback, error) {
	return sshcore.NewTOFUHostKeyCallback(host, port, path)
}

func ensureKnownHostsFile() (string, error) {
	path := scenarioKnownHostsFile()
	if err := sshcore.EnsureKnownHostsFile(path); err != nil {
		return "", fmt.Errorf("initialize cloud SSH known_hosts: %w", err)
	}
	return path, nil
}

func scenarioKnownHostsFile() string {
	stateDir := os.Getenv("VROOLI_STATE_DIR")
	if stateDir == "" {
		stateDir = filepath.Join("data", "ssh")
	}
	return filepath.Join(stateDir, "known_hosts")
}
