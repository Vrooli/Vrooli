package validation

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates root/rel (with parent dirs) containing content.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// compliantTokens defines a full, WCAG-AA-passing color + typography token set.
const compliantTokens = `:root {
  --color-background: #ffffff;
  --color-surface: #f1f5f9;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
}`

// fullyBrandedScenario builds a scenario tree that should pass every rule.
func fullyBrandedScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop"}}`)
	writeFile(t, root, "ui/src/design-tokens.css", compliantTokens)
	writeFile(t, root, "ui/public/logo.svg", "<svg/>")
	writeFile(t, root, "ui/public/favicon.png", "x")
	writeFile(t, root, "ui/src/app.css", "/* brand-manager:primary */ .a{color:red}")
	return root
}

func findingByRule(res *ScanResult, ruleID string) (Finding, bool) {
	for _, f := range res.Findings {
		if f.RuleID == ruleID {
			return f, true
		}
	}
	return Finding{}, false
}

func TestScanFullyBrandedScenarioIsClean(t *testing.T) {
	res := ScanScenario("widget-shop", fullyBrandedScenario(t))
	if len(res.Findings) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(res.Findings), res.Findings)
	}
}

func TestHasDisplayNamePlaceholderFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	// Display name equal to the raw id is a placeholder.
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"widget-shop"}}`)
	res := ScanScenario("widget-shop", root)
	f, ok := findingByRule(res, "has-display-name")
	if !ok {
		t.Fatal("expected has-display-name finding")
	}
	if f.Severity != SeverityError {
		t.Fatalf("display-name severity = %q, want error", f.Severity)
	}
}

func TestHasDisplayNameBracketPlaceholderFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"[Outcome title]"}}`)
	res := ScanScenario("widget-shop", root)
	if _, ok := findingByRule(res, "has-display-name"); !ok {
		t.Fatal("expected has-display-name finding for bracket placeholder")
	}
}

func TestColorSystemMissingTokensFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root { --color-background: #fff; }`)
	res := ScanScenario("widget-shop", root)
	f, ok := findingByRule(res, "has-color-system")
	if !ok {
		t.Fatal("expected has-color-system finding")
	}
	if f.Severity != SeverityWarning || !f.AutofixAvailable {
		t.Fatalf("unexpected color-system finding: %+v", f)
	}
}

func TestTypographyMissingFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
}`)
	res := ScanScenario("widget-shop", root)
	f, ok := findingByRule(res, "has-typography")
	if !ok {
		t.Fatal("expected has-typography finding")
	}
	if f.Severity != SeverityInfo {
		t.Fatalf("typography severity = %q, want info", f.Severity)
	}
}

func TestLogoAndFaviconMissingFire(t *testing.T) {
	root := fullyBrandedScenario(t)
	_ = os.Remove(filepath.Join(root, "ui/public/logo.svg"))
	_ = os.Remove(filepath.Join(root, "ui/public/favicon.png"))
	writeFile(t, root, "ui/index.html", "<html><head></head></html>")
	res := ScanScenario("widget-shop", root)
	if _, ok := findingByRule(res, "has-logo"); !ok {
		t.Fatal("expected has-logo finding")
	}
	if _, ok := findingByRule(res, "has-favicon"); !ok {
		t.Fatal("expected has-favicon finding")
	}
}

func TestFaviconViaIndexHTMLLinkPasses(t *testing.T) {
	root := fullyBrandedScenario(t)
	_ = os.Remove(filepath.Join(root, "ui/public/favicon.png"))
	writeFile(t, root, "ui/index.html", `<html><head><link rel="icon" href="/x.png"/></head></html>`)
	res := ScanScenario("widget-shop", root)
	if _, ok := findingByRule(res, "has-favicon"); ok {
		t.Fatal("favicon referenced via index.html should satisfy the rule")
	}
}

func TestWCAGContrastFailureFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	// Light gray text on white => fails AA normal.
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #cccccc;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
}`)
	res := ScanScenario("widget-shop", root)
	f, ok := findingByRule(res, "wcag-aa-contrast")
	if !ok {
		t.Fatal("expected wcag-aa-contrast finding")
	}
	if f.Evidence["failures"] == nil {
		t.Fatalf("expected contrast failure evidence, got %+v", f.Evidence)
	}
}

func TestBrandMarkersMissingFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	// Replace the marked stylesheet with an unmarked one.
	writeFile(t, root, "ui/src/app.css", ".a{color:red}")
	res := ScanScenario("widget-shop", root)
	f, ok := findingByRule(res, "brand-markers-applied")
	if !ok {
		t.Fatal("expected brand-markers-applied finding")
	}
	if f.Severity != SeverityInfo {
		t.Fatalf("markers severity = %q, want info", f.Severity)
	}
}

func TestBrandMarkersViaManifestPasses(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/app.css", ".a{color:red}") // no CSS marker
	writeFile(t, root, "ui/manifest.json", `{"_brand":{"id":"abc"}}`)
	res := ScanScenario("widget-shop", root)
	if _, ok := findingByRule(res, "brand-markers-applied"); ok {
		t.Fatal("manifest _brand key should satisfy the markers rule")
	}
}

func TestServiceJsonMissingFiresDisplayName(t *testing.T) {
	root := t.TempDir()
	res := ScanScenario("empty", root)
	if _, ok := findingByRule(res, "has-display-name"); !ok {
		t.Fatal("expected has-display-name finding when service.json is absent")
	}
}
