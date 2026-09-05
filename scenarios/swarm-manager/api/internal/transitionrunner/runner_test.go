package transitionrunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeWorkflow struct {
	start      agentmanager.WorkflowStart
	completion agentmanager.InvocationCompletion
	collectErr error
	invocation agentmanager.Invocation
}

func (f *fakeWorkflow) StartWorkflow(_ context.Context, in agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.invocation = in
	return f.start, nil
}

func (f *fakeWorkflow) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return f.completion, f.collectErr
}

func testRegistry(t *testing.T) transitions.Registry {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(`{"schemaVersion":"swarm-transition/v1","key":"capture.classify","subject":"capture","kind":"workflow","workflow":{"owner":"swarm-manager","key":"capture-workflow"},"inputContract":"capture/v1","terminalOutcomes":["classified"],"applyAction":"apply_capture"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := transitions.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func testRunner(t *testing.T) (*Runner, *fakeWorkflow, *string) {
	t.Helper()
	workflow := &fakeWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "exec-1", DefinitionDigest: "sha256:def"}}
	runner := New(testRegistry(t), workflow, transitionrun.NewFileStore(t.TempDir()), nil)
	version := "v1"
	runner.RegisterInput("capture.classify", func(context.Context, string) (Snapshot, error) {
		value, err := structpb.NewValue(map[string]any{"capture": "one"})
		return Snapshot{Input: value, EntityVersion: version}, err
	})
	return runner, workflow, &version
}

func completion(t *testing.T) agentmanager.InvocationCompletion {
	t.Helper()
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "classified"}})
	if err != nil {
		t.Fatal(err)
	}
	return agentmanager.InvocationCompletion{ExecutionID: "exec-1", DefinitionDigest: "sha256:def", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: output}
}

func TestStartUsesDeclaredLocatorAndIdempotencyKey(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	got, err := runner.Start(context.Background(), "capture.classify", "cap-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplyState != transitionrun.ApplyStateClaimed || workflow.invocation.IdempotencyKey != "capture.classify/cap-1/v1" || workflow.invocation.WorkflowKey != "capture-workflow" {
		t.Fatalf("start = %#v, invocation = %#v", got, workflow.invocation)
	}
}

func TestStartPreparedUsesRegistryLifecycleAndPersistsFrontier(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	input, err := structpb.NewValue(map[string]any{"prepared": true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := runner.StartPrepared(context.Background(), "capture.classify", "cap-1", PreparedInput{Input: input, EntityVersion: "v-prepared", FrontierDigest: "frontier-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.EntityVersion != "v-prepared" || got.FrontierDigest != "frontier-1" || workflow.invocation.IdempotencyKey != "capture.classify/cap-1/v-prepared" {
		t.Fatalf("prepared start = %#v, invocation = %#v", got, workflow.invocation)
	}
}

func TestApplyCompletesRegisteredActionAndReplays(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	workflow.completion = completion(t)
	called := 0
	runner.RegisterApply("apply_capture", func(_ context.Context, subject string, outcome Outcome) error {
		called++
		if subject != "cap-1" || outcome.Name != "classified" || outcome.TransitionKey != "capture.classify" {
			t.Fatalf("apply args = %q %#v", subject, outcome)
		}
		return nil
	})
	got, err := runner.Apply(context.Background(), "capture.classify", "exec-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplyState != transitionrun.ApplyStateComplete || got.AppliedTime == "" || called != 1 {
		t.Fatalf("apply = %#v calls=%d", got, called)
	}
	if _, err := runner.Apply(context.Background(), "capture.classify", "exec-1"); err != nil || called != 1 {
		t.Fatalf("replay err=%v calls=%d", err, called)
	}
}

func TestStartPreservesCompletedCorrelationForIdempotentExecution(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	workflow.completion = completion(t)
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	if _, err := runner.Apply(context.Background(), "capture.classify", "exec-1"); err != nil {
		t.Fatal(err)
	}
	got, err := runner.Start(context.Background(), "capture.classify", "cap-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplyState != transitionrun.ApplyStateComplete {
		t.Fatalf("idempotent start rewrote apply state: %#v", got)
	}
}

func TestApplySerializesConcurrentDeliveryOfOneExecution(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	workflow.completion = completion(t)
	var calls int
	var callsMu sync.Mutex
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error {
		callsMu.Lock()
		calls++
		callsMu.Unlock()
		return nil
	})
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := runner.Apply(context.Background(), "capture.classify", "exec-1"); err != nil {
				t.Errorf("apply: %v", err)
			}
		}()
	}
	workers.Wait()
	if calls != 1 {
		t.Fatalf("apply function calls = %d, want 1", calls)
	}
}

func TestApplyRejectsEachGuardReason(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	workflow.completion = completion(t)
	workflow.completion.DefinitionDigest = "sha256:other"
	var digest *transitionrun.DigestMismatchError
	if !errors.As(mustApply(runner), &digest) {
		t.Fatal("expected digest mismatch")
	}
	workflow.completion = completion(t)
	workflow.completion.Status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED
	var status *transitionrun.StatusNotSucceededError
	if !errors.As(mustApply(runner), &status) {
		t.Fatal("expected status rejection")
	}
	workflow.completion = completion(t)
	value, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "failed"}})
	workflow.completion.Output = value
	var outcome *transitionrun.OutcomeNotDeclaredError
	if !errors.As(mustApply(runner), &outcome) {
		t.Fatal("expected outcome rejection")
	}
}

func TestApplyRejectsChangedEntityVersion(t *testing.T) {
	runner, workflow, version := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	workflow.completion = completion(t)
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	*version = "v2"
	var changed *transitionrun.EntityVersionChangedError
	if !errors.As(mustApply(runner), &changed) {
		t.Fatal("expected entity version rejection")
	}
}

func mustApply(r *Runner) error {
	_, err := r.Apply(context.Background(), "capture.classify", "exec-1")
	return err
}

func TestVerifyDispatchTableNamesMissingAction(t *testing.T) {
	runner, _, _ := testRunner(t)
	err := runner.VerifyDispatchTable()
	if err == nil || !strings.Contains(err.Error(), "apply_capture") {
		t.Fatalf("VerifyDispatchTable error = %v", err)
	}
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	if err := runner.VerifyDispatchTable(); err != nil {
		t.Fatal(err)
	}
}

func TestListUnappliedExposesDurableRecoveryCandidate(t *testing.T) {
	runner, _, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	candidates, err := runner.ListUnapplied()
	if err != nil || len(candidates) != 1 || candidates[0].ExecutionID != "exec-1" {
		t.Fatalf("ListUnapplied = %#v, %v", candidates, err)
	}
}

func TestApplyRecordsAttemptAndFailureDiagnostics(t *testing.T) {
	runner, workflow, _ := testRunner(t)
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatal(err)
	}
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error { return nil })
	workflow.collectErr = agentmanager.ErrWorkflowNotReady
	got, err := runner.Apply(context.Background(), "capture.classify", "exec-1")
	if !errors.Is(err, agentmanager.ErrWorkflowNotReady) {
		t.Fatalf("Apply error = %v", err)
	}
	if got.ApplyAttemptCount != 1 || got.LastApplyAttemptTime == "" || !strings.Contains(got.LastApplyError, agentmanager.ErrWorkflowNotReady.Error()) {
		t.Fatalf("failure diagnostics = %#v", got)
	}
}

// TestApplyRejectsSubjectEditedWhileWorkflowRan is the regression test for the
// defect that shipped with the first cut of this runner: six transitions had
// input builders that echoed the version stored on the correlation instead of
// recomputing it. That made the rebuilt version equal to the stored one by
// construction, so this comparison always passed and a subject edited mid-run
// was applied anyway.
func TestApplyRejectsSubjectEditedWhileWorkflowRan(t *testing.T) {
	runner, workflow, version := testRunner(t)
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error {
		t.Fatal("apply must not run for a subject that changed while the workflow ran")
		return nil
	})
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	workflow.completion = succeededCompletion(t, "exec-1", "sha256:def", "classified")

	// The subject changes after the workflow started.
	*version = "v2"

	_, err := runner.Apply(context.Background(), "capture.classify", "exec-1")
	var changed *transitionrun.EntityVersionChangedError
	if !errors.As(err, &changed) {
		t.Fatalf("Apply error = %v, want EntityVersionChangedError", err)
	}
}

// TestApplyRejectsFrontierMovedWhileWorkflowRan covers the second digest the
// rebuild feeds into CanApply. Passing the correlation's own frontier back in
// would compare it to itself and leave FrontierChangedError unreachable.
func TestApplyRejectsFrontierMovedWhileWorkflowRan(t *testing.T) {
	workflow := &fakeWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "exec-1", DefinitionDigest: "sha256:def"}}
	runner := New(testRegistry(t), workflow, transitionrun.NewFileStore(t.TempDir()), nil)
	frontier := "frontier-1"
	runner.RegisterInput("capture.classify", func(context.Context, string) (Snapshot, error) {
		value, err := structpb.NewValue(map[string]any{"capture": "one"})
		return Snapshot{Input: value, EntityVersion: "v1", FrontierDigest: frontier}, err
	})
	runner.RegisterApply("apply_capture", func(context.Context, string, Outcome) error {
		t.Fatal("apply must not run for a subject whose frontier moved")
		return nil
	})
	if _, err := runner.Start(context.Background(), "capture.classify", "cap-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	workflow.completion = succeededCompletion(t, "exec-1", "sha256:def", "classified")

	frontier = "frontier-2"

	_, err := runner.Apply(context.Background(), "capture.classify", "exec-1")
	var moved *transitionrun.FrontierChangedError
	if !errors.As(err, &moved) {
		t.Fatalf("Apply error = %v, want FrontierChangedError", err)
	}
}

func succeededCompletion(t *testing.T, executionID, digest, outcome string) agentmanager.InvocationCompletion {
	t.Helper()
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": outcome}})
	if err != nil {
		t.Fatal(err)
	}
	return agentmanager.InvocationCompletion{
		ExecutionID:      executionID,
		DefinitionDigest: digest,
		Status:           domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		Output:           output,
	}
}
