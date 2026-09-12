/*
Rule: PWA Manifest Install Fields
ID: pwa_manifest_install_fields
Description: A UI's web app manifest must declare identity, launch URL, display
  mode, colors, and icons for install readiness.
Why: Browsers cannot present a scenario as an installable, native-feeling web app
  unless the manifest carries stable identity, launch, scope, color, and icon
  metadata. ui-health owns this contract shape; brand-manager owns asset quality.
Category: pwa
Severity: medium
Slot: [A]
SlotFile: ui/public/site.webmanifest
TechStack: React
Recommendation: Add id, name, short_name, description, start_url, scope,
  display, background_color, theme_color, and icon entries to the web manifest.
Standard: vrooli-pwa-native-readiness-v1

GoodExample:
    { "id": ".", "name": "Demo", "short_name": "Demo", "description": "Demo",
      "start_url": ".", "scope": ".", "display": "standalone",
      "background_color": "#0f172a", "theme_color": "#0f172a",
      "icons": [{ "src": "manifest-icon-192.maskable.png", "sizes": "192x192",
        "type": "image/png", "purpose": "any maskable" }] }

BadExample:
    { "name": "Demo", "display": "browser" }

<test-case id="manifest-install-fields-complete" should-fail="false">
  <description>site.webmanifest carries install identity, launch, colors, and a resolvable icon</description>
  <input>
    [ui/public/site.webmanifest]
    { "id": ".", "name": "Demo", "short_name": "Demo", "description": "Demo",
      "start_url": ".", "scope": ".", "display": "standalone",
      "background_color": "#0f172a", "theme_color": "#0f172a",
      "icons": [{ "src": "manifest-icon-192.maskable.png", "sizes": "192x192",
        "type": "image/png", "purpose": "any maskable" }] }
    [ui/public/manifest-icon-192.maskable.png]
    png
  </input>
</test-case>

<test-case id="manifest-install-fields-missing" should-fail="true">
  <description>manifest lacks required install fields</description>
  <input>
    [ui/public/site.webmanifest]
    { "name": "Demo", "display": "browser", "icons": [{ "src": "missing.png" }] }
  </input>
  <expected-message>short_name</expected-message>
</test-case>
*/

package checks

import (
	"fmt"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("pwa_manifest_install_fields", checkPWAManifestInstallFields)
}

func checkPWAManifestInstallFields(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "pwa_manifest_install_fields"
	mf, ok := readManifestJSON(ctx.ScenarioRoot)
	if !ok {
		return skippedPWA(ruleID, "web app manifest not found")
	}
	var violations []uiinterop.Violation
	for _, field := range []string{"id", "name", "short_name", "description", "start_url", "scope", "display", "background_color", "theme_color"} {
		if strings.TrimSpace(stringField(mf.data, field)) == "" {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Missing manifest "+field, "web app manifest is missing "+field, "Add "+field+" to the web app manifest."))
		}
	}
	if display := stringField(mf.data, "display"); display != "" && display != "standalone" && display != "fullscreen" && display != "minimal-ui" {
		violations = append(violations, pwaViolation(ruleID, mf.rel, "Install display is not standalone", `web app manifest display should be "standalone", "fullscreen", or "minimal-ui" for install readiness`, `Set "display":"standalone" unless the product needs another install display mode.`))
	}
	icons, ok := mf.data["icons"].([]any)
	if !ok || len(icons) == 0 {
		violations = append(violations, pwaViolation(ruleID, mf.rel, "Missing manifest icons", "web app manifest declares no icons", "Add at least one icon with src, sizes, type, and purpose."))
	} else {
		for i, raw := range icons {
			icon, _ := raw.(map[string]any)
			src := stringField(icon, "src")
			if strings.TrimSpace(src) == "" {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Manifest icon missing src", fmt.Sprintf("manifest icon %d is missing src", i), "Add a relative icon src."))
				continue
			}
			if strings.TrimSpace(stringField(icon, "sizes")) == "" || strings.TrimSpace(stringField(icon, "type")) == "" {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Manifest icon incomplete", fmt.Sprintf("manifest icon %d should declare sizes and type", i), "Add sizes and type to each manifest icon."))
			}
			if !manifestAssetExists(ctx.ScenarioRoot, mf.dir, src) {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Manifest icon path unresolved", fmt.Sprintf("manifest icon %q does not resolve relative to the manifest", src), "Move the icon under ui/public or update the manifest icon src."))
			}
		}
	}
	return pwaResult(ruleID, "manifest install fields are complete", "manifest install fields are incomplete", violations)
}
