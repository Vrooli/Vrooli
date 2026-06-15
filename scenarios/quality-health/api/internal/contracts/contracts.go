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
			ID:          "typescript-react-vite-ui",
			Title:       "TypeScript React/Vite UI static quality",
			Category:    "typescript",
			Severity:    "error",
			Language:    "typescript",
			Framework:   "react-vite",
			SurfaceKind: "ui",
			RuleIDs: []string{
				RuleTSConfigStrict,
				RuleESLintSafetyRules,
				RuleTSDangerousPatterns,
				RuleESLintTypedConfig,
				RuleNodeBuildTypecheck,
			},
			Description:      "Enforces strict TypeScript, typed ESLint, safety rules, guardrail comments, typechecked builds, and suppression visibility for React/Vite UI surfaces.",
			WhyItMatters:     "These rules prevent agents from hiding UI runtime crashes by weakening type and lint settings.",
			Remediation:      "Restore the strict config values, keep the required safety comments, and fix source code with null checks, optional chaining, nullish coalescing, and type guards.",
			AutofixAvailable: true,
		},
		{
			ID:          "go-api-cli-quality",
			Title:       "Go API/CLI lint baseline",
			Category:    "go",
			Severity:    "error",
			Language:    "go",
			SurfaceKind: "api,cli",
			RuleIDs: []string{
				RuleGoModPresent,
				RuleGoLintConfigPresent,
				RuleGoLintRequiredLinters,
			},
			Description:  "Enforces Go module and golangci-lint baseline setup for API and CLI surfaces.",
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
