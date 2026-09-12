package transitionrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestSweeperRetriesWorkflowNotReadyAndAppliesRecoveredResult(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	called := 0
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error {
		called++
		return nil
	})
	sweeper := NewSweeper(runner)
	sweeper.ApplyTimeout = time.Second
	workflow.collectErr = agentmanager.ErrWorkflowNotReady
	if applied, err := sweeper.RunOnce(context.Background()); err != nil || applied != 0 {
		t.Fatalf("first sweep = applied %d, err %v", applied, err)
	}
	first, err := runner.ListUnapplied()
	if err != nil || len(first) != 1 || first[0].ApplyAttemptCount != 1 || first[0].LastApplyError == "" {
		t.Fatalf("first sweep journal = %#v, %v", first, err)
	}
	workflow.collectErr = nil
	workflow.completion = completion(t)
	sweeper.RetryDelay = -1
	if applied, err := sweeper.RunOnce(context.Background()); err != nil || applied != 1 {
		t.Fatalf("recovery sweep = applied %d, err %v", applied, err)
	}
	complete, err := runner.store.Get("exec-1")
	if err != nil {
		t.Fatal(err)
	}
	if complete.ApplyState != transitionrun.ApplyStateComplete || complete.ApplyAttemptCount != 2 || complete.LastApplyError != "" || called != 1 {
		t.Fatalf("completed journal = %#v, calls=%d", complete, called)
	}
}

func TestSweeperSpacesRetryableAttempts(t *testing.T) {
	runner, _, _ := testRunner(t)
	sweeper := NewSweeper(runner)
	sweeper.RetryDelay = time.Second
	correlation := transitionrun.Correlation{LastApplyError: agentmanager.ErrNotAvailable.Error(), LastApplyAttemptTime: time.Now().UTC().Format(time.RFC3339Nano), ApplyAttemptCount: 3}
	if sweeper.retryDue(correlation, time.Now().UTC()) {
		t.Fatal("retryable correlation was retried before capped backoff elapsed")
	}
	if !sweeper.retryDue(correlation, time.Now().UTC().Add(5*time.Second)) {
		t.Fatal("retryable correlation remained blocked after capped backoff elapsed")
	}
}

func TestSweeperTreatsEngineUnavailableAsRetryLater(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	workflow.collectErr = agentmanager.ErrNotAvailable
	if applied, err := NewSweeper(runner).RunOnce(context.Background()); err != nil || applied != 0 {
		t.Fatalf("engine outage sweep = applied %d, err %v", applied, err)
	}
	correlation, err := runner.GetCorrelation("exec-1")
	if err != nil {
		t.Fatal(err)
	}
	if correlation.ApplyState != transitionrun.ApplyStateClaimed || correlation.ApplyAttemptCount != 1 || correlation.LastApplyError == "" {
		t.Fatalf("engine outage was not retained for retry: %#v", correlation)
	}
}

func TestSweeperRecoversEveryWorkflowSubjectKind(t *testing.T) {
	// [REQ:SWM-P0-014] The recovery loop is subject-agnostic. Keep this table
	// explicit so a new domain cannot accidentally bypass the shared journal.
	subjects := []struct{ key, subject string }{
		{"capture.classify", "capture"},
		{"plan.author", "backlog-item"},
		{"plan.workshop.review", "plan-workshop-subject"},
		{"review.evidence_request", "review-thread"},
		{"goal.plan", "goal"},
		{"milestone.review", "milestone"},
		{"scenario.spec_sync", "scenario"},
	}
	for _, subject := range subjects {
		t.Run(subject.subject, func(t *testing.T) {
			workflow := &fakeWorkflow{
				start:      agentmanager.WorkflowStart{ExecutionID: "exec-" + subject.subject, DefinitionDigest: "sha256:def"},
				completion: completion(t),
			}
			workflow.completion.ExecutionID = workflow.start.ExecutionID
			runner := New(singleWorkflowRegistry(t, subject.key, subject.subject), workflow, transitionrun.NewFileStore(t.TempDir()), nil)
			runner.RegisterInput(subject.key, func(context.Context, string) (Snapshot, error) {
				value, err := structpb.NewValue(map[string]any{"subject": subject.subject})
				return Snapshot{Input: value, EntityVersion: "v1"}, err
			})
			applied := 0
			runner.RegisterApply("apply_"+subject.subject, func(context.Context, string, Outcome) error { applied++; return nil })
			if _, err := runner.Start(context.Background(), subject.key, subject.subject+"-1"); err != nil {
				t.Fatal(err)
			}
			count, err := NewSweeper(runner).RunOnce(context.Background())
			if err != nil || count != 1 || applied != 1 {
				t.Fatalf("sweep = count %d, applied %d, err %v", count, applied, err)
			}
		})
	}
}

func singleWorkflowRegistry(t *testing.T, key, subject string) transitions.Registry {
	t.Helper()
	dir := t.TempDir()
	definition := fmt.Sprintf(`{"schemaVersion":"swarm-transition/v1","key":%q,"subject":%q,"kind":"workflow","workflow":{"owner":"swarm-manager","key":%q},"inputContract":"test/v1","terminalOutcomes":["classified"],"applyAction":%q}`,
		key, subject, key+"-workflow", "apply_"+subject)
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(definition), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := transitions.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}
