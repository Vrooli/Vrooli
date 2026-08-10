package dependencyhealth

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateToolMacOSAcquisitionsEnumeratesRepositoryTools(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
	if err := validateToolMacOSAcquisitions(repoRoot); err != nil {
		t.Fatalf("repository tool acquisition validation failed: %v", err)
	}
}
