/*
Rule: Helmet Frame Ancestors
ID: interop_helmet_frame_ancestors
Description: Ensure a UI server's Helmet configuration disables frameguard and
  sets a CSP frameAncestors directive that allows trusted loopback iframe
  parents, so the scenario renders inside the Vrooli host frame without
  inviting clickjacking.
Why: Helmet's default X-Frame-Options:DENY blocks the Vrooli shell from
  embedding the scenario UI, while an unrestricted frameAncestors invites
  clickjacking. The balanced configuration disables frameguard and pins CSP
  frameAncestors to 'self' plus the loopback origins the host uses, with
  optional env-driven extras.
Category: interop
Severity: high
Slot: [B]
SlotFile: ui/server.js
TechStack: React
Recommendation: Set `frameguard: false` and a contentSecurityPolicy
  frameAncestors directive including 'self', http://localhost:*,
  http://127.0.0.1:* and http://[::1]:*.
Standard: vrooli-ui-security-v1

GoodExample:
    app.use(helmet({
      frameguard: false,
      contentSecurityPolicy: { useDefaults: true, directives: {
        frameAncestors: ["'self'", 'http://localhost:*', 'http://127.0.0.1:*', 'http://[::1]:*'] } }
    }));

BadExample:
    app.use(helmet());

<test-case id="helmet-configured-correctly" should-fail="false">
  <description>frameguard disabled and frameAncestors allows loopback origins</description>
  <input>
    [ui/server.js]
    const express = require('express');
    const helmet = require('helmet');
    const app = express();
    app.use(helmet({
      frameguard: false,
      contentSecurityPolicy: {
        useDefaults: true,
        directives: {
          defaultSrc: ["'self'"],
          frameAncestors: ["'self'", 'http://localhost:*', 'http://127.0.0.1:*', 'http://[::1]:*']
        }
      }
    }));
  </input>
</test-case>

<test-case id="helmet-missing-configuration" should-fail="true">
  <description>Helmet enabled without frameguard override or frameAncestors directive</description>
  <input>
    [ui/server.js]
    const express = require('express');
    const helmet = require('helmet');
    const app = express();
    app.use(helmet());
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>frameguard</expected-message>
</test-case>

<test-case id="helmet-missing-loopback" should-fail="true">
  <description>frameAncestors defined but missing mandatory loopback origins</description>
  <input>
    [ui/server.js]
    const express = require('express');
    const helmet = require('helmet');
    const app = express();
    app.use(helmet({
      frameguard: false,
      contentSecurityPolicy: {
        useDefaults: true,
        directives: {
          defaultSrc: ["'self'"],
          frameAncestors: ["'self'", 'https://trusted.example']
        }
      }
    }));
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>loopback</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_helmet_frame_ancestors", checkHelmetFrameAncestors)
}

var requiredLoopbackAncestors = []string{
	"http://localhost:*",
	"http://127.0.0.1:*",
	"http://[::1]:*",
}

func checkHelmetFrameAncestors(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_helmet_frame_ancestors"

	var violations []uiinterop.Violation
	evaluated := false

	for _, f := range sourceFiles(ctx, "ui") {
		if !isHelmetServerFile(f.RelPath) {
			continue
		}
		if !strings.Contains(f.Content, "helmet") {
			continue
		}
		evaluated = true
		violations = append(violations, evaluateHelmetFile(ruleID, f)...)
	}

	if !evaluated {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI server file using helmet found",
			Message:    "no UI server file using helmet found; skipping",
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "Helmet iframe protection misconfigured",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "Helmet frameguard disabled and frameAncestors allows loopback origins",
	}
}

func evaluateHelmetFile(ruleID string, f uiinterop.SourceFile) []uiinterop.Violation {
	var violations []uiinterop.Violation

	if !strings.Contains(f.Content, "frameguard: false") {
		violations = append(violations, helmetViolation(ruleID, f.RelPath, lineOf(f.Content, "helmet"),
			"Helmet configuration must disable frameguard (frameguard: false) and rely on CSP frameAncestors for iframe control"))
	}

	if !strings.Contains(f.Content, "frameAncestors") {
		violations = append(violations, helmetViolation(ruleID, f.RelPath, lineOf(f.Content, "contentSecurityPolicy"),
			"Helmet contentSecurityPolicy directives must define frameAncestors to permit trusted parents"))
		return violations
	}

	var missing []string
	for _, host := range requiredLoopbackAncestors {
		if !strings.Contains(f.Content, host) {
			missing = append(missing, host)
		}
	}
	if len(missing) > 0 {
		violations = append(violations, helmetViolation(ruleID, f.RelPath, lineOf(f.Content, "frameAncestors"),
			"frameAncestors must include loopback origins for local iframe embedding: "+strings.Join(missing, ", ")))
	}

	return violations
}

func helmetViolation(ruleID, filePath string, line int, msg string) uiinterop.Violation {
	if line <= 0 {
		line = 1
	}
	return uiinterop.Violation{
		RuleID:         ruleID,
		Severity:       "high",
		Title:          "Helmet iframe protection misconfigured",
		Description:    msg,
		FilePath:       filePath,
		Line:           line,
		Recommendation: "Disable helmet frameguard and configure CSP frameAncestors to include loopback origins plus optional trusted hosts via FRAME_ANCESTORS",
	}
}

// isHelmetServerFile reports whether a relative path is a UI server-style file
// worth evaluating for Helmet configuration.
func isHelmetServerFile(relPath string) bool {
	base := strings.ToLower(filepath.Base(relPath))
	switch filepath.Ext(base) {
	case ".js", ".ts", ".mjs", ".cjs", ".jsx", ".tsx":
	default:
		return false
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.Contains(name, "server") || strings.Contains(name, "proxy") {
		return true
	}
	switch name {
	case "app", "index", "main", "dev":
		return true
	}
	return false
}
