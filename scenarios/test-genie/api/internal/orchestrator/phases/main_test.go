package phases

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"

	"test-genie/internal/orchestrator/phases/seeds"
)

func TestMain(m *testing.M) {
	if handled, code := seeds.RunWorker(os.Args[1:], os.Stderr); handled {
		os.Exit(code)
	}
	os.Exit(storagetest.RunWithIsolatedStorage(m))
}
