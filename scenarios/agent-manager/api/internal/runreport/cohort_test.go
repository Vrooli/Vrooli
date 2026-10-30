package runreport

import (
	"testing"

	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

func TestBuildCohortRanksBoundedSignals(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	cohort := BuildCohort([]*RunReport{
		{RunID: a, Tools: []ToolSummary{{Name: "shell", Failures: 2}}, RepeatedToolCalls: 1, HelpRecoveries: 1, EventsAvailability: Availability{State: AvailabilityAvailable}, ReceiptsAvailability: Availability{State: AvailabilityUnobserved}, TimeAccounting: runsignal.TimeAccounting{ModelGeneratingMS: 4, ModelTokens: 2}},
		{RunID: b, Tools: []ToolSummary{{Name: "shell", Failures: 1}}, ExternalToolCalls: 3, EventsAvailability: Availability{State: AvailabilityAvailable}, ReceiptsAvailability: Availability{State: AvailabilityDegraded}, TimeAccounting: runsignal.TimeAccounting{ToolExecutingMS: 5, ToolTokens: 3}},
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
	if cohort.TimeAccounting.ModelGeneratingMS != 4 || cohort.TimeAccounting.ToolExecutingMS != 5 || cohort.TimeAccounting.Tokens() != 5 {
		t.Fatalf("cohort time accounting=%+v", cohort.TimeAccounting)
	}
}

// A selected run with no shown episodes is a clean run: the cohort stays
// available and has no signal for it.
func TestBuildEpisodeCohortTreatsARunWithoutEpisodesAsClean(t *testing.T) {
	cohort := BuildEpisodeCohort(map[string][]runsignal.FrictionEpisode{"clean-run": {}})
	if cohort.Availability.State != AvailabilityAvailable || len(cohort.Signals) != 0 {
		t.Fatalf("cohort = %#v, want available with no signals", cohort)
	}
}
