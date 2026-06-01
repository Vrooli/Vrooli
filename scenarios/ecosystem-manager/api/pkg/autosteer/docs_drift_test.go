package autosteer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenDocPhrases are stale claims the controller docs must never reassert
// (EM-P5 anti-drift). DTV and test-genie are wired; the loop's degraded-gate
// policy is proceed-cap-flag, not "barred"/"deferred". The list is scoped to
// specific stale phrasings (not every DTV mention) so it does not fire on
// legitimately-deferred work elsewhere (e.g. the Connect-RPC migration). Matched
// case-insensitively.
var forbiddenDocPhrases = []string{
	"not yet wired",
	"not wired.",
	"remain a p2 target",
	"remains a p2 target",
	"p2 dtv gap remains",
	"not wired to development-toolchain-validator",
	"wire development-toolchain-validator into the injected",
	"dtv trust gate | needs",
}

// findControllerDocsDir locates the scenario's docs/ tree from the test's
// working directory. Returns "" (and the test skips) when unreachable.
func findControllerDocsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "docs", "concepts", "CONTROL-MODEL.md")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, "docs")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// TestDocsDoNotReassertDTVUnwired is the EM-P5 anti-drift guard: it walks every
// markdown file under docs/ and fails if any reintroduces a phrase that claims
// DTV / test-genie is unwired or that the controller defers the DTV gate. This
// keeps the pinned mental model honest as future agents edit the docs.
func TestDocsDoNotReassertDTVUnwired(t *testing.T) {
	docsDir := findControllerDocsDir(t)
	if docsDir == "" {
		t.Skip("controller docs/ tree not reachable from this checkout; anti-drift guard skipped")
	}

	var scanned int
	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		scanned++
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: read: %v", path, err)
			return nil
		}
		lower := strings.ToLower(string(raw))
		for _, phrase := range forbiddenDocPhrases {
			if strings.Contains(lower, phrase) {
				rel, _ := filepath.Rel(docsDir, path)
				t.Errorf("docs/%s reasserts the stale phrase %q — DTV/test-genie are wired and the gate is proceed-cap-flag; fix the doc (see CONTROL-MODEL.md)", rel, phrase)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", docsDir, err)
	}
	if scanned == 0 {
		t.Fatalf("no markdown scanned under %s — guard would silently pass", docsDir)
	}
	t.Logf("anti-drift guard: %d markdown files scanned, no stale DTV/test-genie claims", scanned)
}
