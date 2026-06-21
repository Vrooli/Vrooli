package budgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// configSchemaVersion is the on-disk perf-budgets config schema version.
const configSchemaVersion = "1.0.0"

// BudgetsConfigRelPath is the declarative budget config location, relative to a
// scenario root. Budgets live in the scenario's own .vrooli directory so they
// are versioned alongside the scenario and discoverable without a database.
const BudgetsConfigRelPath = ".vrooli/perf-budgets.json"

// configFile is the JSON shape persisted at .vrooli/perf-budgets.json. It holds
// the budgets for the OWNING scenario (keyed by scenario slug so a config can
// also be used to seed a fleet sweep), plus a schema version for forward
// migration.
type configFile struct {
	SchemaVersion string                  `json:"schema_version"`
	Budgets       map[string]budgetRecord `json:"budgets"`
}

// budgetRecord is one scenario's declared thresholds on disk. Zero means unset.
type budgetRecord struct {
	GoBuildMaxMs            int64   `json:"go_build_max_ms,omitempty"`
	UIBuildMaxMs            int64   `json:"ui_build_max_ms,omitempty"`
	BundleMaxBytes          int64   `json:"bundle_max_bytes,omitempty"`
	LCPMaxMs                int64   `json:"lcp_max_ms,omitempty"`
	StartupMaxMs            int64   `json:"startup_max_ms,omitempty"`
	ComponentCommitAvgMaxMs float64 `json:"component_commit_avg_max_ms,omitempty"`
	ComponentCommitMaxMs    float64 `json:"component_commit_max_ms,omitempty"`
	Ratchet                 bool    `json:"ratchet,omitempty"`
}

func (r budgetRecord) toBudget(scenario string) Budget {
	return Budget{
		Scenario:                scenario,
		GoBuildMaxMs:            r.GoBuildMaxMs,
		UIBuildMaxMs:            r.UIBuildMaxMs,
		BundleMaxBytes:          r.BundleMaxBytes,
		LCPMaxMs:                r.LCPMaxMs,
		StartupMaxMs:            r.StartupMaxMs,
		ComponentCommitAvgMaxMs: r.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    r.ComponentCommitMaxMs,
		Ratchet:                 r.Ratchet,
	}
}

func recordFromBudget(b Budget) budgetRecord {
	return budgetRecord{
		GoBuildMaxMs:            b.GoBuildMaxMs,
		UIBuildMaxMs:            b.UIBuildMaxMs,
		BundleMaxBytes:          b.BundleMaxBytes,
		LCPMaxMs:                b.LCPMaxMs,
		StartupMaxMs:            b.StartupMaxMs,
		ComponentCommitAvgMaxMs: b.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    b.ComponentCommitMaxMs,
		Ratchet:                 b.Ratchet,
	}
}

// ConfigStore reads and writes declarative budgets from per-scenario
// .vrooli/perf-budgets.json files under a repo root. It satisfies the BudgetStore
// seam so the service is storage-agnostic; the in-memory Store is used in tests.
//
// Writes are guarded by a mutex so concurrent SetBudget calls don't interleave a
// read-modify-write of the same file.
type ConfigStore struct {
	mu       sync.Mutex
	repoRoot string
	resolve  func(scenario string) (string, error)
}

// NewConfigStore binds a config store to a repo root. scenarioPath resolves a
// scenario slug to its on-disk root (so budgets land in the right scenario's
// .vrooli); when nil it defaults to <repoRoot>/scenarios/<scenario>.
var _ BudgetStore = (*ConfigStore)(nil)

func NewConfigStore(repoRoot string, scenarioPath func(scenario string) (string, error)) *ConfigStore {
	if scenarioPath == nil {
		scenarioPath = func(scenario string) (string, error) {
			if repoRoot == "" {
				return "", errors.New("budgets: repo root is empty")
			}
			return filepath.Join(repoRoot, "scenarios", scenario), nil
		}
	}
	return &ConfigStore{repoRoot: repoRoot, resolve: scenarioPath}
}

func (s *ConfigStore) configPath(scenario string) (string, error) {
	root, err := s.resolve(scenario)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, BudgetsConfigRelPath), nil
}

func (s *ConfigStore) load(scenario string) (configFile, error) {
	path, err := s.configPath(scenario)
	if err != nil {
		return configFile{}, err
	}
	return loadConfigFile(path)
}

// loadConfigFile reads a perf-budgets.json file, defaulting a missing file to an
// empty (current-schema) config rather than an error — a scenario with no budget
// is a valid, common state.
func loadConfigFile(path string) (configFile, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path derived from repo-contract scenario root
	if errors.Is(err, os.ErrNotExist) {
		return configFile{SchemaVersion: configSchemaVersion, Budgets: map[string]budgetRecord{}}, nil
	}
	if err != nil {
		return configFile{}, fmt.Errorf("budgets: read %s: %w", path, err)
	}
	var cfg configFile
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return configFile{}, fmt.Errorf("budgets: parse %s: %w", path, err)
	}
	if cfg.Budgets == nil {
		cfg.Budgets = map[string]budgetRecord{}
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = configSchemaVersion
	}
	return cfg, nil
}

func (s *ConfigStore) save(scenario string, cfg configFile) error {
	path, err := s.configPath(scenario)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("budgets: prepare %s: %w", filepath.Dir(path), err)
	}
	cfg.SchemaVersion = configSchemaVersion
	// Deterministic key order keeps the file diff-stable.
	ordered := configFile{SchemaVersion: cfg.SchemaVersion, Budgets: map[string]budgetRecord{}}
	keys := make([]string, 0, len(cfg.Budgets))
	for k := range cfg.Budgets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ordered.Budgets[k] = cfg.Budgets[k]
	}
	out, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return fmt.Errorf("budgets: marshal config: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil { //nolint:gosec // config is non-secret, world-readable by design
		return fmt.Errorf("budgets: write %s: %w", path, err)
	}
	return nil
}

// Get returns a scenario's declared budget and whether one was declared.
func (s *ConfigStore) Get(_ context.Context, scenario string) (Budget, bool, error) {
	if s == nil {
		return Budget{}, false, errors.New("budgets: nil config store")
	}
	if scenario == "" {
		return Budget{}, false, errors.New("budgets: scenario is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.load(scenario)
	if err != nil {
		return Budget{}, false, err
	}
	rec, ok := cfg.Budgets[scenario]
	if !ok {
		return Budget{Scenario: scenario}, false, nil
	}
	return rec.toBudget(scenario), true, nil
}

// Set writes/updates a scenario's budget. With dryRun it validates (including the
// ratchet rule) but does not persist. The ratchet is enforced against the
// CURRENTLY PERSISTED budget for the scenario.
func (s *ConfigStore) Set(_ context.Context, b Budget, dryRun bool) (Budget, error) {
	if s == nil {
		return Budget{}, errors.New("budgets: nil config store")
	}
	if b.Scenario == "" {
		return Budget{}, errors.New("budgets: scenario is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.load(b.Scenario)
	if err != nil {
		return Budget{}, err
	}
	existing, declared := cfg.Budgets[b.Scenario]
	if declared {
		if err := enforceRatchet(existing.toBudget(b.Scenario), b); err != nil {
			return Budget{}, err
		}
	}
	if dryRun {
		return b, nil
	}
	cfg.Budgets[b.Scenario] = recordFromBudget(b)
	if err := s.save(b.Scenario, cfg); err != nil {
		return Budget{}, err
	}
	return b, nil
}
