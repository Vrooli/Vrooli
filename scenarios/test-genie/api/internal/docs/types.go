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

// DOC: docs/phases/docs/README.md#summary-metrics
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

	// Bidirectional reference tracking
	CodeRefsFound     int `json:"codeRefsFound"`
	CodeRefsBroken    int `json:"codeRefsBroken"`
	DocRefsFound      int `json:"docRefsFound"`
	DocRefsBroken     int `json:"docRefsBroken"`
	CodeFilesScanned  int `json:"codeFilesScanned"`
	MarkedRefsFound   int `json:"markedRefsFound"`
	MarkedRefsBroken  int `json:"markedRefsBroken"`
	MarkedRefsSkipped int `json:"markedRefsSkipped"`
	MarkedRefsUnknown int `json:"markedRefsUnknown"`

	// Manifest tracking
	DocsInManifest    int `json:"docsInManifest"`
	DocsNotInManifest int `json:"docsNotInManifest"`
}

// String returns a short human-readable summary.
func (s Summary) String() string {
	base := fmt.Sprintf("%d files, %d broken links, %d mermaid errors, %d markdown errors",
		s.FilesChecked, s.BrokenLinks, s.MermaidFailures, s.MarkdownFailures)

	// Add reference metrics if any were found
	if s.CodeRefsFound > 0 || s.DocRefsFound > 0 || s.MarkedRefsFound > 0 {
		base += fmt.Sprintf(", code refs: %d found/%d broken, doc refs: %d found/%d broken, marked refs: %d found/%d broken",
			s.CodeRefsFound, s.CodeRefsBroken, s.DocRefsFound, s.DocRefsBroken, s.MarkedRefsFound, s.MarkedRefsBroken)
	}

	return base
}
