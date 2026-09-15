package fleetscheduler_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"test-genie/internal/fleetscheduler"
)

// recordingLauncher stands in for the real runmanager-backed launcher during the
// live smoke. It records which scenarios a cycle selected and "launches" without
// paying the cost of a real ~15-minute suite. The selection pipeline (live SCS
// score-list read → parse → staleness-weighted ranking → single-cycle
// bookkeeping) is the genuinely-unproven new code; the real launcher (Start+Wait
// on the durable run manager) is pre-existing, already-proven infrastructure.
type recordingLauncher struct {
	mu       sync.Mutex
	launched []string
}

func (r *recordingLauncher) Launch(_ context.Context, scenario string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.launched = append(r.launched, scenario)
	return "smoke-" + scenario, nil
}

func (r *recordingLauncher) Await(_ context.Context, _, _ string) (string, error) {
	return "passed", nil
}

// TestFleetSchedulerLiveSmoke exercises the scheduler against the REAL
// scenario-completeness-scoring CLI and live fleet data, then runs one bounded
// cycle (MaxRunsPerCycle=1, MaxConcurrent=1) through a recording launcher.
//
// Gated behind TEST_GENIE_FLEET_SMOKE=1 so it never runs in ordinary `go test`
// (it shells out to the SCS CLI and depends on live host state). Run with:
//
//	TEST_GENIE_FLEET_SMOKE=1 go test ./internal/fleetscheduler/ \
//	    -run TestFleetSchedulerLiveSmoke -v -count=1
func TestFleetSchedulerLiveSmoke(t *testing.T) {
	if os.Getenv("TEST_GENIE_FLEET_SMOKE") != "1" {
		t.Skip("set TEST_GENIE_FLEET_SMOKE=1 to run the live fleet-scheduler smoke")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1) Real priority feed against the live SCS CLI + fleet data.
	src := fleetscheduler.NewCLIPrioritySource(0)
	candidates, err := src.Candidates(ctx)
	if err != nil {
		t.Fatalf("live score-list read failed: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("live score list returned 0 candidates; expected a non-empty fleet")
	}
	t.Logf("live candidates: %d", len(candidates))
	preview := candidates
	if len(preview) > 10 {
		preview = preview[:10]
	}
	for i, c := range preview {
		last := "never"
		if !c.LastRunAt.IsZero() {
			last = c.LastRunAt.Format(time.RFC3339)
		}
		t.Logf("  #%-2d %-36s priority=%.4f importance=%.4f last_run=%s status=%q",
			i+1, c.Scenario, c.Priority, c.Importance, last, c.LastStatus)
	}

	// 2) One bounded cycle through a recording launcher (no real suites).
	rec := &recordingLauncher{}
	sched, err := fleetscheduler.New(fleetscheduler.Config{
		Source:           src,
		Launcher:         rec,
		Interval:         6 * time.Hour,
		MaxConcurrent:    1,
		MaxRunsPerCycle:  1,
		StalenessHorizon: time.Hour, // short horizon: staleness dominates ranking
	})
	if err != nil {
		t.Fatalf("scheduler construction failed: %v", err)
	}

	report, err := sched.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	t.Logf("cycle report: candidates=%d selected=%d launched=%d passed=%d failed=%d skipped=%d errored=%d budget_hit=%t duration=%s",
		report.Candidates, report.Selected, report.Launched, report.Passed,
		report.Failed, report.Skipped, report.Errored, report.BudgetHit, report.Duration)

	// 3) Invariants: exactly one scenario selected, launched, and recorded.
	if report.Candidates != len(candidates) {
		t.Errorf("cycle saw %d candidates, live feed had %d", report.Candidates, len(candidates))
	}
	if report.Selected != 1 {
		t.Errorf("selected=%d, want 1 (MaxRunsPerCycle=1)", report.Selected)
	}
	if report.Launched != 1 {
		t.Errorf("launched=%d, want 1", report.Launched)
	}
	if len(rec.launched) != 1 {
		t.Fatalf("recording launcher saw %d launches, want 1", len(rec.launched))
	}
	t.Logf("cycle picked + launched: %s (highest staleness-weighted priority)", rec.launched[0])
}
