package assessment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaturitySpecRelPath is the retired pre-cutover maturity spec location. The
// loader only uses it as a rejection guard so stale specs cannot silently
// override descriptor-owned maturity metadata.
const MaturitySpecRelPath = ".vrooli/maturity.json"

// TestGenieDescriptorRelPath is the canonical provider-owned Test Genie phase
// descriptor location relative to the scenario directory.
const TestGenieDescriptorRelPath = ".vrooli/test-genie.json"

type testGenieDescriptor struct {
	Scenario string          `json:"scenario"`
	Phase    string          `json:"phase"`
	Maturity json.RawMessage `json:"maturity"`
}

// LoadSpecFromScenario reads and validates the maturity block embedded in the
// canonical `<scenarioDir>/.vrooli/test-genie.json` descriptor. scenarioDir is
// the scenario's root directory (the parent of `.vrooli`), e.g.
// filepath.Join(repoRoot, "scenarios", "proto-health").
func LoadSpecFromScenario(scenarioDir string) (*Spec, error) {
	cleanScenarioDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario dir %s: %w", scenarioDir, err)
	}
	legacyPath := filepath.Join(cleanScenarioDir, filepath.FromSlash(MaturitySpecRelPath))
	if _, err := os.Stat(legacyPath); err == nil {
		return nil, fmt.Errorf("retired maturity spec still exists at %s; move the maturity block into %s", legacyPath, TestGenieDescriptorRelPath)
	}
	path := filepath.Join(cleanScenarioDir, filepath.FromSlash(TestGenieDescriptorRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var descriptor testGenieDescriptor
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	scenario := filepath.Base(cleanScenarioDir)
	if descriptor.Scenario != scenario {
		return nil, fmt.Errorf("descriptor scenario %q must match directory %q", descriptor.Scenario, scenario)
	}
	spec, err := ParseEmbeddedSpec(descriptor.Maturity, descriptor.Scenario, descriptor.Phase)
	if err != nil {
		return nil, fmt.Errorf("parse maturity block in %s: %w", path, err)
	}
	return spec, nil
}

// ParseEmbeddedSpec parses a Test Genie descriptor's embedded maturity block.
// The descriptor owns provider and phase identity, so this helper stamps those
// values into the maturity spec while rejecting stale duplicated identity when
// present and inconsistent.
func ParseEmbeddedSpec(raw []byte, provider, phase string) (*Spec, error) {
	provider = strings.TrimSpace(provider)
	phase = strings.TrimSpace(phase)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if phase == "" {
		return nil, fmt.Errorf("phase is required")
	}
	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}
	if spec.Provider != "" && spec.Provider != provider {
		return nil, fmt.Errorf("provider %q does not match descriptor scenario %q", spec.Provider, provider)
	}
	if spec.Phase != "" && spec.Phase != phase {
		return nil, fmt.Errorf("phase %q does not match descriptor phase %q", spec.Phase, phase)
	}
	spec.Provider = provider
	spec.Phase = phase
	if err := ValidateSpec(spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
