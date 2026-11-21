package ssh

// This file is intentionally a thin compatibility adapter. The implementation
// lives in packages/ssh-core so Bridge and scenario-to-cloud cannot drift in
// host-key trust semantics again.

import (
	"errors"

	sshcore "github.com/vrooli/ssh-core"
	gossh "golang.org/x/crypto/ssh"
)

func newTOFUHostKeyCallback(host string, port int, path string) (gossh.HostKeyCallback, error) {
	return sshcore.NewTOFUHostKeyCallback(host, port, path)
}

func pinnedHostKeyAlgorithms(host string, port int, path string) []string {
	return sshcore.PinnedHostKeyAlgorithms(host, port, path)
}

func hostKeyAlgorithmsForType(keyType string) []string {
	return sshcore.HostKeyAlgorithmsForType(keyType)
}

func ensureKnownHostsFile(path string) error { return sshcore.EnsureKnownHostsFile(path) }

func hostFingerprint(path, host string, port int) string {
	return sshcore.HostFingerprint(path, host, port)
}

func (s *Service) ForgetHostKey(host string, port int) error {
	if host == "" {
		return errors.New("host key review: host is required")
	}
	if port == 0 {
		port = DefaultPort
	}
	return sshcore.ForgetHostKey(s.knownHostsPath(), host, port)
}
