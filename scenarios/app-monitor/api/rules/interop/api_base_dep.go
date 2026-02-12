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
Standard: vrooli-ui-interop-v1

GoodExample:
    {
      "dependencies": {
        "@vrooli/api-base": "workspace:*",
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
        "@vrooli/api-base": "workspace:*",
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

package interop

import (
	"encoding/json"
	"os"
	"path/filepath"

	"app-monitor-api/rules"
)

func init() {
	rules.Register("interop_api_base_dep", checkAPIBaseDep)
}

func checkAPIBaseDep(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_api_base_dep"
	const depName = "@vrooli/api-base"
	pkgPath := filepath.Join(ctx.ScenarioRoot, "ui", "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/package.json not found",
			Message:    "ui/package.json not found; skipping dependency check",
		}
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: "failed to parse ui/package.json: " + err.Error(),
			Violations: []rules.Violation{{
				RuleID:         ruleID,
				Severity:       "critical",
				Title:          "Unparseable package.json",
				Description:    "ui/package.json could not be parsed as JSON",
				FilePath:       "ui/package.json",
				Recommendation: "Fix JSON syntax in ui/package.json",
			}},
		}
	}

	// Check both dependencies and devDependencies.
	if _, ok := pkg.Dependencies[depName]; ok {
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: depName + " found in dependencies",
		}
	}
	if _, ok := pkg.DevDependencies[depName]; ok {
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: depName + " found in devDependencies",
		}
	}

	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: depName + " not found in ui/package.json",
		Violations: []rules.Violation{{
			RuleID:         ruleID,
			Severity:       "critical",
			Title:          "Missing " + depName,
			Description:    depName + " not found in dependencies or devDependencies",
			FilePath:       "ui/package.json",
			Recommendation: "Run `pnpm add " + depName + "` in the ui/ directory",
		}},
	}
}
