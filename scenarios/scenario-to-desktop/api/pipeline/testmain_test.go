package pipeline

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

func TestMain(m *testing.M) {
	// Pipeline tests use explicit legacy fallback fixtures. Keep them isolated
	// from the developer's credential store while production resolves the
	// scoped deployment principal through the authority.
	_ = os.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "1")
	// Host-requirement resolution reads operator state through storage.
	os.Exit(storagetest.RunWithIsolatedStorage(m))
}
