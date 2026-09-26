package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func TestWorkflowPersistedTimelineReadersRedactLegacySensitiveValues(t *testing.T) {
	const secret = "BAS_SYNTHETIC_WORKFLOW_REPLAY_SECRET_017"
	executionID := uuid.New()
	resultPath := filepath.Join(t.TempDir(), "result.json")
	dir := filepath.Dir(resultPath)
	entry := `{"id":"legacy-password","action":{"type":"ACTION_TYPE_INPUT","input":{"selector":"#password","value":"` + secret + `"},"metadata":{"elementSnapshot":{"tagName":"input","innerText":"` + secret + `","attributes":{"type":"password","value":"` + secret + `"}}}}}`
	timelineBytes := []byte(`{"entries":[` + entry + `]}`)
	packageBytes := []byte(`{"timeline":[` + entry + `]}`)
	require.NoError(t, os.WriteFile(resultPath, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "timeline.proto.json"), timelineBytes, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.proto.json"), packageBytes, 0o600))

	service := &WorkflowService{repo: &lifecycleRepository{execution: &database.ExecutionIndex{
		ID: executionID, WorkflowID: uuid.New(), ResultPath: resultPath,
	}}}
	timeline, err := service.GetExecutionTimelineProto(context.Background(), executionID)
	require.NoError(t, err)
	pack, err := service.GetExecutionReplayPackage(context.Background(), executionID)
	require.NoError(t, err)
	for name, message := range map[string]proto.Message{
		"execution timeline": timeline,
		"replay package":     pack,
	} {
		encoded, marshalErr := protojson.Marshal(message)
		require.NoError(t, marshalErr)
		assert.NotContains(t, string(encoded), secret, "%s returned a legacy secret", name)
	}
	for _, name := range []string{"timeline.proto.json", "evidence.proto.json"} {
		stored, readErr := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, readErr)
		assert.Contains(t, string(stored), secret, "reader must not rewrite stored user data")
	}
}

type lifecycleSink struct {
	*events.MemorySink
	closed    []uuid.UUID
	published []contracts.EventEnvelope
}

func (s *lifecycleSink) Publish(ctx context.Context, event contracts.EventEnvelope) error {
	s.published = append(s.published, event)
	return s.MemorySink.Publish(ctx, event)
}

func (s *lifecycleSink) CloseExecution(id uuid.UUID) {
	s.closed = append(s.closed, id)
	s.MemorySink.CloseExecution(id)
}

type lifecycleExecutor struct{ err error }

func (e lifecycleExecutor) Execute(context.Context, executor.Request) error { return e.err }

type completionRepository struct {
	database.Repository
	mu              sync.Mutex
	index           database.ExecutionIndex
	workflow        *database.WorkflowIndex
	project         *database.ProjectIndex
	reads           int
	failTerminal    bool
	failRunning     bool
	readErr         error
	createErr       error
	requireTestMode bool
	wrongReads      int
	wrongWrites     int
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
	if r.createErr != nil {
		return r.createErr
	}
	r.index = *index
	return nil
}

func (r *completionRepository) GetExecution(ctx context.Context, _ uuid.UUID) (*database.ExecutionIndex, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.requireTestMode && !coredb.IsTestMode(ctx) {
		r.wrongReads++
		return nil, errors.New("execution read escaped routed context")
	}
	if r.readErr != nil {
		return nil, r.readErr
	}
	copy := r.index
	return &copy, nil
}

