package validation

import (
	"os"
	"path/filepath"
	"testing"
)

// --- referenced-assets-exist (A) -------------------------------------------

func TestReferencedAssetsExistFiresOnMissingManifestIcon(t *testing.T) {
	root := fullyBrandedScenario(t)
	// The manifest declares icon-512.png; remove the file so the reference dangles.
	if err := os.Remove(filepath.Join(root, "ui/public/public/icon-512.png")); err != nil {
		t.Fatalf("remove icon: %v", err)
	}
	f := mustFire(t, root, "referenced-assets-exist")
	issues, _ := f.Evidence["issues"].(map[string]any)
	if issues["manifest:icon-512.png"] != "missing" {
		t.Fatalf("expected manifest:icon-512.png=missing, got %+v", f.Evidence)
	}
	if f.Severity != SeverityWarning {
		t.Fatalf("severity = %q, want warning", f.Severity)
	}
	if f.AutofixAvailable {
		t.Fatal("referenced-assets-exist must be detect-only (no autofix)")
	}
}

func TestReferencedAssetsExistFiresOnMissingOGImage(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<meta property="og:image" content="/og-card.png" />`)
	writeFile(t, root, "ui/index.html", html)
	f := mustFire(t, root, "referenced-assets-exist")
	issues, _ := f.Evidence["issues"].(map[string]any)
	if issues["meta:og:image"] != "missing" {
		t.Fatalf("expected meta:og:image=missing, got %+v", f.Evidence)
	}
}

func TestReferencedAssetsExistFiresOnEmptyManifestIcon(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/public/icon-512.png", "") // present but zero bytes
	f := mustFire(t, root, "referenced-assets-exist")
	issues, _ := f.Evidence["issues"].(map[string]any)
	if issues["manifest:icon-512.png"] != "empty" {
		t.Fatalf("expected manifest:icon-512.png=empty, got %+v", f.Evidence)
	}
}

func TestReferencedAssetsExistSkipsRemoteRefs(t *testing.T) {
	root := fullyBrandedScenario(t)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<meta property="og:image" content="https://cdn.example.com/og.png" />`)
	writeFile(t, root, "ui/index.html", html)
	// A remote og:image is not a local file we can inspect — must not fire.
	mustNotFire(t, root, "referenced-assets-exist")
}

func TestReferencedAssetsExistDimensionMismatch(t *testing.T) {
	root := fullyBrandedScenario(t)
	// A real 1x1 PNG declared as 512x512 in the manifest is a provable mismatch.
	writeRawFile(t, root, "ui/public/public/icon-512.png", onePixelPNG())
	f := mustFire(t, root, "referenced-assets-exist")
	issues, _ := f.Evidence["issues"].(map[string]any)
	got, _ := issues["manifest:icon-512.png"].(string)
	if got == "" || got[:18] != "dimension_mismatch" {
		t.Fatalf("expected dimension_mismatch, got %+v", f.Evidence)
	}
}

// --- svg-asset-safety (C) --------------------------------------------------

func TestSVGAssetSafetyFiresOnScript(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	f := mustFire(t, root, "svg-asset-safety")
	issues, _ := f.Evidence["issues"].(map[string]any)
	if issues["ui/public/logo.svg"] == nil {
		t.Fatalf("expected logo.svg flagged, got %+v", f.Evidence)
	}
	if f.AutofixAvailable {
		t.Fatal("svg-asset-safety must be detect-only")
	}
}

func TestSVGAssetSafetyFiresOnEventHandler(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/logo.svg", `<svg xmlns="http://www.w3.org/2000/svg" onload="x()"><rect/></svg>`)
	mustFire(t, root, "svg-asset-safety")
}

func TestSVGAssetSafetySilentOnCleanSVG(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/public/logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h10v10H0z"/></svg>`)
	mustNotFire(t, root, "svg-asset-safety")
}

// --- custom-font-loaded (D) ------------------------------------------------

func TestCustomFontLoadedFiresWhenUnloaded(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root {
  --color-background: #ffffff;
  --color-surface: #f1f5f9;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --color-primary-foreground: #ffffff;
  --color-accent: #0e7490;
  --font-sans: Inter, sans-serif;
}`)
	f := mustFire(t, root, "custom-font-loaded")
	fams, _ := f.Evidence["unloaded_families"].([]string)
	if len(fams) != 1 || fams[0] != "inter" {
		t.Fatalf("expected [inter], got %+v", f.Evidence)
	}
	if f.Severity != SeverityInfo {
		t.Fatalf("severity = %q, want info", f.Severity)
	}
	if f.AutofixAvailable {
		t.Fatal("custom-font-loaded must be detect-only")
	}
}

func TestCustomFontLoadedSilentForSystemStack(t *testing.T) {
	root := fullyBrandedScenario(t) // fixture uses a system-ui stack
	mustNotFire(t, root, "custom-font-loaded")
}

func TestCustomFontLoadedSilentWhenFontFaceLoads(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root { --font-sans: Inter, sans-serif; }`)
	writeFile(t, root, "ui/src/fonts.css", `@font-face { font-family: "Inter"; src: url(/fonts/inter.woff2) format("woff2"); }`)
	mustNotFire(t, root, "custom-font-loaded")
}

func TestCustomFontLoadedSilentWhenGoogleFontsLinked(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root { --font-sans: Inter, sans-serif; }`)
	html := injectBeforeHeadCloseRaw(compliantIndexHTML, `<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter" />`)
	writeFile(t, root, "ui/index.html", html)
	mustNotFire(t, root, "custom-font-loaded")
}

func TestCustomFontLoadedSilentWhenFontFileBundled(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/design-tokens.css", `:root { --font-sans: Inter, sans-serif; }`)
	writeFile(t, root, "ui/public/fonts/inter.woff2", "FONTBYTES")
	mustNotFire(t, root, "custom-font-loaded")
}

// --- reduced-motion-support (B) + fixer ------------------------------------

func TestReducedMotionFiresWhenMotionUnaccommodated(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/app.css", `/* brand-manager:primary */ .fade { transition: opacity 200ms ease; }`)
	f := mustFire(t, root, "reduced-motion-support")
	if f.Severity != SeverityInfo {
		t.Fatalf("severity = %q, want info", f.Severity)
	}
}

func TestReducedMotionSilentWithoutMotion(t *testing.T) {
	root := fullyBrandedScenario(t) // app.css has no transitions/animations
	mustNotFire(t, root, "reduced-motion-support")
}

func TestReducedMotionSilentWhenAccommodated(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/app.css", `.fade { transition: opacity 200ms ease; }
@media (prefers-reduced-motion: reduce) { .fade { transition: none; } }`)
	mustNotFire(t, root, "reduced-motion-support")
}

func TestReducedMotionFixerRoundTrip(t *testing.T) {
	root := fullyBrandedScenario(t)
	writeFile(t, root, "ui/src/app.css", `.fade { transition: opacity 200ms ease; }`)
	assertRoundTrip(t, root, "reduced-motion-support")
}

func TestReducedMotionFixerUnavailableWithoutTokensFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>X</title></head></html>`)
	writeFile(t, root, "ui/src/app.css", `.fade { transition: opacity 200ms ease; }`)
	f := mustFire(t, root, "reduced-motion-support")
	if f.AutofixAvailable {
		t.Fatal("reduced-motion fixer must not advertise when there is no design-tokens.css to append to")
	}
}

// --- helpers ---------------------------------------------------------------

func writeRawFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// onePixelPNG returns the bytes of a real 1x1 opaque PNG (decodes to 1x1).
func onePixelPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x18, 0xdd, 0x8d, 0xb0, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
