package checks_test

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestMockTLSDialerGetCertificateError(t *testing.T) {
	dialer := testutil.NewMockTLSDialer()
	dialer.Errors["bad.host:443"] = testutil.ErrConnectionRefused

	_, err := dialer.GetCertificate("bad.host:443")

	if err != testutil.ErrConnectionRefused {
		t.Errorf("error = %v, want %v", err, testutil.ErrConnectionRefused)
	}
}

// TestMockTLSDialerGetCertificateDefault verifies default response