func (r *completionRepository) UpdateExecutionStatus(ctx context.Context, _ uuid.UUID, status string, detail *string, completed *time.Time, updated time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.requireTestMode && (!coredb.IsTestMode(ctx) || ctx.Err() != nil) {
		r.wrongWrites++
		return errors.New("execution write escaped live routed context")
	}
	if completed != nil && r.failTerminal {
		return errors.New("terminal index write failed")
	}
	if status == database.ExecutionStatusRunning && r.failRunning {
		return errors.New("running index write failed")
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
	if err := persistWorkflowVersionSnapshot(project, summary); err != nil {
		t.Fatal(err)
	}
	repo := &completionRepository{workflow: &database.WorkflowIndex{ID: id, ProjectID: &project.ID, FilePath: relative, Version: 1}, project: project}
	service := &WorkflowService{repo: repo, executor: run, executionDataRoot: t.TempDir()}
	invoke := func(ctx context.Context, wait bool) completionResult {
		if mode == "manual" {
			response, err := service.ExecuteWorkflow(ctx, id, map[string]any{"manual-value": "preserved"})
			if err != nil {
				return completionResult{err: err}
			}
			return completionResult{enums.StringToExecutionStatus(response.Status), response.CompletedAt != nil, response.ErrorMessage, nil}
		}
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

// [REQ:BAS-RH-J07] Effects and terminal notifications follow durable status writes.
func TestExecutionStatusPersistenceOwnsEffectsAndNotifications(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		for _, stage := range []string{"running", "terminal", "healthy"} {
			t.Run(mode+"/"+stage, func(t *testing.T) {
				effects := 0
				service, repo, invoke := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { effects++; return nil })
				repo.failRunning, repo.failTerminal = stage == "running", stage == "terminal"
				sink := &lifecycleSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
				service.eventSinkFactory = func() events.Sink { return sink }
				got := invoke(context.Background(), true)
				wantEffects := 1
				if stage == "running" {
					wantEffects = 0
				}
				if effects != wantEffects {
					t.Errorf("executor effects = %d, want %d", effects, wantEffects)
				}
				if stage == "terminal" {
					if got.err == nil {
						t.Error("failed terminal persistence returned no error")
					}
					if len(sink.published) != 0 {
						t.Errorf("broadcast %d terminal receipts without a durable terminal state", len(sink.published))
					}
					return
				}
				wantStatus, wantKind := basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, contracts.EventKindExecutionCompleted
				if stage == "running" {
					wantStatus, wantKind = basbase.ExecutionStatus_EXECUTION_STATUS_FAILED, contracts.EventKindExecutionFailed
				}
				if got.err != nil || got.status != wantStatus || !got.completed {
					t.Errorf("result = %+v, want durable %s", got, wantStatus)
				}
				if len(sink.published) != 1 || sink.published[0].Kind != wantKind {
					t.Errorf("events = %+v, want one %s", sink.published, wantKind)
				}
			})
		}
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

func TestStopExecutionWaitsForRunnerCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		releaseCleanup := make(chan struct{})
		entered := make(chan struct{})
		service, repo, invoke := newCompletionFixture(t, "saved", func(ctx context.Context, _ executor.Request) error {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		})
		service.eventSinkFactory = func() events.Sink {
			return &delayedCompletionSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits), release: releaseCleanup}
		}
		if got := invoke(context.Background(), false); got.err != nil || got.completed {
			t.Fatalf("admit async workflow: %+v", got)
		}
		synctest.Wait()
		select {
		case <-entered:
		default:
			t.Fatal("runner did not enter the action")
		}

		stopResult := make(chan error, 1)
		go func() { stopResult <- service.StopExecution(context.Background(), repo.index.ID) }()
		synctest.Wait()
		if repo.index.Status != database.ExecutionStatusCancelled || repo.index.CompletedAt == nil {
			t.Fatalf("runner did not persist cancellation before cleanup gate: status=%s completed=%v", repo.index.Status, repo.index.CompletedAt)
		}
		select {
		case err := <-stopResult:
			t.Fatalf("StopExecution returned while owned sink cleanup was pending: %v", err)
		default:
		}

		close(releaseCleanup)
		synctest.Wait()
		if err := <-stopResult; err != nil {
			t.Fatalf("StopExecution after cleanup: %v", err)
		}
	})
}

type retainedWorkflowEventSink struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func (s *retainedWorkflowEventSink) Publish(_ context.Context, event contracts.EventEnvelope) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (*retainedWorkflowEventSink) Limits() contracts.EventBufferLimits {
	return contracts.DefaultEventBufferLimits
}

func (*retainedWorkflowEventSink) CloseExecution(uuid.UUID) {}

func (s *retainedWorkflowEventSink) snapshot() []contracts.EventEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]contracts.EventEnvelope(nil), s.events...)
}

