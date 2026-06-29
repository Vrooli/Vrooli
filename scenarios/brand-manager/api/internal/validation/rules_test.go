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

// compliantTokens is a full, WCAG-AA-passing, light-only color + typography set
// (no dark block, so dark-mode/color-scheme rules stay silent).
const compliantTokens = `:root {
  --color-background: #ffffff;
  --color-surface: #f1f5f9;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --color-primary-foreground: #ffffff;
  --color-accent: #0e7490;
  --font-sans: system-ui, sans-serif;
  color-scheme: light;
}`

// compliantIndexHTML declares every head surface the rule set checks, all
// agreeing with the display name "Widget Shop".
const compliantIndexHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="apple-touch-icon" href="/apple-icon-180.png" />
    <link rel="manifest" href="/site.webmanifest" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="Buy widgets fast." />
    <meta name="theme-color" content="#1d4ed8" />
    <meta name="color-scheme" content="light" />
    <meta name="application-name" content="Widget Shop" />
    <meta name="apple-mobile-web-app-title" content="Widget Shop" />
    <meta name="mobile-web-app-capable" content="yes" />
    <meta name="apple-mobile-web-app-capable" content="yes" />
    <meta property="og:type" content="website" />
    <meta property="og:title" content="Widget Shop" />
    <meta property="og:description" content="Buy widgets fast." />
    <meta property="og:site_name" content="Widget Shop" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content="Widget Shop" />
    <meta name="twitter:description" content="Buy widgets fast." />
    <title>Widget Shop</title>
  </head>
  <body><div id="root"></div></body>
</html>`

const compliantManifest = `{
  "name": "Widget Shop",
  "short_name": "Widget Shop",
  "description": "Buy widgets fast.",
  "theme_color": "#1d4ed8",
  "background_color": "#ffffff",
  "display": "standalone",
  "start_url": ".",
  "scope": ".",
  "id": "/",
  "icons": [
    {"src": "icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable"},
    {"src": "icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"}
  ]
}`

// fullyBrandedScenario builds a scenario tree that should pass every rule.
func fullyBrandedScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Buy widgets fast."}}`)
	writeFile(t, root, "ui/src/design-tokens.css", compliantTokens)
	writeFile(t, root, "ui/index.html", compliantIndexHTML)
	writeFile(t, root, "ui/public/site.webmanifest", compliantManifest)
	writeFile(t, root, "ui/public/logo.svg", "<svg/>")
	writeFile(t, root, "ui/public/favicon.svg", "<svg/>")
	writeFile(t, root, "ui/public/apple-icon-180.png", "PNGDATAOPAQUE")
	// The manifest declares these icons; a fully-branded scenario ships the files
	// (non-decodable stub bytes keep the dimension check best-effort/silent).
	writeFile(t, root, "ui/public/icon-192.png", "PNGDATA-192")
	writeFile(t, root, "ui/public/icon-512.png", "PNGDATA-512")
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

func mustFire(t *testing.T, root, ruleID string) Finding {
	t.Helper()
	f, ok := findingByRule(ScanScenario("widget-shop", root), ruleID)
	if !ok {
		t.Fatalf("expected %s finding", ruleID)
	}
	return f
}

func mustNotFire(t *testing.T, root, ruleID string) {
	t.Helper()
	if _, ok := findingByRule(ScanScenario("widget-shop", root), ruleID); ok {
		t.Fatalf("did not expect %s finding", ruleID)
	}
}

func TestScanFullyBrandedScenarioIsClean(t *testing.T) {
	res := ScanScenario("widget-shop", fullyBrandedScenario(t))
	if len(res.Findings) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(res.Findings), res.Findings)
	}
}

// --- surface-conditional ---------------------------------------------------

func TestNonUIScenarioSkipsUIRules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"svc","displayName":"Service","description":"A real service."}}`)
	res := ScanScenario("svc", root)
	for _, f := range res.Findings {
		if f.RuleID != "has-display-name" && f.RuleID != "api-branding" {
			t.Fatalf("UI/CLI rule %s should be skipped without those surfaces", f.RuleID)
		}
	}
}

// --- identity --------------------------------------------------------------

func TestHasDisplayNamePlaceholderFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"widget-shop","description":"Buy widgets fast."}}`)
	f := mustFire(t, root, "has-display-name")
	if f.Severity != SeverityError {
		t.Fatalf("display-name severity = %q, want error", f.Severity)
	}
}

