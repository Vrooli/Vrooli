package repocontractcheck

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunPassesAgainstLiveRepo(t *testing.T) {
	report, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Success {
		t.Fatalf("report.Success = false, checks = %+v", report.Checks)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected checks to be populated")
	}
}

func TestRunRequiresRoot(t *testing.T) {
	if _, err := Run(""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
