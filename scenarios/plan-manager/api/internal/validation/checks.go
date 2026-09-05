package validation

import (
	"fmt"
	"sort"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

// compileValidationChecks is the sole compiler from an effective boundary to
// executable work. It deliberately produces one scenario oracle (not a
// snapshot-status companion) and one informational repo diff at most.
func compileValidationChecks(p planmodel.Plan, refs []planmodel.Reference, boundary planmodel.ChangeBoundary) []ValidationCheck {
	scenarios := map[string]bool{}
	for _, name := range boundary.AffectedScenarios() {
		scenarios[name] = true
	}
	repoLevel := false
	for _, ref := range refs {
		if ref.Kind != planmodel.ReferenceCode || ref.Future {
			continue
		}
		if name := scenarioFromTarget(ref.Target); name != "" {
			scenarios[name] = true
		} else {
			repoLevel = true
		}
	}
	paths := append([]string(nil), boundary.RepoPaths()...)
	if p.RegressionAnchor.Strategy == planmodel.AnchorStrategyHeadShaAllowlist {
		paths = append(paths, p.RegressionAnchor.AllowlistPaths...)
	}
	paths = uniqueSortedStrings(paths)
	baseline := strings.TrimSpace(p.RegressionAnchor.BaselineName)
	var names []string
	for name := range scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	checks := make([]ValidationCheck, 0, len(names)+1)
	if baselineSet := p.BaselineSet; strings.TrimSpace(baselineSet.Name) != "" {
		selected := intersectScenarioTargets(names, baselineSet.ScenarioTargets)
		if len(selected) > 0 {
			checks = append(checks, ValidationCheck{
				Kind: ValidationCheckCollectionDiff, Baseline: baselineSet.Name, Scenarios: selected,
				SemanticKey: "collection-diff:" + baselineSet.Name + ":" + strings.Join(selected, ","),
				Command:     "git-control-tower baseline collection diff --name " + baselineSet.Name + " --scenario " + strings.Join(selected, ","), Oracle: true,
			})
		}
	} else if baseline != "" && !strings.ContainsAny(baseline, " \t\r\n") {
		for _, name := range names {
			checks = append(checks, ValidationCheck{
				Kind: ValidationCheckScenarioDiff, Scenario: name, Baseline: baseline,
				SemanticKey: "scenario-diff:" + name + ":" + baseline,
				Command:     fmt.Sprintf("git-control-tower baseline diff --scenario %s --name %s --wait --json", name, baseline), Oracle: true,
			})
		}
	}
	if len(paths) > 0 || repoLevel {
		cmd := "git diff --stat"
		if sha := strings.TrimSpace(p.RegressionAnchor.HeadSha); sha != "" && !planmodel.ContainsUnresolvedPlaceholder(sha) && strings.ToLower(sha) != "captured at execution start" {
			cmd += " " + sha
		}
		if len(paths) > 0 {
			cmd += " -- " + strings.Join(paths, " ")
		}
		checks = append(checks, ValidationCheck{Kind: ValidationCheckRepoDiff, SemanticKey: "repo-diff:" + strings.Join(paths, ","), Paths: paths, Command: cmd})
	}
	for _, command := range p.RegressionAnchor.Commands {
		// A collection-backed execution has one authoritative behavioral oracle:
		// its durable collection diff. Legacy per-scenario snapshot/diff commands
		// in a rendered regression anchor are retained for historical readability,
		// never reintroduced as hidden worker children or competing evidence.
		if strings.TrimSpace(p.BaselineSet.Name) != "" && legacyCollectionSupersededCommand(command) {
			continue
		}
		check := parseKnownValidationCheck(command)
		if check.SemanticKey == "" {
			check = ValidationCheck{Kind: ValidationCheckCustom, SemanticKey: "custom:" + strings.TrimSpace(command), Command: command, Oracle: isOracleCommand(command)}
		}
		checks = append(checks, check)
	}
	return deduplicateChecks(checks)
}

func legacyCollectionSupersededCommand(command string) bool {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) < 3 || fields[0] != "git-control-tower" || fields[1] != "baseline" {
		return false
	}
	return fields[2] == "diff" || (fields[2] == "snapshot" && len(fields) >= 4 && fields[3] == "status")
}

func intersectScenarioTargets(scope, inventory []string) []string {
	allowed := make(map[string]struct{}, len(inventory))
	for _, scenario := range inventory {
		if scenario = strings.TrimSpace(scenario); scenario != "" {
			allowed[scenario] = struct{}{}
		}
	}
	out := make([]string, 0, len(scope))
	for _, scenario := range scope {
		if _, ok := allowed[scenario]; ok {
			out = append(out, scenario)
		}
	}
	return uniqueSortedStrings(out)
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func parseKnownValidationCheck(command string) ValidationCheck {
	fields := strings.Fields(command)
	if len(fields) < 2 || fields[0] != "git-control-tower" || fields[1] != "baseline" {
		return ValidationCheck{}
	}
	if len(fields) >= 3 && fields[2] == "diff" {
		var scenario, baseline string
		for i := 3; i+1 < len(fields); i++ {
			if fields[i] == "--scenario" {
				scenario = fields[i+1]
			}
			if fields[i] == "--name" {
				baseline = fields[i+1]
			}
		}
		if scenario != "" && baseline != "" {
			return ValidationCheck{Kind: ValidationCheckScenarioDiff, Scenario: scenario, Baseline: baseline, SemanticKey: "scenario-diff:" + scenario + ":" + baseline, Command: fmt.Sprintf("git-control-tower baseline diff --scenario %s --name %s --wait --json", scenario, baseline), Oracle: true}
		}
	}
	return ValidationCheck{}
}

func deduplicateChecks(in []ValidationCheck) []ValidationCheck {
	seen := make(map[string]bool, len(in))
	out := make([]ValidationCheck, 0, len(in))
	for _, check := range in {
		if check.SemanticKey != "" && !seen[check.SemanticKey] {
			seen[check.SemanticKey] = true
			out = append(out, check)
		}
	}
	return out
}