func TestServiceJsonMissingFiresDisplayName(t *testing.T) {
	root := t.TempDir()
	if _, ok := findingByRule(ScanScenario("empty", root), "has-display-name"); !ok {
		t.Fatal("expected has-display-name finding when service.json is absent")
	}
}

// --- visual system ---------------------------------------------------------

func TestColorSystemMissingTokensFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root { --color-background: #fff; }`)
	f := mustFire(t, root, "has-color-system")
	if f.Severity != SeverityWarning {
		t.Fatalf("color-system severity = %q, want warning", f.Severity)
	}
	// An existing-but-incomplete token file is NOT create-only fixable, so the
	// flag must be honest: no advertised fix it cannot perform.
	if f.AutofixAvailable {
		t.Fatal("incomplete (present) token file must not advertise an autofix")
	}
}

func TestTypographyMissingFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
}`)
	f := mustFire(t, root, "has-typography")
	if f.Severity != SeverityInfo {
		t.Fatalf("typography severity = %q, want info", f.Severity)
	}
}

func TestLogoAndFaviconMissingFire(t *testing.T) {
	root := fullyBrandedScenario(t)
	_ = os.Remove(filepath.Join(root, "ui/public/logo.svg"))
	_ = os.Remove(filepath.Join(root, "ui/public/favicon.svg"))
	_ = os.Remove(filepath.Join(root, "ui/public/apple-icon-180.png"))
	writeFile(t, root, "ui/index.html", "<html><head></head></html>")
	mustFire(t, root, "has-logo")
	mustFire(t, root, "has-favicon")
}

func TestFaviconViaIndexHTMLLinkPasses(t *testing.T) {
	root := fullyBrandedScenario(t)
	_ = os.Remove(filepath.Join(root, "ui/public/favicon.svg"))
	mustNotFire(t, root, "has-favicon") // apple-touch-icon link still references an icon
}

func TestWCAGContrastFailureFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #cccccc;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
}`)
	f := mustFire(t, root, "wcag-aa-contrast")
	if f.Evidence["failures"] == nil {
		t.Fatalf("expected contrast failure evidence, got %+v", f.Evidence)
	}
}

func TestPrimaryForegroundContrastPairChecked(t *testing.T) {
	root := fullyBrandedScenario(t)
	// White-ish label on a light primary => button text fails AA.
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #93c5fd;
  --color-primary-foreground: #ffffff;
  --font-sans: Inter, sans-serif;
}`)
	f := mustFire(t, root, "wcag-aa-contrast")
	failures, _ := f.Evidence["failures"].([]map[string]any)
	found := false
	for _, fl := range failures {
		if fl["pair"] == "primary-foreground-on-primary" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected primary-foreground-on-primary failure, got %+v", f.Evidence)
	}
}

func TestBrandMarkersMissingFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/app.css", ".a{color:red}")
	f := mustFire(t, root, "brand-markers-applied")
	if f.Severity != SeverityInfo {
		t.Fatalf("markers severity = %q, want info", f.Severity)
	}
	// Brand-projection without an assigned brand is honestly guidance-only.
	if f.AutofixAvailable {
		t.Fatal("brand-markers-applied must not advertise an autofix in the served (brandless) path")
	}
}

// --- dark mode + color-scheme ----------------------------------------------

func TestDarkModeContrastFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
  color-scheme: light;
}
.dark {
  --color-background: #0b1220;
  --color-foreground: #1e293b;
}`)
	f := mustFire(t, root, "dark-mode-contrast")
	if f.Evidence["scheme"] != "dark" {
		t.Fatalf("expected dark scheme evidence, got %+v", f.Evidence)
	}
}

