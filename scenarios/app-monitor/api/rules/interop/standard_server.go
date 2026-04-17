/*
Rule: Standard Server Functions
ID: interop_standard_server
Description: Verifies that when a server file exists in ui/, it imports
  startScenarioServer or createScenarioServer from @vrooli/api-base/server
  rather than building a server from scratch.
Why: The standard server functions handle proxy headers, CORS, static
  asset serving, and iframe embedding automatically. A server file
  that exists without importing these functions is likely using a
  custom setup that bypasses the Vrooli hosting stack.
Category: interop
Severity: medium
Slot: [C]
SlotFile: ui/server.js
TechStack: *
Recommendation: Import startScenarioServer or createScenarioServer from
  @vrooli/api-base/server. Use createScenarioServer with the setupRoutes
  callback when custom routes are needed.
Standard: vrooli-ui-interop-v1

GoodExample:
    // ui/server.js — standard server with no custom routes
    import { startScenarioServer } from '@vrooli/api-base/server';
    startScenarioServer({ uiPort: process.env.UI_PORT, distDir: './dist' });

BadExample:
    // ui/server.js — exists but does not use standard server functions
    import express from 'express';
    const app = express();
    app.use(express.static('dist'));

<test-case id="standard-server-start" should-fail="false">
  <description>Server file uses startScenarioServer</description>
  <input>
    [ui/server.js]
    import { startScenarioServer } from '@vrooli/api-base/server';
    startScenarioServer({
      uiPort: process.env.UI_PORT,
      distDir: './dist',
      serviceName: 'my-scenario',
    });
  </input>
</test-case>

<test-case id="standard-server-create" should-fail="false">
  <description>Server file uses createScenarioServer</description>
  <input>
    [ui/server.js]
    import { createScenarioServer } from '@vrooli/api-base/server';
    const app = createScenarioServer({
      uiPort: process.env.UI_PORT,
      distDir: './dist',
    });
    app.listen(process.env.UI_PORT);
  </input>
</test-case>

<test-case id="standard-server-no-file" should-fail="false">
  <description>No server file — rule skips</description>
  <input>
    [ui/src/main.tsx]
    import React from 'react';
  </input>
</test-case>

<test-case id="standard-server-missing-import" should-fail="true">
  <description>Server file exists but does not import standard functions</description>
  <input>
    [ui/server.js]
    import express from 'express';
    const app = express();
    app.use(express.static('dist'));
    app.listen(3000);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>standard server function not found</expected-message>
</test-case>

<test-case id="standard-server-empty-file" should-fail="true">
  <description>Server file exists but is nearly empty</description>
  <input>
    [ui/server.js]
    // TODO: set up server
    console.log("starting");
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>standard server function not found</expected-message>
</test-case>
*/

package interop

import (
	"app-monitor-api/rules"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	rules.Register("interop_standard_server", checkStandardServer)
}

func checkStandardServer(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_standard_server"

	for _, name := range serverFileNames {
		p := filepath.Join(ctx.ScenarioRoot, "ui", name)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		content := string(data)
		relPath := "ui/" + name

		if strings.Contains(content, "startScenarioServer") || strings.Contains(content, "createScenarioServer") {
			return rules.RuleResult{
				RuleID:  ruleID,
				Passed:  true,
				Message: "standard server function found in " + relPath,
			}
		}

		line := 1
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: "standard server function not found in " + relPath,
			Violations: []rules.Violation{{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Missing standard server function",
				Description:    relPath + " exists but does not use startScenarioServer or createScenarioServer from @vrooli/api-base/server",
				FilePath:       relPath,
				Line:           line,
				Recommendation: "Import startScenarioServer or createScenarioServer from @vrooli/api-base/server",
			}},
		}
	}

	return rules.RuleResult{
		RuleID:     ruleID,
		Skipped:    true,
		SkipReason: "no server file found",
		Message:    "no server file found in ui/; skipping",
	}
}
