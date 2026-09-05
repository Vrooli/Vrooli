/*
Rule: PWA Launch Scope
ID: pwa_launch_scope
Description: A web app manifest's start_url, scope, and declared launch URLs
  must be same-origin, non-localhost, and in scope.
Why: Install launch, direct navigation, and future deep-link checks depend on a
  coherent manifest scope. Localhost-only or out-of-scope URLs break proxy,
  tunnel, and installed-app contexts.
Category: pwa
Severity: medium
Slot: [A]
SlotFile: ui/public/site.webmanifest
TechStack: React
Recommendation: Keep start_url, scope, shortcuts, share targets, protocol
  handlers, and file handlers relative or same-origin, and ensure launch paths
  stay under manifest scope.
Standard: vrooli-pwa-native-readiness-v1

GoodExample:
    { "start_url": ".", "scope": ".", "display": "standalone" }

BadExample:
    { "start_url": "http://localhost:5173/app", "scope": "/ui/" }

<test-case id="launch-scope-valid" should-fail="false">
  <description>relative start_url and scope are accepted</description>
  <input>
    [ui/public/site.webmanifest]
    { "start_url": ".", "scope": ".", "display": "standalone" }
  </input>
</test-case>

<test-case id="launch-scope-localhost" should-fail="true">
  <description>localhost launch URLs are not proxy/tunnel safe</description>
  <input>
    [ui/public/site.webmanifest]
    { "start_url": "http://localhost:5173/app", "scope": "/" }
  </input>
  <expected-message>localhost</expected-message>
</test-case>
*/

package checks

import (
	"fmt"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("pwa_launch_scope", checkPWALaunchScope)
}

func checkPWALaunchScope(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "pwa_launch_scope"
	mf, ok := readManifestJSON(ctx.ScenarioRoot)
	if !ok {
		return skippedPWA(ruleID, "web app manifest not found")
	}
	var violations []uiinterop.Violation
	startURL := stringField(mf.data, "start_url")
	scope := stringField(mf.data, "scope")
	for field, value := range map[string]string{"start_url": startURL, "scope": scope} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if unsafeLaunchURL(value) {
			violations = append(violations, pwaViolation(ruleID, mf.rel, "Manifest "+field+" is not deployment-safe", fmt.Sprintf("%s %q is absolute localhost or cross-origin", field, value), "Use a relative or same-origin URL that survives proxy, tunnel, and installed contexts."))
		}
	}
	if startURL != "" && scope != "" && !manifestURLInScope(startURL, scope) {
		violations = append(violations, pwaViolation(ruleID, mf.rel, "Manifest start_url is outside scope", fmt.Sprintf("start_url %q is outside scope %q", startURL, scope), "Keep start_url under the manifest scope."))
	}
	return pwaResult(ruleID, "manifest launch scope is coherent", "manifest launch scope has deployment risks", violations)
}
