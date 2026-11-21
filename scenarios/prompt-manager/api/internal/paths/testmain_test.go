package paths

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates contract-routed storage for the package. Under go test the
// storage resolver refuses the operator's live runtime home, so code here that
// resolves per-install storage on its own needs a private root.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