// [REQ:BAS-RH-J07] A real Go workflow cancellation retains an uncertain browser
// outcome, closes the same execution lease and waits for driver resource cleanup.
func TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose(t *testing.T) {
	t.Log("[REQ:BAS-RH-J07] canceled external effect remains uncertain and StopExecution joins leased-driver cleanup")
	runEntered := make(chan struct{})
	runCanceled := make(chan struct{})
	closeEntered := make(chan struct{})
	releaseClose := make(chan struct{})
	closedLease := make(chan struct {
		executionID string
		leaseID     string
	}, 1)
	var startedExecution atomic.Value
	var effects atomic.Int32
	var liveSessions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session/start":
			var request struct {
				ExecutionID string `json:"execution_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode session start: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			startedExecution.Store(request.ExecutionID)
			liveSessions.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session_id": "cancel-session", "lease_id": "lease-owner-1", "active_page_id": "page-1",
			})
		case r.URL.Path == "/session/cancel-session/run":
			var request struct {
				ExecutionID string `json:"execution_id"`
				LeaseID     string `json:"lease_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode instruction: %v", err)
				return
			}
			if request.ExecutionID != startedExecution.Load().(string) || request.LeaseID != "lease-owner-1" {
				t.Errorf("instruction owner = %q/%q, want %q/lease-owner-1", request.ExecutionID, request.LeaseID, startedExecution.Load())
			}
			effects.Add(1) // The external effect occurs before its response is lost.
			close(runEntered)
			<-r.Context().Done()
			close(runCanceled)
		case r.URL.Path == "/session/cancel-session/close":
			var request struct {
				ExecutionID string `json:"execution_id"`
				LeaseID     string `json:"lease_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode session close: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			closedLease <- struct {
				executionID string
				leaseID     string
			}{request.ExecutionID, request.LeaseID}
			close(closeEntered)
			<-releaseClose
			liveSessions.Add(-1)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			t.Errorf("unexpected driver request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	playwright, err := engine.NewPlaywrightEngineWithHTTPClient(server.URL, server.Client(), logrus.New())
	if err != nil {
		t.Fatalf("create Playwright engine: %v", err)
	}
	sink := &retainedWorkflowEventSink{}
	service, repo, invoke := newCompletionFixture(t, "saved", func(ctx context.Context, request executor.Request) error {
		request.EngineName = playwright.Name()
		request.EngineFactory = engine.NewStaticFactory(playwright)
		request.EngineCaps = &contracts.EngineCapabilities{
			SchemaVersion: contracts.CapabilitiesSchemaVersion,
			Engine:        "playwright", MaxConcurrentSessions: 1,
			SupportsHAR: true, SupportsVideo: true, SupportsTracing: true,
		}
		return executor.NewSimpleExecutor(nil).Execute(ctx, request)
	})
	service.eventSinkFactory = func() events.Sink { return sink }
	service.artifactRecorder = executionwriter.NewFileWriter(nil, nil, nil, executionwriter.NewStaticRoot(t.TempDir()))
	if got := invoke(context.Background(), false); got.err != nil || got.completed {
		t.Fatalf("admit async workflow: %+v", got)
	}
	select {
	case <-runEntered:
	case <-time.After(2 * time.Second):
		t.Fatalf("workflow never dispatched the browser instruction: status=%s error=%q events=%+v", repo.index.Status, repo.index.ErrorMessage, sink.snapshot())
	}
	executionID, _ := startedExecution.Load().(string)
	if executionID == "" {
		t.Fatal("driver session start did not retain execution ownership")
	}

	stopStarted := time.Now()
	stopResult := make(chan error, 1)
	go func() { stopResult <- service.StopExecution(context.Background(), repo.index.ID) }()
	select {
	case <-runCanceled:
	case <-time.After(time.Second):
		t.Fatal("accepted cancellation did not interrupt the live instruction within1s")
	}
	inputStoppedAt := time.Now()
	select {
	case <-closeEntered:
	case <-time.After(time.Second):
		t.Fatal("owned session close did not start within1s of StopExecution")
	}
	if time.Since(stopStarted) > time.Second {
		t.Fatalf("StopExecution began driver teardown after %s", time.Since(stopStarted))
	}
	closed := <-closedLease
	if closed.executionID != executionID || closed.leaseID != "lease-owner-1" {
		t.Fatalf("closed lease = %q/%q, want %q/lease-owner-1", closed.executionID, closed.leaseID, executionID)
	}
	if effects.Load() != 1 || liveSessions.Load() != 1 {
		t.Fatalf("before close acknowledgement: effects=%d live sessions=%d, want1/1", effects.Load(), liveSessions.Load())
	}
	resourcesBeforeClose := liveSessions.Load()
	select {
	case err := <-stopResult:
		t.Fatalf("StopExecution returned before driver close acknowledged resource cleanup: %v", err)
	default:
	}

	closeReleasedAt := time.Now()
	close(releaseClose)
	select {
	case err := <-stopResult:
		if err != nil {
			t.Fatalf("StopExecution after driver cleanup: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("terminal cleanup did not finish within5s after driver close was released")
	}
	if repo.index.Status != database.ExecutionStatusCancelled || repo.index.CompletedAt == nil {
		t.Fatalf("terminal receipt = status %s completed %v, want cancelled with completion time", repo.index.Status, repo.index.CompletedAt)
	}
	if effects.Load() != 1 || liveSessions.Load() != 0 {
		t.Fatalf("after terminal cleanup: effects=%d live sessions=%d, want1/0", effects.Load(), liveSessions.Load())
	}
	var uncertain *contracts.StepOutcome
	for _, event := range sink.snapshot() {
		if event.Kind != contracts.EventKindStepFailed {
			continue
		}
		payload, marshalErr := json.Marshal(event.Payload)
		if marshalErr != nil {
			t.Fatalf("marshal failure event: %v", marshalErr)
		}
		var recorded struct {
			Outcome contracts.StepOutcome `json:"outcome"`
		}
		if err := json.Unmarshal(payload, &recorded); err != nil {
			t.Fatalf("decode failure event: %v", err)
		}
		uncertain = &recorded.Outcome
	}
	if uncertain == nil || uncertain.Failure == nil || uncertain.Failure.Code != contracts.FailureCodeInstructionOutcomeUncertain || uncertain.Failure.Retryable {
		t.Fatalf("last step evidence = %+v, want a retained non-retryable uncertain outcome", uncertain)
	}
	inputStoppedMS := float64(inputStoppedAt.Sub(stopStarted).Microseconds()) / 1000
	cleanupMS := float64(time.Since(closeReleasedAt).Microseconds()) / 1000
	if inputStoppedMS > 1000 || cleanupMS > 5000 {
		t.Fatalf("cancellation timings outside J07 band: inputStopped=%.2fms cleanup=%.2fms", inputStoppedMS, cleanupMS)
	}
	if err := cancellationqualification.RecordObservation(t.Name(), "cancellation", cancellationqualification.CaseObservation{
		Observed: true, Passed: true, ExternalEffects: int(effects.Load()), TerminalStatus: "cancelled",
		LiveResourcesBeforeClose: int(resourcesBeforeClose), LiveResourcesAfterClose: int(liveSessions.Load()),
		InputStoppedMS: inputStoppedMS, CleanupMS: cleanupMS, RecoveryMS: 0,
		UncertainEffect: uncertain.Failure.Code == contracts.FailureCodeInstructionOutcomeUncertain,
		RetryAdmitted:   uncertain.Failure.Retryable,
	}); err != nil {
		t.Fatalf("record J07 cancellation observation: %v", err)
	}
}

func TestStopExecutionWaitRespectsCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service := &WorkflowService{}
		executionID := uuid.New()
		done := make(chan struct{})
		cancelled := make(chan struct{})
		service.storeExecutionCancel(executionID, func() { close(cancelled) }, done)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stopResult := make(chan error, 1)
		go func() { stopResult <- service.StopExecution(ctx, executionID) }()
		synctest.Wait()
		select {
		case <-cancelled:
		default:
			t.Fatal("StopExecution did not signal runner cancellation")
		}
		cancel()
		synctest.Wait()
		if err := <-stopResult; !errors.Is(err, context.Canceled) {
			t.Fatalf("StopExecution error after caller cancellation = %v, want context.Canceled", err)
		}
		close(done)
	})
}

func TestStopExecutionWithoutLocalOwnerUsesDurableStatus(t *testing.T) {
	for _, tc := range []struct {
		name      string
		execution *database.ExecutionIndex
		wantError bool
	}{
		{name: "running", execution: &database.ExecutionIndex{Status: database.ExecutionStatusRunning}, wantError: true},
		{name: "pending", execution: &database.ExecutionIndex{Status: database.ExecutionStatusPending}, wantError: true},
		{name: "terminal", execution: &database.ExecutionIndex{Status: database.ExecutionStatusCompleted}},
		{name: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("[REQ:BAS-RH-J07] stop with no local owner checks durable status: %s", tc.name)
			executionID := uuid.New()
			if tc.execution != nil {
				tc.execution.ID = executionID
			}
			service := &WorkflowService{repo: &lifecycleRepository{execution: tc.execution}}
			err := service.StopExecution(context.Background(), executionID)
			if (err != nil) != tc.wantError {
				t.Fatalf("StopExecution error = %v, wantError=%t", err, tc.wantError)
			}
		})
	}
}

// [REQ:BAS-RH-J07] The workflow owner retires decorated sinks on all exits.
func TestWorkflowClosesDecoratedSinkOnEveryExit(t *testing.T) {
	for _, outcome := range []string{"completed", "failed", "cancelled", "compile-failed"} {
		t.Run(outcome, func(t *testing.T) {
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
			service.executeWorkflowAsyncWithOptions(context.Background(), workflow, id, nil, nil, nil, nil, nil, nil, "", nil, "", "", "", false, nil, "", nil)

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
			if len(sink.published) != 1 || sink.published[0].Payload.(map[string]any)["status"] != want {
				t.Fatalf("terminal events = %+v, want one committed %s", sink.published, want)
			}
		})
	}
}

func TestExecutionPanicDoesNotPublishSuccess(t *testing.T) {
	service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { panic("executor panic") })
	id := uuid.New()
	repo.index = database.ExecutionIndex{ID: id, WorkflowID: repo.workflow.ID, Status: database.ExecutionStatusPending}
	sink := &lifecycleSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
	service.eventSinkFactory = func() events.Sink { return sink }
	response, err := service.GetWorkflowAPI(context.Background(), &basapi.GetWorkflowRequest{WorkflowId: repo.workflow.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() != "executor panic" {
				t.Error("executor panic was swallowed or replaced")
			}
		}()
		service.executeWorkflowAsyncWithOptions(context.Background(), response.Workflow, id, nil, nil, nil, nil, nil, nil, "", nil, repo.project.FolderPath, "", "", false, nil, "", nil)
	}()
	if repo.index.Status != database.ExecutionStatusFailed || len(sink.published) != 1 || sink.published[0].Kind != contracts.EventKindExecutionFailed {
		t.Fatalf("panic produced status=%s events=%+v", repo.index.Status, sink.published)
	}
	if len(sink.closed) != 1 {
		t.Fatalf("panic left sink open: %+v", sink.closed)
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
	for _, mode := range []string{"saved", "adhoc", "manual"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				var executionErr error
				var observedStore map[string]any
				_, repo, invoke := newCompletionFixture(t, mode, func(ctx context.Context, req executor.Request) error {
					<-release
					observedStore = req.InitialStore
					executionErr = ctx.Err()
					return nil
				})
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
				if mode == "manual" && observedStore["manual-value"] != "preserved" {
					t.Errorf("manual parameters lost from initial store: %v", observedStore)
				}
			})
		})
	}
}

func invokeMetadataFixture(ctx context.Context, service *WorkflowService, repo *completionRepository, mode string, parameters *basexecution.ExecutionParameters) error {
	switch mode {
	case "manual":
		_, err := service.ExecuteWorkflow(ctx, repo.workflow.ID, jsonValueMapToAnyMap(parameters.InitialStore))
		return err
	case "adhoc":
		workflow, err := service.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: repo.workflow.ID.String()})
		if err != nil {
			return err
		}
		_, err = service.ExecuteAdhocWorkflowAPI(ctx, &basexecution.ExecuteAdhocRequest{FlowDefinition: workflow.Workflow.FlowDefinition, Parameters: parameters})
		return err
	default:
		_, err := service.ExecuteWorkflowAPI(ctx, &basapi.ExecuteWorkflowRequest{WorkflowId: repo.workflow.ID.String(), Parameters: parameters})
		return err
	}
}

// [REQ:BAS-RH-J07] Recoverable inputs and the workflow version survive both
// running and terminal publication. The DB index owns changing lifecycle data.
func TestExecutionPreservesAdmissionMetadata(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc", "manual"} {
		for _, failed := range []bool{false, true} {
			name := mode + "/completed"
			if failed {
				name = mode + "/failed"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					release := make(chan struct{})
					service, repo, _ := newCompletionFixture(t, mode, func(context.Context, executor.Request) error {
						<-release
						if failed {
							return errors.New("fixture action failed")
						}
						return nil
					})
					parameters := &basexecution.ExecutionParameters{InitialStore: convertParamsToProto(map[string]any{"identity": "original"})}
					if mode != "manual" {
						parameters.InitialParams = convertParamsToProto(map[string]any{"input": "preserved"})
						parameters.Env = convertParamsToProto(map[string]any{"fixture": "local"})
					}
					if err := invokeMetadataFixture(context.Background(), service, repo, mode, parameters); err != nil {
						t.Fatal(err)
					}
					admittedBytes, err := os.ReadFile(service.executionSnapshotPath(repo.index.ID))
					if err != nil {
						t.Fatal(err)
					}
					synctest.Wait()
					check := func(stage string, status basbase.ExecutionStatus) {
						got, err := service.HydrateExecutionProto(context.Background(), &repo.index)
						if err != nil {
							t.Errorf("%s hydration: %v", stage, err)
							return
						}
						wantTrigger := basbase.TriggerType_TRIGGER_TYPE_API
						if mode == "manual" {
							wantTrigger = basbase.TriggerType_TRIGGER_TYPE_MANUAL
						}
						if got.Status != status || got.WorkflowVersion != 1 || got.TriggerType != wantTrigger || !proto.Equal(got.Parameters, parameters) {
							t.Errorf("%s lost admitted metadata: status=%s version=%d trigger=%s parameters=%v", stage, got.Status, got.WorkflowVersion, got.TriggerType, got.Parameters)
						}
						checkpoint, err := service.ExtractCheckpointState(context.Background(), repo.index.ID)
						if err != nil {
							t.Errorf("%s checkpoint: %v", stage, err)
							return
						}
						if checkpoint.WorkflowVersion != 1 || !reflect.DeepEqual(checkpoint.Variables, jsonValueMapToAnyMap(parameters.InitialStore)) || !reflect.DeepEqual(checkpoint.Params, jsonValueMapToAnyMap(parameters.InitialParams)) || !reflect.DeepEqual(checkpoint.Env, jsonValueMapToAnyMap(parameters.Env)) {
							t.Errorf("%s checkpoint lost admitted inputs/version: %+v", stage, checkpoint)
						}
					}
					check("running", basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING)
					close(release)
					synctest.Wait()
					status := basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
					if failed {
						status = basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
					}
					check("terminal", status)
					terminalBytes, err := os.ReadFile(service.executionSnapshotPath(repo.index.ID))
					if err != nil || string(terminalBytes) != string(admittedBytes) {
						t.Errorf("lifecycle changed immutable admission bytes: %v", err)
					}
				})
			})
		}
	}
}

func TestExecutionRejectsUnavailableAdmissionStorageBeforeEffects(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc", "manual"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				called := false
				service, repo, _ := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { called = true; return nil })
				root := filepath.Join(t.TempDir(), "unavailable")
				if err := os.WriteFile(root, []byte("preserve this file"), 0o600); err != nil {
					t.Fatal(err)
				}
				service.executionDataRoot = root
				err := invokeMetadataFixture(context.Background(), service, repo, mode, &basexecution.ExecutionParameters{})
				synctest.Wait()
				if err == nil || called || repo.index.ID != uuid.Nil {
					t.Errorf("unrecoverable admission: error=%v, runner called=%t, indexed=%s", err, called, repo.index.ID)
				}
				content, readErr := os.ReadFile(root)
				if readErr != nil || string(content) != "preserve this file" {
					t.Errorf("unrelated storage changed: %q, %v", content, readErr)
				}
			})
		})
	}
}

func prepareResumableMetadataFixture(t *testing.T, service *WorkflowService, repo *completionRepository, parameters *basexecution.ExecutionParameters) uuid.UUID {
	t.Helper()
	response, err := service.ExecuteWorkflowAPI(context.Background(), &basapi.ExecuteWorkflowRequest{
		WorkflowId: repo.workflow.ID.String(), Parameters: parameters, WaitForCompletion: true,
	})
	if err != nil || response.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_FAILED {
		t.Fatalf("prepare failed original execution: %v, %v", response, err)
	}
	id := repo.index.ID
	path := filepath.Join(service.executionDataRoot, id.String(), "timeline.proto.json")
	// An independent successful first step makes this failed execution resumable.
	if err := os.WriteFile(path, []byte(`{"entries":[{"step_index":0,"node_id":"navigate","context":{"success":true}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.index.ResultPath = filepath.Join(filepath.Dir(path), "result.json")
	writer := executionwriter.NewFileWriter(nil, nil, nil, executionwriter.NewStaticRoot(service.executionDataRoot))
	if err := writer.RecordCheckpoint(context.Background(), executionwriter.Checkpoint{SchemaVersion: executionwriter.CheckpointVersion, ExecutionID: id, WorkflowID: repo.workflow.ID, LastStepIndex: 0, NodeID: "navigate", TotalSteps: 1, Store: jsonValueMapToAnyMap(parameters.InitialStore)}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestResumePreservesAdmissionMetadataAndVersionGuard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return errors.New("second step failed") })
		parameters := &basexecution.ExecutionParameters{
			StartUrl:        proto.String("https://fixture.invalid/original"),
			ArtifactConfig:  &basexecution.ArtifactCollectionConfig{Profile: proto.String("none")},
			BrowserProfile:  &basbase.BrowserProfile{ExtraHeaders: map[string]string{"X-Fixture": "preserved"}},
			ContinueOnError: proto.Bool(false),
			InitialStore:    convertParamsToProto(map[string]any{"identity": "original"}),
			InitialParams:   convertParamsToProto(map[string]any{"input": "original"}),
			Env:             convertParamsToProto(map[string]any{"fixture": "local"}),
		}
		original := prepareResumableMetadataFixture(t, service, repo, parameters)
		originalBytes, err := os.ReadFile(service.executionSnapshotPath(original))
		if err != nil {
			t.Fatal(err)
		}
		repo.workflow.Version = 2
		if err := service.ValidateResumable(context.Background(), original); !errors.Is(err, ErrWorkflowChanged) {
			t.Errorf("changed workflow accepted for resume: %v", err)
		}
		repo.workflow.Version = 1
		service.executor = completionExecutor(func(_ context.Context, request executor.Request) error {
			if request.ResumedFromID == nil || *request.ResumedFromID != original || request.InitialStore["identity"] != "original" || request.InitialParams["input"] != "updated" || request.Env["fixture"] != "local" {
				t.Errorf("resume request lost admitted inputs or origin")
			}
			if request.ResumeAfterStep == nil || *request.ResumeAfterStep != 0 || request.StartURL != "https://fixture.invalid/resume" || request.BrowserProfile == nil || request.BrowserProfile.ExtraHeaders["X-Fixture"] != "preserved" || request.ArtifactConfig == nil || request.ArtifactConfig.CollectScreenshots || request.ContinueOnError == nil || *request.ContinueOnError {
				t.Errorf("resume request lost checkpoint or execution settings: %+v", request)
			}
			return nil
		})
		resumed, err := service.ResumeExecution(context.Background(), original, map[string]any{"input": "updated", "resume_url": "https://fixture.invalid/resume"})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		got, err := service.HydrateExecutionProto(context.Background(), &repo.index)
		parameters.StartUrl = proto.String("https://fixture.invalid/resume")
		parameters.InitialParams = convertParamsToProto(map[string]any{"input": "updated"})
		if err != nil || got.GetWorkflowVersion() != 1 || got.GetTriggerType() != basbase.TriggerType_TRIGGER_TYPE_RESUME || got.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || !proto.Equal(got.GetParameters(), parameters) || resumed.ID == original || got.GetResumedFrom() != original.String() {
			t.Errorf("resumed metadata/status = %v, %v", got, err)
		}
		afterBytes, err := os.ReadFile(service.executionSnapshotPath(original))
		if err != nil || string(afterBytes) != string(originalBytes) {
			t.Errorf("resume mutated original receipt: %v", err)
		}
	})
}

