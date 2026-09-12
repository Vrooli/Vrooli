package checks

import (
	"os"
	"path/filepath"
	"testing"

	"ui-health/internal/uiinterop"
)

func writeCheckFixture(t *testing.T, root, relPath, content string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", relPath, err)
	}
}

func TestA11yHarnessAcceptsSharedApiBaseCompanion(t *testing.T) {
	root := t.TempDir()
	writeCheckFixture(t, root, "ui/package.json", `{"devDependencies":{"axe-core":"4.10.2"}}`)
	result := checkA11yHarness(uiinterop.CheckContext{
		ScenarioRoot: root,
		TestSources: []uiinterop.SourceFile{
			{
				RelPath: "ui/src/App.a11y.test.tsx",
				Content: `import { expectNoA11yViolations } from "@vrooli/api-base/testing"; expectNoA11yViolations(document.body);`,
			},
		},
	})
	if !result.Passed {
		t.Fatalf("shared a11y companion result = %+v, want pass", result)
	}
}

func TestA11yHarnessStillRequiresA11yEvidence(t *testing.T) {
	root := t.TempDir()
	writeCheckFixture(t, root, "ui/package.json", `{"devDependencies":{"axe-core":"4.10.2"}}`)
	result := checkA11yHarness(uiinterop.CheckContext{
		ScenarioRoot: root,
		TestSources: []uiinterop.SourceFile{{
			RelPath: "ui/src/App.test.tsx",
			Content: `import { expectNoA11yViolations } from "@vrooli/api-base/testing";`,
		}},
	})
	if result.Passed || len(result.Violations) != 1 {
		t.Fatalf("missing a11y test result = %+v, want one violation", result)
	}
}
