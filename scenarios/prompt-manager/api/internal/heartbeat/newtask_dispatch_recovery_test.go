package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	amapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	amdomain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

type taskDispatchOwnerStub struct {
	task          *Task
	getErr        error
	createErr     error
	getCalls      []string
	createCalls   []*Task
	createMutates bool
}

func (s *taskDispatchOwnerStub) GetTask(_ context.Context, taskID string) (*Task, error) {
	s.getCalls = append(s.getCalls, taskID)
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.task == nil {
		return nil, nil
	}
	return cloneDispatchTask(s.task), nil
}

func (s *taskDispatchOwnerStub) CreateTask(_ context.Context, task *Task) (*Task, error) {
	s.createCalls = append(s.createCalls, cloneDispatchTask(task))
	if s.createMutates {
		s.task = cloneDispatchTask(task)
	}
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.task = cloneDispatchTask(task)
	return cloneDispatchTask(task), nil
}

func cloneDispatchTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.ContextAttachments = append(cloned.ContextAttachments[:0:0], task.ContextAttachments...)
	cloned.PhasePromptIDs = append(cloned.PhasePromptIDs[:0:0], task.PhasePromptIDs...)
	return &cloned
}

func TestResolveTaskDispatchReplaysFrozenUUIDAfterRestart(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	// The restarted caller only has the persisted bytes. A previous accepted
	// create with a lost response is discovered by exact ID and is not repeated.
	owner := &taskDispatchOwnerStub{task: cloneDispatchTask(request)}
	resolved, err := resolveTaskDispatch(context.Background(), owner, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != request.ID || len(owner.createCalls) != 0 {
		t.Fatalf("resolved=%+v create calls=%d; expected the original task and no duplicate", resolved, len(owner.createCalls))
	}
	if len(owner.getCalls) != 1 || owner.getCalls[0] != request.ID {
		t.Fatalf("lookup IDs=%v; expected only exact persisted UUID %q", owner.getCalls, request.ID)
	}
}

func TestResolveTaskDispatchPostsSameUUIDOnlyAfterExact404(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	owner := &taskDispatchOwnerStub{}
	resolved, err := resolveTaskDispatch(context.Background(), owner, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != request.ID || len(owner.createCalls) != 1 || owner.createCalls[0].ID != request.ID {
		t.Fatalf("resolved=%+v create calls=%+v; expected one create with frozen ID %q", resolved, owner.createCalls, request.ID)
	}
	if len(owner.getCalls) != 1 || owner.getCalls[0] != request.ID {
		t.Fatalf("lookup IDs=%v; expected exact persisted UUID %q", owner.getCalls, request.ID)
	}
}

func TestResolveTaskDispatchLostCreateResponseUsesExactLookup(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	owner := &taskDispatchOwnerStub{
		createErr:     NewDispatchUncertainError(errors.New("response lost after owner insert")),
		createMutates: true,
	}
	resolved, err := resolveTaskDispatch(context.Background(), owner, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != request.ID || len(owner.createCalls) != 1 || len(owner.getCalls) != 2 {
		t.Fatalf("resolved=%+v creates=%d lookups=%v", resolved, len(owner.createCalls), owner.getCalls)
	}
	for _, id := range owner.getCalls {
		if id != request.ID {
			t.Fatalf("lookup used replacement ID %q, want %q", id, request.ID)
		}
	}
}

func TestResolveTaskDispatchRealHTTPRecoversLostCreateResponse(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var stored *Task
	var postIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tasks/"+request.ID:
			mu.Lock()
			current := cloneDispatchTask(stored)
			mu.Unlock()
			if current == nil {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{Task: current})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks":
			var body CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Task == nil {
				http.Error(w, "invalid task request", http.StatusBadRequest)
				return
			}
			mu.Lock()
			postIDs = append(postIDs, body.Task.ID)
			stored = cloneDispatchTask(body.Task)
			mu.Unlock()
			// Simulate the owner committing the row and losing the response.
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("test server response writer does not support hijacking")
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("hijack response: %v", err)
				return
			}
			_ = conn.Close()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &AgentManagerClient{
		httpClient:  server.Client(),
		testBaseURL: server.URL,
		sleep:       func(context.Context, time.Duration) error { return nil },
	}
	resolved, err := resolveTaskDispatch(context.Background(), client, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != request.ID {
		t.Fatalf("resolved task ID=%q, want persisted ID %q", resolved.ID, request.ID)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(postIDs) != 1 || postIDs[0] != request.ID {
		t.Fatalf("task POST IDs=%v; expected one POST with frozen ID %q", postIDs, request.ID)
	}
}

func TestResolveTaskDispatchAcceptsNativeQueuedEnumAndRejectsNativeCancelled(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    amdomain.TaskStatus
		wantError bool
	}{
		{name: "queued", status: amdomain.TaskStatus_TASK_STATUS_QUEUED},
		{name: "cancelled", status: amdomain.TaskStatus_TASK_STATUS_CANCELLED, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := dispatchTaskFixture()
			request.ContextAttachments = nil
			snapshot, err := freezeTaskDispatchRequest(request)
			if err != nil {
				t.Fatal(err)
			}
			response, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(&amapi.GetTaskResponse{Task: &amdomain.Task{
				Id: request.ID, Title: request.Title, Description: request.Description,
				ScopePath: request.ScopePath, ProjectRoot: request.ProjectRoot, Status: tc.status,
			}})
			if err != nil {
				t.Fatal(err)
			}
			var postCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(response)
					return
				}
				if r.Method == http.MethodPost {
					postCount.Add(1)
					http.Error(w, "unexpected duplicate task POST", http.StatusConflict)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			client := &AgentManagerClient{httpClient: server.Client(), testBaseURL: server.URL}
			resolved, resolveErr := resolveTaskDispatch(context.Background(), client, snapshot)
			if tc.wantError {
				if resolveErr == nil || !IsDispatchUncertain(resolveErr) {
					t.Fatalf("resolve error=%v; want retained uncertain outcome for native cancelled task", resolveErr)
				}
			} else if resolveErr != nil || resolved == nil || resolved.ID != request.ID {
				t.Fatalf("resolve=(%+v, %v); want matching task from native TASK_STATUS_QUEUED response", resolved, resolveErr)
			}
			if got := postCount.Load(); got != 0 {
				t.Fatalf("native owner status caused %d duplicate task POSTs", got)
			}
		})
	}
}

func TestResolveTaskDispatchDoesNotAdoptForeignSameIDTerms(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	foreign := cloneDispatchTask(request)
	foreign.Description = "different owner-authored terms"
	owner := &taskDispatchOwnerStub{task: foreign}
	if _, err := resolveTaskDispatch(context.Background(), owner, snapshot); err == nil {
		t.Fatal("expected same-ID task with mismatched terms to remain unresolved")
	}
	if len(owner.createCalls) != 0 {
		t.Fatalf("mismatched task caused %d create calls", len(owner.createCalls))
	}
}

func TestResolveTaskDispatchRejectsUnexpectedOwnerPhasePrompts(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	ownerTask := cloneDispatchTask(request)
	ownerTask.PhasePromptIDs = []string{uuid.NewString()}
	owner := &taskDispatchOwnerStub{task: ownerTask}
	if _, err := resolveTaskDispatch(context.Background(), owner, snapshot); err == nil {
		t.Fatal("owner-added executable phase prompt must not match the frozen heartbeat task")
	}
	if len(owner.createCalls) != 0 {
		t.Fatalf("phase mismatch caused %d task create calls", len(owner.createCalls))
	}
}

func TestResolveTaskDispatchDoesNotResurrectTerminalOrIncompatibleTask(t *testing.T) {
	for _, status := range []string{"cancelled", "completed", "failed", "needs_review", "running", ""} {
		t.Run(status, func(t *testing.T) {
			request := dispatchTaskFixture()
			snapshot, err := freezeTaskDispatchRequest(request)
			if err != nil {
				t.Fatal(err)
			}
			ownerTask := cloneDispatchTask(request)
			ownerTask.Status = status
			owner := &taskDispatchOwnerStub{task: ownerTask}
			if _, err := resolveTaskDispatch(context.Background(), owner, snapshot); err == nil {
				t.Fatalf("status %q must not be accepted as a runnable task", status)
			}
			if len(owner.createCalls) != 0 {
				t.Fatalf("status %q caused %d task creates", status, len(owner.createCalls))
			}
		})
	}
}

func TestResolveTaskDispatchMalformedOwnerSuccessIsNotAbsence(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var postCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`)) // malformed success: no task identity
			return
		}
		if r.Method == http.MethodPost {
			postCount++
			w.WriteHeader(http.StatusCreated)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := &AgentManagerClient{
		httpClient:  server.Client(),
		testBaseURL: server.URL,
		sleep:       func(context.Context, time.Duration) error { return nil },
	}
	if _, err := resolveTaskDispatch(context.Background(), client, snapshot); err == nil {
		t.Fatal("malformed HTTP 200 task response must remain unresolved, not imply absence")
	}
	if postCount != 0 {
		t.Fatalf("malformed success caused %d replacement task POSTs", postCount)
	}
}

func TestResolveTaskDispatchDoesNotTreatLookupOutageAsAbsence(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	owner := &taskDispatchOwnerStub{getErr: errors.New("owner unavailable")}
	if _, err := resolveTaskDispatch(context.Background(), owner, snapshot); err == nil {
		t.Fatal("expected lookup outage to retain unresolved dispatch")
	}
	if len(owner.createCalls) != 0 {
		t.Fatalf("lookup outage caused %d create calls", len(owner.createCalls))
	}
}

func TestResolveTaskDispatchPreservesApprovalPending(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	pending := NewApprovalPendingError("creating_task", errors.New("human approval required"))
	owner := &taskDispatchOwnerStub{createErr: pending}
	_, err = resolveTaskDispatch(context.Background(), owner, snapshot)
	if !IsApprovalPending(err) {
		t.Fatalf("error=%v; expected typed approval_pending", err)
	}
	if len(owner.createCalls) != 1 || len(owner.getCalls) != 1 {
		t.Fatalf("pending create must not be treated as confirmed or retried: creates=%d lookups=%v", len(owner.createCalls), owner.getCalls)
	}
}

func TestResolveTaskDispatchCancellationDoesNotBecomeReplayableUncertainty(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	owner := &cancelBeforeTaskCreateOwner{cancel: cancel}
	resolved, resolveErr := resolveTaskDispatch(ctx, owner, snapshot)
	if resolved != nil || !isPreSendCancellation(resolveErr) {
		t.Fatalf("resolve=(%+v, %v); want no task and definitive pre-send cancellation", resolved, resolveErr)
	}
	if owner.createCalls != 1 || owner.getCalls != 1 {
		t.Fatalf("calls: create=%d get=%d; want one preflight read and one guarded create, no follow-up read", owner.createCalls, owner.getCalls)
	}
}

func TestResolveTaskDispatchKeepsAmbiguousCreateAbsentResultUncertain(t *testing.T) {
	request := dispatchTaskFixture()
	snapshot, err := freezeTaskDispatchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	owner := &ambiguousAbsentTaskOwner{createErr: NewDispatchUncertainError(errors.New("response lost"))}
	resolved, resolveErr := resolveTaskDispatch(context.Background(), owner, snapshot)
	if resolved != nil || !IsDispatchUncertain(resolveErr) {
		t.Fatalf("resolve=(%+v, %v); ambiguous POST with immediate 404 must remain uncertain", resolved, resolveErr)
	}
	if owner.createCalls != 1 || owner.getCalls != 2 {
		t.Fatalf("calls: create=%d get=%d; want one POST and pre/post exact-ID reads", owner.createCalls, owner.getCalls)
	}
}

type cancelBeforeTaskCreateOwner struct {
	cancel      context.CancelFunc
	createCalls int
	getCalls    int
}

func (o *cancelBeforeTaskCreateOwner) GetTask(context.Context, string) (*Task, error) {
	o.getCalls++
	return nil, nil
}

func (o *cancelBeforeTaskCreateOwner) CreateTask(ctx context.Context, _ *Task) (*Task, error) {
	o.createCalls++
	o.cancel()
	return nil, NewDispatchRejectedError(context.Canceled)
}

type ambiguousAbsentTaskOwner struct {
	createErr   error
	createCalls int
	getCalls    int
}

func (o *ambiguousAbsentTaskOwner) GetTask(context.Context, string) (*Task, error) {
	o.getCalls++
	return nil, nil
}

func (o *ambiguousAbsentTaskOwner) CreateTask(context.Context, *Task) (*Task, error) {
	o.createCalls++
	return nil, o.createErr
}

func TestParseTaskDispatchRequestRejectsInvalidIdentity(t *testing.T) {
	for _, snapshot := range []string{
		`{"id":"","title":"x"}`,
		`{"id":"00000000-0000-0000-0000-000000000000","title":"x"}`,
		`{"id":"` + uuid.NewString() + `","title":"x"} {}`,
	} {
		if _, err := parseTaskDispatchRequest(snapshot); err == nil {
			t.Errorf("expected invalid snapshot to fail: %s", snapshot)
		}
	}
}

func dispatchTaskFixture() *Task {
	return &Task{
		ID:                 uuid.NewString(),
		Title:              "heartbeat task",
		Description:        "frozen task terms",
		ScopePath:          "/workspace",
		ProjectRoot:        "/workspace/project",
		Status:             "queued",
		ContextAttachments: []byte(`{"refs":["work-1"]}`),
	}
}
