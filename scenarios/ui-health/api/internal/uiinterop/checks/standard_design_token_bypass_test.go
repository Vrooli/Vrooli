package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestDesignTokenBypassChecksVendoredRequiredTokensWithoutDesignDeclaration(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "fixture")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "markdown-renderer")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"libraryId":"react-component-library:markdown-renderer","requiredTokens":["--color-surface-muted","--color-border"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := uiinterop.CheckContext{ScenarioRoot: scenarioRoot, Sources: []uiinterop.SourceFile{
		{RelPath: "ui/src/components/Markdown.tsx", Content: "// @vrooliComponentSource react-component-library:markdown-renderer\nexport const Markdown = () => null;"},
		{RelPath: "ui/src/styles.css", Content: ":root { --color-surface-muted: #111; }"},
	}}
	missing := checkDesignTokenBypass(ctx)
	if missing.Passed || missing.Skipped || len(missing.Violations) != 1 {
		t.Fatalf("missing-token result = %+v, want one failure", missing)
	}
	if got := missing.Violations[0].Description; got == "" || !strings.Contains(got, "--color-border") {
		t.Fatalf("missing-token description %q does not name token", got)
	}

	ctx.Sources[1].Content = ":root { --color-surface-muted: #111; --color-border: #222; }"
	passing := checkDesignTokenBypass(ctx)
	if !passing.Passed || passing.Skipped {
		t.Fatalf("defined-token result = %+v, want pass", passing)
	}
}
