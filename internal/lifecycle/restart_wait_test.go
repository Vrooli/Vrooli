package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// A replacement may spend minutes preparing while the old process serves.
// Wait must attach to that attempt, including when preparation then fails.
func TestRestartWaitObservesPreparationInsteadOfPreviousSuccess(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	manifest := lifecycleFixtureManifest("alpha")
	manifest.HostTools = []hostreqspec.Declaration{{Name: "go", Required: true, Reason: "replacement compiler", When: []string{"develop"}}}
	writeLifecycleFixtureManifest(t, root, manifest)
	store := openWaitTestStore(t, home)
	previous := seedRunningStartOperation(t, store, "alpha", 99999)
	previous.Status = scenarioruntime.StartOperationStatusSucceeded
	previous.Verdict = WaitVerdictHealthy
	finished := time.Now().UTC()
	previous.FinishedAt = &finished
	if _, err := store.UpdateStartOperation(t.Context(), previous); err != nil {
		t.Fatal(err)
	}
	clock := newFakeAwaitClock()
	clock.now = finished.Add(time.Second)
	var runner *Runner
	preflightFailure := errors.New("replacement host requirement unavailable")
	var observedID string
	runner = newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.now, deps.sleep = clock.Now, clock.Sleep
		deps.isPIDRunning = func(int) bool { return true }
		deps.enforceHostRequirements = func(vrooliruntime.Options) (vrooliruntime.Report, error) {
			current := latestStartOperation(t, home, "alpha")
			if current.OperationID == previous.OperationID || current.Status != scenarioruntime.StartOperationStatusRunning || current.Operation != lifecycleOperationRestart {
				t.Fatalf("preflight observed stale operation: %+v", current)
			}
			observedID = current.OperationID
			outcome, err := runner.WaitScenario("alpha", WaitOptions{Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Attached || !outcome.TimedOut || outcome.Operation == nil || outcome.Operation.OperationID != current.OperationID {
				t.Fatalf("wait during preparation = %+v; want attachment to %s", outcome, current.OperationID)
			}
			return vrooliruntime.Report{}, preflightFailure
		}
	})
	if _, err := runner.Restart("alpha", StartOptions{}); !errors.Is(err, preflightFailure) {
		t.Fatalf("Restart: %v", err)
	}
	final := latestStartOperation(t, home, "alpha")
	if observedID == "" || final.OperationID != observedID || final.Status != scenarioruntime.StartOperationStatusFailed || !strings.Contains(final.Error, preflightFailure.Error()) {
		t.Fatalf("preflight result = %+v; want exact failed attempt %s", final, observedID)
	}
	outcome, err := runner.WaitScenario("alpha", WaitOptions{})
	if err != nil || outcome.Verdict != WaitVerdictFailed || outcome.Operation == nil || outcome.Operation.OperationID != observedID {
		t.Fatalf("wait after preflight failure = %+v, %v", outcome, err)
	}
}

func TestWaitScenarioDoesNotTrustHistoricalSuccessWhenRegistryReadFails(t *testing.T) {
	runner, _, home := newWaitTestRunner(t, func(int) bool { return true })
	store := openWaitTestStore(t, home)
	op := seedRunningStartOperation(t, store, "alpha", 99999)
	op.Status, op.Verdict = scenarioruntime.StartOperationStatusSucceeded, WaitVerdictHealthy
	finished := time.Now().UTC()
	op.FinishedAt = &finished
	if _, err := store.UpdateStartOperation(t.Context(), op); err != nil {
		t.Fatal(err)
	}
	registryFailure := errors.New("runtime registry cannot be read")
	registry := runner.runtimeDeps().runtimeRegistry
	reads := 0
	runner.deps.runtimeRegistry = func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
		reads++
		if reads > 1 {
			return nil, registryFailure
		}
		return registry(ctx, home)
	}
	outcome, err := runner.WaitScenario("alpha", WaitOptions{})
	if !errors.Is(err, registryFailure) || outcome.Healthy() || outcome.Verdict == WaitVerdictNotRunning {
		t.Fatalf("unverified historical success = %+v, %v", outcome, err)
	}
}

func TestWaitScenarioKeepsExactAttemptWhenSuccessorSupersedesIt(t *testing.T) {
	runner, clock, home := newWaitTestRunner(t, func(int) bool { return true })
	store := openWaitTestStore(t, home)
	attached := seedRunningStartOperation(t, store, "alpha", 99999)
	var successor scenarioruntime.StartOperation
	runner.deps.sleep = func(d time.Duration) {
		clock.Sleep(d)
		var err error
		successor, err = store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{Scenario: "alpha", StartedAt: clock.Now()})
		if err != nil {
			t.Fatal(err)
		}
	}
	outcome, err := runner.WaitScenario("alpha", WaitOptions{Timeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Attached || outcome.TimedOut || outcome.Verdict != WaitVerdictAbandoned || outcome.Operation == nil || outcome.Operation.OperationID != attached.OperationID {
		t.Fatalf("superseded attachment = %+v; want exact original %s", outcome, attached.OperationID)
	}
	current, err := store.GetLatestStartOperation(t.Context(), "alpha", "live")
	if err != nil || current.OperationID != successor.OperationID || current.Status != scenarioruntime.StartOperationStatusRunning {
		t.Fatalf("wait modified successor: %+v, %v", current, err)
	}
}

func TestRestartWaitRecorderCompletesStopWithoutDuplicatingIt(t *testing.T) {
	rec := &startOperationRecorder{}
	started := time.Now().UTC()
	rec.beginStep(startStepStop, started)
	rec.markCompletedStep(startStepStop, started, started.Add(time.Second))
	if len(rec.steps) != 1 || rec.steps[0].Status != scenarioruntime.StartStepDone || rec.steps[0].EndedAt == nil {
		t.Fatalf("stop history = %+v", rec.steps)
	}
}

type restartRecordFailureStore struct {
	scenarioRuntimeStore
	err error
}

func (s *restartRecordFailureStore) BeginStartOperation(context.Context, scenarioruntime.StartOperation) (scenarioruntime.StartOperation, error) {
	return scenarioruntime.StartOperation{}, s.err
}

func TestRestartWaitRequiresDurableAdmissionBeforePreparation(t *testing.T) {
	for _, failBegin := range []bool{false, true} {
		t.Run(map[bool]string{false: "open registry", true: "begin attempt"}[failBegin], func(t *testing.T) {
			runner, _, _ := newWaitTestRunner(t, func(int) bool { return true })
			failure := errors.New("durable attempt unavailable")
			registry := runner.runtimeDeps().runtimeRegistry
			runner.deps.runtimeRegistry = func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
				if !failBegin {
					return nil, failure
				}
				store, err := registry(ctx, home)
				return &restartRecordFailureStore{scenarioRuntimeStore: store, err: failure}, err
			}
			preparation := &fakeEngagementResolver{}
			runner.Engagements = preparation
			if _, err := runner.Restart("alpha", StartOptions{}); !errors.Is(err, failure) {
				t.Fatalf("Restart: %v", err)
			}
			if len(preparation.asked) != 0 {
				t.Fatal("replacement preparation began without durable admission")
			}
		})
	}
}