func TestExecutionIndexFailurePreservesAdmissionReceiptWithoutEffects(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc", "manual", "resumed"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, repo, _ := newCompletionFixture(t, mode, func(context.Context, executor.Request) error { return errors.New("second step failed") })
				parameters := &basexecution.ExecutionParameters{InitialStore: convertParamsToProto(map[string]any{"identity": "preserved"})}
				original := uuid.Nil
				if mode == "resumed" {
					original = prepareResumableMetadataFixture(t, service, repo, parameters)
				}
				called := false
				service.executor = completionExecutor(func(context.Context, executor.Request) error { called = true; return nil })
				want := errors.New("index commit outcome uncertain")
				repo.createErr = want
				var err error
				if mode == "resumed" {
					_, err = service.ResumeExecution(context.Background(), original, nil)
				} else {
					err = invokeMetadataFixture(context.Background(), service, repo, mode, parameters)
				}
				synctest.Wait()
				if !errors.Is(err, want) || called {
					t.Fatalf("index failure: %v, runner called=%t", err, called)
				}
				paths, globErr := filepath.Glob(filepath.Join(service.executionDataRoot, "*", executionSnapshotFileName))
				if globErr != nil {
					t.Fatal(globErr)
				}
				attempts := 0
				for _, path := range paths {
					if filepath.Base(filepath.Dir(path)) == original.String() {
						continue
					}
					attempts++
					raw, readErr := os.ReadFile(path)
					var receipt basexecution.Execution
					if readErr != nil || protojson.Unmarshal(raw, &receipt) != nil || !proto.Equal(receipt.Parameters, parameters) || receipt.WorkflowVersion != 1 || !strings.Contains(err.Error(), receipt.ExecutionId) {
						t.Errorf("failed admission lost its identifiable recovery receipt: %v", readErr)
					}
					info, statErr := os.Stat(path)
					if statErr != nil || info.Mode().Perm() != 0o600 {
						t.Errorf("recovery inputs must be owner-readable only: %v", statErr)
					}
				}
				if attempts != 1 {
					t.Errorf("retained admission receipts = %d, want one", attempts)
				}
			})
		})
	}
}

