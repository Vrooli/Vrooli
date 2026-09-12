package repocontext

import (
	"fmt"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Context is the sole repo-contract-backed authority for repo-aware behavior in
// scenario-auditor. It is immutable after construction.
type Context struct {
	repoRoot            string
	contract            *repocontract.Contract
	scenariosRoot       string
	scenarioAuditorRoot string
}

func FromEnvOrCWD() (*Context, error) {
	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("repocontext: load from env or cwd: %w", err)
	}
	return fromLoaded(repoRoot, contract)
}

func FromRepoRoot(root string) (*Context, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("repocontext: repo root is required")
	}

	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return nil, fmt.Errorf("repocontext: load contract: %w", err)
	}
	return fromLoaded(root, contract)
}

func fromLoaded(root string, contract *repocontract.Contract) (*Context, error) {
	if contract == nil {
		return nil, fmt.Errorf("repocontext: contract is required")
	}

	cleanRoot := filepath.Clean(root)
	layout := contract.Layout()
	scenariosRoot := filepath.Join(cleanRoot, filepath.FromSlash(layout.ScenarioDir))
	scenarioAuditorRoot, err := contract.ScenarioRoot(cleanRoot, "scenario-auditor")
	if err != nil {
		return nil, fmt.Errorf("repocontext: resolve scenario-auditor root: %w", err)
	}

	return &Context{
		repoRoot:            cleanRoot,
		contract:            contract,
		scenariosRoot:       scenariosRoot,
		scenarioAuditorRoot: scenarioAuditorRoot,
	}, nil
}

func (c *Context) RepoRoot() string {
	return c.repoRoot
}

func (c *Context) Contract() *repocontract.Contract {
	return c.contract
}

func (c *Context) ScenarioAuditorRoot() string {
	return c.scenarioAuditorRoot
}

func (c *Context) ScenariosRoot() string {
	return c.scenariosRoot
}

func (c *Context) ResolveScenarioPath(name string) (string, error) {
	path, err := c.contract.ScenarioRoot(c.repoRoot, name)
	if err != nil {
		return "", fmt.Errorf("repocontext: resolve scenario path: %w", err)
	}
	return path, nil
}

func (c *Context) RelativeToRepoRoot(path string) string {
	rel, err := filepath.Rel(c.repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
