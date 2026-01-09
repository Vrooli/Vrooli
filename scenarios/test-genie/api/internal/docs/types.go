package docs

import (
	"fmt"

	"test-genie/internal/shared"
)

// Re-export shared types for consistency.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	RunResult       = shared.RunResult[Summary]
)

const (
	FailureClassNone              = shared.FailureClassNone
	FailureClassMisconfiguration  = shared.FailureClassMisconfiguration
	FailureClassSystem            = shared.FailureClassSystem
	FailureClassMissingDependency = shared.FailureClassMissingDependency

	ObservationSection = shared.ObservationSection
	ObservationSuccess = shared.ObservationSuccess
	ObservationWarning = shared.ObservationWarning
	ObservationError   = shared.ObservationError
	ObservationInfo    = shared.ObservationInfo
	ObservationSkip    = shared.ObservationSkip
)

var (
	NewSectionObservation = shared.NewSectionObservation
	NewSuccessObservation = shared.NewSuccessObservation
	NewWarningObservation = shared.NewWarningObservation
	NewErrorObservation   = shared.NewErrorObservation
	NewInfoObservation    = shared.NewInfoObservation
	NewSkipObservation    = shared.NewSkipObservation
)

// Summary aggregates key counts for docs validation.
type Summary struct {
	FilesChecked     int `json:"filesChecked"`
	ExternalLinks    int `json:"externalLinks"`
	LocalLinks       int `json:"localLinks"`
	BrokenLinks      int `json:"brokenLinks"`
	MermaidValidated int `json:"mermaidValidated"`
	AbsolutePathHits int `json:"absolutePathHits"`
	MarkdownWarnings int `json:"markdownWarnings"`
	MarkdownFailures int `json:"markdownFailures"`
	ExternalWarnings int `json:"externalWarnings"`
	ExternalFailures int `json:"externalFailures"`
	MermaidFailures  int `json:"mermaidFailures"`
	AbsoluteFailures int `json:"absoluteFailures"`
}

// String returns a short human-readable summary.
func (s Summary) String() string {
	return fmt.Sprintf("%d files, %d broken links, %d mermaid errors, %d markdown errors",
		s.FilesChecked, s.BrokenLinks, s.MermaidFailures, s.MarkdownFailures)
}
