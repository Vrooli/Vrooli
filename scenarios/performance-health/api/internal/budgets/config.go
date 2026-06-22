package budgets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TestingConfigRelPath is the single source of truth for a scenario's
// performance budgets, relative to its root. Budgets live under the
// `performance.budgets` block of the scenario's own `.vrooli/testing.json` —
// the same file every other test-genie phase is configured from — so there is
// exactly one config surface and no second budget file to drift out of sync.
const TestingConfigRelPath = ".vrooli/testing.json"

// budgetRecord is one scenario's declared thresholds as they sit on disk under
// `performance.budgets`. Zero means unset. Units are milliseconds and bytes
// throughout (the historical `_seconds` axes are gone).
type budgetRecord struct {
	GoBuildMaxMs            int64                       `json:"go_build_max_ms,omitempty"`
	UIBuildMaxMs            int64                       `json:"ui_build_max_ms,omitempty"`
	BundleMaxBytes          int64                       `json:"bundle_max_bytes,omitempty"`
	LCPMaxMs                int64                       `json:"lcp_max_ms,omitempty"`
	StartupMaxMs            int64                       `json:"startup_max_ms,omitempty"`
	ComponentCommitAvgMaxMs float64                     `json:"component_commit_avg_max_ms,omitempty"`
	ComponentCommitMaxMs    float64                     `json:"component_commit_max_ms,omitempty"`
	Ratchet                 bool                        `json:"ratchet,omitempty"`
	Flows                   map[string]flowBudgetRecord `json:"flows,omitempty"`
}

// flowBudgetRecord is one interaction flow's thresholds as they sit on disk
// under `performance.budgets.flows.<slug>`. Only the per-flow axes (LCP +
// component-commit avg/max) are persisted; build/bundle/startup stay scenario-
// level.
type flowBudgetRecord struct {
	LCPMaxMs                int64   `json:"lcp_max_ms,omitempty"`
	ComponentCommitAvgMaxMs float64 `json:"component_commit_avg_max_ms,omitempty"`
	ComponentCommitMaxMs    float64 `json:"component_commit_max_ms,omitempty"`
}

func (r budgetRecord) toBudget(scenario string) Budget {
	b := Budget{
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
	if len(r.Flows) > 0 {
		b.Flows = make(map[string]FlowBudget, len(r.Flows))
		for slug, fr := range r.Flows {
			b.Flows[slug] = FlowBudget(fr)
		}
	}
	return b
}

func recordFromBudget(b Budget) budgetRecord {
	rec := budgetRecord{
		GoBuildMaxMs:            b.GoBuildMaxMs,
		UIBuildMaxMs:            b.UIBuildMaxMs,
		BundleMaxBytes:          b.BundleMaxBytes,
		LCPMaxMs:                b.LCPMaxMs,
		StartupMaxMs:            b.StartupMaxMs,
		ComponentCommitAvgMaxMs: b.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    b.ComponentCommitMaxMs,
		Ratchet:                 b.Ratchet,
	}
	if len(b.Flows) > 0 {
		rec.Flows = make(map[string]flowBudgetRecord, len(b.Flows))
		for slug, fb := range b.Flows {
			rec.Flows[slug] = flowBudgetRecord(fb)
		}
	}
	return rec
}

// ConfigStore reads and writes declarative budgets from the `performance.budgets`
// block of each scenario's `.vrooli/testing.json`. It satisfies the BudgetStore
// seam so the service is storage-agnostic; the in-memory Store is used in tests.
//
// Writes are a structured read-modify-write that preserves every sibling key
// (and their order) in testing.json — only the `performance.budgets` block is
// touched. A mutex guards concurrent SetBudget calls against an interleaved
// read-modify-write of the same file.
type ConfigStore struct {
	mu       sync.Mutex
	repoRoot string
	resolve  func(scenario string) (string, error)
}

var _ BudgetStore = (*ConfigStore)(nil)

// NewConfigStore binds a config store to a repo root. scenarioPath resolves a
// scenario slug to its on-disk root (so budgets are read from the right
// scenario's testing.json); when nil it defaults to <repoRoot>/scenarios/<scenario>.
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
	return filepath.Join(root, TestingConfigRelPath), nil
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
	path, err := s.configPath(scenario)
	if err != nil {
		return Budget{}, false, err
	}
	rec, declared, err := loadBudgetRecord(path)
	if err != nil {
		return Budget{}, false, err
	}
	if !declared {
		return Budget{Scenario: scenario}, false, nil
	}
	return rec.toBudget(scenario), true, nil
}

