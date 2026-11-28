package gates

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates storage: gates read and promote the asset stamp report
// under the scenario's state root, which would otherwise be the operator's.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
