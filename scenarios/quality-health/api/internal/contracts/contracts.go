package contracts

import (
	"strings"

	"quality-health/internal/rules"
)

const (
	RuleTSConfigStrict            = rules.RuleTSConfigStrict
	RuleESLintSafetyRules         = rules.RuleESLintSafetyRules
	RuleTSDangerousPatterns       = rules.RuleTSDangerousPatterns
	RuleESLintTypedConfig         = rules.RuleESLintTypedConfig
	RuleNodeBuildTypecheck        = rules.RuleNodeBuildTypecheck
	RuleTestingConfigStrict       = rules.RuleTestingConfigStrict
	RuleGoModPresent              = rules.RuleGoModPresent
	RuleGoLintConfigPresent       = rules.RuleGoLintConfigPresent
	RuleGoLintRequiredLinters     = rules.RuleGoLintRequiredLinters
	RuleGoDangerousPatterns       = rules.RuleGoDangerousPatterns
	RuleScenarioPrivilegeBoundary = rules.RuleScenarioPrivilegeBoundary
	RuleMakefileQualityGates      = rules.RuleMakefileQualityGates
	RuleShellSyntaxLint           = rules.RuleShellSyntaxLint
	RuleCoverageGap               = rules.RuleCoverageGap
)

const (
	FixClassAutofix       = rules.FixClassAutofix
	FixClassDetectionOnly = rules.FixClassDetectionOnly
)

type Rule = rules.Rule

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
	FixClass         string
}

func Rules() []Rule {
	return rules.Registry()
}

func Registry() []Contract {
	return enrichContracts([]Contract{
		{
			ID:           "typescript-static-quality",
			Title:        "TypeScript/JavaScript static quality",
			Category:     "typescript",
			Severity:     "error",
			Language:     "typescript",
			Framework:    "",
			SurfaceKind:  "",
			Description:  "Enforces strict TypeScript, typed ESLint, safety rules, guardrail comments, typechecked builds, and suppression visibility for any TypeScript or JavaScript surface.",
			WhyItMatters: "These rules prevent agents from hiding runtime crashes by weakening type and lint settings.",
			Remediation:  "Restore the strict config values, keep the required safety comments, and fix source code with null checks, optional chaining, nullish coalescing, and type guards.",
		},
		{
			ID:           "go-static-quality",
			Title:        "Go lint baseline",
			Category:     "go",
			Severity:     "error",
			Language:     "go",
			SurfaceKind:  "",
			Description:  "Enforces Go module and golangci-lint baseline setup for any Go surface.",
			WhyItMatters: "Without local module and linter configuration, Go quality checks become environment-dependent and easier to bypass.",
			Remediation:  "Keep go.mod and .golangci.yml next to each Go surface and enable the baseline linters.",
		},
		{
			ID:           "scenario-quality-gates",
			Title:        "Scenario-level quality gates",
			Category:     "scenario",
			Severity:     "error",
			SurfaceKind:  "scenario",
			Description:  "Enforces .vrooli/testing.json strict lint handlers and Makefile quality targets for discovered language surfaces.",
			WhyItMatters: "Scenario-level gates keep Test Genie and local developer commands from silently accepting weak lint/type behavior.",
			Remediation:  "Set strict lint handlers in .vrooli/testing.json and ensure Makefile quality targets run real lint/type/format commands.",
		},
	})
}

func enrichContracts(in []Contract) []Contract {
	byContract := map[string][]rules.Rule{}
	for _, rule := range rules.Registry() {
		byContract[rule.ContractID] = append(byContract[rule.ContractID], rule)
	}
	for i := range in {
		contractRules := byContract[in[i].ID]
		allAutofix := len(contractRules) > 0
		anyAutofix := false
		for _, rule := range contractRules {
			in[i].RuleIDs = append(in[i].RuleIDs, rule.ID)
			if rule.FixClass != FixClassAutofix {
				allAutofix = false
				continue
			}
			anyAutofix = true
		}
		in[i].AutofixAvailable = anyAutofix
		if allAutofix {
			in[i].FixClass = FixClassAutofix
		} else {
			in[i].FixClass = FixClassDetectionOnly
		}
	}
	return in
}

func ByRuleID(ruleID string) (Rule, bool) {
	rule, ok := rules.ByID(ruleID)
	if !ok {
		return Rule{}, false
	}
	if c, ok := ByRule(ruleID); ok {
		rule.WhyItMatters = c.WhyItMatters
		rule.Remediation = c.Remediation
	}
	return rule, true
}

func List(language, framework, surfaceKind string, ruleIDs []string) []Contract {
	wantRules := map[string]bool{}
	for _, id := range ruleIDs {
		wantRules[strings.TrimSpace(id)] = true
	}
	var out []Contract
	for _, c := range Registry() {
		if !contractMatches(c, language, framework, surfaceKind) {
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

func contractMatches(c Contract, language, framework, surfaceKind string) bool {
	if language != "" && c.Language != "" && !strings.EqualFold(c.Language, rules.NormalizeLanguage(language)) {
		return false
	}
	if framework != "" && c.Framework != "" && !strings.EqualFold(c.Framework, framework) {
		return false
	}
	if surfaceKind != "" && c.SurfaceKind != "" && !strings.Contains(c.SurfaceKind, surfaceKind) {
		return false
	}
	return true
}

func contractHasAnyRule(c Contract, want map[string]bool) bool {
	for _, id := range c.RuleIDs {
		if want[id] {
			return true
		}
	}
	return false
}
