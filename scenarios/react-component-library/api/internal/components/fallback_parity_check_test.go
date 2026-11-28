package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The publish check rejects a draft whose fallback disagrees with the
// canonical vocabulary and accepts the same draft once the fallback is gone.
func TestDraftFallbackParityGuardsThePublishPath(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("templates/design/_base/tokens.css", ":root {\n  --space-md: 24px;\n}\n")
	library := filepath.Join(root, "scenarios", "react-component-library", "library")
	draft := "components/Panel/versions/1.0.1-draft.1"
	write("scenarios/react-component-library/library/"+draft+"/Panel.tsx", "const css = `gap: var(--space-md, 1rem);`;")

	verdict, message := draftFallbackParity(library, draft)
	if verdict != "failed" || !strings.Contains(message, "--space-md") || !strings.Contains(message, "24px") {
		t.Fatalf("disagreeing fallback: verdict=%s message=%q", verdict, message)
	}

	write("scenarios/react-component-library/library/"+draft+"/Panel.tsx", "const css = `gap: var(--space-md);`;")
	if verdict, message := draftFallbackParity(library, draft); verdict != "passed" {
		t.Fatalf("fallback-free draft: verdict=%s message=%q", verdict, message)
	}
}
