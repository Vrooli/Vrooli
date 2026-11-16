package decisionpage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A page whose literal colours work in one scheme only is shown by the Hub in
// that scheme whatever the Hub's theme; the preview reads the page the Hub's
// way (pagescheme) and shows it the same way.
func TestThePreviewShowsALockedPageInItsOwnScheme(t *testing.T) {
	kit := t.TempDir()
	if err := os.WriteFile(filepath.Join(kit, "kit.css"), []byte(":root{--hub-fg:#0f172a;color:var(--hub-fg)}"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked, err := Preview([]byte(`<!doctype html><html><head><style>body{background:#fff}</style></head><body>x</body></html>`), kit, "dark")
	if err != nil || !strings.Contains(locked, `data-theme="light" data-hub-scheme="light"`) {
		t.Fatalf("a light-only page in the dark Hub = %v\n%s", err, locked)
	}
	follows, err := Preview([]byte(`<!doctype html><html><head></head><body style="background:var(--hub-surface)">x</body></html>`), kit, "dark")
	if err != nil || !strings.Contains(follows, `<html data-theme="dark">`) {
		t.Fatalf("a page that follows the Hub = %v\n%s", err, follows)
	}
}
