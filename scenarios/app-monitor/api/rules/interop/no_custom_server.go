/*
Rule: No Custom Server
ID: interop_no_custom_server
Description: Ensures the scenario UI does not ship a custom server file
  (e.g. express(), createServer, http.listen) that would conflict
  with the Vrooli hosting layer.
Why: Vrooli serves embedded UIs through its own static-file proxy.
  A custom server in ui/server.* would bind its own port, bypass
  the proxy, and break iframe embedding. The scenario should rely
  on the Vrooli hosting stack for serving built assets.
Category: interop
Severity: medium
Slot: [C]
SlotFile: ui/server.js
TechStack: *
Recommendation: Remove the custom server file and let Vrooli serve your
  built assets. If you need SSR, use a Vite SSR plugin instead
  of a standalone server.
Standard: vrooli-ui-interop-v1

GoodExample:
    // No ui/server.js file at all - Vrooli serves the UI

BadExample:
    // ui/server.js
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
  <description>ui/server.ts uses createServer</description>
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

package interop

import (
	"os"
	"path/filepath"
	"regexp"

	"app-monitor-api/rules"
)

var noCustomServerPattern = regexp.MustCompile(`(?i)(express\(\)|createServer|http\.listen|\.listen\(\d)`)

func init() {
	rules.Register("interop_no_custom_server", checkNoCustomServer)
}

func checkNoCustomServer(ctx rules.CheckContext) rules.RuleResult {
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
			return rules.RuleResult{
				RuleID:  ruleID,
				Passed:  false,
				Message: "custom server code detected in " + relPath,
				Violations: []rules.Violation{{
					RuleID:         ruleID,
					Severity:       "medium",
					Title:          "Custom server detected",
					Description:    "custom server code detected in " + relPath + ": " + matches,
					FilePath:       relPath,
					Line:           line,
					CodeSnippet:    matches,
					Recommendation: "Remove " + relPath + " and let Vrooli serve your built assets",
				}},
			}
		}

		// Server file exists but has no custom server patterns - that is fine.
	}

	// No server files found or none contain custom server patterns.
	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no custom server detected",
	}
}
