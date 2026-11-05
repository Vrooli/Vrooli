package advisory

import (
	"fmt"
	"strings"
)

type FindingSeverity string

const (
	SeverityInfo     FindingSeverity = "info"
	SeverityWarning  FindingSeverity = "warning"
	SeverityCritical FindingSeverity = "critical"
)

type FindingStatus string

const (
	FindingOpen      FindingStatus = "open"
	FindingDismissed FindingStatus = "dismissed"
	FindingDisputed  FindingStatus = "disputed"
	FindingConfirmed FindingStatus = "confirmed"
	FindingFixed     FindingStatus = "fixed"
)

type Finding struct {
	ID          string          `json:"id"`
	Severity    FindingSeverity `json:"severity"`
	Path        string          `json:"path"`
	EvidenceRef string          `json:"evidence_ref"`
	Revision    string          `json:"revision"`
	Reason      string          `json:"reason"`
	Action      string          `json:"action"`
	Status      FindingStatus   `json:"status"`
}

type ReviewResult struct {
	SubjectDigest string    `json:"subject_digest"`
	Revision      string    `json:"revision"`
	Findings      []Finding `json:"findings"`
	ExaminedFiles int       `json:"examined_files"`
	OmittedFiles  int       `json:"omitted_files"`
	CoverageNote  string    `json:"coverage_note,omitempty"`
}

func (r ReviewResult) Validate() error {
	if !strings.HasPrefix(r.SubjectDigest, "sha256:") || strings.TrimSpace(r.Revision) == "" || r.ExaminedFiles < 0 || r.OmittedFiles < 0 {
		return fmt.Errorf("review requires subject, revision, and non-negative coverage")
	}
	if r.OmittedFiles > 0 && strings.TrimSpace(r.CoverageNote) == "" {
		return fmt.Errorf("omitted review files require a coverage note")
	}
	for _, f := range r.Findings {
		if f.ID == "" || f.Path == "" || f.EvidenceRef == "" || f.Revision != r.Revision || f.Reason == "" || f.Action == "" {
			return fmt.Errorf("finding %q is not actionable and revision-bound", f.ID)
		}
		if strings.HasPrefix(f.Path, "/") || strings.Contains(f.Path, "..") {
			return fmt.Errorf("finding %q has unsafe path", f.ID)
		}
		switch f.Severity {
		case SeverityInfo, SeverityWarning, SeverityCritical:
		default:
			return fmt.Errorf("finding %q has invalid severity", f.ID)
		}
	}
	return nil
}
