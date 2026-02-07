package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var knownHostsWriteMu sync.Mutex

// NewTOFUHostKeyCallback returns a known_hosts-backed callback that mirrors
// OpenSSH StrictHostKeyChecking=accept-new behavior:
// - unknown hosts are added on first connect
// - changed host keys are rejected
func NewTOFUHostKeyCallback(host string, port int) (gossh.HostKeyCallback, error) {
	knownHostsPath, err := ensureKnownHostsFile()
	if err != nil {
		return nil, err
	}
	return NewTOFUHostKeyCallbackForPath(host, port, knownHostsPath)
}

// NewTOFUHostKeyCallbackForPath is equivalent to NewTOFUHostKeyCallback, but
// writes to a caller-provided known_hosts file path (test seam).
func NewTOFUHostKeyCallbackForPath(host string, port int, knownHostsPath string) (gossh.HostKeyCallback, error) {
	baseCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}

	if port == 0 {
		port = DefaultPort
	}

	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		if err := baseCallback(hostname, remote, key); err == nil {
			return nil
		} else {
			var keyErr *knownhosts.KeyError
			if !asKnownHostsKeyError(err, &keyErr) {
				return err
			}
			// Existing host entry mismatch: reject.
			if len(keyErr.Want) > 0 {
				return err
			}
		}

		// Unknown host: trust on first use and persist to known_hosts.
		address := net.JoinHostPort(host, strconv.Itoa(port))
		normalizedAddress := knownhosts.Normalize(address)
		line := knownhosts.Line([]string{normalizedAddress}, key)
		if err := appendKnownHostLine(knownHostsPath, line); err != nil {
			return fmt.Errorf("persist host key: %w", err)
		}

		// Re-load and verify to guarantee the persisted key matches.
		verifyCallback, err := knownhosts.New(knownHostsPath)
		if err != nil {
			return fmt.Errorf("reload known_hosts: %w", err)
		}
		return verifyCallback(hostname, remote, key)
	}, nil
}

func ensureKnownHostsFile() (string, error) {
	sshDir, err := GetSSHDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("create ~/.ssh: %w", err)
	}
	path := filepath.Join(sshDir, "known_hosts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return "", fmt.Errorf("create known_hosts: %w", err)
		}
	}
	return path, nil
}

func appendKnownHostLine(path, line string) error {
	knownHostsWriteMu.Lock()
	defer knownHostsWriteMu.Unlock()

	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString(line); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	return nil
}

func asKnownHostsKeyError(err error, out **knownhosts.KeyError) bool {
	if err == nil {
		return false
	}
	keyErr, ok := err.(*knownhosts.KeyError)
	if !ok {
		return false
	}
	*out = keyErr
	return true
}
