// Package fleet answers deterministic, structured storage-inventory queries
// across the whole fleet: which engines each scenario uses, whether its
// test-isolation seams are statically proven (the safety throughline), whether
// it adopted the variant-aware namespace helpers, its deploy stage, and whether
// a data-persisting scenario declares a backup target. These are exact,
// structured queries (not semantic), so storage-health is NOT a search-hub data
// provider; the CLI verbs are discoverable through cli-health's command index.
//
// Per-scenario classification sits behind the Classifier seam (the production
// classifier composes the validation engine + backup detection and lives in
// handlers/fleet), so this package stays testable with a fake classifier. The
// Store seam persists the latest snapshot to storage-health's own SQLite store
// — dogfooding the per-domain-schema conventions it enforces.
package fleet

import (
	"context"
	"errors"
	"sort"
	"time"
)

// ScenarioEntry is one scenario's storage rollup.
type ScenarioEntry struct {
	Scenario         string
	Engines          []string
	PrimaryEngine    string
	Language         string
	StorageStage     string
	IsolationReady   bool
	IsolationReason  string
	NamespaceAdopted bool
	HasBackupTarget  bool
	FindingCount     int
	ErrorCount       int
	AutofixableCount int
}

// EngineCount counts scenarios using a given engine.
type EngineCount struct {
	Engine        string
	ScenarioCount int
}

// StageCount counts scenarios at a given deploy stage.
type StageCount struct {
	Stage         string
	ScenarioCount int
}

// ScanError records a scenario enumerated but not classifiable.
type ScanError struct {
	Scenario string
	Reason   string
}

// Result is a fleet scan rollup.
type Result struct {
	Entries               []ScenarioEntry
	EngineDistribution    []EngineCount
	StageDistribution     []StageCount
	ScenarioCount         int
	IsolationUnreadyCount int
	NoBackupCount         int
	FindingCount          int
	Errors                []ScanError
	ScannedAt             time.Time
}

// Classifier produces one ScenarioEntry per scenario. The real classifier
// composes the validation engine + backup detection; tests drive a fake.
type Classifier interface {
	Classify(ctx context.Context, scenario string) (ScenarioEntry, error)
}

// Enumerator lists the scenarios to scan when none are requested.
type Enumerator interface {
	List(ctx context.Context) ([]string, error)
}

// Store persists and reads the latest fleet snapshot. A nil store disables
// persistence (Scan still computes + returns; Inventory reports empty).
type Store interface {
	Save(ctx context.Context, res Result) error
	Load(ctx context.Context) (Result, error)
}

// Clock is the minimal wall-clock seam used to stamp a snapshot.
type Clock interface {
	Now() time.Time
}

// Service is the fleet engine.
type Service struct {
	classifier Classifier
	enumerator Enumerator
	store      Store
	clock      Clock
}

// NewService wires a fleet Service. store and clock may be nil (no persistence
// / zero-value timestamps).
func NewService(classifier Classifier, enumerator Enumerator, store Store, clk Clock) *Service {
	return &Service{classifier: classifier, enumerator: enumerator, store: store, clock: clk}
}

// Scan classifies the requested scenarios (or every enumerated scenario), rolls
// the entries up into distributions + offender counts, persists the snapshot
// (best-effort), and returns it.
func (s *Service) Scan(ctx context.Context, scenarios []string) (Result, error) {
	if s == nil || s.classifier == nil {
		return Result{}, errors.New("fleet: service not wired")
	}
	targets := scenarios
	if len(targets) == 0 {
		if s.enumerator == nil {
			return Result{}, errors.New("fleet: no scenarios requested and no enumerator wired")
		}
		listed, err := s.enumerator.List(ctx)
		if err != nil {
			return Result{}, err
		}
		targets = listed
	}
	sort.Strings(targets)

	res := Result{}
	engineCounts := map[string]int{}
	stageCounts := map[string]int{}
	for _, scenario := range targets {
		entry, err := s.classifier.Classify(ctx, scenario)
		if err != nil {
			res.Errors = append(res.Errors, ScanError{Scenario: scenario, Reason: err.Error()})
			continue
		}
		res.Entries = append(res.Entries, entry)
		res.ScenarioCount++
		res.FindingCount += entry.FindingCount
		if !entry.IsolationReady {
			res.IsolationUnreadyCount++
		}
		if !entry.HasBackupTarget && entry.dataPersisting() {
			res.NoBackupCount++
		}
		seenEngine := map[string]struct{}{}
		for _, e := range entry.Engines {
			if _, dup := seenEngine[e]; dup {
				continue
			}
			seenEngine[e] = struct{}{}
			engineCounts[e]++
		}
		stage := entry.StorageStage
		if stage == "" {
			stage = "greenfield"
		}
		stageCounts[stage]++
	}
	res.EngineDistribution = engineDistribution(engineCounts)
	res.StageDistribution = stageDistribution(stageCounts)
	if s.clock != nil {
		res.ScannedAt = s.clock.Now()
	}

	if s.store != nil {
		if err := s.store.Save(ctx, res); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// Inventory returns the latest persisted snapshot. With no store wired it
// returns an empty (never-scanned) result.
func (s *Service) Inventory(ctx context.Context) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, nil
	}
	return s.store.Load(ctx)
}

// dataPersisting reports whether the entry's engines include a durable store.
// Mirrors validation.IsDataPersisting on the string engine names.
func (e ScenarioEntry) dataPersisting() bool {
	for _, eng := range e.Engines {
		switch eng {
		case "sqlite", "postgres", "qdrant", "file":
			return true
		}
	}
	return false
}

// IsolationUnready returns the scenarios whose isolation seams are not proven,
// sorted by scenario name. The safety offender query.
func (r Result) IsolationUnready() []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		if !e.IsolationReady {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

// ByEngine returns the scenarios using the given engine, sorted by name.
func (r Result) ByEngine(engine string) []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		for _, got := range e.Engines {
			if got == engine {
				out = append(out, e)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

// NoBackup returns the data-persisting scenarios with no declared backup
// target, sorted by name.
func (r Result) NoBackup() []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		if !e.HasBackupTarget && e.dataPersisting() {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

func engineDistribution(counts map[string]int) []EngineCount {
	out := make([]EngineCount, 0, len(counts))
	for engine, n := range counts {
		out = append(out, EngineCount{Engine: engine, ScenarioCount: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScenarioCount != out[j].ScenarioCount {
			return out[i].ScenarioCount > out[j].ScenarioCount
		}
		return out[i].Engine < out[j].Engine
	})
	return out
}

func stageDistribution(counts map[string]int) []StageCount {
	out := make([]StageCount, 0, len(counts))
	for stage, n := range counts {
		out = append(out, StageCount{Stage: stage, ScenarioCount: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScenarioCount != out[j].ScenarioCount {
			return out[i].ScenarioCount > out[j].ScenarioCount
		}
		return out[i].Stage < out[j].Stage
	})
	return out
}
