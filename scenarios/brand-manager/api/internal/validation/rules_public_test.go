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
  "start_url": ".", "scope": ".",
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

// TestPublicAssetConventionSilentForCanonicalRefs confirms the canonical forms
// pass: relative public/ head links, an absolute /public/ og:image, manifest
// icon srcs beside the manifest, and a "../" launch target.
func TestPublicAssetConventionSilentForCanonicalRefs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head>
    <link rel="icon" href="public/favicon.svg" />
    <link rel="manifest" href="public/site.webmanifest" />
    <meta property="og:image" content="/public/og-image.png" />
    <title>X</title></head></html>`)
	writeFile(t, root, "ui/public/public/site.webmanifest", `{"name":"X","start_url":"../","scope":"../","icons":[{"src":"icon.png","sizes":"192x192"}]}`)
	mustNotFire(t, root, "public-asset-convention")
}

// TestPublicAssetConventionFlagsProxyUnsafeRefs covers references that reach
// /public/ only from the host root: absolute /public/ head links, a relative
// link that resolves to the gated root, root-absolute manifest icon srcs, and a
// manifest under /public/ that launches into "/" instead of "../".
func TestPublicAssetConventionFlagsProxyUnsafeRefs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head>
    <link rel="icon" href="favicon.svg" />
    <link rel="icon" sizes="32x32" href="/public/favicon-32.png" />
    <link rel="manifest" href="/public/site.webmanifest" />
    <meta property="og:image" content="og-image.png" />
    <title>X</title></head></html>`)
	writeFile(t, root, "ui/public/public/site.webmanifest", `{"name":"X","start_url":"/","scope":"/","icons":[{"src":"/public/icon.png","sizes":"192x192"}]}`)
	f := mustFire(t, root, "public-asset-convention")
	refs, _ := f.Evidence["root_refs"].([]string)
	for _, want := range []string{"favicon.svg", "/public/favicon-32.png", "/public/site.webmanifest", "og-image.png", "/public/icon.png"} {
		if !containsString(refs, want) {
			t.Fatalf("root_refs = %v, want %q flagged", refs, want)
		}
	}
	if launch, _ := f.Evidence["manifest_launch"].(bool); !launch {
		t.Fatalf("expected the / launch target flagged, evidence %+v", f.Evidence)
	}

	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	html, _ := readFile(root, "ui/index.html")
	for _, want := range []string{`href="public/favicon.svg"`, `href="public/favicon-32.png"`, `href="public/site.webmanifest"`, `content="/public/og-image.png"`} {
		if !contains(html, want) {
			t.Fatalf("index.html missing canonical ref %q:\n%s", want, html)
		}
	}
	manifest, _ := readFile(root, "ui/public/public/site.webmanifest")
	if !contains(manifest, `"src": "icon.png"`) || !contains(manifest, `"start_url": "../"`) || !contains(manifest, `"scope": "../"`) {
		t.Fatalf("manifest not repointed:\n%s", manifest)
	}
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
// public/..., and the relocated manifest's launch target is the app root "../".
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

	// index.html references repointed at the relative public/ surface.
	html, _ := readFile(root, "ui/index.html")
	for _, want := range []string{`href="public/favicon.svg"`, `href="public/apple-icon-180.png"`, `href="public/site.webmanifest"`} {
		if !contains(html, want) {
			t.Fatalf("index.html missing repointed ref %q:\n%s", want, html)
		}
	}
	if contains(html, `href="/favicon.svg"`) {
		t.Fatal("index.html still references the root /favicon.svg")
	}

	// Relocated manifest launches into the app root, not the /public/ folder.
	manifest, _ := readFile(root, "ui/public/public/site.webmanifest")
	if !contains(manifest, `"start_url": "../"`) || !contains(manifest, `"scope": "../"`) {
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

// TestPublicAssetConventionFixerNeverOverwritesThePublicCopy guards the
// relocation: a root copy that differs from the served /public/ file is
// refused before anything changes, and an identical one is simply removed.
func TestPublicAssetConventionFixerNeverOverwritesThePublicCopy(t *testing.T) {
	conflict := rootAssetScenario(t)
	writeFile(t, conflict, "ui/public/public/site.webmanifest", `{"name":"served"}`)
	if _, _, err := BuildFixCandidates(conflict, []string{"public-asset-convention"}, true); err == nil || !contains(err.Error(), "different content") {
		t.Fatalf("apply over a differing /public/ manifest: err = %v, want a refusal", err)
	}
	if served, _ := readFile(conflict, "ui/public/public/site.webmanifest"); served != `{"name":"served"}` {
		t.Fatalf("the served manifest was overwritten: %s", served)
	}
	if !fileExists(t, conflict, "ui/public/favicon.svg") {
		t.Fatal("a refused apply must leave the tree untouched")
	}

	duplicate := rootAssetScenario(t)
	writeFile(t, duplicate, "ui/public/public/favicon.svg", "<svg/>") // identical to the root copy
	if _, _, err := BuildFixCandidates(duplicate, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply with an identical duplicate: %v", err)
	}
	if fileExists(t, duplicate, "ui/public/favicon.svg") || !fileExists(t, duplicate, "ui/public/public/favicon.svg") {
		t.Fatal("an identical root duplicate should be removed and the served copy kept")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// A link's rel, not its filename, makes it branding: prompt-manager's
// "/public/prompt-manager-icon.svg" fits none of the conventional name globs
// and was left absolute. A root-served custom icon is relocated with its link,
// and a bundled /assets/ target, which the convention cannot place, is left
// alone.
func TestPublicAssetConventionCoversCustomNamedBrandingLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head>
    <link rel="icon" type="image/svg+xml" href="/public/x-icon.svg" />
    <link rel="apple-touch-icon" href="/x-touch.png" />
    <link rel="mask-icon" href="/assets/x-mask.svg" />
    <link rel="manifest" href="public/site.webmanifest" />
    <title>X</title></head></html>`)
	writeFile(t, root, "ui/public/public/x-icon.svg", "<svg/>")
	writeFile(t, root, "ui/public/x-touch.png", "PNGDATA")
	writeFile(t, root, "ui/public/assets/x-mask.svg", "<svg/>")
	writeFile(t, root, "ui/public/public/site.webmanifest", `{"name":"X","start_url":"../","scope":"../","icons":[{"src":"x-icon.svg","sizes":"192x192"}]}`)
	mustFire(t, root, "public-asset-convention")

	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	html, _ := readFile(root, "ui/index.html")
	for _, want := range []string{`href="public/x-icon.svg"`, `href="public/x-touch.png"`, `href="/assets/x-mask.svg"`} {
		if !contains(html, want) {
			t.Fatalf("index.html missing %q:\n%s", want, html)
		}
	}
	if !fileExists(t, root, "ui/public/public/x-touch.png") || fileExists(t, root, "ui/public/x-touch.png") {
		t.Fatal("the root-served custom icon should move under ui/public/public with its link")
	}
	mustNotFire(t, root, "public-asset-convention")
}

// A relocated file that a service worker precaches at its old URL makes
// cache.addAll reject, so the worker never installs. The fixer rewrites those
// entries with the move.
func TestPublicAssetConventionRepointsServiceWorkerPrecache(t *testing.T) {
	root := rootAssetScenario(t)
	writeFile(t, root, "ui/public/sw.js", `const CACHE_NAME = "shell-v1";
const APP_SHELL_URLS = ["./", "./site.webmanifest"];
self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL_URLS)));
});`)
	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	sw, _ := readFile(root, "ui/public/sw.js")
	if !contains(sw, `const APP_SHELL_URLS = ["./", "./public/site.webmanifest"];`) {
		t.Fatalf("precache entry not repointed:\n%s", sw)
	}
	mustNotFire(t, root, "public-asset-convention")
}

// Every same-origin form of a moved file is rewritten, in the shipped sw.js
// and in a worker the page registers under another name; other entries stay.
func TestPublicAssetConventionRepointsEveryPrecachedMove(t *testing.T) {
	root := rootAssetScenario(t)
	writeFile(t, root, "ui/index.html", `<!doctype html>
