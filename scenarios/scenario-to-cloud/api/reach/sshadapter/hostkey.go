package sshadapter

import (
	"fmt"
	"os"
	"path/filepath"

	sshcore "github.com/vrooli/ssh-core"
	gossh "golang.org/x/crypto/ssh"
)

// KnownHostsFile is the scenario-owned trust store every SSH invocation
// (command and session) reads and extends on first use.
func KnownHostsFile() string {
	stateDir := os.Getenv("VROOLI_STATE_DIR")
	if stateDir == "" {
		stateDir = filepath.Join("data", "ssh")
	}
	return filepath.Join(stateDir, "known_hosts")
}

// NewTOFUHostKeyCallback is the trust-on-first-use policy from
// packages/ssh-core over the scenario's known_hosts store.
func NewTOFUHostKeyCallback(host string, port int) (gossh.HostKeyCallback, error) {
	path := KnownHostsFile()
	if err := sshcore.EnsureKnownHostsFile(path); err != nil {
		return nil, fmt.Errorf("initialize cloud SSH known_hosts: %w", err)
	}
	return sshcore.NewTOFUHostKeyCallback(host, port, path)
}
