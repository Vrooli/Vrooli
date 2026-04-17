/*
Rule: Bridge App ID Parameter
ID: interop_bridge_app_id
Description: Ensures the initIframeBridgeChild call includes the appId
  parameter so the host shell can uniquely identify this scenario's
  iframe among multiple embedded UIs.
Why: Without appId the host shell cannot route messages to the correct
  embedded UI. When multiple scenarios are loaded simultaneously, a
  missing appId causes message mis-delivery and state corruption.
Category: interop
Severity: medium
Slot: [D]
SlotFile: ui/src/main.tsx
TechStack: iframe-bridge
Recommendation: Pass appId in the config object, e.g.
  initIframeBridgeChild({ appId: 'my-scenario' }).
Standard: vrooli-ui-interop-v1

GoodExample:
    initIframeBridgeChild({ appId: 'my-scenario', captureLogs: true });

BadExample:
    initIframeBridgeChild();
    // or
    initIframeBridgeChild({});

<test-case id="bridge-app-id-present" should-fail="false">
  <description>initIframeBridgeChild call includes appId</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    initIframeBridgeChild({ appId: 'my-scenario', captureLogs: true });
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="bridge-app-id-multiline" should-fail="false">
  <description>initIframeBridgeChild call includes appId on separate line</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    initIframeBridgeChild({
      appId: 'my-scenario',
      captureLogs: true,
    });
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="bridge-app-id-missing" should-fail="true">
  <description>initIframeBridgeChild called without appId</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    initIframeBridgeChild({});
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>appId not found</expected-message>
</test-case>
*/

package interop

import (
	"app-monitor-api/rules"
	"regexp"
	"strings"
)

var appIDPattern = regexp.MustCompile(`appId\s*:`)

func init() {
	rules.Register("interop_bridge_app_id", checkBridgeAppID)
}

func checkBridgeAppID(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_bridge_app_id"

	content, relPath, _, err := findMainEntry(ctx.ScenarioRoot)
	if err != nil {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI entry file found",
			Message:    "no UI entry file found; skipping appId check",
		}
	}

	// Skip if no bridge init at all (that is a different rule's concern).
	if !strings.Contains(content, "initIframeBridgeChild") {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "initIframeBridgeChild not present",
			Message:    "initIframeBridgeChild not found; skipping appId check",
		}
	}

	// First check single-line pattern: appId in the same call.
	if appIDPattern.MatchString(content) {
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "appId found in initIframeBridgeChild call in " + relPath,
		}
	}

	// Also check multi-line: look within 200 chars after initIframeBridgeChild.
	idx := strings.Index(content, "initIframeBridgeChild")
	if idx >= 0 {
		end := idx + 200
		if end > len(content) {
			end = len(content)
		}
		window := content[idx:end]
		if appIDPattern.MatchString(window) {
			return rules.RuleResult{
				RuleID:  ruleID,
				Passed:  true,
				Message: "appId found in initIframeBridgeChild call in " + relPath,
			}
		}
	}

	line := lineOf(content, "initIframeBridgeChild")
	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "appId not found in initIframeBridgeChild call in " + relPath,
		Violations: []rules.Violation{{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Missing appId in bridge init",
			Description:    "appId not found in initIframeBridgeChild call",
			FilePath:       relPath,
			Line:           line,
			Recommendation: "Add appId to the config: initIframeBridgeChild({ appId: '<scenario>' })",
		}},
	}
}
