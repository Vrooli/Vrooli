/*
Rule: Bridge Initialization
ID: interop_bridge_init
Description: Ensures initIframeBridgeChild() is called in the UI entry file
  so that the scenario UI establishes a communication channel with
  the Vrooli host shell on startup.
Why: The iframe bridge must be initialized before any other app code runs.
  If initIframeBridgeChild is missing the host cannot inject
  configuration, receive telemetry, or coordinate navigation with
  the embedded UI.
Category: interop
Severity: critical
Slot: [D]
SlotFile: ui/src/main.tsx
TechStack: iframe-bridge
Recommendation: Import initIframeBridgeChild from '@vrooli/iframe-bridge' and
  call it at the top of your main entry file, before ReactDOM.render
  or equivalent.
Standard: vrooli-ui-interop-v1

GoodExample:
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    initIframeBridgeChild({ appId: 'my-app' });
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);

BadExample:
    // No bridge initialization at all
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);

<test-case id="bridge-init-present" should-fail="false">
  <description>main.tsx calls initIframeBridgeChild</description>
  <input>
    [ui/src/main.tsx]
    import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: 'my-app' });
    }
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="bridge-init-missing" should-fail="true">
  <description>main.tsx does not call initIframeBridgeChild</description>
  <input>
    [ui/src/main.tsx]
    import React from 'react';
    import ReactDOM from 'react-dom/client';
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>initIframeBridgeChild not found</expected-message>
</test-case>
*/

package checks

import (
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_bridge_init", checkBridgeInit)
}

func checkBridgeInit(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_bridge_init"

	content, relPath, _, err := findMainEntry(ctx.ScenarioRoot)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI entry file found",
			Message:    "no UI entry file found; skipping bridge init check",
		}
	}

	if strings.Contains(content, "initIframeBridgeChild") {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "initIframeBridgeChild found in " + relPath,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "initIframeBridgeChild not found in " + relPath,
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "critical",
			Title:          "Missing bridge initialization",
			Description:    "initIframeBridgeChild not found in " + relPath,
			FilePath:       relPath,
			Recommendation: "Add initIframeBridgeChild({ appId: '<scenario>' }) to " + relPath,
		}},
	}
}
