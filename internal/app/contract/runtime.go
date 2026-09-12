package contractapp

import (
	"slices"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func NewDefaultService() Service {
	return Service{
		ResolveRootFn:     ResolveRoot,
		ShowFn:            LoadShowOutput,
		ResolveScenarioFn: ResolveScenario,
		MatchGlobFn:       MatchGlob,
	}
}

func ResolveRoot() (string, error) {
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func LoadShowOutput() (ShowOutput, error) {
	contract, root, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return ShowOutput{}, err
	}
	return ShowOutput{
		Success:      true,
		Root:         root,
		ContractPath: repocontractmeta.ContractPath(root),
		Schema:       contract.Schema(),
		Version:      contract.Version(),
		Platform:     contract.Platform(),
		Markers:      contract.RootMarkers(),
		Layout:       contract.Layout(),
		Scenario:     contract.Scenario(),
		Resource:     contract.Resource(),
		Globs:        contract.Globs(),
		Environment:  contract.EnvironmentVariables(),
		Sandbox: ShowSandbox{
			FullRepoScopes:      contract.SandboxFullRepoScopes(),
			ScenarioScopePrefix: contract.SandboxScenarioScopePrefix(),
		},
		Profiles: LoadProfiles(contract),
	}, nil
}

func ResolveScenario(root, scenarioName, fileKey string) (ResolveScenarioOutput, error) {
	var (
		resolved string
		err      error
	)
	if fileKey == "" {
		resolved, err = repocontract.ResolveScenarioPath(root, scenarioName)
	} else {
		resolved, err = repocontract.ResolveScenarioFile(root, scenarioName, fileKey)
	}
	if err != nil {
		return ResolveScenarioOutput{}, err
	}
	return ResolveScenarioOutput{
		Success:  true,
		Root:     root,
		Scenario: scenarioName,
		File:     fileKey,
		Path:     resolved,
	}, nil
}

func MatchGlob(pattern, path string) (MatchGlobOutput, error) {
	matched, err := repocontract.MatchRepoGlob(pattern, path)
	if err != nil {
		return MatchGlobOutput{}, err
	}
	return MatchGlobOutput{
		Success: true,
		Pattern: pattern,
		Path:    path,
		Matched: matched,
	}, nil
}

func LoadProfiles(contract *repocontract.Contract) map[string]repocontract.Profile {
	names := []string{repocontractmeta.MiniBundleProfile}
	profiles := make(map[string]repocontract.Profile, len(names))
	for _, name := range names {
		profile, err := contract.Profile(name)
		if err == nil {
			profiles[name] = profile
		}
	}
	return profiles
}

func SortedProfileNames(profiles map[string]repocontract.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
