package staleness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckerDisabled(t *testing.T) {
	if NewChecker(CheckerConfig{Disabled: true}).CheckAndMaybeRebuild() {
		t.Fatal("disabled checker must not restart")
	}
}

func TestCheckerLifecycleManifestIsVerificationOnly(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "api")
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	var logs string
	checker := NewChecker(CheckerConfig{
		BinaryPath:       binary,
		APIDir:           root,
		LifecycleManaged: true,
		Logger: func(format string, args ...interface{}) {
			logs += format
		},
	})
	if checker.CheckAndMaybeRebuild() {
		t.Fatal("manifest verification must never re-exec")
	}
	if logs != "" {
		t.Fatalf("missing manifest should defer quietly, got %q", logs)
	}
}

func TestCheckerDoesNotRequireGoModule(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "api")
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if NewChecker(CheckerConfig{BinaryPath: binary, APIDir: root}).CheckAndMaybeRebuild() {
		t.Fatal("unmanaged verification must not restart")
	}
}
