// Package advisor turns per-scenario storage facts into actionable guidance:
// migration-hygiene grading against deploy stage, and Postgres→SQLite
// engine-fitness ranking. Both are deterministic, structured queries — no AI,
// no semantic ranking.
//
// Per-scenario facts come from the Reader seam (the production reader composes
// the validation engine and lives in handlers/advisor), so this package stays
// testable with a fake reader.
package advisor

import (
	"context"
	"errors"
	"sort"
)

// ScenarioFacts is the normalized storage fact set the advisor reasons over.
type ScenarioFacts struct {
	Scenario            string
	Engines             []string
	StorageStage        string
	HasMigrations       bool
	HasAlterInSchema    bool
	NonIdempotentSchema bool
	// DebtNotes are stage-aware, human-readable migration-debt notes derived
	// from the validation findings (MIGRATION_DEBT + the schema-hygiene codes).
	DebtNotes []string
}

func (f ScenarioFacts) usesEngine(engine string) bool {
	for _, e := range f.Engines {
		if e == engine {
			return true
		}
	}
	return false
}

// MigrationHygiene is one scenario's migration posture.
type MigrationHygiene struct {
	Scenario            string
	StorageStage        string
	HasMigrations       bool
	HasAlterInSchema    bool
	NonIdempotentSchema bool
	MigrationDebt       int
	Notes               []string
}

// MigrationResult is the AnalyzeMigrations rollup.
type MigrationResult struct {
	Entries             []MigrationHygiene
	ScenarioCount       int
	WithMigrationsCount int
	DebtCount           int
	Errors              []ScanError
}

// EngineCandidate is one scenario's engine-fitness recommendation.
type EngineCandidate struct {
	Scenario          string
	CurrentEngine     string
	RecommendedEngine string
	FitnessScore      float64
	Rationale         string
	Autofixable       bool
	Blockers          []string
}

// AdviseResult is the AdviseEngines rollup.
type AdviseResult struct {
	Candidates    []EngineCandidate
	ScenarioCount int
	Errors        []ScanError
}

// ScanError records a scenario enumerated but not analyzable.
type ScanError struct {
	Scenario string
	Reason   string
}

// Reader resolves a scenario's normalized storage facts.
type Reader interface {
	Read(ctx context.Context, scenario string) (ScenarioFacts, error)
}

// Enumerator lists the scenarios to analyze when none are requested.
type Enumerator interface {
	List(ctx context.Context) ([]string, error)
}

// Service is the advisor engine.
type Service struct {
	reader     Reader
	enumerator Enumerator
}

// NewService wires an advisor Service over the reader + enumerator seams.
func NewService(reader Reader, enumerator Enumerator) *Service {
	return &Service{reader: reader, enumerator: enumerator}
}

func (s *Service) targets(ctx context.Context, requested []string) ([]string, error) {
	if len(requested) > 0 {
		out := append([]string(nil), requested...)
		sort.Strings(out)
		return out, nil
	}
	if s.enumerator == nil {
		return nil, errors.New("advisor: no scenarios requested and no enumerator wired")
	}
	listed, err := s.enumerator.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(listed)
	return listed, nil
}

// AnalyzeMigrations grades migration hygiene for the requested (or all)
// scenarios against their deploy stage.
func (s *Service) AnalyzeMigrations(ctx context.Context, scenarios []string) (MigrationResult, error) {
	if s == nil || s.reader == nil {
		return MigrationResult{}, errors.New("advisor: service not wired")
	}
	targets, err := s.targets(ctx, scenarios)
	if err != nil {
		return MigrationResult{}, err
	}
	res := MigrationResult{}
	for _, scenario := range targets {
		facts, err := s.reader.Read(ctx, scenario)
		if err != nil {
			res.Errors = append(res.Errors, ScanError{Scenario: scenario, Reason: err.Error()})
			continue
		}
		h := MigrationHygiene{
			Scenario:            scenario,
			StorageStage:        facts.StorageStage,
			HasMigrations:       facts.HasMigrations,
			HasAlterInSchema:    facts.HasAlterInSchema,
			NonIdempotentSchema: facts.NonIdempotentSchema,
			Notes:               facts.DebtNotes,
			MigrationDebt:       len(facts.DebtNotes),
		}
		res.Entries = append(res.Entries, h)
		res.ScenarioCount++
		if h.HasMigrations {
			res.WithMigrationsCount++
		}
		if h.MigrationDebt > 0 {
			res.DebtCount++
		}
	}
	return res, nil
}

// AdviseEngines ranks Postgres→SQLite migration candidates by fitness for the
// requested (or all) scenarios. Only scenarios with an actionable
// recommendation appear, strongest candidate first.
func (s *Service) AdviseEngines(ctx context.Context, scenarios []string) (AdviseResult, error) {
	if s == nil || s.reader == nil {
		return AdviseResult{}, errors.New("advisor: service not wired")
	}
	targets, err := s.targets(ctx, scenarios)
	if err != nil {
		return AdviseResult{}, err
	}
	res := AdviseResult{}
	for _, scenario := range targets {
		facts, err := s.reader.Read(ctx, scenario)
		if err != nil {
			res.Errors = append(res.Errors, ScanError{Scenario: scenario, Reason: err.Error()})
			continue
		}
		res.ScenarioCount++
		if cand, ok := scorePostgresToSQLite(facts); ok {
			res.Candidates = append(res.Candidates, cand)
		}
	}
	sort.SliceStable(res.Candidates, func(i, j int) bool {
		if res.Candidates[i].FitnessScore != res.Candidates[j].FitnessScore {
			return res.Candidates[i].FitnessScore > res.Candidates[j].FitnessScore
		}
		return res.Candidates[i].Scenario < res.Candidates[j].Scenario
	})
	return res, nil
}

// scorePostgresToSQLite scores a scenario's fitness to migrate Postgres→SQLite.
// Vrooli is local-first: a single-node scenario carrying Postgres usually pays
// an external-service cost it does not need. The score is highest for
// not-yet-deployed scenarios (cheapest to switch) and is dampened by a blocker
// for already-deployed ones (a live data move + downtime window). Returns
// ok=false for scenarios that do not use Postgres.
func scorePostgresToSQLite(facts ScenarioFacts) (EngineCandidate, bool) {
	if !facts.usesEngine("postgres") {
		return EngineCandidate{}, false
	}
	score := 0.5
	var blockers []string
	switch facts.StorageStage {
	case "greenfield", "":
		score += 0.3 // not yet deployed — cheapest moment to choose SQLite
	case "pilot":
		score += 0.15
	case "production":
		blockers = append(blockers, "deployed on Postgres: migration needs a one-time data move and a brief downtime window")
	case "sunset":
		blockers = append(blockers, "scenario is being decommissioned: migration is likely not worth the effort")
		score -= 0.2
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return EngineCandidate{
		Scenario:          facts.Scenario,
		CurrentEngine:     "postgres",
		RecommendedEngine: "sqlite",
		FitnessScore:      score,
		Rationale:         "Vrooli is local-first; a single-node scenario carrying Postgres pays an external-service cost. SQLite removes that dependency, speeds startup, and matches the platform substrate. Engine migration is a deliberate change — review before adopting.",
		Autofixable:       false,
		Blockers:          blockers,
	}, true
}
