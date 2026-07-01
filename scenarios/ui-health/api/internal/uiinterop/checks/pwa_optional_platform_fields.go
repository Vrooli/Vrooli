/*
Rule: PWA Optional Platform Fields
ID: pwa_optional_platform_fields
Description: Optional manifest platform capabilities are validated when declared
  but are not required for every scenario.
Why: Shortcuts, share targets, protocol handlers, file handlers, display
  overrides, and launch handlers can improve native feel, but they are product
  decisions. ui-health should catch malformed declarations without forcing their
  presence.
Category: pwa
Severity: low
Slot: [A]
SlotFile: ui/public/site.webmanifest
TechStack: React
Recommendation: When declaring optional platform fields, provide valid URLs,
  labels, methods, and allowed values; omit fields that the product does not use.
Standard: vrooli-pwa-native-readiness-v1

GoodExample:
    { "shortcuts": [{ "name": "Open", "url": "." }],
      "share_target": { "action": ".", "method": "POST", "enctype": "multipart/form-data",
        "params": { "title": "title", "text": "text" } },
      "protocol_handlers": [{ "protocol": "web+demo", "url": "./open?value=%s" }],
      "file_handlers": [{ "action": ".", "accept": { "text/plain": [".txt"] } }],
      "related_applications": [{ "platform": "webapp", "url": "." }],
      "display_override": ["standalone"], "launch_handler": { "client_mode": "navigate-existing" } }

BadExample:
    { "shortcuts": [{ "name": "", "url": "http://localhost:5173" }],
      "display_override": ["teleport"] }

<test-case id="optional-fields-valid" should-fail="false">
  <description>declared optional fields have safe values</description>
  <input>
    [ui/public/site.webmanifest]
    { "shortcuts": [{ "name": "Open", "url": "." }],
      "protocol_handlers": [{ "protocol": "web+demo", "url": "./open?value=%s" }],
      "file_handlers": [{ "action": ".", "accept": { "text/plain": [".txt"] } }],
      "display_override": ["standalone"], "launch_handler": { "client_mode": "navigate-existing" } }
  </input>
</test-case>

<test-case id="optional-fields-invalid" should-fail="true">
  <description>declared optional fields are malformed</description>
  <input>
    [ui/public/site.webmanifest]
    { "shortcuts": [{ "name": "", "url": "http://localhost:5173" }],
      "display_override": ["teleport"] }
  </input>
  <expected-message>shortcut</expected-message>
</test-case>
*/

package checks

import (
	"fmt"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("pwa_optional_platform_fields", checkPWAOptionalPlatformFields)
}

