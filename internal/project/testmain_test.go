package project

import (
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// TestMain isolates per-install storage for the whole package. Operator state
// resolves from the environment rather than the temporary repository a test
// builds, so without isolation these tests read, and through first-read
// adoption wrote, the operator's live choices.
func TestMain(m *testing.M) {
	os.Exit(testenv.RunWithIsolatedStorage(m))
}