func TestColorSchemeDeclaredFiresWhenDarkBlockUndeclared(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
}
.dark {
  --color-background: #0b1220;
  --color-foreground: #e2e8f0;
}`)
	// index.html in the fixture declares color-scheme; remove it so the rule fires.
	writeFile(t, root, "ui/index.html", "<html><head><title>Widget Shop</title></head></html>")
	mustFire(t, root, "color-scheme-declared")
}

// --- consistency + residue -------------------------------------------------

func TestNameConsistencyFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := compliantIndexHTML
	writeFile(t, root, "ui/index.html", replaceFirst(html, "<title>Widget Shop</title>", "<title>Wgt Shop</title>"))
	f := mustFire(t, root, "name-consistency")
	if f.Evidence["mismatches"] == nil {
		t.Fatalf("expected mismatch evidence, got %+v", f.Evidence)
	}
}

func TestThemeColorConsistencyFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", replaceFirst(compliantIndexHTML,
		`<meta name="theme-color" content="#1d4ed8" />`, `<meta name="theme-color" content="#ff0000" />`))
	mustFire(t, root, "theme-color-consistency")
}

func TestTemplateResidueFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", replaceFirst(compliantIndexHTML,
		"<title>Widget Shop</title>", "<title>Vite + React</title>"))
	writeFile(t, root, "ui/public/vite.svg", "<svg/>")
	f := mustFire(t, root, "no-template-residue")
	if f.Evidence["residues"] == nil {
		t.Fatalf("expected residue evidence, got %+v", f.Evidence)
	}
}

// --- PWA / mobile ----------------------------------------------------------

func TestThemeColorPresentFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="theme-color"`))
	mustFire(t, root, "theme-color-present")
}

func TestStandaloneCapableFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="apple-mobile-web-app-capable"`))
	mustFire(t, root, "standalone-capable")
}

func TestIOSStatusBarSafeAreaFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />`)
	writeFile(t, root, "ui/index.html", html)
	mustFire(t, root, "ios-statusbar-safe-area")
}

func TestManifestCompletenessFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/site.webmanifest", `{"name":"Widget Shop"}`)
	f := mustFire(t, root, "manifest-completeness")
	if f.Evidence["missing"] == nil {
		t.Fatalf("expected missing-keys evidence, got %+v", f.Evidence)
	}
}

// --- social ----------------------------------------------------------------

func TestOpenGraphFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `property="og:title"`))
	mustFire(t, root, "open-graph")
}

func TestTwitterCardFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="twitter:card"`))
	mustFire(t, root, "twitter-card")
}

// --- asset validity --------------------------------------------------------

func TestAssetValidityMaskableMissingFires(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/site.webmanifest", `{
  "name": "Widget Shop", "short_name": "Widget Shop", "description": "Buy widgets fast.",
  "theme_color": "#1d4ed8", "background_color": "#ffffff", "display": "standalone",
  "start_url": ".", "id": "/",
  "icons": [{"src": "icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"}]
}`)
	f := mustFire(t, root, "asset-validity")
	issues, _ := f.Evidence["issues"].(map[string]any)
	if issues["no_maskable_icon"] == nil {
		t.Fatalf("expected no_maskable_icon issue, got %+v", f.Evidence)
	}
}

// --- CLI / API -------------------------------------------------------------

func TestCLIBrandingFiresWithoutDisplayName(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "cli/manifest.json", `{"name":"widget-shop","description":"Template scenario CLI surface."}`)
	mustFire(t, root, "cli-branding")
}

func TestCLIBrandingSilentWithDisplayName(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "cli/manifest.json", `{"name":"widget-shop","description":"Widget Shop command surface."}`)
	mustNotFire(t, root, "cli-branding")
}

func TestAPIBrandingFiresOnTemplateDescription(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Template scenario for the notes domain."}}`)
	mustFire(t, root, "api-branding")
}

// --- small string helpers for fixtures -------------------------------------

func replaceFirst(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// removeLine drops the first line containing marker.
func removeLine(s, marker string) string {
	lines := splitLines(s)
	out := make([]string, 0, len(lines))
	dropped := false
	for _, l := range lines {
		if !dropped && indexOf(l, marker) >= 0 {
			dropped = true
			continue
		}
		out = append(out, l)
	}
	return joinLines(out)
}

func injectBeforeHeadCloseRaw(s, line string) string {
	return replaceFirst(s, "  </head>", "    "+line+"\n  </head>")
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
