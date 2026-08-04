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

import (
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

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
	Code string `json:"code"`
	// Severity is the analyzer-assigned severity. When zero-valued the maturity
	// engine falls back to the catalog's severity_default for the code.
	Severity Severity `json:"severity"`
	// Title is a short human headline.
	Title string `json:"title"`
	// Message is the detailed, instructive explanation — for isolation/safety
	// findings this MUST be loud: why it was flagged, why it matters (real-data
	// risk), and the exact remediation (which seam to wire / which autofix).
	Message string `json:"message"`
	// Location is a repo-relative file (optionally :line) the finding points at.
	Location string `json:"location,omitempty"`
	// Remediation is the concrete fix instruction.
	Remediation string `json:"remediation,omitempty"`
	// AutofixAvailable marks that a registered storage-manager autofix can
	// remediate this instance. Wired in the autofix phase.
	AutofixAvailable bool `json:"autofix_available,omitempty"`
	// Analyzer is the name of the analyzer that produced the finding (for
	// deterministic ordering + provenance).
	Analyzer string `json:"analyzer,omitempty"`
	// Subject identifies the owner this finding is about when a fleet-level
	// scenario validation reports on another manifest-backed target.
	Subject *commonv1.ValidationTarget `json:"subject,omitempty"`
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
	Scenario string `json:"scenario"`
	// OwnerKind and OwnerID identify the native manifest validated. Scenario is
	// retained as the shared response field for compatibility with Test Genie.
	OwnerKind corestorage.OwnerKind `json:"owner_kind"`
	OwnerID   string                `json:"owner_id"`
	Platform  corestorage.Platform  `json:"platform"`
	Status    string                `json:"status"`
	// Analyzers records both executed and kind-gated analyzers so consumers can
	// distinguish not-applicable from an analyzer that silently disappeared.
	Analyzers []AnalyzerResult `json:"analyzers,omitempty"`
	// ScenarioDir is the absolute path to the scenario directory on disk.
	ScenarioDir string `json:"scenario_dir,omitempty"`
	// Language is the detected API-surface language ("go", "typescript",
	// "python", or "" when undetermined). Drives Go-only analyzer gating.
	Language string `json:"language,omitempty"`
	// Engines is the set of storage engines the scenario declares/uses, in a
	// stable order.
	Engines []Engine `json:"engines,omitempty"`
	// StorageStage is the derived deploy/greenfield stage (greenfield, pilot,
	// production, sunset) — informational; migration findings reference it.
	StorageStage string `json:"storage_stage,omitempty"`
	// HasMigrations reports whether a committed migrations/ directory exists.
	HasMigrations bool `json:"has_migrations"`
	// Findings are the aggregated, deterministically-sorted findings.
	Findings []Finding `json:"findings,omitempty"`
}

type AnalyzerResult struct {
	Name        string   `json:"name"`
	Applicable  bool     `json:"applicable"`
	Reason      string   `json:"reason,omitempty"`
	FindingCode []string `json:"finding_codes,omitempty"`
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
