package seeds

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

func TestMain(m *testing.M) {
	if handled, code := RunWorker(os.Args[1:], os.Stderr); handled {
		os.Exit(code)
	}
	os.Exit(storagetest.RunWithIsolatedStorage(m))
}
