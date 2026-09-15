package runs

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeFleetSource struct {
	runs []execution.ScenarioRunRollup
	obs  []execution.PhaseObservation
}

func (f fakeFleetSource) AggregateScenarioRuns(context.Context, time.Time, int) ([]execution.ScenarioRunRollup, error) {
	return f.runs, nil
}

func (f fakeFleetSource) AggregatePhaseObservations(context.Context, time.Time, int) ([]execution.PhaseObservation, error) {
	return f.obs, nil
}

func TestGetFleetHealthUnimplementedWithoutSource(t *testing.T) {
	svc := NewService(t.TempDir(), nil, nil, nil)
	_, err := svc.GetFleetHealth(context.Background(), connect.NewRequest(&runspb.GetFleetHealthRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("err code = %v, want Unimplemented", connect.CodeOf(err))
	}
}

func TestGetFleetHealthAssemblesPayload(t *testing.T) {
	now := time.Now().UTC()
	src := fakeFleetSource{
		runs: []execution.ScenarioRunRollup{
			{ScenarioName: "flaky", Runs: 3, Passed: 1, LastCompletedAt: now.Add(-time.Hour), LastOutcome: "failed"},
			{ScenarioName: "healthy", Runs: 2, Passed: 2, LastCompletedAt: now.Add(-2 * time.Hour), LastOutcome: "passed"},
		},
		obs: []execution.PhaseObservation{
			{ScenarioName: "flaky", Status: "failed", FindingSource: "standards"},
		},
	}
	svc := NewService(t.TempDir(), nil, nil, nil)
	svc.SetFleetSource(src, func(context.Context) ([]string, error) {
		return []string{"flaky", "healthy", "untouched"}, nil
	})

	resp, err := svc.GetFleetHealth(context.Background(), connect.NewRequest(&runspb.GetFleetHealthRequest{
		WindowDays:    14,
		IncludeRoster: true,
	}))
	if err != nil {
		t.Fatalf("GetFleetHealth: %v", err)
	}
	fh := resp.Msg.GetFleetHealth()
	if fh == nil {
		t.Fatal("fleet_health missing")
	}
	if fh.GetScenariosTested() != 2 || fh.GetScenariosTotal() != 3 {
		t.Fatalf("tested=%d total=%d, want 2/3", fh.GetScenariosTested(), fh.GetScenariosTotal())
	}
	if fh.GetCapturedAt() == "" {
		t.Fatal("captured_at empty; fleet data must be as-of stamped")
	}
	if len(fh.GetScenarios()) == 0 || fh.GetScenarios()[0].GetTarget() != "flaky" {
		t.Fatalf("most-errored first wrong: %+v", fh.GetScenarios())
	}
	if got := fh.GetNeverTestedInWindow(); len(got) != 1 || got[0] != "untouched" {
		t.Fatalf("never_tested = %v, want [untouched]", got)
	}
	if fh.GetScenarios()[0].GetLastRunAt() == "" {
		t.Fatal("flaky last_run_at empty; per-scenario staleness must be stamped")
	}
}
