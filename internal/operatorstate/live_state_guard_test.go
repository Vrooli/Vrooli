package operatorstate

import (
	"context"
	"os/user"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// A temporary RepoRoot does not place the document, so without the backstop
// this test would read (and on first read, seed) the operator's real file.
func TestPathRefusesTheLiveOperatorStateUnderTest(t *testing.T) {
	account, err := user.Current()
	if err != nil {
		t.Skipf("account home unavailable: %v", err)
	}
	t.Setenv("HOME", account.HomeDir)
	for _, key := range []string{"VROOLI_STORAGE_ROOT", "VROOLI_STATE_ROOT"} {
		t.Setenv(key, "")
	}
	service := New(Config{RepoRoot: t.TempDir()})
	if _, err := service.Path(context.Background()); err == nil || !strings.Contains(err.Error(), "live storage") {
		t.Fatalf("Path() error = %v, want the live-state refusal", err)
	}
	if _, err := service.Load(context.Background()); err == nil {
		t.Fatal("Load() read the operator's live state from a test")
	}

	root := testenv.IsolateStorage(t)
	path, err := service.Path(context.Background())
	if err != nil {
		t.Fatalf("Path() with isolated storage: %v", err)
	}
	if !strings.HasPrefix(path, root) {
		t.Fatalf("Path() = %q, want beneath isolated root %q", path, root)
	}
}
