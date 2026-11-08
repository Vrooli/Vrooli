package resources

import (
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// TestMain isolates per-install storage for the whole package. Controller
// reads adopt repository resource choices into operator state, which resolves
// from the environment rather than the temporary repository a test passes in;
// without isolation these tests seeded fixture resources into the operator's
// live state and then failed against them.
func TestMain(m *testing.M) {
	os.Exit(testenv.RunWithIsolatedStorage(m))
}
