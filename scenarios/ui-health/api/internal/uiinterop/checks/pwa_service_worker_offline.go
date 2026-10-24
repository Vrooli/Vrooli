/*
Rule: PWA Service Worker Offline Shell
ID: pwa_service_worker_offline
Description: Install-targeted UIs should declare a service worker registration
  and a service worker source or public artifact for reload-safe offline launch.
Why: Manifest install metadata only controls installability. A native-feeling web
  app also needs a service worker so reloads and transient offline launches do
  not collapse into a browser error page.
Category: pwa
Severity: medium
Slot: [A]
SlotFile: ui/src/main.tsx
TechStack: React
Recommendation: Register a deterministic service worker and ship the matching
  service worker artifact. Treat app-shell caching strategy as a product choice.
Standard: vrooli-pwa-native-readiness-v1

GoodExample:
    if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js")

BadExample:
    createRoot(document.getElementById("root")!).render(<App />)

<test-case id="service-worker-present" should-fail="false">
  <description>main entry registers a service worker and public sw.js exists</description>
  <input>
    [ui/src/main.tsx]
    if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js")
    [ui/public/sw.js]
    self.addEventListener("fetch", () => {})
  </input>
</test-case>

<test-case id="service-worker-missing" should-fail="true">
  <description>main entry has no service worker registration</description>
  <input>
    [ui/src/main.tsx]
    createRoot(document.getElementById("root")!).render(<App />)
  </input>
  <expected-message>service worker</expected-message>
</test-case>
*/

package checks

import (
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("pwa_service_worker_offline", checkPWAServiceWorkerOffline)
}

func checkPWAServiceWorkerOffline(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "pwa_service_worker_offline"
	if len(sourceFiles(ctx, "ui")) == 0 {
		return skippedPWA(ruleID, "ui source not found")
	}
	registered := false
	for _, f := range sourceFiles(ctx, "ui/src") {
		content := f.Content
		if stringsContainsServiceWorkerRegistration(content) {
			registered = true
			break
		}
	}
	source := serviceWorkerSourceExists(ctx.ScenarioRoot)
	var violations []uiinterop.Violation
	if !registered {
		violations = append(violations, pwaViolation(ruleID, "ui/src", "Missing service worker registration", "UI source does not register a service worker", "Register a deterministic service worker for install/offline readiness."))
	}
	if !source {
		violations = append(violations, pwaViolation(ruleID, "ui", "Missing service worker artifact", "UI has no standard service worker source or public artifact", "Add ui/public/sw.js or a canonical service worker source."))
	}
	return pwaResult(ruleID, "service worker registration and artifact are present", "service worker offline shell is incomplete", violations)
}

func stringsContainsServiceWorkerRegistration(content string) bool {
	return strings.Contains(content, "serviceWorker.register") || strings.Contains(content, "navigator.serviceWorker")
}