func checkPWAOptionalPlatformFields(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "pwa_optional_platform_fields"
	mf, ok := readManifestJSON(ctx.ScenarioRoot)
	if !ok {
		return skippedPWA(ruleID, "web app manifest not found")
	}
	var violations []uiinterop.Violation
	if shortcuts, ok := mf.data["shortcuts"].([]any); ok {
		for i, raw := range shortcuts {
			shortcut, _ := raw.(map[string]any)
			urlValue := stringField(shortcut, "url")
			if strings.TrimSpace(stringField(shortcut, "name")) == "" || malformedManifestLaunchURL(urlValue) || !manifestURLInScope(urlValue, stringField(mf.data, "scope")) {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Malformed shortcut", fmt.Sprintf("shortcut %d needs a non-empty name and a deployment-safe URL", i), "Use a relative shortcut URL and a user-facing shortcut name."))
			}
			if icons, ok := shortcut["icons"].([]any); ok {
				violations = append(violations, validateOptionalIcons(ctx, ruleID, mf, fmt.Sprintf("shortcut %d", i), icons)...)
			}
		}
	}
	if share, ok := mf.data["share_target"].(map[string]any); ok {
		method := strings.ToUpper(stringField(share, "method"))
		if method != "" && method != "GET" && method != "POST" {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid share_target method", "share_target.method must be GET or POST", "Use GET or POST for share_target.method."))
		}
		if malformedManifestLaunchURL(stringField(share, "action")) {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid share_target action", malformedOptionalURL("share_target.action", stringField(share, "action")), "Use a relative or same-origin share_target action URL."))
		}
		if method == "POST" {
			enctype := stringField(share, "enctype")
			if enctype != "" && enctype != "application/x-www-form-urlencoded" && enctype != "multipart/form-data" {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid share_target enctype", fmt.Sprintf("share_target.enctype %q is not supported", enctype), "Use application/x-www-form-urlencoded or multipart/form-data for POST share targets."))
			}
		}
		if params, ok := share["params"].(map[string]any); ok {
			if len(params) == 0 {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Empty share_target params", "share_target.params is declared but empty", "Map at least one incoming share field such as title, text, url, or files."))
			}
		}
	}
	if handlers, ok := mf.data["protocol_handlers"].([]any); ok {
		for i, raw := range handlers {
			handler, _ := raw.(map[string]any)
			protocol := stringField(handler, "protocol")
			urlValue := stringField(handler, "url")
			if !strings.HasPrefix(protocol, "web+") || malformedManifestLaunchURL(strings.ReplaceAll(urlValue, "%s", "value")) || !strings.Contains(urlValue, "%s") {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Malformed protocol handler", fmt.Sprintf("protocol handler %d needs a web+ protocol and URL containing %%s", i), "Use a web+ protocol and a relative handler URL containing %s."))
			}
		}
	}
	if handlers, ok := mf.data["file_handlers"].([]any); ok {
		for i, raw := range handlers {
			handler, _ := raw.(map[string]any)
			if malformedManifestLaunchURL(stringField(handler, "action")) {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid file handler action", malformedOptionalURL("file_handlers.action", stringField(handler, "action")), "Use a relative file handler action URL."))
			}
			accept, ok := handler["accept"].(map[string]any)
			if !ok || len(accept) == 0 {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid file handler accept map", fmt.Sprintf("file handler %d needs a non-empty accept map", i), "Declare MIME types and extensions accepted by the file handler."))
			}
		}
	}
	if applications, ok := mf.data["related_applications"].([]any); ok {
		for i, raw := range applications {
			application, _ := raw.(map[string]any)
			if stringField(application, "platform") == "" {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid related application", fmt.Sprintf("related application %d is missing platform", i), "Declare the target platform for each related application."))
			}
			if urlValue := stringField(application, "url"); urlValue != "" && malformedManifestLaunchURL(urlValue) {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid related application URL", malformedOptionalURL("related_applications.url", urlValue), "Use a deployment-safe related application URL."))
			}
		}
	}
	if overrides, ok := mf.data["display_override"].([]any); ok {
		for _, raw := range overrides {
			value, _ := raw.(string)
			if !allowedDisplayOverride(value) {
				violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid display_override value", fmt.Sprintf("display_override %q is not a known value", value), "Use fullscreen, standalone, minimal-ui, browser, window-controls-overlay, tabbed, or borderless."))
			}
		}
	}
	if launch, ok := mf.data["launch_handler"].(map[string]any); ok {
		mode := stringField(launch, "client_mode")
		if mode != "" && mode != "auto" && mode != "focus-existing" && mode != "navigate-existing" && mode != "navigate-new" {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Invalid launch_handler client_mode", fmt.Sprintf("launch_handler.client_mode %q is not known", mode), "Use auto, focus-existing, navigate-existing, or navigate-new."))
		}
	}
	return pwaResult(ruleID, "declared optional platform fields are valid", "declared optional platform fields are malformed", violations)
}

func validateOptionalIcons(ctx uiinterop.CheckContext, ruleID string, mf manifestRead, owner string, icons []any) []uiinterop.Violation {
	var violations []uiinterop.Violation
	for i, raw := range icons {
		icon, _ := raw.(map[string]any)
		src := stringField(icon, "src")
		if src == "" || stringField(icon, "sizes") == "" || stringField(icon, "type") == "" {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Malformed optional icon", fmt.Sprintf("%s icon %d needs src, sizes, and type", owner, i), "Declare complete icon metadata or omit optional icons."))
			continue
		}
		if !manifestAssetExists(ctx.ScenarioRoot, mf.dir, src) {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Optional icon path unresolved", fmt.Sprintf("%s icon %q does not resolve relative to the manifest", owner, src), "Move the icon under ui/public or update the optional icon src."))
		}
	}
	return violations
}

func malformedManifestLaunchURL(raw string) bool {
	return raw == "" || unsafeLaunchURL(raw)
}
