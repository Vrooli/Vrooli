package ssh

import (
	"errors"
	"testing"
)

func TestClassifyError_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		errStr     string
		host       string
		wantStatus string
	}{
		{
			name:       "host key changed",
			errStr:     "Host key verification failed.",
			host:       "example.com",
			wantStatus: StatusHostKeyChanged,
		},
		{
			name:       "permission denied",
			errStr:     "Permission denied (publickey).",
			host:       "example.com",
			wantStatus: StatusAuthFailed,
		},
		{
			name:       "unable to authenticate",
			errStr:     "unable to authenticate",
			host:       "example.com",
			wantStatus: StatusAuthFailed,
		},
		{
			name:       "no supported methods remain",
			errStr:     "no supported methods remain",
			host:       "example.com",
			wantStatus: StatusAuthFailed,
		},
		{
			name:       "connection refused",
			errStr:     "connection refused",
			host:       "example.com",
			wantStatus: StatusHostUnreachable,
		},
		{
			name:       "timeout ipv4",
			errStr:     "i/o timeout",
			host:       "192.168.1.1",
			wantStatus: StatusTimeout,
		},
		{
			name:       "timeout ipv6",
			errStr:     "i/o timeout",
			host:       "2001:db8::1",
			wantStatus: StatusIPv6Unavailable,
		},
		{
			name:       "connection timed out",
			errStr:     "connection timed out",
			host:       "10.0.0.1",
			wantStatus: StatusTimeout,
		},
		{
			name:       "no route to host ipv4",
			errStr:     "no route to host",
			host:       "10.0.0.1",
			wantStatus: StatusHostUnreachable,
		},
		{
			name:       "no route to host ipv6",
			errStr:     "no route to host",
			host:       "::1",
			wantStatus: StatusIPv6Unavailable,
		},
		{
			name:       "network unreachable ipv4",
			errStr:     "network is unreachable",
			host:       "10.0.0.1",
			wantStatus: StatusHostUnreachable,
		},
		{
			name:       "network unreachable ipv6",
			errStr:     "network is unreachable",
			host:       "fe80::1",
			wantStatus: StatusIPv6Unavailable,
		},
		{
			name:       "unknown error",
			errStr:     "something unexpected happened",
			host:       "example.com",
			wantStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshErr := ClassifyError(tt.errStr, tt.host, "default hint")
			status := StatusFromError(sshErr)
			if status != tt.wantStatus {
				t.Errorf("ClassifyError(%q, %q) status = %q, want %q", tt.errStr, tt.host, status, tt.wantStatus)
			}
			if sshErr.Message == "" {
				t.Error("ClassifyError should always set Message")
			}
			if sshErr.Hint == "" {
				t.Error("ClassifyError should always set Hint")
			}
		})
	}
}

func TestClassifyError_Retryability(t *testing.T) {
	t.Parallel()

	retryable := []struct {
		errStr string
		host   string
	}{
		{"i/o timeout", "10.0.0.1"},
		{"connection timed out", "10.0.0.1"},
		{"connection refused", "10.0.0.1"},
		{"no route to host", "10.0.0.1"},
		{"network is unreachable", "10.0.0.1"},
		{"i/o timeout", "::1"},
		{"no route to host", "::1"},
	}
	for _, tt := range retryable {
		sshErr := ClassifyError(tt.errStr, tt.host, "")
		if !sshErr.Retryable {
			t.Errorf("ClassifyError(%q, %q) should be retryable", tt.errStr, tt.host)
		}
	}

	nonRetryable := []struct {
		errStr string
		host   string
	}{
		{"Permission denied", "10.0.0.1"},
		{"Host key verification failed", "10.0.0.1"},
	}
	for _, tt := range nonRetryable {
		sshErr := ClassifyError(tt.errStr, tt.host, "")
		if sshErr.Retryable {
			t.Errorf("ClassifyError(%q, %q) should NOT be retryable", tt.errStr, tt.host)
		}
	}
}

func TestClassifyError_ErrorsIs(t *testing.T) {
	t.Parallel()

	sshErr := ClassifyError("Permission denied", "example.com", "")
	if !errors.Is(sshErr, ErrAuth) {
		t.Error("ClassifyError for 'Permission denied' should unwrap to ErrAuth")
	}

	sshErr2 := ClassifyError("i/o timeout", "10.0.0.1", "")
	if !errors.Is(sshErr2, ErrTimeout) {
		t.Error("ClassifyError for 'i/o timeout' on IPv4 should unwrap to ErrTimeout")
	}
}

func TestIsIPv6(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "ipv4 address", host: "192.168.1.1", want: false},
		{name: "ipv6 full", host: "2001:0db8:85a3:0000:0000:8a2e:0370:7334", want: true},
		{name: "ipv6 compressed", host: "2001:db8::1", want: true},
		{name: "ipv6 loopback", host: "::1", want: true},
		{name: "hostname", host: "example.com", want: false},
		{name: "empty string", host: "", want: false},
		{name: "ipv4 mapped ipv6", host: "::ffff:192.168.1.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIPv6(tt.host)
			if got != tt.want {
				t.Errorf("IsIPv6(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
