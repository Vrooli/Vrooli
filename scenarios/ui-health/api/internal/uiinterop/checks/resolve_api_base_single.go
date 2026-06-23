/*
Rule: Resolve API Base Single Import
ID: interop_resolve_api_base_single
Description: Ensures resolveApiBase is imported in at most 2 files under
  ui/src/ to prevent scattered API base resolution that is hard to
  maintain and update.
Why: When resolveApiBase is called in many files, each call site may
  drift in how it uses the returned URL. A single or dual-file pattern
  (one config file + one hook/wrapper) keeps the API base centralized
  and easy to change.
Category: interop
Severity: high
Slot: [F]
SlotFile: ui/src/
TechStack: api-base
Recommendation: Consolidate resolveApiBase usage into a single config or
  hook file (e.g., ui/src/config/api.ts) and import the resolved value
  from there everywhere else.
Standard: vrooli-ui-interop-v1

GoodExample:
    // ui/src/config/api.ts — single source of truth
    import { resolveApiBase } from "@vrooli/api-base";
    export const API_URL = resolveApiBase();

    // ui/src/hooks/useApi.ts — allowed second usage
    import { resolveApiBase } from "@vrooli/api-base";
    export const useApi = () => { ... };

BadExample:
    // resolveApiBase scattered across 4+ files
    // ui/src/pages/Home.tsx
    import { resolveApiBase } from "@vrooli/api-base";
    // ui/src/pages/Settings.tsx
    import { resolveApiBase } from "@vrooli/api-base";
    // ui/src/components/Header.tsx
    import { resolveApiBase } from "@vrooli/api-base";

<test-case id="api-base-single-file" should-fail="false">
  <description>resolveApiBase used in exactly 1 file</description>
  <input>
    [ui/src/config/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const API_URL = resolveApiBase();
  </input>
</test-case>

<test-case id="api-base-two-files" should-fail="false">
  <description>resolveApiBase used in exactly 2 files</description>
  <input>
    [ui/src/config/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const API_URL = resolveApiBase();
    [ui/src/hooks/useApi.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export function useApi() { return resolveApiBase(); }
  </input>
</test-case>

<test-case id="api-base-not-found" should-fail="true">
  <description>resolveApiBase not used anywhere</description>
  <input>
    [ui/src/App.tsx]
    import React from "react";
    export default function App() { return <div />; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>not found</expected-message>
</test-case>

<test-case id="api-base-too-many" should-fail="true">
  <description>resolveApiBase scattered across 3 files</description>
  <input>
    [ui/src/config/api.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const API_URL = resolveApiBase();
    [ui/src/pages/Home.tsx]
    import { resolveApiBase } from "@vrooli/api-base";
    const url = resolveApiBase();
    [ui/src/pages/Settings.tsx]
    import { resolveApiBase } from "@vrooli/api-base";
    const url = resolveApiBase();
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>found in 3 files</expected-message>
</test-case>

<test-case id="api-base-test-setup-excluded" should-fail="false">
  <description>A test-harness setup file that mocks resolveApiBase is test infrastructure and does not count toward the production usage limit</description>
  <input>
    [ui/src/lib/api-internals.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const API_BASE = resolveApiBase({ appendSuffix: true });
    [ui/src/lib/connect.ts]
    import { resolveApiBase } from "@vrooli/api-base";
    export const transport = createTransport({ baseUrl: resolveApiBase() });
    [ui/src/test-setup.ts]
    vi.mock("./lib/api-internals", () => ({
      resolveApiBase: () => "https://example.test/api/v1",
    }));
  </input>
</test-case>
*/

package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_resolve_api_base_single", checkResolveApiBaseSingle)
}

func checkResolveApiBaseSingle(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_resolve_api_base_single"

	srcDir := filepath.Join(ctx.ScenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/src/ directory not found",
			Message:    "ui/src/ directory not found; skipping",
		}
	}

	var filesWithUsage []string

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

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(data), "resolveApiBase") {
			rel, _ := filepath.Rel(ctx.ScenarioRoot, path)
			filesWithUsage = append(filesWithUsage, rel)
		}
		return nil
	})

	count := len(filesWithUsage)

	if count == 0 {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: "resolveApiBase not found in any file under ui/src/",
			Violations: []uiinterop.Violation{{
				RuleID:         ruleID,
				Severity:       "high",
				Title:          "resolveApiBase not found",
				Description:    "No file under ui/src/ references resolveApiBase",
				FilePath:       "ui/src/",
				Recommendation: "Import and call resolveApiBase from @vrooli/api-base in a central config file",
			}},
		}
	}

	if count <= 2 {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: fmt.Sprintf("resolveApiBase found in %d file(s): %s", count, strings.Join(filesWithUsage, ", ")),
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: fmt.Sprintf("resolveApiBase found in %d files; expected at most 2", count),
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "high",
			Title:          "resolveApiBase used in too many files",
			Description:    fmt.Sprintf("resolveApiBase found in %d files: %s", count, strings.Join(filesWithUsage, ", ")),
			FilePath:       "ui/src/",
			Recommendation: "Consolidate resolveApiBase into a single config/hook file and import the resolved value elsewhere",
		}},
	}
}
