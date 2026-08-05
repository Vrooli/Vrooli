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
	findings, err := ValidateTokens(liveRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		t.Fatalf("live token gate findings: %+v", findings)
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
	findings, err := ValidateFixtures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "catalog.fixture_adversarial" {
		t.Fatalf("findings = %+v", findings)
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
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Button.tsx"), []byte("export const Button = () => <button className=\"px-3\" />"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := ValidateTokens(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "catalog.tokens_literal" {
		t.Fatalf("findings = %+v", findings)
	}
}
