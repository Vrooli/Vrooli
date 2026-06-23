/*
Rule: No Custom Server
ID: interop_no_custom_server
Description: Ensures the scenario UI does not use a custom server
  (e.g. express(), http.createServer, http.listen) that bypasses
  the standard Vrooli server functions. Scenarios should use
  startScenarioServer() or createScenarioServer() from
  @vrooli/api-base/server instead.
Why: Custom servers bypass the Vrooli hosting stack — proxy headers,
  CORS, static asset serving, and iframe embedding all break. The
  standard server functions (startScenarioServer, createScenarioServer)
  handle these automatically. If custom route setup is needed, use
  createScenarioServer with the setupRoutes callback.
Category: interop
Severity: medium
Slot: [C]
SlotFile: ui/server.js
TechStack: *
Recommendation: Replace the custom server with startScenarioServer() or
  createScenarioServer() from @vrooli/api-base/server. Use the
  setupRoutes callback for custom routes.
Standard: vrooli-ui-interop-v1

GoodExample:
    // ui/server.js — uses the standard Vrooli server function
    import { startScenarioServer } from '@vrooli/api-base/server';
    startScenarioServer({ uiPort: process.env.UI_PORT, distDir: './dist' });

BadExample:
    // ui/server.js — custom Express server bypasses Vrooli hosting
    const express = require('express');
    const app = express();
    app.use(express.static('dist'));
    app.listen(3000);

<test-case id="no-server-file" should-fail="false">
  <description>No server file exists in ui/</description>
  <input>
    [ui/src/main.tsx]
    import React from 'react';
    ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
  </input>
</test-case>

<test-case id="standard-server-start" should-fail="false">
  <description>ui/server.js uses startScenarioServer (no custom patterns)</description>
  <input>
    [ui/server.js]
    import { startScenarioServer } from '@vrooli/api-base/server';
    startScenarioServer({ uiPort: process.env.UI_PORT, distDir: './dist' });
  </input>
</test-case>

<test-case id="standard-server-create" should-fail="false">
  <description>ui/server.js uses createScenarioServer with setupRoutes</description>
  <input>
    [ui/server.js]
    import { createScenarioServer } from '@vrooli/api-base/server';
    const app = createScenarioServer({
      uiPort: process.env.UI_PORT,
      distDir: './dist',
      setupRoutes: (expressApp) => {
        expressApp.get('/custom', (req, res) => res.json({ ok: true }));
      },
    });
    app.listen(process.env.UI_PORT);
  </input>
</test-case>

<test-case id="custom-server-express" should-fail="true">
  <description>ui/server.js uses express()</description>
  <input>
    [ui/server.js]
    const express = require('express');
    const app = express();
    app.use(express.static('dist'));
    app.listen(3000);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>custom server code detected</expected-message>
</test-case>

<test-case id="custom-server-create-server" should-fail="true">
  <description>ui/server.ts uses http.createServer</description>
  <input>
    [ui/server.ts]
    import http from 'http';
    const server = http.createServer((req, res) => {
      res.end('hello');
    });
    server.listen(8080);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>custom server code detected</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"regexp"

	"ui-health/internal/uiinterop"
)

var noCustomServerPattern = regexp.MustCompile(`(?i)(express\(\)|createServer|http\.listen|\.listen\(\d)`)

func init() {
	uiinterop.Register("interop_no_custom_server", checkNoCustomServer)
}

func checkNoCustomServer(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_no_custom_server"

	for _, name := range serverFileNames {
		p := filepath.Join(ctx.ScenarioRoot, "ui", name)
		data, err := os.ReadFile(p)
		if err != nil {
			continue // file does not exist, try next
		}

		content := string(data)
		relPath := "ui/" + name

		if noCustomServerPattern.MatchString(content) {
			matches := noCustomServerPattern.FindString(content)
			line := lineOf(content, matches)
			return uiinterop.RuleResult{
				RuleID:  ruleID,
				Passed:  false,
				Message: "custom server code detected in " + relPath,
				Violations: []uiinterop.Violation{{
					RuleID:         ruleID,
					Severity:       "medium",
					Title:          "Custom server detected",
					Description:    "custom server code detected in " + relPath + ": " + matches,
					FilePath:       relPath,
					Line:           line,
					CodeSnippet:    matches,
					Recommendation: "Replace custom server code in " + relPath + " with startScenarioServer() or createScenarioServer() from @vrooli/api-base/server",
				}},
			}
		}

		// Server file exists but has no custom server patterns - that is fine.
	}

	// No server files found or none contain custom server patterns.
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no custom server detected",
	}
}
