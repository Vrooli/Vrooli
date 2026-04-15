// Package path provides utilities for detecting and working with Vrooli paths.
package path

import (
	repocontract "github.com/vrooli/repo-contract-go"
)

// DetectVrooliRoot finds the canonical Vrooli repo root.
func DetectVrooliRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

// DetectScenariosRoot returns the contract-defined scenarios root.
func DetectScenariosRoot() string {
	root := DetectVrooliRoot()
	if root == "" {
		return ""
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return ""
	}
	scenariosRoot, err := contract.TopLevelDir(root, "scenarios")
	if err != nil {
		return ""
	}
	return scenariosRoot
}

// ResolveScenarioRoot returns the canonical root for a scenario.
func ResolveScenarioRoot(scenario string) string {
	root := DetectVrooliRoot()
	if root == "" {
		return ""
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(root, scenario)
	if err != nil {
		return ""
	}
	return scenarioRoot
}
