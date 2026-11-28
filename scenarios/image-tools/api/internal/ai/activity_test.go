package ai

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"image-tools/internal/capacity"
)

// activityBroker is a fakeBroker that records the idleness the engine reports,
// as the host broker does: a claim reads idle until its owner says otherwise.
type activityBroker struct {
	*fakeBroker
	mu     sync.Mutex
	state  string
	calls  []bool
	failed error
}

func newActivityBroker() *activityBroker {
	return &activityBroker{fakeBroker: &fakeBroker{}, state: "idle"}
}

func (b *activityBroker) Activity(_ context.Context, lease capacity.Lease, active bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if lease.ClaimID == "" {
		return errors.New("no claim")
	}
	if b.failed != nil {
		return b.failed
	}
	b.calls = append(b.calls, active)
	b.state = "idle"
	if active {
		b.state = "active"
	}
	return nil
}

func (b *activityBroker) current() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func TestAGPUJobsClaimReadsActiveWhileItRuns(t *testing.T) {
	broker := newActivityBroker()
	var duringRun string
	fp := &fakeProvider{onExecute: func() { duringRun = broker.current() }}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)
	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatal(err)
	}
	if duringRun != "active" {
		t.Fatalf("the GPU claim read %q while the job ran: the broker would treat a running job as reclaimable", duringRun)
	}
	if len(broker.calls) != 1 || !broker.calls[0] {
		t.Fatalf("activity reports = %v, want one active report at admission (the release ends the claim)", broker.calls)
	}
}

func TestAFailedActivityReportNeverStopsTheJob(t *testing.T) {
	broker := newActivityBroker()
	broker.failed = errors.New("broker unreachable")
	fp := &fakeProvider{}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)
	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatalf("an unreported claim must not fail the work: %v", err)
	}
	if fp.execN != 1 {
		t.Fatalf("job did not run: %d executions", fp.execN)
	}
}

func TestABrokerThatCannotReportActivityIsNotAnError(t *testing.T) {
	fp := &fakeProvider{}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, &fakeBroker{})
	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatal(err)
	}
}

// reporterLease is a renewable memory lease whose owner reports idleness.
type reporterLease struct {
	memoryTestLease
	calls []bool
	err   error
}

func (l *reporterLease) ReportActivity(_ context.Context, active bool) error {
	l.calls = append(l.calls, active)
	return l.err
}

func TestLeaseGroupReportsEveryReportingLeaseAndIgnoresTheRest(t *testing.T) {
	group := newLeaseGroup(context.Background())
	first, second := &reporterLease{}, &reporterLease{err: errors.New("stale generation")}
	group.renew("RAM", first)
	group.renew("CPU", second)
	group.renew("model files", &memoryTestLease{})
	err := group.setActivity(context.Background(), true)
	if err == nil {
		t.Fatal("a failed report was swallowed instead of surfaced to the caller")
	}
	if len(first.calls) != 1 || !first.calls[0] || len(second.calls) != 1 {
		t.Fatalf("every reporting lease must be told, even after another failed: %v %v", first.calls, second.calls)
	}
	if got := reportActivity(context.Background(), group, true); got == "active" {
		t.Fatalf("a failed report was described as %q", got)
	}
	second.err = nil
	if got := reportActivity(context.Background(), group, false); got != "idle" {
		t.Fatalf("outcome = %q", got)
	}
}

func TestAWarmResidentReadsActiveWhileAJobServesOnItAndIdleDuringTheIdleWindow(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	runner.idleTimeout = time.Minute
	t.Cleanup(func() { _ = runner.Close() })
	claim := &reporterLease{}
	acquire := func(ctx context.Context) (*leaseGroup, error) {
		group := newLeaseGroup(ctx)
		group.renew("GPU residency", claim)
		return group, nil
	}
	for i, wantCalls := range [][]bool{{true, false}, {true, false, true, false}} {
		meta := map[string]string{}
		admitted, err := runner.reserve(context.Background(), "model", "job", acquire, meta)
		if err != nil {
			t.Fatal(err)
		}
		if meta[residentActivityMetadataKey] != "active" {
			t.Fatalf("job %d: resident activity meta = %q", i, meta[residentActivityMetadataKey])
		}
		if last := claim.calls[len(claim.calls)-1]; !last {
			t.Fatalf("job %d: the resident claim was not active while the job served on it: %v", i, claim.calls)
		}
		if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, string(rune('a'+i)))}); err != nil {
			t.Fatal(err)
		}
		if err := admitted.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(claim.calls) != len(wantCalls) {
			t.Fatalf("job %d: reports = %v, want %v", i, claim.calls, wantCalls)
		}
		for j := range wantCalls {
			if claim.calls[j] != wantCalls[j] {
				t.Fatalf("job %d: reports = %v, want %v", i, claim.calls, wantCalls)
			}
		}
	}
}
