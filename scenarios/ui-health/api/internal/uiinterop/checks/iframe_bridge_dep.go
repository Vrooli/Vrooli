/*
Rule: Iframe Bridge Dependency
ID: interop_iframe_bridge_dep
Description: Ensures @vrooli/iframe-bridge is listed in ui/package.json
  dependencies so the UI can be embedded in the Vrooli dashboard
  and communicate with the host frame.
Why: The iframe-bridge package provides the bidirectional messaging
  channel between the embedded scenario UI and the Vrooli shell.
  Without it the UI cannot receive configuration, emit telemetry,
  or participate in cross-scenario workflows.
Category: interop
Severity: critical
Slot: [A]
SlotFile: ui/package.json
TechStack: *
Recommendation: Run `pnpm add @vrooli/iframe-bridge` in the ui/ directory,
  then call initIframeBridgeChild() in your entry file.
Standard: vrooli-ui-interop-v1

GoodExample:
    {
      "dependencies": {
        "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge",
        "react": "^18.2.0"
      }
    }

BadExample:
    {
      "dependencies": {
        "react": "^18.2.0"
      }
    }

<test-case id="iframe-bridge-dep-present" should-fail="false">
  <description>package.json lists @vrooli/iframe-bridge in dependencies</description>
  <input>
    [ui/package.json]
    {
      "name": "my-scenario-ui",
      "dependencies": {
        "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge",
        "react": "^18.2.0"
      }
    }
  </input>
</test-case>

<test-case id="iframe-bridge-dep-in-dev" should-fail="false">
  <description>package.json lists @vrooli/iframe-bridge in devDependencies</description>
  <input>
    [ui/package.json]
    {
      "name": "my-scenario-ui",
      "devDependencies": {
        "@vrooli/iframe-bridge": "^2.0.0"
      }
    }
  </input>
</test-case>

<test-case id="iframe-bridge-dep-missing" should-fail="true">
  <description>package.json has no @vrooli/iframe-bridge dependency</description>
  <input>
    [ui/package.json]
    {
      "name": "my-scenario-ui",
      "dependencies": {
        "react": "^18.2.0"
      }
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>@vrooli/iframe-bridge not found</expected-message>
</test-case>
*/

package checks

import "ui-health/internal/uiinterop"

func init() {
	uiinterop.Register("interop_iframe_bridge_dep", checkIframeBridgeDep)
}

// checkIframeBridgeDep verifies @vrooli/iframe-bridge is declared in
// ui/package.json. The read/parse/lookup body is shared with interop_api_base_dep
// via checkPackageJSONDependency (see helpers.go).
func checkIframeBridgeDep(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	return checkPackageJSONDependency(ctx, "interop_iframe_bridge_dep", "@vrooli/iframe-bridge")
}
