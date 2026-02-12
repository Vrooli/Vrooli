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
        "@vrooli/iframe-bridge": "workspace:*",
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
        "@vrooli/iframe-bridge": "workspace:*",
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

package interop

import (
	"encoding/json"
	"os"
	"path/filepath"

	"app-monitor-api/rules"
)

func init() {
	rules.Register("interop_iframe_bridge_dep", checkIframeBridgeDep)
}

func checkIframeBridgeDep(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_iframe_bridge_dep"
	const depName = "@vrooli/iframe-bridge"
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
