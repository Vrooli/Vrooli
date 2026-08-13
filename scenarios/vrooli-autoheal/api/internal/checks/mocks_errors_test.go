package checks_test

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestMockTLSDialerGetCertificateDefault(t *testing.T) {
	dialer := testutil.NewMockTLSDialer()
	dialer.DefaultCert = &testutil.CertInfo{
		Subject:   "default.com",
		DaysUntil: 60,
	}

	cert, err := dialer.GetCertificate("any.host:443")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cert.Subject != "default.com" {
		t.Errorf("Subject = %q, want %q", cert.Subject, "default.com")
	}
}

// TestMockTLSDialerGetCertificateFallback verifies final fallback
func TestMockTLSDialerGetCertificateFallback(t *testing.T) {
	dialer := testutil.NewMockTLSDialer()
	// No configured response, no default cert

	cert, err := dialer.GetCertificate("any.host:443")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cert.Subject != "example.com" {
		t.Errorf("Subject = %q, want %q (fallback)", cert.Subject, "example.com")
	}
	if cert.DaysUntil != 90 {
		t.Errorf("DaysUntil = %d, want 90 (fallback)", cert.DaysUntil)
	}
}

// TestMockTLSDialerGetCertificateDefaultError verifies default error
func TestMockTLSDialerGetCertificateDefaultError(t *testing.T) {
	dialer := testutil.NewMockTLSDialer()
	dialer.DefaultError = testutil.ErrTimeout

	_, err := dialer.GetCertificate("any.host:443")

	if err != testutil.ErrTimeout {
		t.Errorf("error = %v, want %v", err, testutil.ErrTimeout)
	}
}

// =============================================================================
// Common Error Variables Tests
// =============================================================================

// TestCommonErrors verifies error variables are defined
func TestCommonErrors(t *testing.T) {
	errors := map[string]error{
		"testutil.ErrConnectionRefused": testutil.ErrConnectionRefused,
		"testutil.ErrTimeout":           testutil.ErrTimeout,
		"testutil.ErrCommandNotFound":   testutil.ErrCommandNotFound,
		"testutil.ErrDNSLookupFailed":   testutil.ErrDNSLookupFailed,
		"testutil.ErrFileNotFound":      testutil.ErrFileNotFound,
		"testutil.ErrPermissionDenied":  testutil.ErrPermissionDenied,
	}

	for name, err := range errors {
		if err == nil {
			t.Errorf("%s should not be nil", name)
		}
		if err.Error() == "" {
			t.Errorf("%s should have non-empty message", name)
		}
	}
}
