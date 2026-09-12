package validation

import "testing"

// rootAssetScenario builds a scenario that serves its branding/PWA/OG assets at
// the site root (not under /public/) and references them at the root — the
// pre-convention layout the public-asset-convention rule flags.
func rootAssetScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"widget-shop","displayName":"Widget Shop","description":"Buy widgets fast."}}`)
	writeFile(t, root, "ui/src/design-tokens.css", compliantTokens)
	writeFile(t, root, "ui/index.html", `<!doctype html>
<html lang="en">
  <head>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="apple-touch-icon" href="/apple-icon-180.png" />
    <link rel="manifest" href="/site.webmanifest" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="theme-color" content="#1d4ed8" />
    <title>Widget Shop</title>
  </head>
  <body><div id="root"></div></body>
</html>`)
	writeFile(t, root, "ui/public/site.webmanifest", `{
  "name": "Widget Shop", "short_name": "Widget Shop", "description": "Buy widgets fast.",
  "theme_color": "#1d4ed8", "background_color": "#ffffff", "display": "standalone",
  "start_url": ".", "scope": ".", "id": "/",
  "icons": [
    {"src": "icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable"},
    {"src": "icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"}
  ]
}`)
	writeFile(t, root, "ui/public/logo.svg", "<svg/>")
	writeFile(t, root, "ui/public/favicon.svg", "<svg/>")
	writeFile(t, root, "ui/public/apple-icon-180.png", "PNGDATAOPAQUE")
	writeFile(t, root, "ui/public/icon-192.png", "PNGDATA-192")
	writeFile(t, root, "ui/public/icon-512.png", "PNGDATA-512")
	return root
}

// TestPublicAssetConventionFiresOnRootAssets is detection part (a)+(b): the rule
// flags both the root-served files and the root-absolute references to them.
func TestPublicAssetConventionFiresOnRootAssets(t *testing.T) {
	root := rootAssetScenario(t)
	f := mustFire(t, root, "public-asset-convention")
	if f.Severity != SeverityWarning {
		t.Fatalf("severity = %q, want warning", f.Severity)
	}
	files, _ := f.Evidence["root_files"].([]string)
	if len(files) == 0 {
		t.Fatalf("expected root_files evidence, got %+v", f.Evidence)
	}
	refs, _ := f.Evidence["root_refs"].([]string)
	if len(refs) == 0 {
		t.Fatalf("expected root_refs evidence, got %+v", f.Evidence)
	}
	if !f.AutofixAvailable {
		t.Fatal("public-asset-convention should advertise an autofix on a root-asset scenario")
	}
}

// TestPublicAssetConventionPassesOnCompliantScenario is the "passes on a
// /public/-correct fixture" check: the canonical fully-branded scenario serves
// every branding asset under /public/, so the rule must stay silent.
func TestPublicAssetConventionPassesOnCompliantScenario(t *testing.T) {
	mustNotFire(t, fullyBrandedScenario(t), "public-asset-convention")
}

// TestPublicAssetConventionSilentForRelativeRefs confirms relative references
// (resolved against the document/manifest, convention-neutral) and assets already
// nested under /public/ do not trip the rule.
func TestPublicAssetConventionSilentForRelativeRefs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head>
    <link rel="icon" href="favicon.svg" />
    <link rel="manifest" href="/public/site.webmanifest" />
    <title>X</title></head></html>`)
	writeFile(t, root, "ui/public/public/site.webmanifest", `{"name":"X","icons":[{"src":"icon.png","sizes":"192x192"}]}`)
	mustNotFire(t, root, "public-asset-convention")
}

// TestPublicAssetConventionFixerRoundTrip is the full preview→apply→clear→
// idempotent cycle: the fixer relocates the files under /public/ and repoints the
// references, after which the rule clears and a re-run is a no-op.
func TestPublicAssetConventionFixerRoundTrip(t *testing.T) {
	assertRoundTrip(t, rootAssetScenario(t), "public-asset-convention")
}

// TestPublicAssetConventionFixerRelocatesAndRepoints asserts the concrete edits:
// the files move under ui/public/public/, the index.html links are repointed at
// /public/, and the relocated manifest's launch target is pinned to the app root.
func TestPublicAssetConventionFixerRelocatesAndRepoints(t *testing.T) {
	root := rootAssetScenario(t)
	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Files relocated under /public/, removed from the root.
	for _, base := range []string{"favicon.svg", "apple-icon-180.png", "icon-192.png", "icon-512.png", "site.webmanifest", "logo.svg"} {
		if !fileExists(t, root, "ui/public/public/"+base) {
			t.Fatalf("expected %s relocated under ui/public/public/", base)
		}
		if fileExists(t, root, "ui/public/"+base) {
			t.Fatalf("expected %s removed from ui/public/ root", base)
		}
	}

	// index.html references repointed at /public/.
	html, _ := readFile(root, "ui/index.html")
	for _, want := range []string{`href="/public/favicon.svg"`, `href="/public/apple-icon-180.png"`, `href="/public/site.webmanifest"`} {
		if !contains(html, want) {
			t.Fatalf("index.html missing repointed ref %q:\n%s", want, html)
		}
	}
	if contains(html, `href="/favicon.svg"`) {
		t.Fatal("index.html still references the root /favicon.svg")
	}

	// Relocated manifest launches into the app root, not the /public/ folder.
	manifest, _ := readFile(root, "ui/public/public/site.webmanifest")
	if !contains(manifest, `"start_url": "/"`) || !contains(manifest, `"scope": "/"`) {
		t.Fatalf("manifest start_url/scope not pinned to app root:\n%s", manifest)
	}
}

// TestPublicAssetConventionBiconditional guards the advertised==implemented
// invariant for the new rule across a firing and a compliant fixture.
func TestPublicAssetConventionBiconditional(t *testing.T) {
	for _, root := range []string{rootAssetScenario(t), fullyBrandedScenario(t)} {
		f, fired := findingByRule(ScanScenario("x", root), "public-asset-convention")
		cands, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, false)
		if err != nil {
			t.Fatalf("BuildFixCandidates: %v", err)
		}
		switch {
		case fired && f.AutofixAvailable && len(cands) != 1:
			t.Fatalf("rule fired + advertises autofix but BuildFixCandidates returned %d candidates", len(cands))
		case fired && !f.AutofixAvailable:
			t.Fatal("rule fired but did not advertise the implemented autofix")
		case !fired && len(cands) != 0:
			t.Fatalf("rule did not fire but BuildFixCandidates returned %d candidates", len(cands))
		}
	}
}

func fileExists(t *testing.T, root, rel string) bool {
	t.Helper()
	_, ok := readFile(root, rel)
	return ok
}

// contains is a tiny substring helper local to this file's assertions.
func contains(s, sub string) bool { return indexOf(s, sub) >= 0 }
