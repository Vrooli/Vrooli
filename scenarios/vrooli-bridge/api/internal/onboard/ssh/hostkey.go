package ssh

import (
	"bytes"
	"errors"
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

// newTOFUHostKeyCallback returns a known_hosts-backed callback that mirrors
// OpenSSH StrictHostKeyChecking=accept-new against the bridge-owned
// known_hosts file:
//   - unknown hosts are trusted on first use and persisted
//   - a changed host key for a known host is rejected
func newTOFUHostKeyCallback(host string, port int, knownHostsPath string) (gossh.HostKeyCallback, error) {
	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return nil, err
	}
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

		// Unknown host: trust on first use and persist.
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

// ensureKnownHostsFile creates the state dir (0700) and an empty known_hosts
// (0600) if absent.
func ensureKnownHostsFile(path string) error {
	if err := ensureDir0700(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create ssh state dir: %w", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return fmt.Errorf("create known_hosts: %w", err)
		}
	}
	return nil
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

// hostFingerprint returns the SHA256 fingerprint of the persisted host key for
// host:port from the bridge known_hosts, or "" if not recorded (best-effort,
// cosmetic).
func hostFingerprint(knownHostsPath, host string, port int) string {
	data, err := os.ReadFile(knownHostsPath)
	if err != nil {
		return ""
	}
	if port == 0 {
		port = DefaultPort
	}
	target := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		_, hosts, key, _, _, err := gossh.ParseKnownHosts(line)
		if err != nil {
			continue
		}
		for _, h := range hosts {
			if h == target {
				return gossh.FingerprintSHA256(key)
			}
		}
	}
	return ""
}
