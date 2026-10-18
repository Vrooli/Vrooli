package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
)

type fakeTokenSource struct {
	token string
	err   error
}

func (f fakeTokenSource) Token(context.Context) (string, error) { return f.token, f.err }

// An unattended heartbeat with no human/run proof forwards Prompt Manager's
// service bearer, so Agent Manager sees a verified service identity.
func TestPrepareCreateRunCallerForwardsServiceBearer(t *testing.T) {
	c := &AgentManagerClient{serviceToken: fakeTokenSource{token: "svc-token"}}
	ctx, err := c.prepareCreateRunCaller(context.Background())
	if err != nil {
		t.Fatalf("prepare refused: %v", err)
	}
	proof := createRunCaller(ctx)
	if proof.authorization != "Bearer svc-token" {
		t.Fatalf("service bearer not forwarded as authorization: %q", proof.authorization)
	}
	if proof.runIdentity != "" {
		t.Fatalf("service path must not set a run identity: %q", proof.runIdentity)
	}
}

// A human owner proof already in context is never replaced by the service
// identity — attribution stays with the human.
func TestPrepareCreateRunCallerKeepsExistingHumanProof(t *testing.T) {
	c := &AgentManagerClient{
		serviceToken:         fakeTokenSource{token: "svc-token"},
		createCallerVerifier: replayCallerVerifier{subject: "human-a"},
	}
	seeded := withCreateRunCaller(context.Background(), "Bearer human-token", "")
	ctx, err := c.prepareCreateRunCaller(seeded)
	if err != nil {
		t.Fatalf("prepare refused: %v", err)
	}
	if proof := createRunCaller(ctx); proof.authorization != "Bearer human-token" {
		t.Fatalf("human proof was overridden by the service identity: %q", proof.authorization)
	}
}

// A configured service mode never falls back to anonymous authority, even
// when the operator has AUTH-01 enforcement disabled.
func TestPrepareCreateRunCallerServiceMintFailureIsHardRefusal(t *testing.T) {
	for _, enforcement := range []string{"", "true"} {
		t.Run("auth01="+enforcement, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, enforcement)
			c := &AgentManagerClient{serviceToken: fakeTokenSource{err: errors.New("credential not issued")}}
			if _, err := c.prepareCreateRunCaller(context.Background()); err == nil {
				t.Fatal("service credential failure must refuse before owner I/O")
			}
			serviceMarker := withCreateRunCallerOrigin(context.Background(), "", "", createRunCallerOriginService)
			if _, err := c.prepareCreateRunCaller(serviceMarker); err == nil {
				t.Fatal("queued service mode must not fall back when fresh mint fails")
			}
		})
	}
}

func TestServiceTokenSourceCachesUntilRefreshMargin(t *testing.T) {
	now := time.Now()
	calls := 0
	src := newServiceTokenSource(
		func() (string, error) { return "cred", nil },
		func(context.Context, string) (string, time.Time, error) {
			calls++
			return "tok", now.Add(time.Hour), nil
		},
		func() time.Time { return now },
	)
	for i := 0; i < 3; i++ {
		if tok, err := src.Token(context.Background()); err != nil || tok != "tok" {
			t.Fatalf("token: %q %v", tok, err)
		}
	}
	if calls != 1 {
		t.Fatalf("expected a single exchange while cached, got %d", calls)
	}
}

func TestEnqueueServiceMintFailureHasNoOwnerRequests(t *testing.T) {
	for _, enforcement := range []string{"", "true"} {
		t.Run("auth01="+enforcement, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, enforcement)
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			client := &AgentManagerClient{
				serviceToken: fakeTokenSource{err: errors.New("credential not issued")},
				httpClient:   server.Client(),
				testBaseURL:  server.URL,
			}
			q := newTeamExecutionContext("team", &captureExecutor{}, t.TempDir(), client)
			if _, err := q.Enqueue(context.Background(), "agent", "profile"); err == nil {
				t.Fatal("enqueue must refuse when service mint fails")
			}
			if requests != 0 || len(q.queue) != 0 || len(q.running) != 0 {
				t.Fatalf("failed mint had owner or queue effects: requests=%d queue=%d running=%d", requests, len(q.queue), len(q.running))
			}
		})
	}
}

type mutexTokenSource struct {
	mu    sync.Mutex
	calls int
}

