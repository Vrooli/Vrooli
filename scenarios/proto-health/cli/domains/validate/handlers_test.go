package validate

import (
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestSeverityCountUsesCanonicalFindingSeverityKeys(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{
		FindingsBySeverity: map[string]int32{
			architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String():   18,
			architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String(): 7,
			"SEVERITY_ERROR": 99,
		},
	}

	if got := severityCount(assessment, architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR); got != 18 {
		t.Fatalf("severityCount(ERROR) = %d, want 18", got)
	}
	if got := severityCount(assessment, architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING); got != 7 {
		t.Fatalf("severityCount(WARNING) = %d, want 7", got)
	}
}

func TestFailedFindingCountIncludesBlockers(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{
		FindingsBySeverity: map[string]int32{
			architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String():   2,
			architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER.String(): 1,
			architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String(): 5,
		},
	}

	if got := failedFindingCount(assessment); got != 3 {
		t.Fatalf("failedFindingCount() = %d, want 3", got)
	}
}
