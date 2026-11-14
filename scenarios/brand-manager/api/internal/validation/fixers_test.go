package validation

import (
	"os"
	"path/filepath"
	"testing"
)

// --- autofix_available ⇔ fixer-exists invariant ----------------------------

// TestAutofixAvailableBiconditional is the structural guard the plan requires:
// for every finding a scan emits, AutofixAvailable must be true exactly when
// BuildFixCandidates for that one rule yields a candidate, and false when it
// yields none. This makes the advertised flag and the implemented fixer provably
// in lockstep.
func TestAutofixAvailableBiconditional(t *testing.T) {
	for _, root := range []string{minimalUIScenario(t), fullyBrandedScenario(t), residueScenario(t)} {
		res := ScanScenario("x", root)
		for _, f := range res.Findings {
			cands, _, err := BuildFixCandidates(root, []string{f.RuleID}, false)
			if err != nil {
				t.Fatalf("%s: BuildFixCandidates: %v", f.RuleID, err)
			}
			switch {
			case f.AutofixAvailable && len(cands) != 1:
				t.Fatalf("rule %s advertises autofix but BuildFixCandidates returned %d candidates", f.RuleID, len(cands))
			case !f.AutofixAvailable && len(cands) != 0:
				t.Fatalf("rule %s advertises no autofix but BuildFixCandidates returned %d candidates", f.RuleID, len(cands))
			}
		}
	}
}

// minimalUIScenario has a UI surface but only a display name — so it fires a wide
// spread of UI rules with a mix of fixable and guidance-only outcomes.
func minimalUIScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Buy widgets fast."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>Widget Shop</title></head><body></body></html>`)
	return root
}

func residueScenario(t *testing.T) string {
	t.Helper()
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", replaceFirst(compliantIndexHTML, "<title>Widget Shop</title>", "<title>Vite + React</title>"))
	writeFile(t, root, "ui/public/vite.svg", "<svg/>")
	return root
}

// TestEveryFixerRuleExistsInRegistry guards that fixerByID never references a
// rule that the scan does not emit (a fixer for a non-existent rule could never
// be advertised, hiding a typo).
func TestEveryFixerRuleExistsInRegistry(t *testing.T) {
	known := map[string]bool{}
	for _, s := range specs {
		known[s.id] = true
	}
	for id := range fixerByID {
		if !known[id] {
			t.Fatalf("fixerByID has %q with no matching rule in specs", id)
		}
	}
}

// --- color-system create-only fixer ----------------------------------------

func TestColorSystemFixerCreatesTokensWhenAbsentAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/index.html", "<html><head></head></html>") // establish the UI surface
	if _, ok := findingByRule(ScanScenario("x", root), "has-color-system"); !ok {
		t.Fatal("precondition: expected has-color-system finding")
	}

	cands, _, err := BuildFixCandidates(root, []string{"has-color-system"}, false)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(cands) != 1 || cands[0].Applied {
		t.Fatalf("preview candidates = %+v", cands)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(designSystemCSSRel))); !os.IsNotExist(err) {
		t.Fatal("preview must not write the token file")
	}

	applied, _, err := BuildFixCandidates(root, []string{"has-color-system"}, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 || !applied[0].Applied {
		t.Fatalf("apply candidates = %+v", applied)
	}
	if _, ok := findingByRule(ScanScenario("x", root), "has-color-system"); ok {
		t.Fatal("has-color-system finding should clear after ApplyFix")
	}

	again, _, err := BuildFixCandidates(root, []string{"has-color-system"}, true)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("re-apply should propose no candidates, got %+v", again)
	}
}

func TestNonFixableRuleReportsMessage(t *testing.T) {
	root := minimalUIScenario(t)
	cands, messages, err := BuildFixCandidates(root, []string{"has-logo"}, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("has-logo should yield no candidates, got %+v", cands)
	}
	if len(messages) == 0 {
		t.Fatal("expected a message explaining has-logo has no deterministic fixer")
	}
}

