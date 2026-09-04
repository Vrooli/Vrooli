package ssh

import (
	"errors"
	"testing"
)

func TestSSHError_ErrorReturnsMessage(t *testing.T) {
	t.Parallel()

	e := &SSHError{
		Category: ErrAuth,
		Message:  "authentication failed for host",
		Hint:     "check your key",
	}
	if e.Error() != "authentication failed for host" {
		t.Errorf("Error() = %q, want %q", e.Error(), "authentication failed for host")
	}
}

func TestSSHError_UnwrapReturnsCategory(t *testing.T) {
	t.Parallel()

	e := &SSHError{Category: ErrTimeout, Message: "timed out"}
	if !errors.Is(e, ErrTimeout) {
		t.Error("errors.Is(sshErr, ErrTimeout) should be true")
	}
	if errors.Is(e, ErrAuth) {
		t.Error("errors.Is(sshErr, ErrAuth) should be false")
	}
}

func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		errStr        string
		host          string
		wantCategory  error
		wantRetryable bool
	}{
		{
			name:          "host key changed",
			errStr:        "Host key verification failed.",
			host:          "example.com",
			wantCategory:  ErrHostKey,
			wantRetryable: false,
		},
		{
			name:          "permission denied",
			errStr:        "Permission denied (publickey).",
			host:          "example.com",
			wantCategory:  ErrAuth,
			wantRetryable: false,
		},
		{
			name:          "connection refused",
			errStr:        "connection refused",
			host:          "example.com",
			wantCategory:  ErrUnreachable,
			wantRetryable: true,
		},
		{
			name:          "timeout ipv4",
			errStr:        "i/o timeout",
			host:          "192.168.1.1",
			wantCategory:  ErrTimeout,
			wantRetryable: true,
		},
		{
			name:          "timeout ipv6",
			errStr:        "i/o timeout",
			host:          "2001:db8::1",
			wantCategory:  ErrIPv6,
			wantRetryable: true,
		},
		{
			name:          "no route ipv6",
			errStr:        "no route to host",
			host:          "::1",
			wantCategory:  ErrIPv6,
			wantRetryable: true,
		},
		{
			name:          "no route ipv4",
			errStr:        "no route to host",
			host:          "10.0.0.1",
			wantCategory:  ErrUnreachable,
			wantRetryable: true,
		},
		{
			name:          "disk space",
			errStr:        "write error: No space left on device",
			host:          "example.com",
			wantCategory:  ErrDiskSpace,
			wantRetryable: false,
		},
		{
			name:          "dns resolution hostname",
			errStr:        "ssh: Could not resolve hostname bad.example.com",
			host:          "bad.example.com",
			wantCategory:  ErrDNS,
			wantRetryable: false,
		},
		{
			name:          "dns name or service not known",
			errStr:        "ssh: getaddrinfo: Name or service not known",
			host:          "bad.example.com",
			wantCategory:  ErrDNS,
			wantRetryable: false,
		},
		{
			name:          "key invalid format",
			errStr:        "load key \"/root/.ssh/id_ed25519\": invalid format",
			host:          "example.com",
			wantCategory:  ErrKeyFormat,
			wantRetryable: false,
		},
		{
			name:          "key bad permissions",
			errStr:        "bad permissions on /root/.ssh/id_ed25519",
			host:          "example.com",
			wantCategory:  ErrKeyFormat,
			wantRetryable: false,
		},
		{
			name:          "connection reset",
			errStr:        "Connection reset by peer",
			host:          "example.com",
			wantCategory:  ErrUnreachable,
			wantRetryable: true,
		},
		{
			name:          "broken pipe",
			errStr:        "write: broken pipe",
			host:          "example.com",
			wantCategory:  ErrUnreachable,
			wantRetryable: true,
		},
		{
			name:          "too many auth failures",
			errStr:        "Received disconnect: Too many authentication failures",
			host:          "example.com",
			wantCategory:  ErrAuth,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.errStr, tt.host, "default hint")
			if !errors.Is(result, tt.wantCategory) {
				t.Errorf("ClassifyError(%q, %q) category mismatch: got %v, want %v",
					tt.errStr, tt.host, result.Category, tt.wantCategory)
			}
			if result.Retryable != tt.wantRetryable {
				t.Errorf("ClassifyError(%q, %q) retryable = %v, want %v",
					tt.errStr, tt.host, result.Retryable, tt.wantRetryable)
			}
			if result.Message == "" {
				t.Error("ClassifyError should always set Message")
			}
			if result.Hint == "" {
				t.Error("ClassifyError should always set Hint")
			}
			if result.Host != tt.host {
				t.Errorf("ClassifyError Host = %q, want %q", result.Host, tt.host)
			}
		})
	}
}

