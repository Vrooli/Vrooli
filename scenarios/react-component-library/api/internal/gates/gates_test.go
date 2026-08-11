package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func liveRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
}

func TestLiveTokenGate(t *testing.T) {
	result, err := ValidateTokens(liveRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected < 170 {
		t.Fatalf("live token gate inspected %d active implementation files; expected the indexed active corpus", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("live token gate reported findings in the active corpus: %+v", result.Findings)
	}
}

func TestLiveAPIGateInspectsClosureImplementations(t *testing.T) {
	result, err := ValidateAPI(liveRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected == 0 {
		t.Fatal("live api gate inspected zero closure implementations")
	}
}

func TestAPIGateRejectsUndeclaredImplementationVocabulary(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	manifestDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	asset := `{"asset":{"id":"controls.button","kind":"component"},"api":{"variants":{"tone":["danger"]},"modes":["controlled"],"parts":["icon"]}}`
	if err := os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(asset), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "Button.tsx"), []byte(`export function Button() { return <button>save</button>; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateAPI(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 3 {
		t.Fatalf("result = %+v, want three vocabulary findings", result)
	}
}

func TestFixtureGateRejectsMissingAdversarialShape(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "fixtures")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "one.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"fixtures.one","kind":"fixture"},"fixture":{"dataShapes":["typical"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateFixtures(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_adversarial" {
		t.Fatalf("result = %+v", result)
	}
}

func TestTokenGateRejectsLiteralDimensionInImplementation(t *testing.T) {
	root := t.TempDir()
	for _, kit := range []string{"one", "two", "three"} {
		path := filepath.Join(root, "templates", "design", kit, "adapters", "react-vite-tailwind")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "tokens.css"), []byte(strings.Join([]string{
			"--space-3xs: 4px; --space-2xs: 8px; --space-xs: 12px; --space-sm: 16px; --space-md: 24px; --space-lg: 32px; --space-xl: 40px; --space-2xl: 48px;",
			"--text-display: x; --text-title: x; --text-heading: x; --text-body: x; --text-subheading: x; --text-body-sm: x; --text-label: x; --text-caption: x;",
			"--elev-flat: x; --elev-raised: x; --elev-overlay: x; --elev-modal: x; --layer-base: x; --layer-dropdown: x; --layer-sticky: x; --layer-overlay: x; --layer-modal: x; --layer-toast: x; --layer-tooltip: x; --border-hairline: x; --border-strong: x; --opacity-disabled: x; --opacity-muted: x; --opacity-scrim: x; --dur-instant: x;",
		}, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Button.tsx"), []byte("export const Button = () => <button className=\"px-3\" />"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateTokens(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.tokens_literal" {
		t.Fatalf("result = %+v", result)
	}
}

func TestLifecycleGateRejectsMissingCleanup(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(`export const Watcher = () => { if (typeof window !== "undefined") window.addEventListener("resize", () => {}); return null; };`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.lifecycle_cleanup" {
		t.Fatalf("result = %+v", result)
	}
}

func TestLifecycleGateIgnoresStoriesAndEffectScopedBrowserAccess(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `import { useEffect } from "react";
export const Watcher = () => {
  useEffect(() => {
    const timer = window.setTimeout(() => {}, 10);
    return () => window.clearTimeout(timer);
  }, []);
  return null;
};`
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "story.tsx"), []byte(`export const Story = () => window.setTimeout(() => {}, 10);`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want only runtime source inspected with no findings", result)
	}
}

func TestLifecycleGateRejectsRenderTimeBrowserAccess(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `export const Watcher = () => <output>{window.innerWidth}</output>;`
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.lifecycle_ssr" {
		t.Fatalf("result = %+v, want render-time SSR finding", result)
	}
}

func TestLifecycleGateAcceptsDocumentGuard(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "hooks", "useLocale", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `export function useLocale() {
  return typeof document === "undefined" ? "en" : document.documentElement.lang || "en";
}`
	if err := os.WriteFile(filepath.Join(source, "useLocale.ts"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want guarded document access to pass", result)
	}
}

func TestEveryGateRejectsZeroInspectedInputs(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name string
		run  func(string) (Result, error)
	}{
		{name: "api", run: ValidateAPI},
		{name: "tokens", run: ValidateTokens},
		{name: "lifecycle", run: ValidateLifecycle},
		{name: "fixtures", run: ValidateFixtures},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run(root)
			if err != nil {
				t.Fatal(err)
			}
			if result.Inspected != 0 || len(result.Findings) == 0 {
				t.Fatalf("result = %+v, want zero-input finding", result)
			}
		})
	}
}

func TestFixtureGateRejectsDataSourceWithoutTypeArgument(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "fixtures")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"kind":"catalog-asset","asset":{"id":"fixtures.one","kind":"fixture"},"fixture":{"dataShapes":["typical","failure"],"satisfies":{"capability":"data-source","typeArguments":[]}}}`
	if err := os.WriteFile(filepath.Join(assets, "one.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateFixtures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_data_source" {
		t.Fatalf("result = %+v", result)
	}
}
