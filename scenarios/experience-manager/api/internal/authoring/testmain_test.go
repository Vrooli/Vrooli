package authoring

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates storage: rendering writes wireframe artifacts under the
// scenario's storage root, which would otherwise be the operator's live cache.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