func (r *completionRepository) GetProjectByName(context.Context, string) (*database.ProjectIndex, error) {
	return nil, errors.New("seed fixture omitted by context-boundary test")
}

// [REQ:BAS-RH-J07] Request cancellation detaches, explicit stop cancels,
// and both paths publish terminal state through the original storage route.
func TestResumeRetainsRoutedContextAfterAdmission(t *testing.T) {
	for _, stop := range []bool{false, true} {
		name := "completed"
		if stop {
			name = "stopped"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return errors.New("second step failed") })
				original := prepareResumableMetadataFixture(t, service, repo, &basexecution.ExecutionParameters{})
				repo.requireTestMode = true
				release := make(chan struct{})
				called := false
				service.executor = completionExecutor(func(ctx context.Context, req executor.Request) error {
					called = true
					if !coredb.IsTestMode(ctx) || ctx.Err() != nil {
						t.Errorf("resumed context lost route or retained request cancellation: %v", ctx.Err())
					}
					if req.BrowserProfile == nil || req.BrowserProfile.ExtraHeaders["X-Vrooli-Test-Mode"] != "1" {
						t.Error("resumed browser request lost routed test header")
					}
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})
				ctx, cancel := context.WithCancel(coredb.WithTestMode(context.Background()))
				resumed, err := service.ResumeExecution(ctx, original, nil)
				cancel()
				if err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				if !called || repo.index.CompletedAt != nil {
					t.Fatalf("caller cancellation stopped admitted execution: called=%t completed=%v", called, repo.index.CompletedAt)
				}
				want := database.ExecutionStatusCompleted
				if stop {
					want = database.ExecutionStatusCancelled
					stopDone := make(chan error, 1)
					go func() { stopDone <- service.StopExecution(coredb.WithTestMode(context.Background()), resumed.ID) }()
					synctest.Wait()
					if err := <-stopDone; err != nil {
						t.Fatal(err)
					}
				} else {
					close(release)
				}
				synctest.Wait()
				if repo.wrongReads != 0 || repo.wrongWrites != 0 || repo.index.CompletedAt == nil || repo.index.Status != want {
					t.Errorf("resumed terminal state escaped route or cancellation: reads=%d writes=%d completed=%v status=%s want=%s", repo.wrongReads, repo.wrongWrites, repo.index.CompletedAt, repo.index.Status, want)
				}
			})
		})
	}
}

