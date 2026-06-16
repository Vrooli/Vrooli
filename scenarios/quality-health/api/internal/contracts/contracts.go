package contracts

import "strings"

const (
	RuleTSConfigStrict        = "TS_CONFIG_STRICT"
	RuleESLintSafetyRules     = "ESLINT_SAFETY_RULES"
	RuleTSDangerousPatterns   = "TS_DANGEROUS_PATTERNS"
	RuleESLintTypedConfig     = "ESLINT_TYPED_CONFIG"
	RuleNodeBuildTypecheck    = "NODE_BUILD_TYPECHECK"
	RuleTestingConfigStrict   = "TESTING_CONFIG_LINT_STRICT"
	RuleGoModPresent          = "GO_MOD_PRESENT_FOR_API_OR_CLI"
	RuleGoLintConfigPresent   = "GO_LINT_CONFIG_PRESENT"
	RuleGoLintRequiredLinters = "GO_LINT_REQUIRED_LINTERS"
	RuleMakefileQualityGates  = "MAKEFILE_QUALITY_GATES"
	// RuleCoverageGap marks a discovered surface for which no quality contract
	// pack applies. It is an informational honesty signal, not a registry
	// contract, so it is intentionally absent from Registry().
	RuleCoverageGap = "QUALITY_COVERAGE_GAP"
)

type Contract struct {
	ID               string
	Title            string
	Category         string
	Severity         string
	Language         string
	Framework        string
	SurfaceKind      string
	RuleIDs          []string
	Description      string
	WhyItMatters     string
	Remediation      string
	AutofixAvailable bool
}

func Registry() []Contract {
	return []Contract{
		{
			// Language-keyed: applies to any TypeScript or JavaScript surface
			// regardless of surface name or framework. Framework/SurfaceKind are
			// left empty (wildcard) so a future pack can narrow without forcing
			// this one to. The owner of applicability is the language/tooling,
			// not the folder name.
			ID:          "typescript-static-quality",
			Title:       "TypeScript/JavaScript static quality",
			Category:    "typescript",
			Severity:    "error",
			Language:    "typescript",
			Framework:   "",
			SurfaceKind: "",
			RuleIDs: []string{
				RuleTSConfigStrict,
				RuleESLintSafetyRules,
				RuleTSDangerousPatterns,
				RuleESLintTypedConfig,
				RuleNodeBuildTypecheck,
			},
			Description:      "Enforces strict TypeScript, typed ESLint, safety rules, guardrail comments, typechecked builds, and suppression visibility for any TypeScript or JavaScript surface.",
			WhyItMatters:     "These rules prevent agents from hiding runtime crashes by weakening type and lint settings.",
			Remediation:      "Restore the strict config values, keep the required safety comments, and fix source code with null checks, optional chaining, nullish coalescing, and type guards.",
			AutofixAvailable: true,
		},
		{
			// Language-keyed: applies to any Go surface regardless of surface
			// name (api/cli/worker/...). SurfaceKind is left empty (wildcard).
			ID:          "go-static-quality",
			Title:       "Go lint baseline",
			Category:    "go",
			Severity:    "error",
			Language:    "go",
			SurfaceKind: "",
			RuleIDs: []string{
				RuleGoModPresent,
				RuleGoLintConfigPresent,
				RuleGoLintRequiredLinters,
			},
			Description:  "Enforces Go module and golangci-lint baseline setup for any Go surface.",
			WhyItMatters: "Without local module and linter configuration, Go quality checks become environment-dependent and easier to bypass.",
			Remediation:  "Keep go.mod and .golangci.yml next to each Go surface and enable the baseline linters.",
		},
		{
			ID:          "scenario-quality-gates",
			Title:       "Scenario-level quality gates",
			Category:    "scenario",
			Severity:    "error",
			SurfaceKind: "scenario",
			RuleIDs: []string{
				RuleTestingConfigStrict,
				RuleMakefileQualityGates,
			},
			Description:  "Enforces .vrooli/testing.json strict lint handlers and Makefile quality targets for discovered language surfaces.",
			WhyItMatters: "Scenario-level gates keep Test Genie and local developer commands from silently accepting weak lint/type behavior.",
			Remediation:  "Set strict lint handlers in .vrooli/testing.json and ensure Makefile quality targets run real lint/type/format commands.",
		},
	}
}

func List(language, framework, surfaceKind string, ruleIDs []string) []Contract {
	wantRules := map[string]bool{}
	for _, id := range ruleIDs {
		wantRules[strings.TrimSpace(id)] = true
	}
	var out []Contract
	for _, c := range Registry() {
		if language != "" && c.Language != "" && !strings.EqualFold(c.Language, language) {
			continue
		}
		if framework != "" && c.Framework != "" && !strings.EqualFold(c.Framework, framework) {
			continue
		}
		if surfaceKind != "" && c.SurfaceKind != "" && !strings.Contains(c.SurfaceKind, surfaceKind) {
			continue
		}
		if len(wantRules) > 0 && !contractHasAnyRule(c, wantRules) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func ByRule(ruleID string) (Contract, bool) {
	for _, c := range Registry() {
		for _, id := range c.RuleIDs {
			if id == ruleID {
				return c, true
			}
		}
	}
	return Contract{}, false
}

func contractHasAnyRule(c Contract, want map[string]bool) bool {
	for _, id := range c.RuleIDs {
		if want[id] {
			return true
		}
	}
	return false
}
