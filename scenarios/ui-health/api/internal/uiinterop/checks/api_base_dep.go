/*
Rule: API Base Dependency
ID: interop_api_base_dep
Description: Ensures @vrooli/api-base is listed in ui/package.json dependencies
  so the UI can communicate with the scenario's API layer through
  the standardized Vrooli API client.
Why: Without @vrooli/api-base the UI has no typed, versioned client for
  calling scenario API endpoints. Raw fetch calls bypass retry logic,
  auth token refresh, and error normalization that the shared package
  provides.
Category: interop
Severity: critical
Slot: [A]
SlotFile: ui/package.json
TechStack: *
Recommendation: Run `pnpm add @vrooli/api-base` in the ui/ directory, then
  import and configure the client in your app bootstrap.
Standard: ui-health-v1

GoodExample:
    {
      "dependencies": {
        "@vrooli/api-base": "file:../../../packages/api-base",
        "react": "^18.2.0"
      }
    }

BadExample:
    {
      "dependencies": {
        "react": "^18.2.0"
      }
    }

<test-case id="api-base-dep-present" should-fail="false">
  <description>package.json lists @vrooli/api-base in dependencies</description>
  <input>
    [ui/package.json]
    {
      "name": "my-scenario-ui",
      "dependencies": {
        "@vrooli/api-base": "file:../../../packages/api-base",
        "react": "^18.2.0"
      }
    }
  </input>
</test-case>

<test-case id="api-base-dep-in-dev" should-fail="false">
  <description>package.json lists @vrooli/api-base in devDependencies</description>
  <input>
    [ui/package.json]
    {
      "name": "my-scenario-ui",
      "devDependencies": {
        "@vrooli/api-base": "^1.0.0"
      }
    }
  </input>
</test-case>

<test-case id="api-base-dep-missing" should-fail="true">
  <description>package.json has no @vrooli/api-base dependency</description>
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
  <expected-message>@vrooli/api-base not found</expected-message>
</test-case>
*/

package checks

import "ui-health/internal/uiinterop"

func init() {
	uiinterop.Register("interop_api_base_dep", checkAPIBaseDep)
}

// checkAPIBaseDep verifies @vrooli/api-base is declared in ui/package.json. The
// read/parse/lookup body is shared with interop_iframe_bridge_dep via
// checkPackageJSONDependency (see helpers.go).
func checkAPIBaseDep(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	return checkPackageJSONDependency(ctx, "interop_api_base_dep", "@vrooli/api-base")
}
