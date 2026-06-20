package focusvisible

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFocusVisibleStylesUsesScenarioGlobalPolicy(t *testing.T) {
	root := t.TempDir()
	stylesPath := filepath.Join(root, "ui", "src", "styles.css")
	if err := os.MkdirAll(filepath.Dir(stylesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stylesPath, []byte(`
:where(a, button, input, textarea, select, [tabindex]):focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 2px;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	componentPath := filepath.Join(root, "ui", "src", "components", "LocaleSwitcher.tsx")
	source := []byte(`export function LocaleSwitcher() {
  return <button type="button" className="rounded-md px-2">EN</button>;
}`)

	violations := CheckFocusVisibleStyles(source, componentPath)
	if len(violations) != 0 {
		t.Fatalf("expected global focus-visible policy to satisfy component, got %+v", violations)
	}
}

func TestCheckFocusVisibleStylesFlagsInteractiveWithoutLocalOrGlobalPolicy(t *testing.T) {
	source := []byte(`export function LocaleSwitcher() {
  return <button type="button" className="rounded-md px-2">EN</button>;
}`)

	violations := CheckFocusVisibleStyles(source, filepath.Join(t.TempDir(), "ui", "src", "components", "LocaleSwitcher.tsx"))
	if len(violations) != 1 {
		t.Fatalf("expected one focus-visible violation, got %d: %+v", len(violations), violations)
	}
}
