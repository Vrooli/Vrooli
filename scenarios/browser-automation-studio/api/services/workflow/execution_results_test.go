package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

type lifecycleRepository struct {
	database.Repository
	execution *database.ExecutionIndex
}

func (r *lifecycleRepository) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	return r.execution, nil
}

func (r *lifecycleRepository) UpdateExecutionStatus(_ context.Context, _ uuid.UUID, status string, _ *string, _ *time.Time, _ time.Time) error {
	r.execution.Status = status
	return nil
}

type lifecycleSink struct {
	*events.MemorySink
	closed []uuid.UUID
}

func (s *lifecycleSink) CloseExecution(id uuid.UUID) {
	s.closed = append(s.closed, id)
	s.MemorySink.CloseExecution(id)
}

type lifecycleExecutor struct{ err error }

func (e lifecycleExecutor) Execute(context.Context, executor.Request) error { return e.err }

type completionRepository struct {
	database.Repository
	mu           sync.Mutex
	index        database.ExecutionIndex
	workflow     *database.WorkflowIndex
	project      *database.ProjectIndex
	reads        int
	failTerminal bool
	readErr      error
}

func (r *completionRepository) GetWorkflow(context.Context, uuid.UUID) (*database.WorkflowIndex, error) {
	return r.workflow, nil
}

func (r *completionRepository) GetWorkflowByName(context.Context, string, string) (*database.WorkflowIndex, error) {
	return r.workflow, nil
}

func (r *completionRepository) GetProject(context.Context, uuid.UUID) (*database.ProjectIndex, error) {
	return r.project, nil
}

func (r *completionRepository) CreateExecution(_ context.Context, index *database.ExecutionIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index = *index
	return nil
}

func (r *completionRepository) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.readErr != nil {
		return nil, r.readErr
	}
	copy := r.index
	return &copy, nil
}

func (r *completionRepository) UpdateExecutionStatus(_ context.Context, _ uuid.UUID, status string, detail *string, completed *time.Time, updated time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if completed != nil && r.failTerminal {
		return errors.New("terminal index write failed")
	}
	r.index.Status, r.index.CompletedAt, r.index.UpdatedAt = status, completed, updated
	if detail != nil {
		r.index.ErrorMessage = *detail
	}
	return nil
}

type completionExecutor func(context.Context, executor.Request) error

func (f completionExecutor) Execute(ctx context.Context, req executor.Request) error {
	return f(ctx, req)
}

type completionResult struct {
	status    basbase.ExecutionStatus
	completed bool
	detail    string
	err       error
}

func newCompletionFixture(t *testing.T, mode string, run completionExecutor) (*WorkflowService, *completionRepository, func(context.Context, bool) completionResult) {
	t.Helper()
	project := &database.ProjectIndex{ID: uuid.New(), FolderPath: t.TempDir()}
	id := uuid.New()
	flow := &basworkflows.WorkflowDefinitionV2{Nodes: []*basworkflows.WorkflowNodeV2{{Id: "navigate", Action: &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
		Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}},
	}}}}
	summary := &basapi.WorkflowSummary{Id: id.String(), ProjectId: project.ID.String(), Name: "completion fixture", Version: 1, FlowDefinition: flow}
	_, relative, err := WriteWorkflowSummaryFile(project, summary, "")
	if err != nil {
		t.Fatal(err)
	}
	repo := &completionRepository{workflow: &database.WorkflowIndex{ID: id, ProjectID: &project.ID, FilePath: relative, Version: 1}, project: project}
	service := &WorkflowService{repo: repo, executor: run, executionDataRoot: t.TempDir()}
	invoke := func(ctx context.Context, wait bool) completionResult {
		if mode == "saved" {
			response, err := service.ExecuteWorkflowAPI(ctx, &basapi.ExecuteWorkflowRequest{WorkflowId: id.String(), WaitForCompletion: wait})
			return completionResult{response.GetStatus(), response.GetCompletedAt() != nil, response.GetError(), err}
		}
		response, err := service.ExecuteAdhocWorkflowAPI(ctx, &basexecution.ExecuteAdhocRequest{FlowDefinition: flow, WaitForCompletion: wait})
		return completionResult{response.GetStatus(), response.GetCompletedAt() != nil, response.GetError(), err}
	}
	return service, repo, invoke
}

