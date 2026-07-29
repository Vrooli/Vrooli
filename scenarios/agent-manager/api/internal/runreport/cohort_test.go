package runreport

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildCohortRanksBoundedSignals(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	cohort := BuildCohort([]*RunReport{
		{RunID: a, Tools: []ToolSummary{{Name: "shell", Failures: 2}}, RepeatedToolCalls: 1, HelpRecoveries: 1, EventsAvailability: Availability{State: "available"}, ReceiptsAvailability: Availability{State: "unobserved"}},
		{RunID: b, Tools: []ToolSummary{{Name: "shell", Failures: 1}}, ExternalToolCalls: 3, EventsAvailability: Availability{State: "available"}, ReceiptsAvailability: Availability{State: "degraded"}},
	})
	if cohort.ClassifierVersion != ClassifierVersion || len(cohort.RunIDs) != 2 {
		t.Fatalf("cohort=%+v", cohort)
	}
	if cohort.Signals[0].Kind != "command_failure" || cohort.Signals[0].Impact != 3 {
		t.Fatalf("signals=%+v", cohort.Signals)
	}
	var failures *CohortSignal
	for i := range cohort.Signals {
		if cohort.Signals[i].Kind == "command_failure" {
			failures = &cohort.Signals[i]
		}
	}
	if failures == nil || failures.Count != 2 || failures.Confidence != "high" {
		t.Fatalf("failure signal=%+v", failures)
	}
}
