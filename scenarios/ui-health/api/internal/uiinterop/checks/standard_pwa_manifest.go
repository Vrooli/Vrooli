/*
Rule: PWA Manifest & Viewport
ID: standard_pwa_manifest
Description: A UI's ui/index.html must declare the installable-web-app basics —
  a responsive viewport meta tag, a web-app manifest link, and at least one icon
  link — and the referenced webmanifest must set "display":"standalone". These
  are the static markers of a PWA-ready, mobile-correct shell.
Why: Without a viewport meta the page renders at desktop width on phones and the
  layout is unusable; without a linked webmanifest declaring standalone display
  the app cannot be installed to a home screen and always shows browser chrome;
  without icon links the installed app and browser tab have no identity. These
  tags are cheap, standard, and expected of every Vrooli scenario UI.
Category: pwa
Severity: medium
Slot: [A]
SlotFile: ui/index.html
TechStack: React
Recommendation: Add <meta name="viewport" content="width=device-width,
  initial-scale=1.0">, <link rel="manifest" href="/site.webmanifest">, and an
  icon <link rel="icon"> to ui/index.html; set "display":"standalone" in the
  webmanifest. See the react-vite template's ui/index.html.
Standard: vrooli-ui-pwa-v1

GoodExample:
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <link rel="manifest" href="/site.webmanifest" />
    <link rel="icon" href="/favicon-196.png" />

BadExample:
    <head><title>App</title></head>   // no viewport, no manifest, no icon

<test-case id="pwa-manifest-complete" should-fail="false">
  <description>index.html declares viewport, manifest, and icon; webmanifest is standalone</description>
  <input>
    [ui/index.html]
    <!doctype html><html><head>
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <link rel="manifest" href="/site.webmanifest" />
    <link rel="icon" href="/favicon-196.png" />
    </head><body></body></html>
    [ui/public/site.webmanifest]
    { "display": "standalone" }
  </input>
</test-case>

<test-case id="pwa-manifest-no-html" should-fail="false">
  <description>No ui/index.html; PWA markers not applicable</description>
  <input>
    [api/main.go]
    package main
  </input>
</test-case>

<test-case id="pwa-manifest-missing-viewport" should-fail="true">
  <description>index.html is missing the viewport meta tag</description>
  <input>
    [ui/index.html]
    <!doctype html><html><head>
    <link rel="manifest" href="/site.webmanifest" />
    <link rel="icon" href="/favicon-196.png" />
    </head><body></body></html>
    [ui/public/site.webmanifest]
    { "display": "standalone" }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>viewport</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_pwa_manifest", checkPWAManifest)
}

func checkPWAManifest(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_pwa_manifest"
	const indexRel = "ui/index.html"

	data, err := os.ReadFile(filepath.Join(ctx.ScenarioRoot, filepath.FromSlash(indexRel)))
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/index.html not found",
			Message:    "no ui/index.html; skipping PWA manifest check",
		}
	}
	html := strings.ToLower(string(data))

	var violations []uiinterop.Violation
	add := func(title, desc, rec string) {
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          title,
			Description:    desc,
			FilePath:       indexRel,
			Recommendation: rec,
		})
	}

	if !strings.Contains(html, `name="viewport"`) && !strings.Contains(html, "name='viewport'") {
		add("Missing viewport meta", indexRel+" has no responsive viewport meta tag",
			`Add <meta name="viewport" content="width=device-width, initial-scale=1.0"> to <head>`)
	}
	if !strings.Contains(html, `rel="manifest"`) && !strings.Contains(html, "rel='manifest'") {
		add("Missing webmanifest link", indexRel+" does not link a web-app manifest",
			`Add <link rel="manifest" href="/site.webmanifest"> to <head>`)
	}
	if !strings.Contains(html, `rel="icon"`) && !strings.Contains(html, `rel="apple-touch-icon"`) &&
		!strings.Contains(html, "rel='icon'") && !strings.Contains(html, "rel='apple-touch-icon'") {
		add("Missing icon link", indexRel+" declares no favicon or apple-touch-icon link",
			`Add <link rel="icon" href="/favicon-196.png"> to <head>`)
	}

	// The webmanifest, if linked + present, should declare standalone display.
	if manifest, ok := readWebmanifest(ctx.ScenarioRoot); ok {
		if !strings.Contains(strings.ToLower(manifest), `"display"`) ||
			!strings.Contains(strings.ToLower(manifest), "standalone") {
			add("Webmanifest not standalone", "the web-app manifest does not set \"display\":\"standalone\"",
				`Set "display":"standalone" in the webmanifest`)
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "PWA manifest / viewport markers are incomplete",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "viewport, webmanifest link, and icon present; manifest is standalone",
	}
}

// webmanifestCandidates are the common locations a Vite UI publishes its
// web-app manifest at.
var webmanifestCandidates = []string{
	"ui/public/site.webmanifest",
	"ui/public/manifest.webmanifest",
	"ui/public/manifest.json",
	"ui/site.webmanifest",
}

// readWebmanifest returns the first webmanifest found under the scenario, if any.
func readWebmanifest(root string) (string, bool) {
	for _, rel := range webmanifestCandidates {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err == nil {
			return string(data), true
		}
	}
	return "", false
}