func TestResumeRejectsMissingWorkflowRevisionBeforeAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return errors.New("second step failed") })
		original := prepareResumableMetadataFixture(t, service, repo, &basexecution.ExecutionParameters{})
		snapshot, err := service.HydrateExecutionProto(context.Background(), &repo.index)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.WorkflowVersion = 0
		raw, err := protojson.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(service.executionSnapshotPath(original), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		called := false
		service.executor = completionExecutor(func(context.Context, executor.Request) error { called = true; return nil })
		_, err = service.ResumeExecution(context.Background(), original, nil)
		synctest.Wait()
		if !errors.Is(err, ErrExecutionNotResumable) || called || repo.index.ID != original {
			t.Errorf("resume guessed missing revision: error=%v called=%t index=%s original=%s", err, called, repo.index.ID, original)
		}
	})
}

// [REQ:BAS-RH-J07] Resume uses committed state, and rejects unusable recovery
// evidence before admitting a replacement execution.
func TestResumeRequiresMatchingCommittedStore(t *testing.T) {
	for _, fault := range []string{"none", "missing", "malformed", "version", "execution", "workflow", "cursor", "node", "store"} {
		t.Run(fault, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return errors.New("later step failed") })
				original := prepareResumableMetadataFixture(t, service, repo, &basexecution.ExecutionParameters{InitialStore: convertParamsToProto(map[string]any{"value": "original"})})
				path := filepath.Join(service.executionDataRoot, original.String(), executionwriter.CheckpointFileName)
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var cp map[string]any
				if err := json.Unmarshal(raw, &cp); err != nil {
					t.Fatal(err)
				}
				cp["store"] = map[string]any{"value": "committed", "nested": map[string]any{"value": "preserved"}}
				switch fault {
				case "version":
					cp["schema_version"] = 999
				case "execution":
					cp["execution_id"] = uuid.NewString()
				case "workflow":
					cp["workflow_id"] = uuid.NewString()
				case "cursor":
					cp["last_step_index"] = 7
				case "node":
					cp["node_id"] = "wrong-node"
				case "store":
					delete(cp, "store")
				}
				raw, err = json.Marshal(cp)
				if err != nil {
					t.Fatal(err)
				}
				if fault == "malformed" {
					raw = []byte("{incomplete")
				}
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				if fault == "missing" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
				called := false
				service.executor = completionExecutor(func(_ context.Context, req executor.Request) error {
					called = true
					if !reflect.DeepEqual(req.InitialStore, cp["store"]) {
						t.Errorf("resume substituted initial values: got=%v want=%v", req.InitialStore, cp["store"])
					}
					return nil
				})
				_, err = service.ResumeExecution(context.Background(), original, nil)
				synctest.Wait()
				if fault == "none" {
					if err != nil || !called || repo.index.ID == original {
						t.Errorf("valid committed state rejected: %v called=%t", err, called)
					}
				} else if !errors.Is(err, ErrExecutionNotResumable) || called || repo.index.ID != original {
					t.Errorf("unusable state admitted: error=%v called=%t original=%s current=%s", err, called, original, repo.index.ID)
				}
			})
		})
	}
}

