/*
Rule: Iframe Guard Check
ID: interop_iframe_guard
Description: Ensures initIframeBridgeChild is guarded by an iframe detection
  check such as `window.parent !== window` to prevent bridge
  initialization when the UI runs standalone.
Why: When a developer opens the scenario UI directly in a browser tab
  (not embedded in Vrooli), the bridge attempts to postMessage to a
  non-existent parent. This causes silent errors and can block
  startup. Guarding with an iframe check lets the UI work in both
  embedded and standalone modes.
Category: interop
Severity: high
Slot: [D]
SlotFile: ui/src/main.tsx
TechStack: iframe-bridge
Recommendation: Wrap the initIframeBridgeChild call in a guard:
  if (window.parent !== window) { initIframeBridgeChild({...}); }
Standard: ui-health-v1

GoodExample:
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-scenario' });
    }

BadExample:
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    // No guard - always initializes even outside iframe
    initIframeBridgeChild({ appId: 'my-scenario' });

<test-case id="iframe-guard-present" should-fail="false">
  <description>Bridge init is guarded with window.parent !== window</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-scenario' });
    }
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="iframe-guard-top-self" should-fail="false">
  <description>Bridge init is guarded with window.top !== window.self</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.top !== window.self) {
      initIframeBridgeChild({ appId: 'my-scenario' });
    }
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="iframe-guard-missing" should-fail="true">
  <description>Bridge init has no iframe guard</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    initIframeBridgeChild({ appId: 'my-scenario' });
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>iframe guard not found</expected-message>
</test-case>
*/

package checks

import (
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

var iframeGuardPattern = regexp.MustCompile(
	`window\.parent\s*(!==|!=|===|==)\s*window` +
		`|window\s*(!==|!=|===|==)\s*window\.parent` +
		`|window\.top\s*(!==|!=|===|==)\s*window\.self` +
		`|window\.self\s*(!==|!=|===|==)\s*window\.top`,
)

func init() {
	uiinterop.Register("interop_iframe_guard", checkIframeGuard)
}

func checkIframeGuard(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_iframe_guard"

	content, relPath, _, err := findMainEntry(ctx.ScenarioRoot)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI entry file found",
			Message:    "no UI entry file found; skipping iframe guard check",
		}
	}

	// Skip if no bridge init at all (that is a different rule's concern).
	if !strings.Contains(content, "initIframeBridgeChild") {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "initIframeBridgeChild not present",
			Message:    "initIframeBridgeChild not found; skipping iframe guard check",
		}
	}

	if iframeGuardPattern.MatchString(content) {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "iframe guard found in " + relPath,
		}
	}

	line := lineOf(content, "initIframeBridgeChild")
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "iframe guard not found near initIframeBridgeChild in " + relPath,
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "high",
			Title:          "Missing iframe guard",
			Description:    "initIframeBridgeChild is not guarded by an iframe detection check",
			FilePath:       relPath,
			Line:           line,
			Recommendation: "Wrap with: if (window.parent !== window) { initIframeBridgeChild({...}); }",
		}},
	}
}
