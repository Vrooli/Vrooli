package credentials

import (
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// TestMain isolates per-install storage for the whole package. Credential
// inventory honors resource enablement through operator state, which resolves
// from the environment rather than the temporary repository a test builds, so
// without isolation the operator's live choices override the fixture.
func TestMain(m *testing.M) {
	os.Exit(testenv.RunWithIsolatedStorage(m))
}
