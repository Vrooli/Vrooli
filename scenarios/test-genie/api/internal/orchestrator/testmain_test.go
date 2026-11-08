package orchestrator

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain runs the package under a private HOME. Under go test the storage
// resolver refuses the operator's live runtime home; these tests resolve
// default-layout paths (some through an injected environment, some beneath a
// HOME they set), which a global storage root would override.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithPrivateHome(m)) }
