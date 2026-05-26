// Package manifest is the domain-scoped home for the per-scenario
// architecture manifest. Phase 2 ships the types + a service stub;
// Phase 4 adds the parser, validator, persistence cache, and overlay.
package manifest

import (
	"fmt"
	"time"
)

// ManifestVersion identifies a backwards-incompatible manifest shape.
type ManifestVersion string

const (
	ManifestVersionUnspecified ManifestVersion = ""
	ManifestVersionV1          ManifestVersion = "v1"
)

// DiagnosticSeverity classifies a manifest validation finding.
type DiagnosticSeverity string

const (
	DiagnosticSeverityInfo  DiagnosticSeverity = "info"
	DiagnosticSeverityWarn  DiagnosticSeverity = "warn"
	DiagnosticSeverityError DiagnosticSeverity = "error"
)

// Threshold is a tier boundary for the aggregator.
type Threshold struct {
	Tier     string
	MinValue float64
}

// SignalWeights is the per-scenario weight overlay.
type SignalWeights struct {
	Weights map[string]float64
}

// TransitionalDeclaration marks an intentional deviation as legitimate.
type TransitionalDeclaration struct {
	ID          string
	Kind        string
	Locator     string
	Rationale   string
	ExpiresWhen string
}

// DomainSpec describes one declared domain.
type DomainSpec struct {
	Name                  string
	Paths                 []string
	AllowedDependencies   []string
	Glossary              []string
	SignalWeightOverrides SignalWeights
}

// ManifestDefinition is the root document.
type ManifestDefinition struct {
	Version         ManifestVersion
	Scenario        string
	Domains         []DomainSpec
	SharedSubstrate []string
	SignalWeights   SignalWeights
	Thresholds      []Threshold
	Transitional    []TransitionalDeclaration
	ParsedAt        time.Time
	ContentHash     string
}

// Diagnostic is one validation finding.
type Diagnostic struct {
	Severity DiagnosticSeverity
	Path     string
	Line     int
	Column   int
	Message  string
	Code     string
}

// ErrManifestNotFound is the typed sentinel returned by GetManifest
// when no manifest has been validated for the requested scenario.
type ErrManifestNotFound struct {
	Scenario string
}

func (e ErrManifestNotFound) Error() string {
	return fmt.Sprintf("manifest for scenario %q not found; run validate first", e.Scenario)
}

// ErrInvalidManifest is the typed sentinel returned by ValidateManifest
// when one or more error-severity diagnostics were emitted.
type ErrInvalidManifest struct {
	Diagnostics []Diagnostic
}

func (e ErrInvalidManifest) Error() string {
	if len(e.Diagnostics) == 0 {
		return "manifest invalid"
	}
	return fmt.Sprintf("manifest invalid: %d diagnostic(s); first: %s", len(e.Diagnostics), e.Diagnostics[0].Message)
}
