/*
Rule: Secure Tunnel Proxy
ID: interop_secure_tunnel
Description: Ensures custom server files route API calls through the
  proxyToApi function so that requests are tunneled securely through
  the Vrooli infrastructure rather than hitting localhost directly.
Why: When a scenario uses a custom server (Express, http.createServer,
  etc.), API requests must be proxied through proxyToApi to ensure
  they traverse the Cloudflare tunnel and respect auth, CORS, and
  rate-limit headers. Direct forwarding bypasses these protections.
Category: interop
Severity: high
Slot: [C]
SlotFile: ui/server.js
TechStack: *
Recommendation: Define a proxyToApi function in your custom server that
  forwards /api requests through the Vrooli tunnel endpoint.
Standard: ui-health-v1

GoodExample:
    import express from "express";
    const app = express();

    async function proxyToApi(req, res) {
      const upstream = process.env.VROOLI_API_URL + req.url;
      const resp = await fetch(upstream, { headers: req.headers });
      res.status(resp.status).send(await resp.text());
    }

    app.use("/api", proxyToApi);
    app.listen(3000);

BadExample:
    import express from "express";
    const app = express();

    // No proxyToApi — API calls go directly to localhost
    app.use("/api", (req, res) => {
      fetch("http://localhost:4000" + req.url).then(r => r.text()).then(t => res.send(t));
    });
    app.listen(3000);

<test-case id="secure-tunnel-proxy-present" should-fail="false">
  <description>Custom server defines proxyToApi function</description>
  <input>
    [ui/server.js]
    import express from "express";
    const app = express();

    async function proxyToApi(req, res) {
      const upstream = process.env.VROOLI_API_URL + req.url;
      const resp = await fetch(upstream, { headers: req.headers });
      res.status(resp.status).send(await resp.text());
    }

    app.use("/api", proxyToApi);
    app.listen(3000);
  </input>
</test-case>

<test-case id="secure-tunnel-proxy-const" should-fail="false">
  <description>Custom server defines proxyToApi as const arrow function</description>
  <input>
    [ui/server.ts]
    import express from "express";
    const app = express();

    const proxyToApi = async (req, res) => {
      const upstream = process.env.VROOLI_API_URL + req.url;
      const resp = await fetch(upstream, { headers: req.headers });
      res.status(resp.status).send(await resp.text());
    };

    app.use("/api", proxyToApi);
    app.listen(3000);
  </input>
</test-case>

<test-case id="secure-tunnel-no-custom-server" should-fail="false">
  <description>No custom server file exists — rule is skipped</description>
  <input>
    [ui/src/main.tsx]
    import React from "react";
  </input>
</test-case>

<test-case id="secure-tunnel-proxy-missing" should-fail="true">
  <description>Custom Express server without proxyToApi</description>
  <input>
    [ui/server.js]
    import express from "express";
    const app = express();

    app.use("/api", (req, res) => {
      fetch("http://localhost:4000" + req.url).then(r => r.text()).then(t => res.send(t));
    });
    app.listen(3000);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>proxyToApi function not found</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"regexp"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_secure_tunnel", checkSecureTunnel)
}

var (
	customServerPattern = regexp.MustCompile(`(?i)(express\(\)|createServer|http\.listen|\.listen\(\d)`)
	proxyToApiPattern   = regexp.MustCompile(`(?m)^(?:\s*(?:async\s+)?function\s+proxyToApi\s*\(|\s*(?:const|let|var)\s+proxyToApi\s*=)`)
)

func checkSecureTunnel(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_secure_tunnel"

	for _, name := range serverFileNames {
		serverPath := filepath.Join(ctx.ScenarioRoot, "ui", name)
		data, err := os.ReadFile(serverPath)
		if err != nil {
			continue
		}

		content := string(data)

		// Check if this is actually a custom server.
		if !customServerPattern.MatchString(content) {
			return uiinterop.RuleResult{
				RuleID:     ruleID,
				Skipped:    true,
				SkipReason: "not a custom server",
				Message:    "ui/" + name + " does not appear to be a custom server",
			}
		}

		// Custom server found — check for proxyToApi.
		if proxyToApiPattern.MatchString(content) {
			return uiinterop.RuleResult{
				RuleID:  ruleID,
				Passed:  true,
				Message: "proxyToApi function found in ui/" + name,
			}
		}

		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: "proxyToApi function not found in custom server ui/" + name,
			Violations: []uiinterop.Violation{{
				RuleID:         ruleID,
				Severity:       "high",
				Title:          "Missing proxyToApi in custom server",
				Description:    "Custom server ui/" + name + " does not define a proxyToApi function to route API calls through the Vrooli tunnel",
				FilePath:       "ui/" + name,
				Recommendation: "Define a proxyToApi function that forwards /api requests through the Vrooli tunnel endpoint",
			}},
		}
	}

	return uiinterop.RuleResult{
		RuleID:     ruleID,
		Skipped:    true,
		SkipReason: "no custom server file found",
		Message:    "no custom server file found; skipping",
	}
}