type failAfterFirstTokenSource struct {
	mu    sync.Mutex
	calls int
}

func (s *failAfterFirstTokenSource) Token(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls == 1 {
		return "preflight-token", nil
	}
	return "", errors.New("credential refresh failed")
}

func (s *failAfterFirstTokenSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *mutexTokenSource) Token(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return "service-token-" + strconv.Itoa(s.calls), nil
}

func (s *mutexTokenSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestQueuedServiceModeRefreshesRequestCredential(t *testing.T) {
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	tokens := &mutexTokenSource{}
	var mu sync.Mutex
	var authHeaders []string
	var taskID string
	var runTaskID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Method == http.MethodPost && (r.URL.Path == "/api/v1/tasks" || r.URL.Path == "/api/v1/runs") {
			authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/tasks":
			var request CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Task == nil {
				mu.Unlock()
				http.Error(w, "invalid task request", http.StatusBadRequest)
				return
			}
			taskID = request.Task.ID
			request.Task.Status = "queued"
			response, _ := json.Marshal(CreateTaskResponse{Task: request.Task})
			_, _ = w.Write(response)
		case "/api/v1/runs":
			var request CreateRunRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				mu.Unlock()
				http.Error(w, "invalid run request", http.StatusBadRequest)
				return
			}
			runTaskID = request.TaskID
			_, _ = w.Write([]byte(`{"run":{"id":"run-1","task_id":"` + request.TaskID + `","status":"RUN_STATUS_COMPLETE"}}`))
		case "/api/v1/runs/run-1":
			_, _ = w.Write([]byte(`{"run":{"id":"run-1","task_id":"` + taskID + `","status":"RUN_STATUS_COMPLETE"}}`))
		default:
			mu.Unlock()
			http.NotFound(w, r)
			return
		}
		mu.Unlock()
	}))
	defer server.Close()
	client := &AgentManagerClient{serviceToken: tokens, httpClient: server.Client(), testBaseURL: server.URL}
	teamStore, agentStore, root := setupExecutorTestEnv(t)
	executor := newTestExecutor(t, teamStore, agentStore, client, root, NewRunRegistry(t.TempDir()), nil)
	queueDir := t.TempDir()
	q := newTeamExecutionContext("team-1", executor, queueDir, client)
	completed := make(chan struct{}, 1)
	executor.OnComplete = func(_ string, agentID string) {
		q.OnMemberComplete(agentID)
		completed <- struct{}{}
	}
	q.running["occupied"] = runningEntry{}
	if _, err := q.Enqueue(context.Background(), "agent-1", "profile"); err != nil {
		t.Fatalf("enqueue service work: %v", err)
	}
	queuedProof := createRunCaller(q.queue[0].caller)
	if queuedProof.origin != createRunCallerOriginService || queuedProof.authorization != "" || queuedProof.runIdentity != "" {
		t.Fatalf("queue must retain only service mode, got %+v", queuedProof)
	}
	encoded, _ := json.Marshal(q.queue)
	if strings.Contains(string(encoded), "service-token") {
		t.Fatal("service credential was persisted in queued work")
	}
	// Remove the fixture's capacity blocker and persist a restart image with
	// only the original queued service obligation.
	q.mu.Lock()
	delete(q.running, "occupied")
	if err := q.persistLocked(); err != nil {
		q.mu.Unlock()
		t.Fatalf("persist queue fixture: %v", err)
	}
	q.mu.Unlock()
	queueFile, err := os.ReadFile(q.queueFilePath())
	if err != nil {
		t.Fatalf("read persisted queue fixture: %v", err)
	}
	if strings.Contains(string(queueFile), "service-token") || !strings.Contains(string(queueFile), `"CreateRunCallerMode": "service"`) {
		t.Fatalf("persisted queue must retain service mode only, got %s", queueFile)
	}
	restarted := newTeamExecutionContext("team-1", executor, queueDir, client)
	restarted.Recover(context.Background())
	if len(restarted.queue) != 1 {
		t.Fatalf("restart must retain original service queued work, got %d entries", len(restarted.queue))
	}
	recoveredProof := createRunCaller(restarted.queue[0].caller)
	if recoveredProof.origin != createRunCallerOriginService || recoveredProof.authorization != "" {
		t.Fatalf("restart must restore only the service mode marker, got %+v", recoveredProof)
	}
	executor.OnComplete = func(_ string, agentID string) {
		restarted.OnMemberComplete(agentID)
		completed <- struct{}{}
	}
	restarted.Configure("fifo", 1)
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("queued service execution did not finish")
	}
	restarted.Shutdown()
	mu.Lock()
	defer mu.Unlock()
	if calls := tokens.count(); calls != 2 {
		t.Fatalf("expected Enqueue qualification and promotion refresh, got %d token calls", calls)
	}
	if len(authHeaders) != 2 || authHeaders[0] != "Bearer service-token-2" || authHeaders[1] != "Bearer service-token-2" {
		t.Fatalf("task/run requests did not receive fresh service identity: %v", authHeaders)
	}
	if taskID == "" || runTaskID != taskID {
		t.Fatalf("CreateRun did not reference the task created by this dispatch: task=%q run.task=%q", taskID, runTaskID)
	}
}

