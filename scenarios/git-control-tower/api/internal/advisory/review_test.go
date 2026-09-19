package advisory

import "testing"

func TestReviewResultRequiresActionableRevisionBoundFindings(t *testing.T) {
	result := ReviewResult{SubjectDigest: "sha256:s", Revision: "head-1", ExaminedFiles: 1, Findings: []Finding{{ID: "f-1", Severity: SeverityWarning, Path: "api/a.go", EvidenceRef: "hunk:1", Revision: "head-1", Reason: "error is ignored", Action: "handle the error", Status: FindingOpen}}}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	result.Findings[0].Revision = "head-0"
	if err := result.Validate(); err == nil {
		t.Fatal("stale finding accepted")
	}
}

func TestReviewCoverageDoesNotRequireZeroOmissions(t *testing.T) {
	result := ReviewResult{SubjectDigest: "sha256:s", Revision: "head-1", ExaminedFiles: 1, OmittedFiles: 1, CoverageNote: "binary file omitted"}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestReviewCoverageRequiresVisibleOmissionReason(t *testing.T) {
	result := ReviewResult{SubjectDigest: "sha256:s", Revision: "head-1", ExaminedFiles: 1, OmittedFiles: 1}
	if err := result.Validate(); err == nil {
		t.Fatal("review with omitted files and no coverage note was accepted")
	}
}

func TestReviewRejectsUnsafeFindingPath(t *testing.T) {
	result := ReviewResult{SubjectDigest: "sha256:s", Revision: "head-1", ExaminedFiles: 1, Findings: []Finding{{ID: "f-1", Severity: SeverityWarning, Path: "../secret", EvidenceRef: "hunk:1", Revision: "head-1", Reason: "unsafe", Action: "remove"}}}
	if err := result.Validate(); err == nil {
		t.Fatal("unsafe finding path was accepted")
	}
}