<html lang="en">
  <head>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="manifest" href="/site.webmanifest" />
    <title>Widget Shop</title>
  </head>
  <body><div id="root"></div>
  <script>navigator.serviceWorker.register("./offline-worker.js");</script></body>
</html>`)
	worker := `self.addEventListener("install", (event) => {
  event.waitUntil(caches.open("v1").then((cache) => cache.addAll([
    "./",
    "./index.html",
    "./site.webmanifest",
    "/favicon.svg",
    'icon-192.png',
  ])));
});`
	writeFile(t, root, "ui/public/offline-worker.js", worker)
	writeFile(t, root, "ui/public/sw.js", `const SHELL = ["./", "/icon-512.png"];
self.addEventListener("install", (e) => e.waitUntil(caches.open("v").then((c) => c.addAll(SHELL))));`)

	cands, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, false)
	if err != nil || len(cands) != 1 || !contains(cands[0].After, "sw ui/public/offline-worker.js ./public/site.webmanifest") {
		t.Fatalf("preview should list the worker rewrites, got %+v err %v", cands, err)
	}
	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := readFile(root, "ui/public/offline-worker.js")
	for _, want := range []string{`"./index.html"`, `"./public/site.webmanifest"`, `"/public/favicon.svg"`, `'public/icon-192.png'`} {
		if !contains(got, want) {
			t.Fatalf("registered worker missing %s:\n%s", want, got)
		}
	}
	sw, _ := readFile(root, "ui/public/sw.js")
	if !contains(sw, `const SHELL = ["./", "/public/icon-512.png"];`) {
		t.Fatalf("sw.js not repointed:\n%s", sw)
	}
}

// A precache list the fixer cannot read statically is refused before
// anything moves.
func TestPublicAssetConventionRefusesAComputedPrecacheList(t *testing.T) {
	root := rootAssetScenario(t)
	writeFile(t, root, "ui/public/sw.js", `const BASE = "./";
