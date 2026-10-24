package ssh

import (
	"errors"
	"testing"
)

func TestClassifyErrorMapsToCategoryAndStatus(t *testing.T) {
	cases := []struct {
		name       string
		errStr     string
		host       string
		wantCat    error
		wantStatus string
		retryable  bool
	}{
		{"host key changed", "Host key verification failed.", "h", ErrHostKey, StatusHostKeyChanged, false},
		{"permission denied", "Permission denied (publickey).", "h", ErrAuth, StatusAuthFailed, false},
		{"password auth", "ssh: unable to authenticate", "h", ErrAuth, StatusAuthFailed, false},
		{"refused", "dial tcp: connection refused", "h", ErrUnreachable, StatusHostUnreachable, true},
		{"timeout", "i/o timeout", "h", ErrTimeout, StatusTimeout, true},
		{"ipv6 timeout", "i/o timeout", "2001:db8::1", ErrIPv6, StatusIPv6Unavailable, true},
		{"no route", "no route to host", "h", ErrUnreachable, StatusHostUnreachable, true},
		{"disk full", "no space left on device", "h", ErrDiskSpace, StatusDiskFull, false},
		{"dns", "could not resolve hostname foo", "h", ErrDNS, StatusDNSFailed, false},
		{"key format", "Load key: invalid format", "h", ErrKeyFormat, StatusKeyError, false},
		{"reset", "connection reset by peer", "h", ErrUnreachable, StatusHostUnreachable, true},
		{"too many", "Too many authentication failures", "h", ErrAuth, StatusAuthFailed, false},
		{"unknown", "some weird failure", "h", ErrCommand, StatusError, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyError(tc.errStr, tc.host, "hint")
			if !errors.Is(got, tc.wantCat) {
				t.Errorf("category = %v, want %v", got.Category, tc.wantCat)
			}
			if s := StatusFromError(got); s != tc.wantStatus {
				t.Errorf("status = %q, want %q", s, tc.wantStatus)
			}
			if got.Retryable != tc.retryable {
				t.Errorf("retryable = %v, want %v", got.Retryable, tc.retryable)
			}
		})
	}
}

func TestStatusFromErrorNilIsSuccess(t *testing.T) {
	if s := StatusFromError(nil); s != StatusSuccess {
		t.Errorf("nil status = %q, want %q", s, StatusSuccess)
	}
}
