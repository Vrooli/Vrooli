package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RunAll runs all rules matching the scenario's tech stack and returns results.
func RunAll(scenarioRoot, scenarioName string) []RuleResult {
	stack := enrichTechStack(scenarioRoot)
	rules := ForTechStack(stack)

	ctx := CheckContext{
		ScenarioRoot: scenarioRoot,
		TechStack:    stack,
		ScenarioName: scenarioName,
	}

	results := make([]RuleResult, 0, len(rules))
	for _, r := range rules {
		result := r.Check(ctx)
		result.RuleID = r.Def.ID
		results = append(results, result)
	}
	return results
}

// enrichTechStack reads ui/package.json and derives synthetic tech stack signals.
func enrichTechStack(scenarioRoot string) []string {
	stack := []string{}

	pkgPath := filepath.Join(scenarioRoot, "ui", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return stack
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return stack
	}

	allDeps := map[string]bool{}
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDependencies {
		allDeps[k] = true
	}

	if allDeps["@vrooli/iframe-bridge"] {
		stack = append(stack, "iframe-bridge")
	}
	if allDeps["@vrooli/api-base"] {
		stack = append(stack, "api-base")
	}
	if allDeps["react"] || allDeps["react-dom"] {
		stack = append(stack, "React")
	}
	if allDeps["vite"] {
		stack = append(stack, "Vite")
	}

	return stack
}
