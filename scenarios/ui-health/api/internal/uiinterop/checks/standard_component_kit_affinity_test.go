package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestComponentKitAffinityRejectsPrivateTokenWithoutNativeAffinity(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "fixture")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Card")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"libraryId":"react-component-library:Card","designStyles":[{"styleId":"vrooli-default","affinity":"native"},{"styleId":"vrooli-command-display","affinity":"compatible"}]}`
	if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	uiSourceDir := filepath.Join(scenarioRoot, "ui", "src", "components")
	if err := os.MkdirAll(uiSourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSourceDir, "Card.tsx"), []byte("// @vrooliComponentSource react-component-library:Card\nexport const Card = () => <div className=\"display-glow\" />;"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := uiinterop.CheckContext{ScenarioRoot: scenarioRoot}
	result := checkComponentKitAffinity(ctx)
	if result.Passed || result.Skipped || len(result.Violations) != 1 {
		t.Fatalf("result = %+v, want one affinity violation", result)
	}
	if !strings.Contains(result.Violations[0].Description, "vrooli-command-display") {
		t.Fatalf("description = %q, want kit id", result.Violations[0].Description)
	}
}

func TestComponentKitAffinityAcceptsNativePrivateToken(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "fixture")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Card")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"libraryId":"react-component-library:Card","designStyles":[{"styleId":"vrooli-command-display","affinity":"native"}]}`
	if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	uiSourceDir := filepath.Join(scenarioRoot, "ui", "src", "components")
	if err := os.MkdirAll(uiSourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSourceDir, "Card.tsx"), []byte("// @vrooliComponentSource react-component-library:Card\nexport const Card = () => <div className=\"display-glow\" />;"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := uiinterop.CheckContext{ScenarioRoot: scenarioRoot}
	result := checkComponentKitAffinity(ctx)
	if !result.Passed || result.Skipped || len(result.Violations) != 0 {
		t.Fatalf("result = %+v, want pass", result)
	}
}

func TestComponentKitAffinityRejectsDefaultSidebarWithoutNativeAffinity(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "fixture")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Shell")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"libraryId":"react-component-library:Shell","designStyles":[{"styleId":"vrooli-default","affinity":"compatible"}]}`
	if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	uiSourceDir := filepath.Join(scenarioRoot, "ui", "src", "components")
	if err := os.MkdirAll(uiSourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSourceDir, "Shell.tsx"), []byte("// @vrooliComponentSource react-component-library:Shell\nexport const Shell = () => <aside className=\"w-sidebar font-sans\" />;"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := checkComponentKitAffinity(uiinterop.CheckContext{ScenarioRoot: scenarioRoot})
	if result.Passed || result.Skipped || len(result.Violations) != 1 {
		t.Fatalf("result = %+v, want one default-kit affinity violation", result)
	}
	if !strings.Contains(result.Violations[0].Description, "vrooli-default") {
		t.Fatalf("description = %q, want default kit id", result.Violations[0].Description)
	}
}
