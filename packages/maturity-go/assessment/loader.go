package assessment

import (
	"fmt"
	"os"
	"path/filepath"
)

// MaturitySpecRelPath is the canonical location of a provider's maturity spec,
// relative to the scenario directory.
const MaturitySpecRelPath = ".vrooli/maturity.json"

// LoadSpecFromScenario reads and validates the canonical
// `<scenarioDir>/.vrooli/maturity.json` spec. It is the single shared loader for
// provider scenarios; each provider previously hand-rolled an identical
// loadMaturitySpec. scenarioDir is the scenario's root directory (the parent of
// `.vrooli`), e.g. filepath.Join(repoRoot, "scenarios", "proto-health").
func LoadSpecFromScenario(scenarioDir string) (*Spec, error) {
	path := filepath.Join(scenarioDir, filepath.FromSlash(MaturitySpecRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	spec, err := ParseSpec(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return spec, nil
}
