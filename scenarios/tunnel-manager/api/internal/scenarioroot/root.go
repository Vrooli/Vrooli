package scenarioroot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Resolver is the single Tunnel Manager seam for contract-owned scenario
// paths. It keeps repository resolution errors instead of silently falling
// back to a cwd-relative directory; callers receive the error at the
// operation that needs the path.
type Resolver struct {
	repoRoot      string
	contract      *repocontract.Contract
	scenariosRoot string
	initErr       error
	// override is true when VROOLI_SCENARIOS_ROOT supplies a validated,
	// standalone scenarios directory without a repository contract.
	override bool
}

// New resolves the active scenario layout once. VROOLI_SCENARIOS_ROOT is a
// validated deployment override; ordinary lifecycle-managed processes use the
// repository contract and its canonical environment resolution.
func New() *Resolver {
	if override := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_ROOT")); override != "" {
		root, err := filepath.Abs(override)
		if err != nil {
			return &Resolver{initErr: fmt.Errorf("resolve VROOLI_SCENARIOS_ROOT %q: %w", override, err)}
		}
		info, err := os.Stat(root)
		if err != nil {
			return &Resolver{initErr: fmt.Errorf("stat VROOLI_SCENARIOS_ROOT %q: %w", root, err)}
		}
		if !info.IsDir() {
			return &Resolver{initErr: fmt.Errorf("VROOLI_SCENARIOS_ROOT %q is not a directory", root)}
		}
		return &Resolver{scenariosRoot: root, override: true}
	}

	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return &Resolver{initErr: fmt.Errorf("resolve repository contract: %w", err)}
	}
	scenariosRoot, err := contract.TopLevelDir(repoRoot, "scenarios")
	if err != nil {
		return &Resolver{initErr: fmt.Errorf("resolve scenarios directory: %w", err)}
	}
	return &Resolver{repoRoot: repoRoot, contract: contract, scenariosRoot: scenariosRoot}
}

// ScenariosRoot returns the validated scenarios directory.
func (r *Resolver) ScenariosRoot() (string, error) {
	if r == nil {
		return "", fmt.Errorf("scenario path resolver is nil")
	}
	if r.initErr != nil {
		return "", r.initErr
	}
	return r.scenariosRoot, nil
}

// ServiceFile resolves a scenario's service manifest through the repository
// contract. An explicit scenarios-root override uses the documented default
// service manifest path beneath that root.
func (r *Resolver) ServiceFile(scenario string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("scenario path resolver is nil")
	}
	if r.initErr != nil {
		return "", r.initErr
	}
	if r.override {
		name := strings.TrimSpace(scenario)
		clean := filepath.Clean(name)
		if name == "" || clean == "." || filepath.IsAbs(name) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != name {
			return "", fmt.Errorf("invalid scenario name %q", scenario)
		}
		return filepath.Join(r.scenariosRoot, clean, ".vrooli", "service.json"), nil
	}
	return r.contract.ScenarioFile(r.repoRoot, scenario, "service")
}