// Set writes/updates a scenario's budget under `performance.budgets`, preserving
// every other testing.json key. With dryRun it validates (including the ratchet
// rule against the currently persisted budget) but does not persist.
func (s *ConfigStore) Set(_ context.Context, b Budget, dryRun bool) (Budget, error) {
	if s == nil {
		return Budget{}, errors.New("budgets: nil config store")
	}
	if b.Scenario == "" {
		return Budget{}, errors.New("budgets: scenario is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.configPath(b.Scenario)
	if err != nil {
		return Budget{}, err
	}
	existing, declared, err := loadBudgetRecord(path)
	if err != nil {
		return Budget{}, err
	}
	if declared {
		if err := enforceRatchet(existing.toBudget(b.Scenario), b); err != nil {
			return Budget{}, err
		}
	}
	if dryRun {
		return b, nil
	}
	if err := writeBudgetRecord(path, recordFromBudget(b)); err != nil {
		return Budget{}, err
	}
	return b, nil
}

// loadBudgetRecord reads `performance.budgets` from testing.json at path. A
// missing file or absent block is a valid, common state (no budget declared),
// reported as declared=false rather than an error.
func loadBudgetRecord(path string) (budgetRecord, bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path derived from repo-contract scenario root
	if errors.Is(err, os.ErrNotExist) {
		return budgetRecord{}, false, nil
	}
	if err != nil {
		return budgetRecord{}, false, fmt.Errorf("budgets: read %s: %w", path, err)
	}
	var top orderedObject
	if err := json.Unmarshal(raw, &top); err != nil {
		return budgetRecord{}, false, fmt.Errorf("budgets: parse %s: %w", path, err)
	}
	perfRaw, ok := top.get("performance")
	if !ok {
		return budgetRecord{}, false, nil
	}
	var perf orderedObject
	if err := json.Unmarshal(perfRaw, &perf); err != nil {
		return budgetRecord{}, false, fmt.Errorf("budgets: parse %s performance block: %w", path, err)
	}
	budgetsRaw, ok := perf.get("budgets")
	if !ok {
		return budgetRecord{}, false, nil
	}
	var rec budgetRecord
	if err := json.Unmarshal(budgetsRaw, &rec); err != nil {
		return budgetRecord{}, false, fmt.Errorf("budgets: parse %s performance.budgets: %w", path, err)
	}
	return rec, true, nil
}

// writeBudgetRecord performs the structured read-modify-write: it loads the full
// testing.json, replaces only `performance.budgets`, and writes it back with all
// sibling keys and their order preserved. A missing file is created with just
// the performance.budgets block.
func writeBudgetRecord(path string, rec budgetRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("budgets: prepare %s: %w", filepath.Dir(path), err)
	}
	var top orderedObject
	if raw, err := os.ReadFile(path); err == nil { //nolint:gosec // path derived from repo-contract scenario root
		if err := json.Unmarshal(raw, &top); err != nil {
			return fmt.Errorf("budgets: parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("budgets: read %s: %w", path, err)
	}

	var perf orderedObject
	if perfRaw, ok := top.get("performance"); ok {
		if err := json.Unmarshal(perfRaw, &perf); err != nil {
			return fmt.Errorf("budgets: parse %s performance block: %w", path, err)
		}
	}
	budgetsJSON, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("budgets: marshal budgets: %w", err)
	}
	perf.set("budgets", budgetsJSON)
	perfJSON, err := json.Marshal(perf)
	if err != nil {
		return fmt.Errorf("budgets: marshal performance block: %w", err)
	}
	top.set("performance", perfJSON)

	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return fmt.Errorf("budgets: marshal testing.json: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil { //nolint:gosec // config is non-secret, world-readable by design
		return fmt.Errorf("budgets: write %s: %w", path, err)
	}
	return nil
}

// orderedObject is a JSON object that preserves key insertion order across a
// read-modify-write, so editing one nested key never reshuffles a human-authored
// config file. Only the keys actually changed produce a diff.
type orderedObject struct {
	keys   []string
	values map[string]json.RawMessage
}

func (o *orderedObject) get(key string) (json.RawMessage, bool) {
	v, ok := o.values[key]
	return v, ok
}

func (o *orderedObject) set(key string, value json.RawMessage) {
	if o.values == nil {
		o.values = map[string]json.RawMessage{}
	}
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *orderedObject) UnmarshalJSON(b []byte) error {
	o.keys = o.keys[:0]
	o.values = map[string]json.RawMessage{}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("budgets: expected JSON object, got %v", tok)
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("budgets: expected object key, got %v", keyTok)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		if _, exists := o.values[key]; !exists {
			o.keys = append(o.keys, key)
		}
		o.values[key] = raw
	}
	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return err
	}
	return nil
}

func (o orderedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(o.values[key])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
