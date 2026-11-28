package fleet

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates contract-routed storage so fleet checks cannot read the
// operator's live requirements-sync evidence during a test.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
