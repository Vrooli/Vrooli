package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreshMaintainerRejectsSymlinkedSafeAreaCSSDestination(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />`)
	writeFile(t, root, "ui/index.html", html)

	outside := t.TempDir()
	sentinelPath := filepath.Join(outside, "design-tokens.css")
	sentinel := []byte("outside sentinel: leave unchanged\n")
	if err := os.WriteFile(sentinelPath, sentinel, 0o600); err != nil {
		t.Fatalf("write outside sentinel: %v", err)
	}
	tokensPath := filepath.Join(root, "ui/src/design-tokens.css")
	if err := os.Remove(tokensPath); err != nil {
		t.Fatalf("remove scenario design tokens: %v", err)
	}
	if err := os.Symlink(sentinelPath, tokensPath); err != nil {
		t.Fatalf("symlink scenario design tokens to outside sentinel: %v", err)
	}

	_, _, err := BuildFixCandidates(root, []string{"ios-statusbar-safe-area"}, true)
	if err == nil {
		t.Fatal("BuildFixCandidates accepted symlinked safe-area CSS destination")
	}
	if !strings.Contains(err.Error(), "refuse symlink component") {
		t.Fatalf("expected symlink refusal error, got: %v", err)
	}

	after, readErr := os.ReadFile(sentinelPath)
	if readErr != nil {
		t.Fatalf("read outside sentinel after fixer: %v", readErr)
	}
	if string(after) != string(sentinel) {
		t.Fatalf("outside sentinel changed: got %q, want %q", after, sentinel)
	}
}