func TestQueuedServiceRefreshFailureHasNoOwnerRequests(t *testing.T) {
	for _, enforcement := range []string{"", "true"} {
		t.Run("auth01="+enforcement, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, enforcement)
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				http.Error(w, "unexpected owner request", http.StatusInternalServerError)
			}))
			defer server.Close()
			tokens := &failAfterFirstTokenSource{}
			client := &AgentManagerClient{serviceToken: tokens, httpClient: server.Client(), testBaseURL: server.URL}
			teamStore, agentStore, root := setupExecutorTestEnv(t)
			executor := newTestExecutor(t, teamStore, agentStore, client, root, nil, nil)
			queueDir := t.TempDir()
			q := newTeamExecutionContext("team-1", executor, queueDir, client)
			q.running["occupied"] = runningEntry{}
			if _, err := q.Enqueue(context.Background(), "agent-1", "profile"); err != nil {
				t.Fatalf("queue service dispatch: %v", err)
			}
			q.mu.Lock()
			delete(q.running, "occupied")
			if err := q.persistLocked(); err != nil {
				q.mu.Unlock()
				t.Fatalf("persist queued service fixture: %v", err)
			}
			q.mu.Unlock()
			restarted := newTeamExecutionContext("team-1", executor, queueDir, client)
			restarted.Recover(context.Background())
			restarted.Configure("fifo", 1)
			restarted.executionWG.Wait()
			restarted.Shutdown()
			if tokens.count() != 2 || requests != 0 {
				t.Fatalf("failed restart/promotion refresh was not refused before owner I/O: token calls=%d owner requests=%d", tokens.count(), requests)
			}
		})
	}
}

func TestEnqueuedServiceTaskApprovalAndRefusalDoNotCreateRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantHealth string
	}{
		{name: "approval_pending", status: http.StatusAccepted, body: `{"status":"approval_pending","detail":"operator approval pending"}`, wantHealth: "approval_pending"},
		{name: "service_forbidden", status: http.StatusForbidden, body: `{"error":"service authorization refused"}`, wantHealth: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
			tokens := &mutexTokenSource{}
			taskPosts := 0
			runPosts := 0
			var taskAuthorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks" {
					taskPosts++
					taskAuthorization = r.Header.Get("Authorization")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs" {
					runPosts++
				}
				http.NotFound(w, r)
			}))
			defer server.Close()
			client := &AgentManagerClient{serviceToken: tokens, httpClient: server.Client(), testBaseURL: server.URL}
			teamStore, agentStore, root := setupExecutorTestEnv(t)
			executor := newTestExecutor(t, teamStore, agentStore, client, root, nil, nil)
			q := newTeamExecutionContext("team-1", executor, t.TempDir(), client)
			result, err := q.Enqueue(context.Background(), "agent-1", "profile")
			if err != nil || result.Status != "started" {
				t.Fatalf("enqueue returned result=%+v err=%v", result, err)
			}
			q.executionWG.Wait()
			if taskPosts != 1 || runPosts != 0 || taskAuthorization != "Bearer service-token-2" || tokens.count() != 2 {
				t.Fatalf("service task-stage outcome posts task/run=%d/%d auth=%q token calls=%d", taskPosts, runPosts, taskAuthorization, tokens.count())
			}
			config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
			if err != nil || config.LastExecution == nil || config.LastExecution.Status != tc.wantHealth {
				t.Fatalf("task-stage %s not retained in owner health: config=%+v err=%v", tc.name, config, err)
			}
		})
	}
}