// [REQ:BAS-RH-J07] A later action with an uncertain browser effect must not be
// replayed from an earlier successful checkpoint without reconciliation.
func TestResumeRejectsUncertainBrowserEffectAfterCheckpoint(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error {
			return errors.New("fixture failure")
		})
		workflowPath := filepath.Join(repo.project.FolderPath, repo.workflow.FilePath)
		workflowBytes, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatal(err)
		}
		var workflow basapi.WorkflowSummary
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(workflowBytes, &workflow); err != nil {
			t.Fatal(err)
		}
		workflow.Version = 2
		workflow.FlowDefinition.Nodes = append(workflow.FlowDefinition.Nodes, &basworkflows.WorkflowNodeV2{
			Id: "submit",
			Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_CLICK,
				Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#submit"}},
			},
		})
		workflow.FlowDefinition.Edges = []*basworkflows.WorkflowEdgeV2{{Id: "navigate-submit", Source: "navigate", Target: "submit"}}
		_, relativePath, err := WriteWorkflowSummaryFile(repo.project, &workflow, repo.workflow.FilePath)
		if err != nil {
			t.Fatal(err)
		}
		repo.workflow.FilePath = relativePath
		repo.workflow.Version = int(workflow.Version)
		if err := persistWorkflowVersionSnapshot(repo.project, &workflow); err != nil {
			t.Fatal(err)
		}

		original := prepareResumableMetadataFixture(t, service, repo, &basexecution.ExecutionParameters{})
		timelinePath := filepath.Join(service.executionDataRoot, original.String(), "timeline.proto.json")
		timeline := []byte(`{"entries":[{"step_index":0,"node_id":"navigate","context":{"success":true}},{"step_index":1,"node_id":"submit","context":{"success":false,"error":"browser response was cancelled after dispatch","error_code":"INSTRUCTION_OUTCOME_UNCERTAIN"}}]}`)
		if err := os.WriteFile(timelinePath, timeline, 0o600); err != nil {
			t.Fatal(err)
		}

		runnerCalled := false
		service.executor = completionExecutor(func(context.Context, executor.Request) error {
			runnerCalled = true
			return nil
		})
		_, err = service.ResumeExecution(context.Background(), original, nil)
		synctest.Wait()
		if !errors.Is(err, ErrExecutionNotResumable) || !strings.Contains(strings.ToLower(err.Error()), "uncertain") {
			t.Fatalf("resume error = %v, want an explicit uncertain-effect refusal", err)
		}
		if runnerCalled || repo.index.ID != original {
			t.Fatalf("uncertain operation admitted replacement execution: runner=%t index=%s original=%s", runnerCalled, repo.index.ID, original)
		}
	})
}
