/*
Rule: Capture Features Enabled
ID: interop_capture_enabled
Description: Ensures captureLogs and captureNetwork are not explicitly
  disabled in the initIframeBridgeChild configuration, so the
  host shell receives runtime diagnostics from the embedded UI.
Why: captureLogs and captureNetwork are the primary telemetry channels
  from embedded UIs to the Vrooli dashboard. Disabling either one
  creates a blind spot: the host cannot detect runtime errors,
  failed API calls, or performance regressions in the scenario UI.
Category: interop
Severity: medium
Slot: [D]
SlotFile: ui/src/main.tsx
TechStack: iframe-bridge
Recommendation: Remove `captureLogs: false` and `captureNetwork: false` from
  the bridge config. Both default to true when omitted.
Standard: vrooli-ui-interop-v1

GoodExample:
    initIframeBridgeChild({
      appId: 'my-scenario',
      captureLogs: true,
      captureNetwork: true,
    });

BadExample:
    initIframeBridgeChild({
      appId: 'my-scenario',
      captureLogs: false,
      captureNetwork: false,
    });

<test-case id="capture-both-enabled" should-fail="false">
  <description>Both captureLogs and captureNetwork are enabled (or omitted)</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-app', captureLogs: true });
    }
  </input>
</test-case>

<test-case id="capture-omitted" should-fail="false">
  <description>captureLogs and captureNetwork omitted (default to true)</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-app' });
    }
  </input>
</test-case>

<test-case id="capture-logs-disabled" should-fail="true">
  <description>captureLogs explicitly set to false</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-app', captureLogs: false });
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>captureLogs is disabled</expected-message>
</test-case>

<test-case id="capture-network-disabled" should-fail="true">
  <description>captureNetwork explicitly set to false</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({
        appId: 'my-app',
        captureNetwork: false,
      });
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>captureNetwork is disabled</expected-message>
</test-case>
*/

package interop

import (
	"app-monitor-api/rules"
	"regexp"
	"strings"
)

var (
	captureLogsDisabledPattern    = regexp.MustCompile(`(?s)captureLogs\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
	captureNetworkDisabledPattern = regexp.MustCompile(`(?s)captureNetwork\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
)

func init() {
	rules.Register("interop_capture_enabled", checkCaptureEnabled)
}

func checkCaptureEnabled(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_capture_enabled"

	content, relPath, _, err := findMainEntry(ctx.ScenarioRoot)
	if err != nil {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI entry file found",
			Message:    "no UI entry file found; skipping capture check",
		}
	}

	// Skip if no bridge init at all (that is a different rule's concern).
	if !strings.Contains(content, "initIframeBridgeChild") {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "initIframeBridgeChild not present",
			Message:    "initIframeBridgeChild not found; skipping capture check",
		}
	}

	var violations []rules.Violation

	if captureLogsDisabledPattern.MatchString(content) {
		line := lineOf(content, "captureLogs")
		violations = append(violations, rules.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "captureLogs is disabled",
			Description:    "captureLogs is disabled in initIframeBridgeChild config",
			FilePath:       relPath,
			Line:           line,
			Recommendation: "Remove `captureLogs: false` or set it to true",
		})
	}

	if captureNetworkDisabledPattern.MatchString(content) {
		line := lineOf(content, "captureNetwork")
		violations = append(violations, rules.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "captureNetwork is disabled",
			Description:    "captureNetwork is disabled in initIframeBridgeChild config",
			FilePath:       relPath,
			Line:           line,
			Recommendation: "Remove `captureNetwork: false` or set it to true",
		})
	}

	if len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.Title
		}
		return rules.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    strings.Join(msgs, "; ") + " in " + relPath,
			Violations: violations,
		}
	}

	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "capture features are enabled in " + relPath,
	}
}
