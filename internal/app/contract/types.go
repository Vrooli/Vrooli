package contractapp

import (
	repocontract "github.com/vrooli/repo-contract-go"
)

type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type Report struct {
	Root         string        `json:"root"`
	ContractPath string        `json:"contract_path"`
	Success      bool          `json:"success"`
	Checks       []CheckResult `json:"checks"`
}

type ValidationOutput struct {
	Success bool            `json:"success"`
	Root    string          `json:"root"`
	Schema  ValidationCheck `json:"schema"`
	Report  Report          `json:"report"`
}

type ValidationCheck struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type ShowOutput struct {
	Success      bool                            `json:"success"`
	Root         string                          `json:"root"`
	ContractPath string                          `json:"contract_path"`
	Schema       string                          `json:"schema"`
	Version      string                          `json:"version"`
	Platform     repocontract.Platform           `json:"platform"`
	Markers      repocontract.RootMarkers        `json:"markers"`
	Layout       repocontract.Layout             `json:"layout"`
	Scenario     repocontract.ScenarioSpec       `json:"scenario"`
	Resource     repocontract.ResourceSpec       `json:"resource"`
	Globs        repocontract.GlobSpec           `json:"globs"`
	Environment  map[string]string               `json:"environment"`
	Sandbox      ShowSandbox                     `json:"sandbox"`
	Profiles     map[string]repocontract.Profile `json:"profiles"`
}

type ShowSandbox struct {
	FullRepoScopes      []string `json:"full_repo_scopes"`
	ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
}

type ResolveScenarioOutput struct {
	Success  bool   `json:"success"`
	Root     string `json:"root"`
	Scenario string `json:"scenario"`
	File     string `json:"file"`
	Path     string `json:"path"`
}

type MatchGlobOutput struct {
	Success bool   `json:"success"`
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Matched bool   `json:"matched"`
}
