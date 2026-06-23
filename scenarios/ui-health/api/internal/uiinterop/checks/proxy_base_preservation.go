/*
Rule: Proxy Base Preservation
ID: interop_proxy_base_preserved
Description: Flags files that call resolveApiBase but then rebuild the API
  URL using window.location.origin, which discards the proxy-aware base
  that resolveApiBase computed.
Why: resolveApiBase returns a URL that accounts for iframe embedding,
  Cloudflare tunnel paths, and custom proxy prefixes. Overwriting it
  with window.location.origin loses all of that context and breaks
  the UI when it runs behind a reverse proxy or inside an iframe.
Category: interop
Severity: high
Slot: [F]
SlotFile: ui/src/
TechStack: api-base
Recommendation: Remove the window.location.origin override and use the
  value from resolveApiBase() directly for all API calls.
Standard: vrooli-ui-interop-v1

GoodExample:
    import { resolveApiBase } from "@vrooli/api-base";
    const API = resolveApiBase();
    fetch(`${API}/health`);

BadExample:
    import { resolveApiBase } from "@vrooli/api-base";
    resolveApiBase(); // called but then ignored
    const API = window.location.origin + "/api";
    fetch(`${API}/health`);

<test-case id="proxy-base-clean" should-fail="false">
  <description>File uses resolveApiBase without window.location.origin override</description>
  <input>
    [ui/src/config/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const API = resolveApiBase();
  </input>
</test-case>

<test-case id="proxy-base-no-resolve" should-fail="false">
  <description>File uses window.location.origin but does not call resolveApiBase</description>
  <input>
    [ui/src/utils/url.ts]
    const baseUrl = window.location.origin;
    export const getUrl = (path) => baseUrl + path;
  </input>
</test-case>

<test-case id="proxy-base-overridden" should-fail="true">
  <description>File calls resolveApiBase but overrides with window.location.origin</description>
  <input>
    [ui/src/config/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    resolveApiBase();
    const API = window.location.origin + "/api";
    export { API };
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>window.location.origin</expected-message>
</test-case>

<test-case id="proxy-base-overridden-with-wrapper" should-fail="true">
  <description>File rebuilds URL with a wrapper function around window.location.origin</description>
  <input>
    [ui/src/api/client.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    const base = resolveApiBase();
    let apiUrl = getOrigin(window.location.origin);
    export { apiUrl };
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>window.location.origin</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_proxy_base_preserved", checkProxyBasePreserved)
}

var windowOriginPattern = regexp.MustCompile(`(?m)(?:const|let|var)\s+([a-zA-Z_$][\w$]*)\s*=\s*(?:[a-zA-Z_$][\w$]*\()?\s*window\.location\.origin`)

func checkProxyBasePreserved(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_proxy_base_preserved"

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

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(data)

		// Only flag files that also use resolveApiBase.
		if !strings.Contains(content, "resolveApiBase(") {
			return nil
		}

		if windowOriginPattern.MatchString(content) {
			rel, _ := filepath.Rel(ctx.ScenarioRoot, path)
			line := 0
			lines := strings.Split(content, "\n")
			for i, l := range lines {
				if windowOriginPattern.MatchString(l) {
					line = i + 1
					break
				}
			}
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "high",
				Title:          "Proxy base overridden with window.location.origin",
				Description:    rel + " calls resolveApiBase but also rebuilds the URL using window.location.origin",
				FilePath:       rel,
				Line:           line,
				Recommendation: "Remove the window.location.origin override and use resolveApiBase() directly",
			})
		}
		return nil
	})

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "found files that override resolveApiBase with window.location.origin",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no proxy base overrides found in ui/src/",
	}
}