const APP_SHELL_URLS = [BASE, BASE + "site.webmanifest"];
self.addEventListener("install", (event) => {
  event.waitUntil(caches.open("v1").then((cache) => cache.addAll(APP_SHELL_URLS)));
});`)
	cands, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, false)
	if err != nil || len(cands) != 1 || !contains(cands[0].Description, "Refused") {
		t.Fatalf("preview should announce the refusal, got %+v err %v", cands, err)
	}
	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err == nil || !contains(err.Error(), "refused") {
		t.Fatalf("apply should refuse, got err %v", err)
	}
	if !fileExists(t, root, "ui/public/site.webmanifest") || fileExists(t, root, "ui/public/public/site.webmanifest") {
		t.Fatal("a refused apply must leave the tree untouched")
	}
}

// One asset named by a head link and by og:image has two canonical forms:
// relative for the link, absolute for the social image. Rewriting the link
// must not drag the already-canonical og:image along with it.
func TestPublicAssetConventionKeepsSocialImagesAbsolute(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"x","displayName":"X","description":"A real x."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head>
    <meta property="og:image" content="/public/logo.svg" />
    <meta name="twitter:image" content="/public/logo.svg" />
    <link rel="apple-touch-icon" href="/public/logo.svg" />
    <title>X</title></head></html>`)
	writeFile(t, root, "ui/public/public/logo.svg", "<svg/>")
	if _, _, err := BuildFixCandidates(root, []string{"public-asset-convention"}, true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	html, _ := readFile(root, "ui/index.html")
	for _, want := range []string{`href="public/logo.svg"`, `property="og:image" content="/public/logo.svg"`, `name="twitter:image" content="/public/logo.svg"`} {
		if !contains(html, want) {
			t.Fatalf("index.html missing %q:\n%s", want, html)
		}
	}
	mustNotFire(t, root, "public-asset-convention")
}
