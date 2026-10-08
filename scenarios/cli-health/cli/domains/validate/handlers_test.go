package validate

import "testing"

type severityFixture map[string]int32

func (f severityFixture) GetFindingsBySeverity() map[string]int32 { return f }

func TestSeverityCountAcceptsSerializedFindingSeverityKeys(t *testing.T) {
	assessment := severityFixture{
		"FINDING_SEVERITY_ERROR":   60,
		"FINDING_SEVERITY_WARNING": 81,
	}

	if got := severityCount(assessment, "SEVERITY_ERROR"); got != 60 {
		t.Fatalf("error count = %d, want 60", got)
	}
	if got := severityCount(assessment, "FINDING_SEVERITY_WARNING"); got != 81 {
		t.Fatalf("warning count = %d, want 81", got)
	}
}
