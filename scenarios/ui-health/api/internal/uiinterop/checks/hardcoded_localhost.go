/*
Rule: Hardcoded Localhost
ID: interop_hardcoded_localhost
Description: Scans ui/src/ for hardcoded localhost:PORT references that
  bypass the Vrooli proxy and tunnel infrastructure.
Why: Hardcoded localhost URLs break when the scenario runs behind
  Cloudflare tunnel or inside an iframe. All network calls must go
  through the resolved API base so they work in every deployment
  context (local dev, tunnel, desktop app, SaaS).
Category: interop
Severity: high
Slot: [F]
SlotFile: ui/src/
TechStack: *
Recommendation: Replace localhost:PORT references with the value returned
  by resolveApiBase() from @vrooli/api-base.
Standard: vrooli-ui-interop-v1

GoodExample:
    import { resolveApiBase } from "@vrooli/api-base";
    const API = resolveApiBase();
    fetch(`${API}/health`);

BadExample:
    // Direct localhost reference
    fetch("http://localhost:4000/health");

<test-case id="no-localhost-refs" should-fail="false">
  <description>Source files have no localhost:PORT references</description>
  <input>
    [ui/src/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    const API = resolveApiBase();
    export const fetchHealth = () => fetch(`${API}/health`);
  </input>
</test-case>

<test-case id="localhost-in-comment-only" should-fail="false">
  <description>localhost appears only in comments, which are ignored</description>
  <input>
    [ui/src/api.ts]
    // Previously used http://localhost:4000/api
    // old endpoint: localhost:3000
    import { resolveApiBase } from "@vrooli/api-base";
    const API = resolveApiBase();
  </input>
</test-case>

<test-case id="localhost-in-source" should-fail="true">
  <description>Source file contains hardcoded localhost:PORT</description>
  <input>
    [ui/src/api.ts]
    const API_URL = "http://localhost:4000";
    export const fetchHealth = () => fetch(`${API_URL}/health`);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>localhost:PORT</expected-message>
</test-case>

<test-case id="localhost-in-multiple-files" should-fail="true">
  <description>Multiple source files contain hardcoded localhost</description>
  <input>
    [ui/src/api.ts]
    const API = "http://localhost:4000";
    [ui/src/socket.ts]
    const WS = "ws://localhost:4001";
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>localhost:PORT</expected-message>
</test-case>
*/

package checks

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_hardcoded_localhost", checkHardcodedLocalhost)
}

var localhostPattern = regexp.MustCompile(`localhost:\d+`)

func checkHardcodedLocalhost(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_hardcoded_localhost"

	srcDir := filepath.Join(ctx.ScenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/src/ directory not found",
			Message:    "ui/src/ directory not found; skipping",
		}
	}

	var violations []uiinterop.Violation

	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if _, skip := skipDirectories[info.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(info.Name())
		if _, ok := scanExtensions[ext]; !ok {
			return nil
		}
		if isTestFile(info.Name()) {
			return nil
		}

		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Skip comment-only lines.
			if strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "/*") ||
				strings.HasPrefix(trimmed, "*") {
				continue
			}

			if localhostPattern.MatchString(line) {
				rel, _ := filepath.Rel(ctx.ScenarioRoot, path)
				violations = append(violations, uiinterop.Violation{
					RuleID:         ruleID,
					Severity:       "high",
					Title:          "Hardcoded localhost:PORT reference",
					Description:    "Found localhost:PORT in " + rel + " at line " + strings.TrimSpace(line),
					FilePath:       rel,
					Line:           lineNum,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Replace with resolveApiBase() from @vrooli/api-base",
				})
			}
		}
		return nil
	})

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "found hardcoded localhost:PORT references in ui/src/",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no hardcoded localhost:PORT references found in ui/src/",
	}
}