// --- self-contained fixer round-trips --------------------------------------

// assertRoundTrip verifies the full preview→apply→clear→idempotent cycle for a
// fixer whose rule should fully clear after one apply.
func assertRoundTrip(t *testing.T, root, ruleID string) {
	t.Helper()
	f := mustFire(t, root, ruleID)
	if !f.AutofixAvailable {
		t.Fatalf("%s should advertise an autofix on this fixture", ruleID)
	}
	cands, _, err := BuildFixCandidates(root, []string{ruleID}, false)
	if err != nil || len(cands) != 1 {
		t.Fatalf("%s preview: cands=%+v err=%v", ruleID, cands, err)
	}
	if cands[0].After == "" {
		t.Fatalf("%s preview candidate has empty After", ruleID)
	}
	if cands[0].Applied {
		t.Fatalf("%s preview must not be applied", ruleID)
	}
	applied, _, err := BuildFixCandidates(root, []string{ruleID}, true)
	if err != nil || len(applied) != 1 || !applied[0].Applied {
		t.Fatalf("%s apply: applied=%+v err=%v", ruleID, applied, err)
	}
	mustNotFire(t, root, ruleID)
	again, _, err := BuildFixCandidates(root, []string{ruleID}, true)
	if err != nil {
		t.Fatalf("%s re-apply: %v", ruleID, err)
	}
	if len(again) != 0 {
		t.Fatalf("%s re-apply should be a no-op, got %+v", ruleID, again)
	}
}

func TestFaviconFixerRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Buy widgets fast."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>Widget Shop</title></head></html>`)
	writeFile(t, root, "ui/public/logo.svg", "<svg/>")
	assertRoundTrip(t, root, "has-favicon")
}

func TestFaviconFixerUnavailableWithoutLogo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Buy widgets fast."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>Widget Shop</title></head></html>`)
	f := mustFire(t, root, "has-favicon")
	if f.AutofixAvailable {
		t.Fatal("has-favicon must not advertise an autofix when there is no logo to derive from")
	}
}

func TestThemeColorPresentFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="theme-color"`))
	assertRoundTrip(t, root, "theme-color-present")
}

func TestStandaloneFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="apple-mobile-web-app-capable"`))
	assertRoundTrip(t, root, "standalone-capable")
}

func TestOpenGraphFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `property="og:title"`))
	assertRoundTrip(t, root, "open-graph")
}

func TestTwitterCardFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="twitter:card"`))
	assertRoundTrip(t, root, "twitter-card")
}

func TestSafeAreaFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />`)
	writeFile(t, root, "ui/index.html", html)
	assertRoundTrip(t, root, "ios-statusbar-safe-area")
}

func TestNameConsistencyFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", replaceFirst(compliantIndexHTML, "<title>Widget Shop</title>", "<title>Wgt</title>"))
	assertRoundTrip(t, root, "name-consistency")
}

func TestThemeColorConsistencyFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/index.html", replaceFirst(compliantIndexHTML,
		`<meta name="theme-color" content="#1d4ed8" />`, `<meta name="theme-color" content="#ff0000" />`))
	assertRoundTrip(t, root, "theme-color-consistency")
}

func TestColorSchemeFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --font-sans: Inter, sans-serif;
}
.dark { --color-background: #0b1220; --color-foreground: #e2e8f0; }`)
	writeFile(t, root, "ui/index.html", removeLine(compliantIndexHTML, `name="color-scheme"`))
	assertRoundTrip(t, root, "color-scheme-declared")
}

func TestTemplateResidueFixerRoundTrip(t *testing.T) {
	root := residueScenario(t)
	assertRoundTrip(t, root, "no-template-residue")
}

func TestManifestCompletenessFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/public/site.webmanifest", `{
  "theme_color": "#1d4ed8", "background_color": "#ffffff",
  "icons": [
    {"src": "icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable"},
    {"src": "icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"}
  ]
}`)
	assertRoundTrip(t, root, "manifest-completeness")
}