// [REQ:BAS-RH-J07] Waiting observes the owned runner's completion without a
// polling delay or repeated reads of an execution that is still running.
func TestSynchronousExecutionCompletion(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		for _, tc := range []struct {
			name   string
			delay  time.Duration
			runErr error
			status basbase.ExecutionStatus
		}{
			{"immediate", 0, nil, basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
			{"delayed", 1025 * time.Millisecond, nil, basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
			{"failed", 0, errors.New("action failed"), basbase.ExecutionStatus_EXECUTION_STATUS_FAILED},
			{"cancelled", 0, context.Canceled, basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					_, repo, invoke := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { time.Sleep(tc.delay); return tc.runErr })
					start := time.Now()
					got := invoke(context.Background(), true)
					if got.err != nil || !got.completed || got.status != tc.status {
						t.Fatalf("completion = %+v, want %s with terminal timestamp", got, tc.status)
					}
					wantDetail := ""
					if tc.runErr != nil {
						wantDetail = tc.runErr.Error()
					}
					if tc.status == basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED {
						wantDetail = "execution cancelled"
					}
					if got.detail != wantDetail {
						t.Errorf("execution failure detail = %q, want %q", got.detail, wantDetail)
					}
					if elapsed := time.Since(start); elapsed != tc.delay {
						t.Errorf("completion added %v beyond runner duration", elapsed-tc.delay)
					}
					if repo.reads != 2 {
						t.Errorf("execution reads = %d; want initial runner read and one terminal receipt", repo.reads)
					}
				})
			})
		}
	}
}

func TestSynchronousExecutionRejectsMissingTerminalReceipt(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				_, repo, invoke := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { return nil })
				repo.failTerminal = true
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				got := invoke(ctx, true)
				if got.err == nil || errors.Is(got.err, context.DeadlineExceeded) {
					t.Fatalf("missing terminal receipt must fail when the runner exits, got %+v", got)
				}
			})
		})
	}
}

type delayedCompletionSink struct {
	*events.MemorySink
	release <-chan struct{}
}

func (s *delayedCompletionSink) CloseExecution(id uuid.UUID) {
	<-s.release
	s.MemorySink.CloseExecution(id)
}

func TestSynchronousExecutionWaitsForOwnedTeardown(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, _, invoke := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { return nil })
				release := make(chan struct{})
				sink := &delayedCompletionSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits), release: release}
				service.eventSinkFactory = func() events.Sink { return sink }
				result := make(chan completionResult, 1)
				go func() { result <- invoke(context.Background(), true) }()
				synctest.Wait()
				time.Sleep(300 * time.Millisecond)
				synctest.Wait()
				returned := false
				select {
				case got := <-result:
					returned = true
					t.Errorf("returned before owned teardown finished: %+v", got)
				default:
				}
				close(release)
				synctest.Wait()
				if !returned {
					got := <-result
					if got.err != nil || !got.completed || got.status != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
						t.Errorf("completion after teardown = %+v", got)
					}
				}
			})
		})
	}
}

func TestSynchronousWaitCancellationPreservesDetachedExecution(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				var executionErr error
				_, repo, invoke := newCompletionFixture(t, mode, func(ctx context.Context, _ executor.Request) error { <-release; executionErr = ctx.Err(); return nil })
				ctx, cancel := context.WithCancel(context.Background())
				result := make(chan completionResult, 1)
				go func() { result <- invoke(ctx, true) }()
				synctest.Wait()
				cancel()
				synctest.Wait()
				got := <-result
				if !errors.Is(got.err, context.Canceled) {
					t.Errorf("cancelled waiter error = %v", got.err)
				}
				close(release)
				synctest.Wait()
				if executionErr != nil || repo.index.CompletedAt == nil {
					t.Fatalf("detached execution did not finish: context=%v status=%s", executionErr, repo.index.Status)
				}
			})
		})
	}
}

// [REQ:BAS-RH-J07] The workflow owner retires decorated sinks on all exits.
func TestWorkflowClosesDecoratedSinkOnEveryExit(t *testing.T) {
	for _, mode := range []string{"fresh", "resumed"} {
		for _, outcome := range []string{"completed", "failed", "cancelled", "compile-failed"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				id, workflowID := uuid.New(), uuid.New()
				repo := &lifecycleRepository{execution: &database.ExecutionIndex{ID: id, WorkflowID: workflowID}}
				sink := &lifecycleSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
				runner := lifecycleExecutor{}
				if outcome == "failed" {
					runner.err = errors.New("synthetic action failure")
				} else if outcome == "cancelled" {
					runner.err = context.Canceled
				}
				service := &WorkflowService{
					repo: repo, executor: runner, executionDataRoot: t.TempDir(),
					eventSinkFactory: func() events.Sink { return uxcollector.NewCollector(sink, nil) },
				}
				workflow := &basapi.WorkflowSummary{Id: workflowID.String(), FlowDefinition: &basworkflows.WorkflowDefinitionV2{
					Nodes: []*basworkflows.WorkflowNodeV2{{Id: "navigate", Action: &basactions.ActionDefinition{
						Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
						Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}},
					}}},
				}}
				if outcome == "compile-failed" {
					workflow.FlowDefinition = nil
				}
				if mode == "resumed" {
					service.executeResumedWorkflowAsync(context.Background(), workflow, id, &CheckpointState{}, nil, nil, "")
				} else {
					service.executeWorkflowAsyncWithOptions(context.Background(), workflow, id, nil, nil, nil, nil, nil, nil, nil, "", "", "", false, nil, "", nil)
				}
				if len(sink.closed) != 1 || sink.closed[0] != id {
					t.Fatalf("sink closed for %v; wanted exactly %s", sink.closed, id)
				}
				want := outcome
				if outcome == "compile-failed" {
					want = "failed"
				}
				if repo.execution.Status != want {
					t.Fatalf("execution status = %s, want %s", repo.execution.Status, want)
				}
			})
		}
	}
}

