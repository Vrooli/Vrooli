package pipeline

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Pipeline tests use explicit LPBS_SERVICE_SECRET fixtures. Keep them
	// isolated from the developer's real credential store while production
	// code continues to resolve the shared identity through the authority.
	_ = os.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "1")
	os.Exit(m.Run())
}
