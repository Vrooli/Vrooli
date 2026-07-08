/*
Rule: Router Basename Prop
ID: interop_router_basename
Description: Checks that BrowserRouter (or its alias) includes a basename
  prop so the UI can be mounted at a sub-path behind the Vrooli proxy.
Why: Without basename, all route matching assumes the app lives at "/".
  When Vrooli deploys the scenario at a sub-path (e.g., /scenarios/foo/),
  every route and link will be wrong. The basename prop adjusts the
  router's internal prefix to match the deployment path.
Category: interop
Severity: high
Slot: [E]
SlotFile: ui/src/App.tsx
TechStack: React
Recommendation: Add a proxy-aware basename prop to BrowserRouter using
  getProxyInfo() from @vrooli/api-base to resolve the deployment path.
Standard: vrooli-ui-interop-v1

GoodExample:
    import { BrowserRouter } from "react-router-dom";
    import { getProxyInfo } from "@vrooli/api-base";

    function getRouterBasename() {
      const proxyInfo = getProxyInfo();
      const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
      return proxyPath ? proxyPath.replace(/\/+$/, "") : "";
    }

    function App() {
      return (
        <BrowserRouter basename={getRouterBasename()}>
          <Routes>...</Routes>
        </BrowserRouter>
      );
    }

BadExample:
    import { BrowserRouter } from "react-router-dom";
    function App() {
      return (
        <BrowserRouter>
          <Routes>...</Routes>
        </BrowserRouter>
      );
    }

<test-case id="router-has-basename" should-fail="false">
  <description>BrowserRouter has a proxy-aware basename prop</description>
  <input>
    [ui/src/App.tsx]
    import { BrowserRouter } from "react-router-dom";
    import { getProxyInfo } from "@vrooli/api-base";
    function getRouterBasename() {
      const proxyInfo = getProxyInfo();
      const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
      return proxyPath ? proxyPath.replace(/\/+$/, "") : "";
    }
    function App() {
      return (
        <BrowserRouter basename={getRouterBasename()}>
          <Routes />
        </BrowserRouter>
      );
    }
    export default App;
  </input>
</test-case>

<test-case id="router-aliased-with-basename" should-fail="false">
  <description>Aliased BrowserRouter has a basename prop</description>
  <input>
    [ui/src/App.tsx]
    import { BrowserRouter as Router } from "react-router-dom";
    function App() {
      return (
        <Router basename="/">
          <Routes />
        </Router>
      );
    }
    export default App;
  </input>
</test-case>

<test-case id="router-no-browser-router" should-fail="false">
  <description>No BrowserRouter used — may use HashRouter; rule skips</description>
  <input>
    [ui/src/App.tsx]
    import { HashRouter } from "react-router-dom";
    function App() {
      return (
        <HashRouter>
          <Routes />
        </HashRouter>
      );
    }
    export default App;
  </input>
</test-case>

<test-case id="router-missing-basename" should-fail="true">
  <description>BrowserRouter without basename prop</description>
  <input>
    [ui/src/App.tsx]
    import { BrowserRouter } from "react-router-dom";
    function App() {
      return (
        <BrowserRouter>
          <Routes />
        </BrowserRouter>
      );
    }
    export default App;
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>basename prop not found</expected-message>
</test-case>
*/

package checks

import (
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_router_basename", checkRouterBasename)
}

var browserRouterImport = regexp.MustCompile(`import\s+\{[^}]*BrowserRouter(?:\s+as\s+(\w+))?[^}]*\}\s+from\s+['"]react-router-dom['"]`)

func checkRouterBasename(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_router_basename"

	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/src/ directory not found",
			Message:    "ui/src/ directory not found; skipping",
		}
	}

	type routerHit struct {
		relPath     string
		content     string
		alias       string // empty means BrowserRouter used directly
		line        int
		hasBasename bool
	}

	var hits []routerHit

	for _, f := range files {
		// Check for BrowserRouter import.
		m := browserRouterImport.FindStringSubmatch(f.Content)
		if m == nil {
			continue
		}

		alias := "BrowserRouter"
		if m[1] != "" {
			alias = m[1]
		}

		// Build a pattern for the JSX usage of this router name.
		jsxPattern := regexp.MustCompile(`<` + regexp.QuoteMeta(alias) + `[\s>]`)
		basenamePattern := regexp.MustCompile(`<` + regexp.QuoteMeta(alias) + `\s[^>]*basename[\s=]`)

		lines := strings.Split(f.Content, "\n")
		for i, line := range lines {
			if jsxPattern.MatchString(line) {
				// Check if basename appears on this line or nearby (multi-line JSX).
				// We check a window of lines around the JSX tag.
				windowStart := i
				windowEnd := i + 5
				if windowEnd > len(lines) {
					windowEnd = len(lines)
				}
				window := strings.Join(lines[windowStart:windowEnd], "\n")

				hits = append(hits, routerHit{
					relPath:     f.RelPath,
					content:     f.Content,
					alias:       alias,
					line:        i + 1,
					hasBasename: basenamePattern.MatchString(window),
				})
				break // Only check first JSX usage per file.
			}
		}
	}

	if len(hits) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "may use HashRouter or MemoryRouter",
			Message:    "no BrowserRouter usage found; may use HashRouter or MemoryRouter",
		}
	}

	var violations []uiinterop.Violation
	for _, h := range hits {
		if !h.hasBasename {
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "high",
				Title:          "BrowserRouter missing basename prop",
				Description:    "basename prop not found on <" + h.alias + "> in " + h.relPath,
				FilePath:       h.relPath,
				Line:           h.line,
				Recommendation: "Add a proxy-aware basename prop using getProxyInfo() from @vrooli/api-base to <" + h.alias + ">",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "BrowserRouter found without basename prop",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "BrowserRouter has basename prop",
	}
}