func TestRequiredVideoArtifactContract(t *testing.T) {
	if err := requiredVideoArtifactError(true, nil, nil); err == nil {
		t.Fatal("required video with no artifact must fail")
	}
	if err := requiredVideoArtifactError(true, []ExecutionVideoArtifact{{ArtifactID: "video-1"}}, nil); err != nil {
		t.Fatalf("required video with an artifact failed: %v", err)
	}
	want := errors.New("artifact lookup failed")
	if err := requiredVideoArtifactError(true, nil, want); !errors.Is(err, want) {
		t.Fatalf("artifact lookup error = %v, want wrapped %v", err, want)
	}
	if err := requiredVideoArtifactError(false, nil, nil); err != nil {
		t.Fatalf("optional video must not fail: %v", err)
	}
}

func TestListExecutionArtifactsDoesNotExposeCapturePath(t *testing.T) {
	// enforces invariant: protectedEvidenceHasNoPublicLocation
	root := t.TempDir()
	executionID := uuid.New()
	dir := filepath.Join(root, executionID.String(), "artifacts", "har")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capture.har"), []byte(`{"log":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	service := &WorkflowService{executionDataRoot: root}
	artifacts, err := service.listExecutionArtifacts(context.Background(), executionID, "har")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %d", len(artifacts))
	}
	artifact := artifacts[0]
	if _, ok := artifact.Payload["path"]; ok {
		t.Fatalf("payload leaked local path: %#v", artifact.Payload)
	}
	if artifact.StorageURL != "" || artifact.Payload["access_policy"] != "ACCESS_POLICY_PROTECTED_STORAGE_ONLY" || artifact.Payload["sha256"] == "" {
		t.Fatalf("artifact evidence metadata = %#v", artifact)
	}
}

func TestExecutionOutcomeDistinguishesCancellationFromFailure(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{"success", context.Background(), nil, "completed"},
		{"completion wins stop race", cancelled, nil, "completed"},
		{"typed cancellation", context.Background(), context.Canceled, "cancelled"},
		{"driver error after cancellation", cancelled, errors.New("browser request interrupted"), "cancelled"},
		{"deadline is failure", context.Background(), context.DeadlineExceeded, "failed"},
		{"cancel text is not cancellation", context.Background(), errors.New("cancel button selector missing"), "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, detail := executionOutcome(tc.ctx, tc.err)
			if got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
			if tc.want == "completed" && detail != "" {
				t.Fatalf("success has error %q", detail)
			}
			if tc.want == "failed" && detail != tc.err.Error() {
				t.Fatalf("failure detail lost: %q", detail)
			}
		})
	}
}

func TestSynchronousExecutionReportsEarlyRunnerReadFailure(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				called := false
				_, repo, invoke := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { called = true; return nil })
				want := errors.New("execution index unavailable")
				repo.readErr = want
				start := time.Now()
				got := invoke(context.Background(), true)
				if !errors.Is(got.err, want) || called || time.Since(start) != 0 {
					t.Fatalf("early runner failure = %+v, executor called=%t, elapsed=%s", got, called, time.Since(start))
				}
			})
		})
	}
}

func TestAsynchronousExecutionStillReturnsBeforeRunnerFinishes(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				var executionErr error
				_, repo, invoke := newCompletionFixture(t, mode, func(ctx context.Context, _ executor.Request) error { <-release; executionErr = ctx.Err(); return nil })
				ctx, cancel := context.WithCancel(context.Background())
				start := time.Now()
				got := invoke(ctx, false)
				cancel()
				synctest.Wait()
				wantStatus := basbase.ExecutionStatus_EXECUTION_STATUS_PENDING
				if mode == "adhoc" {
					wantStatus = basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
				}
				if got.err != nil || got.completed || got.status != wantStatus || time.Since(start) != 0 {
					t.Errorf("asynchronous admission = %+v, elapsed=%s", got, time.Since(start))
				}
				close(release)
				synctest.Wait()
				if executionErr != nil || repo.index.CompletedAt == nil {
					t.Errorf("asynchronous runner failed: context=%v status=%s", executionErr, repo.index.Status)
				}
			})
		})
	}
}
