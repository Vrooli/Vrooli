package checks

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates contract-routed storage: the checks resolve test-genie's
// evidence root, which would otherwise be the operator's live evidence.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
