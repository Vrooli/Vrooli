package validation

import (
	"context"
	"sort"

	corestorage "github.com/vrooli/api-core/storage"
)

// AnalyzerContext is the read-only input every analyzer receives. It is built
// once per ValidateScenario call from code-facts + service.json + on-disk
// inspection, so analyzers never re-resolve the scenario themselves.
type AnalyzerContext struct {
	// RepoRoot is the repository root used to resolve owner manifests and
	// relative storage declarations.
	RepoRoot string
	// Scenario is the validated scenario id.
	Scenario string
	// ScenarioDir is the absolute path to scenarios/<scenario>.
	ScenarioDir string
	// APIDir is the absolute path to the scenario's API surface directory
	// (scenarios/<scenario>/api), or "" when there is no API surface.
	APIDir string
	// Language is the detected API-surface language ("go", "typescript",
	// "python", or "").
	Language string
	// Engines is the set of storage engines the scenario declares/uses.
	Engines []Engine
	// Domains is the set of code-facts domains resolved for the scenario
	// (FACT_FAMILY_FILE_DOMAIN), used by per-domain schema analyzers.
	Domains []string
	// StorageStage is the derived deploy stage (greenfield/pilot/production/sunset).
	StorageStage string
	// HasMigrations reports whether a committed migrations/ directory exists.
	HasMigrations bool
	// Owner is the normalized native manifest for the validated scenario.
	Owner *corestorage.OwnerManifest
}

// IsGo reports whether the API surface is Go.
func (c AnalyzerContext) IsGo() bool {
	return c.Language == "go"
}

// HasEngine reports whether the context classified the given engine.
func (c AnalyzerContext) HasEngine(e Engine) bool {
	for _, got := range c.Engines {
		if got == e {
			return true
		}
	}
	return false
}

// HasRelationalStore reports whether the scenario uses a SQL engine (SQLite or
// Postgres) — the gate for routed-isolation and schema-layout analyzers.
func (c AnalyzerContext) HasRelationalStore() bool {
	return c.HasEngine(EngineSQLite) || c.HasEngine(EnginePostgres)
}

// Analyzer is the registry seam every storage check implements. An analyzer is
// pure: it inspects the AnalyzerContext (and the files it points at) and
// returns findings, never mutating state. Applies gates an analyzer out
// cheaply (e.g. Go-only, relational-only) before the more expensive Analyze.
type Analyzer interface {
	// Name is a short, stable identifier (used for ordering + provenance).
	Name() string
	// Applies reports whether this analyzer should run for the given context.
	Applies(ctx AnalyzerContext) bool
	// Analyze inspects the context and returns any findings.
	Analyze(c context.Context, ac AnalyzerContext) ([]Finding, error)
}

// registry holds every analyzer registered via register(). Analyzer files
// (schema_*.go, isolation_*.go, hygiene_*.go) self-register in their package
// init() so a new analyzer tier is added by dropping in a file — no central
// edit, no merge conflict between independently-developed tiers.
var registry []Analyzer

// register adds an analyzer to the package registry. Call it from an analyzer
// file's init(). Duplicate Name()s are a programming error and will surface as
// duplicate findings; keep names unique.
func register(a Analyzer) {
	registry = append(registry, a)
}

// DefaultAnalyzers returns the registered analyzer set in a stable, Name-sorted
// order, so the report is deterministic regardless of init() ordering across
// files.
//
// Phase 2 registers ZERO analyzers (the registry seam alone); each later tier
// registers its analyzers via init(), so this returns them automatically.
func DefaultAnalyzers() []Analyzer {
	out := append([]Analyzer(nil), registry...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
