// Package sshcore contains the security-sensitive SSH primitives shared by
// Bridge and scenario-to-cloud.  Consumers supply their own state path; the
// package never silently writes to an operator's ~/.ssh directory.
package sshcore

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const DefaultPort = 22

var knownHostsWriteMu sync.Mutex

// EnsureKnownHostsFile creates a private state directory and known_hosts file.
func EnsureKnownHostsFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create ssh state dir: %w", err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secure ssh state dir: %w", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return fmt.Errorf("create known_hosts: %w", err)
		}
	}
	return os.Chmod(path, 0o600)
}

// NewTOFUHostKeyCallback accepts an unknown key once and persists it, while
// rejecting any key change for an already-known host.
func NewTOFUHostKeyCallback(host string, port int, path string) (gossh.HostKeyCallback, error) {
	if err := EnsureKnownHostsFile(path); err != nil {
		return nil, err
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}
	if port == 0 {
		port = DefaultPort
	}
	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		if err := base(hostname, remote, key); err == nil {
			return nil
		} else {
			var keyErr *knownhosts.KeyError
			if !asKeyError(err, &keyErr) || len(keyErr.Want) > 0 {
				return err
			}
			// knownhosts can classify a changed key as an unknown-key error when
			// the callback receives a bracketed host/port spelling. Consult the
			// normalized file entry before accepting anything, or a changed key
			// could be appended as a second TOFU record.
			target := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
			if exists, sameKey := knownHostKey(path, target, key); exists {
				if sameKey {
					return nil
				}
				return err
			}
		}
		line := knownhosts.Line([]string{knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))}, key)
		if err := appendLine(path, line); err != nil {
			return fmt.Errorf("persist host key: %w", err)
		}
		verify, err := knownhosts.New(path)
		if err != nil {
			return fmt.Errorf("reload known_hosts: %w", err)
		}
		return verify(hostname, remote, key)
	}, nil
}

// PinnedHostKeyAlgorithms returns the algorithms already pinned for host:port.
func PinnedHostKeyAlgorithms(host string, port int, path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if port == 0 {
		port = DefaultPort
	}
	target := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	seen := map[string]bool{}
	var out []string
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		_, hosts, key, _, _, err := gossh.ParseKnownHosts(line)
		if err != nil {
			continue
		}
		for _, token := range hosts {
			if !KnownHostMatches(token, target) {
				continue
			}
			for _, algorithm := range HostKeyAlgorithmsForType(key.Type()) {
				if !seen[algorithm] {
					seen[algorithm] = true
					out = append(out, algorithm)
				}
			}
			break
		}
	}
	return out
}

// HostFingerprint returns the SHA-256 fingerprint of a pinned host key.
func HostFingerprint(path, host string, port int) string {
	data, err := os.ReadFile(path)
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
		for _, token := range hosts {
			if KnownHostMatches(token, target) {
				return gossh.FingerprintSHA256(key)
			}
		}
	}
	return ""
}

// ForgetHostKey removes only entries matching host:port, preserving unrelated
// trust records and comments.
func ForgetHostKey(path, host string, port int) error {
	if strings.TrimSpace(host) == "" {
		return errors.New("host key review: host is required")
	}
	if err := EnsureKnownHostsFile(path); err != nil {
		return err
	}
	if port == 0 {
		port = DefaultPort
	}
	target := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	knownHostsWriteMu.Lock()
	defer knownHostsWriteMu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	kept := make([][]byte, 0)
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			kept = append(kept, raw)
			continue
		}
		_, hosts, _, _, _, parseErr := gossh.ParseKnownHosts(line)
		remove := false
		if parseErr == nil {
			for _, token := range hosts {
				if KnownHostMatches(token, target) {
					remove = true
					break
				}
			}
		}
		if !remove {
			kept = append(kept, raw)
		}
	}
	out := bytes.Join(kept, []byte("\n"))
	if len(out) > 0 {
		out = append(out, '\n')
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// KnownHostMatches handles plaintext and OpenSSH hashed host tokens.
func KnownHostMatches(token, target string) bool {
	if !strings.HasPrefix(token, "|1|") {
		return token == target
	}
	parts := strings.Split(token, "|")
	if len(parts) != 4 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	mac := hmac.New(sha1.New, salt)
	_, _ = mac.Write([]byte(target))
	return hmac.Equal(mac.Sum(nil), want)
}

func HostKeyAlgorithmsForType(keyType string) []string {
	if keyType == gossh.KeyAlgoRSA {
		return []string{gossh.KeyAlgoRSASHA256, gossh.KeyAlgoRSASHA512, gossh.KeyAlgoRSA}
	}
	return []string{keyType}
}

func knownHostKey(path, target string, want gossh.PublicKey) (exists, sameKey bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		_, hosts, key, _, _, err := gossh.ParseKnownHosts(line)
		if err != nil {
			continue
		}
		for _, token := range hosts {
			if KnownHostMatches(token, target) {
				return true, bytes.Equal(key.Marshal(), want.Marshal())
			}
		}
	}
	return false, false
}

func asKeyError(err error, out **knownhosts.KeyError) bool {
	keyErr, ok := err.(*knownhosts.KeyError)
	if ok {
		*out = keyErr
	}
	return ok
}

func appendLine(path, line string) error {
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
	_, err = f.WriteString(line + "\n")
	return err
}
