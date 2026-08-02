// Package validation is storage-manager's storage-judgment engine. It detects
// a target scenario's storage surface (engines + API language), runs the
// applicable static analyzers, and aggregates normalized findings into a
// Report. The Connect handler and the CLI are thin translation layers over
// Service.ValidateScenario.
//
// Phase 2 establishes the seams (Service, Analyzer registry, code-facts-backed
// language detection) with ZERO real analyzers, so the producer is green
// end-to-end and emits a clean assessment. Tiers of analyzers land in later
// phases by registering into DefaultAnalyzers.
package validation

import "strings"

// Severity is storage-manager's internal severity ladder. It is mapped to the
// shared FindingSeverity vocabulary (SEVERITY_ERROR/WARNING/INFO) by the
// Connect handler when building the maturity assessment.
type Severity int

const (
	// SeverityInfo is advisory context — never gates a phase.
	SeverityInfo Severity = iota
	// SeverityWarning is a real gap that is informational rather than blocking.
	SeverityWarning
	// SeverityError blocks local maturity at the finding's level and, for the
	// L2 isolation rung, fails the test-genie storage phase (which fail-closes
	// the destructive playbooks phase).
	SeverityError
)

// Token returns the shared severity token the maturity assessment expects.
func (s Severity) Token() string {
	switch s {
	case SeverityError:
		return "SEVERITY_ERROR"
	case SeverityWarning:
		return "SEVERITY_WARNING"
	case SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

// Finding is a single storage-judgment observation. Code matches a key in
// .vrooli/maturity.json so the maturity engine resolves its level/impact.
type Finding struct {
	// Code is the stable finding code (e.g. ROUTED_SEAMS_UNWIRED). It MUST
	// exist in the scenario's maturity.json findings catalog, otherwise the
	// maturity engine resolves it via the fallback policy.
	Code string
	// Severity is the analyzer-assigned severity. When zero-valued the maturity
	// engine falls back to the catalog's severity_default for the code.
	Severity Severity
	// Title is a short human headline.
	Title string
	// Message is the detailed, instructive explanation — for isolation/safety
	// findings this MUST be loud: why it was flagged, why it matters (real-data
	// risk), and the exact remediation (which seam to wire / which autofix).
	Message string
	// Location is a repo-relative file (optionally :line) the finding points at.
	Location string
	// Remediation is the concrete fix instruction.
	Remediation string
	// AutofixAvailable marks that a registered storage-manager autofix can
	// remediate this instance. Wired in the autofix phase.
	AutofixAvailable bool
	// Analyzer is the name of the analyzer that produced the finding (for
	// deterministic ordering + provenance).
	Analyzer string
}

// Engine enumerates the storage engines storage-manager classifies.
type Engine string

const (
	EngineSQLite   Engine = "sqlite"
	EnginePostgres Engine = "postgres"
	EngineQdrant   Engine = "qdrant"
	EngineRedis    Engine = "redis"
	EngineFile     Engine = "file"
)

// Report is the result of validating one scenario.
type Report struct {
	// Scenario is the validated scenario id.
	Scenario string
	// ScenarioDir is the absolute path to the scenario directory on disk.
	ScenarioDir string
	// Language is the detected API-surface language ("go", "typescript",
	// "python", or "" when undetermined). Drives Go-only analyzer gating.
	Language string
	// Engines is the set of storage engines the scenario declares/uses, in a
	// stable order.
	Engines []Engine
	// StorageStage is the derived deploy/greenfield stage (greenfield, pilot,
	// production, sunset) — informational; migration findings reference it.
	StorageStage string
	// HasMigrations reports whether a committed migrations/ directory exists.
	HasMigrations bool
	// Findings are the aggregated, deterministically-sorted findings.
	Findings []Finding
}

// HasEngine reports whether the report classified the given engine.
func (r Report) HasEngine(e Engine) bool {
	for _, got := range r.Engines {
		if got == e {
			return true
		}
	}
	return false
}

// IsGo reports whether the API surface language is Go (case-insensitive).
func (r Report) IsGo() bool { return strings.EqualFold(r.Language, "go") }
