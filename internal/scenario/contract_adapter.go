package scenario

import (
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

type scenarioContractPaths interface {
	ScenarioBaseDir(root string) string
	ScenarioRootPath(root, name string) string
	ScenarioServicePath(root, name, scenarioPath string) string
	ScenarioDirName(root string) string
	ScenarioScopePrefix(root string) string
	IsFullRepoScope(root, scope string) bool
}

var contractPaths scenarioContractPaths = repoContractPaths{}

type repoContractPaths struct{}

func (repoContractPaths) ScenarioBaseDir(root string) string {
	fallback := filepath.Join(root, repocontractmeta.ScenarioDir)

	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return fallback
	}
	path, err := contract.TopLevelDir(root, repocontractmeta.ScenarioDir)
	if err != nil {
		return fallback
	}
	return path
}

func (repoContractPaths) ScenarioRootPath(root, name string) string {
	if path, err := repocontract.ResolveScenarioPath(root, name); err == nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(root, repocontractmeta.ScenarioDir, name))
}

func (repoContractPaths) ScenarioServicePath(root, name, scenarioPath string) string {
	if path, err := repocontract.ResolveScenarioFile(root, name, "service"); err == nil {
		return filepath.Clean(path)
	}
	return filepath.Join(scenarioPath, filepath.FromSlash(defaultScenarioServiceRelPath))
}

func (repoContractPaths) ScenarioDirName(root string) string {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return repocontractmeta.ScenarioDir
	}
	return filepath.ToSlash(contract.Layout().ScenarioDir)
}

func (repoContractPaths) ScenarioScopePrefix(root string) string {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return repocontractmeta.ScenarioDir
	}
	prefix := filepath.ToSlash(contract.SandboxScenarioScopePrefix())
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return filepath.ToSlash(contract.Layout().ScenarioDir)
	}
	return prefix
}

func (repoContractPaths) IsFullRepoScope(root, scope string) bool {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		scope = strings.TrimSpace(strings.TrimSuffix(filepath.ToSlash(scope), "/"))
		return scope == "" || scope == "." || scope == "/"
	}
	return contract.IsFullRepoScope(scope)
}
