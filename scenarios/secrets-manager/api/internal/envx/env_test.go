package envx_test

import (
	"testing"

	"secrets-manager-api/internal/envx"
	"secrets-manager-api/internal/testutil/mocks"
)

func TestFakeReaderSatisfiesReaderAndReturnsConfiguredValues(t *testing.T) {
	reader := mocks.FakeEnv{"API_KEY": "test-value"}
	var seam envx.Reader = reader
	if got := seam.Getenv("API_KEY"); got != "test-value" {
		t.Fatalf("Getenv(API_KEY) = %q", got)
	}
	if got := seam.Getenv("MISSING"); got != "" {
		t.Fatalf("Getenv(MISSING) = %q", got)
	}
}