func TestClassifyError_LibraryHostKeyMismatchIsSecurityError(t *testing.T) {
	err := ClassifyError("ssh: handshake failed: knownhosts: key mismatch", "example.com", "")
	if !errors.Is(err, ErrHostKey) || err.Retryable {
		t.Fatalf("library mismatch = %#v, want non-retryable host-key error", err)
	}
}

func TestClassifyError_HostKeyHintContainsHost(t *testing.T) {
	t.Parallel()

	result := ClassifyError("Host key verification failed.", "myhost.com", "")
	if result.Hint == "" || !containsSubstr(result.Hint, "myhost.com") {
		t.Errorf("host_key_changed hint should contain hostname: %q", result.Hint)
	}
}

func TestStatusFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		category   error
		wantStatus string
	}{
		{name: "auth", category: ErrAuth, wantStatus: StatusAuthFailed},
		{name: "host key", category: ErrHostKey, wantStatus: StatusHostKeyChanged},
		{name: "timeout", category: ErrTimeout, wantStatus: StatusTimeout},
		{name: "unreachable", category: ErrUnreachable, wantStatus: StatusHostUnreachable},
		{name: "ipv6", category: ErrIPv6, wantStatus: StatusIPv6Unavailable},
		{name: "command", category: ErrCommand, wantStatus: StatusError},
		{name: "disk space", category: ErrDiskSpace, wantStatus: StatusDiskFull},
		{name: "dns", category: ErrDNS, wantStatus: StatusDNSFailed},
		{name: "key format", category: ErrKeyFormat, wantStatus: StatusKeyError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &SSHError{Category: tt.category}
			status := StatusFromError(err)
			if status != tt.wantStatus {
				t.Errorf("StatusFromError(%v) = %q, want %q", tt.category, status, tt.wantStatus)
			}
		})
	}
}

func TestNewCommandError(t *testing.T) {
	t.Parallel()

	original := errors.New("exit status 1")
	res := Result{Stderr: "command not found", ExitCode: 127}
	err := newCommandError(original, res, "example.com")

	if err == nil {
		t.Fatal("newCommandError returned nil")
	}
	if !errors.Is(err, ErrCommand) {
		t.Error("newCommandError should wrap ErrCommand")
	}
	if err.ExitCode != 127 {
		t.Errorf("ExitCode = %d, want 127", err.ExitCode)
	}
	if err.Host != "example.com" {
		t.Errorf("Host = %q, want %q", err.Host, "example.com")
	}
	if err.Message == "" {
		t.Error("newCommandError should set Message")
	}
}

func TestNewCommandError_ClassifiesStderr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		stderr       string
		wantCategory error
	}{
		{
			name:         "disk full stderr",
			stderr:       "tar: write error: No space left on device",
			wantCategory: ErrDiskSpace,
		},
		{
			name:         "dns failure in stderr",
			stderr:       "ssh: Could not resolve hostname bogus.example.com: Name or service not known",
			wantCategory: ErrDNS,
		},
		{
			name:         "generic stderr stays ErrCommand",
			stderr:       "some random error output",
			wantCategory: ErrCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := errors.New("exit status 1")
			res := Result{Stderr: tt.stderr, ExitCode: 1}
			err := newCommandError(original, res, "example.com")
			if !errors.Is(err, tt.wantCategory) {
				t.Errorf("newCommandError with stderr %q: got category %v, want %v",
					tt.stderr, err.Category, tt.wantCategory)
			}
			if err.ExitCode != 1 {
				t.Errorf("ExitCode = %d, want 1", err.ExitCode)
			}
		})
	}
}

func TestErrorInfoFromSSHError(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		info := ErrorInfoFromSSHError(nil)
		if info != nil {
			t.Errorf("ErrorInfoFromSSHError(nil) = %v, want nil", info)
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		sshErr := &SSHError{
			Category:  ErrDiskSpace,
			Message:   "No space left on device",
			Hint:      "Run df -h",
			Retryable: false,
			ExitCode:  1,
			Host:      "example.com",
		}
		info := ErrorInfoFromSSHError(sshErr)
		if info == nil {
			t.Fatal("ErrorInfoFromSSHError returned nil for non-nil input")
		}
		if info.Message != sshErr.Message {
			t.Errorf("Message = %q, want %q", info.Message, sshErr.Message)
		}
		if info.Category != StatusDiskFull {
			t.Errorf("Category = %q, want %q", info.Category, StatusDiskFull)
		}
		if info.Hint != sshErr.Hint {
			t.Errorf("Hint = %q, want %q", info.Hint, sshErr.Hint)
		}
		if info.Retryable != sshErr.Retryable {
			t.Errorf("Retryable = %v, want %v", info.Retryable, sshErr.Retryable)
		}
		if info.ExitCode != sshErr.ExitCode {
			t.Errorf("ExitCode = %d, want %d", info.ExitCode, sshErr.ExitCode)
		}
	})
}
