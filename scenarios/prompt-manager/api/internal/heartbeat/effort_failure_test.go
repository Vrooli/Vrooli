package heartbeat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/vrooli/api-core/effortauthority"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiniteQueueSaveFailurePreventsWorkerAndPreservesPreparation(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(fmt.Sprint(queued), func(t *testing.T) {
			f, a, _, p, _, _ := finiteAuthorityFixture(t, false)
			cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
			ctx, err := f.runtime.prepareFiniteCaller(context.Background(), "committee", "lead", cfg)
			if err != nil {
				t.Fatal(err)
			}
			// Prepare first, matching the actual runtime ordering.
			state, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
			reserveFiniteLeader(state)
			if err = f.runtime.reserveEffortLeader(ctx, cfg, state); err != nil {
				t.Fatal(err)
			}
			before, _ := a.Store.Get(ctx, p.ID)
			bad := filepath.Join(t.TempDir(), "blocked")
			if err = os.WriteFile(bad, []byte("not a directory"), 0600); err != nil {
				t.Fatal(err)
			}
			exec := &captureExecutor{}
			client := &AgentManagerClient{efforts: f.runtime.Efforts}
			queue := newTeamExecutionContext("committee", exec, bad, client)
			if queued {
				queue.running["other"] = runningEntry{ProfileKey: "ordinary-existing"}
			}
			if _, err = queue.Enqueue(ctx, "lead", "qualified-profile"); !IsDispatchUncertain(err) {
				t.Fatalf("save failure not held: %v", err)
			}
			queue.Shutdown()
			after, _ := a.Store.Get(ctx, p.ID)
			if exec.callCount() != 0 || effortauthority.Digest(before) != effortauthority.Digest(after) || len(queue.queue) != 0 {
				t.Fatal("failed save dispatched, changed authority or retained false queue success")
			}
			// Repair persistence and reuse the exact preparation; it consumes no new slot.
			queue.persistDir = t.TempDir()
			if _, err = queue.Enqueue(ctx, "lead", "qualified-profile"); err != nil {
				t.Fatal(err)
			}
			queue.Shutdown()
			after, _ = a.Store.Get(ctx, p.ID)
			if effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("queue repair charged or released budget")
			}
		})
	}
}
func TestFiniteQueueDequeueSaveFailureHoldsOriginalObligation(t *testing.T) {
	f, _, _, _, _, _ := finiteAuthorityFixture(t, false)
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	ctx, err := f.runtime.prepareFiniteCaller(context.Background(), "committee", "lead", cfg)
	if err != nil {
		t.Fatal(err)
	}
	exec := &captureExecutor{}
	queue := newTeamExecutionContext("committee", exec, t.TempDir(), &AgentManagerClient{efforts: f.runtime.Efforts})
	queue.running["other"] = runningEntry{ProfileKey: "ordinary"}
	if _, err = queue.Enqueue(ctx, "lead", "qualified-profile"); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "blocked")
	os.WriteFile(bad, []byte("blocked"), 0600)
	queue.persistDir = bad
	queue.OnMemberComplete("other")
	queue.Shutdown()
	if exec.callCount() != 0 || len(queue.queue) != 1 || queue.queue[0].AgentID != "lead" || queue.running["lead"].ProfileKey != "" {
		t.Fatal("failed dequeue save lost or dispatched finite obligation")
	}
	queue.persistDir = t.TempDir()
	queue.mu.Lock()
	dispatch := queue.dispatchAvailableLocked()
	queue.mu.Unlock()
	queue.startExecutions(dispatch)
	queue.Shutdown()
	if exec.callCount() != 1 {
		t.Fatal("repaired queue did not dispatch original obligation once")
	}
}
func TestFiniteTransportEchoProofRedactedAndNoRedirectOrRetry(t *testing.T) {
	for _, mode := range []string{"echo", "encoded", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _, p, _, _ := finiteAuthorityFixture(t, false)
			cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
			ctx, err := f.runtime.prepareFiniteCaller(context.Background(), "committee", "lead", cfg)
			if err != nil {
				t.Fatal(err)
			}
			task := &Task{ID: "fixture-task", Title: "approved", Description: "approved", ProjectRoot: p.Repository, ScopePath: p.Repository}
			f.agent.getTasks = map[string]*Task{task.ID: task}
			f.runtime.Efforts.Tasks = f.agent
			hits := 0
			leaked := ""
			targetHits := 0
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetHits++ }))
			defer target.Close()
			native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				leaked = r.Header.Get(effortauthority.Header)
				if mode == "redirect" {
					http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				if mode == "encoded" {
					fmt.Fprint(w, base64.StdEncoding.EncodeToString([]byte(leaked)))
				} else {
					fmt.Fprint(w, leaked)
				}
			}))
			defer native.Close()
			client := &AgentManagerClient{httpClient: native.Client(), testBaseURL: native.URL, efforts: f.runtime.Efforts}
			_, err = client.CreateRun(ctx, &CreateRunRequest{TaskID: task.ID, ProfileRef: &ProfileRef{ProfileKey: "qualified-profile"}, IdempotencyKey: "proof-test"})
			if err == nil || leaked == "" || hits != 1 || targetHits != 0 || strings.Contains(err.Error(), leaked) || strings.Contains(err.Error(), base64.StdEncoding.EncodeToString([]byte(leaked))) {
				t.Fatalf("finite proof leaked/retried/redirected: hits=%d target=%d", hits, targetHits)
			}
		})
	}
}

func TestFiniteQueueRecoveryRequiresOriginalCurrentCustody(t *testing.T) {
	for _, mode := range []string{"missing-custody", "changed-binding", "expired", "revoked-owner"} {
		t.Run(mode, func(t *testing.T) {
			f, a, owner, p, _, now := finiteAuthorityFixture(t, false)
			cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
			ctx, e := f.runtime.prepareFiniteCaller(context.Background(), "committee", "lead", cfg)
			if e != nil {
				t.Fatal(e)
			}
			posts := 0
			native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts++
				}
				w.WriteHeader(404)
			}))
			defer native.Close()
			client := &AgentManagerClient{httpClient: native.Client(), testBaseURL: native.URL, efforts: f.runtime.Efforts}
			dir := t.TempDir()
			queue := newTeamExecutionContext("committee", &captureExecutor{}, dir, client)
			queue.maxConcurrentRuns = 0
			if _, e = queue.Enqueue(ctx, "lead", "qualified-profile"); e != nil {
				t.Fatal(e)
			}
			before, _ := a.Store.Get(ctx, p.ID)
			switch mode {
			case "missing-custody":
				f.runtime.Efforts.Signer = nil
			case "changed-binding":
				b := f.runtime.Efforts.Bindings["committee/lead"]
				b.Epoch++
				f.runtime.Efforts.Bindings["committee/lead"] = b
			case "expired":
				*now = p.Deadline
			case "revoked-owner":
				owner.enabled = false
			}
			exec := &captureExecutor{}
			restored := newTeamExecutionContext("committee", exec, dir, client)
			restored.Recover(context.Background())
			restored.mu.Lock()
			restored.maxConcurrentRuns = 1
			dispatch := restored.dispatchAvailableLocked()
			restored.mu.Unlock()
			restored.startExecutions(dispatch)
			restored.Shutdown()
			after, _ := a.Store.Get(ctx, p.ID)
			if posts != 0 || exec.callCount() != 0 || len(restored.queue) != 1 || restored.queue[0].State != "caller_required" || effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("recovery replaced authority, dispatched or lost held obligation")
			}
		})
	}
}

type failingFiniteQueue struct{ delegate *finiteQueueFake }

func (q failingFiniteQueue) Enqueue(context.Context, string, string, string) (*EnqueueResult, error) {
	return nil, fmt.Errorf("disposable enqueue failure")
}
func (q failingFiniteQueue) Status(team string) TeamExecutionStatus { return q.delegate.Status(team) }
func (q failingFiniteQueue) OnComplete(team, agent string)          { q.delegate.OnComplete(team, agent) }
func TestFinitePreparationQueueFailureRepairsSameDurableLeader(t *testing.T) {
	f, a, _, p, _, _ := finiteAuthorityFixture(t, false)
	f.runtime.Queue = failingFiniteQueue{f.queue}
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e == nil {
		t.Fatal("failed enqueue claimed success")
	}
	before, _ := a.Store.Get(context.Background(), p.ID)
	state, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if len(before.Reservations) != 1 || state.ID == "" || len(f.agent.createTaskCalls) != 0 {
		t.Fatal("queue failure lost durable preparation or created task")
	}
	f.runtime.Queue = f.queue
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	after, _ := a.Store.Get(context.Background(), p.ID)
	next, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if effortauthority.Digest(before) != effortauthority.Digest(after) || state.ID != next.ID || f.queue.enqueues != 1 {
		t.Fatal("queue repair replaced intent or charged second slot")
	}
}
func TestFiniteTaskCreateFailureRetainsPreparationAndDoesNotRetryUnknownTask(t *testing.T) {
	f, a, _, p, _, _ := finiteAuthorityFixture(t, false)
	f.agent.createTaskErr = fmt.Errorf("disposable task response lost")
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	if _, e := f.runtime.Executor.Execute(context.Background(), "committee", "lead", "qualified-profile"); e == nil {
		t.Fatal("failed task create claimed success")
	}
	before, _ := a.Store.Get(context.Background(), p.ID)
	f.agent.createTaskErr = nil
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	after, _ := a.Store.Get(context.Background(), p.ID)
	if len(f.agent.createTaskCalls) != 1 || len(f.agent.createRunCalls) != 0 || effortauthority.Digest(before) != effortauthority.Digest(after) {
		t.Fatal("unknown task response caused retry/native effects or budget release")
	}
}

func TestFiniteStateSaveFailureRetainsUnknownPreparationWithoutEffects(t *testing.T) {
	f, a, _, p, _, _ := finiteAuthorityFixture(t, false)
	runtimeRoot := f.teams.GetMemberLogPath("committee", "lead", "fixture")
	for n := 0; n < 6; n++ {
		runtimeRoot = filepath.Dir(runtimeRoot)
	}
	blocked := filepath.Join(runtimeRoot, "finite-leaders")
	originalSigner := f.runtime.Efforts.Signer
	fired := false
	f.runtime.Efforts.Signer = finiteSignerFault{originalSigner, func() {
		if !fired {
			fired = true
			if e := os.WriteFile(blocked, []byte("block state directory creation after state read"), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}}
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e == nil {
		t.Fatal("failed state save claimed success")
	}
	before, e := a.Store.Get(context.Background(), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(before.Reservations) != 1 || f.queue.enqueues != 0 || len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("state-save failure lost occupied preparation or dispatched")
	}
	// A crash can lose evidence at this boundary. Repairing the directory does
	// not prove that no task ever existed: preserve the original unknown slot.
	if e = os.Remove(blocked); e != nil {
		t.Fatal(e)
	}
	f.runtime.Efforts.Signer = originalSigner
	if _, e = f.runtime.Tick(context.Background(), "committee", "lead"); e == nil {
		t.Fatal("missing durable leader renewed unknown preparation")
	}
	after, _ := a.Store.Get(context.Background(), p.ID)
	if effortauthority.Digest(before) != effortauthority.Digest(after) || f.queue.enqueues != 0 || len(f.agent.createTaskCalls) != 0 {
		t.Fatal("unknown preparation repaired by duplicate effects")
	}
}

func TestFiniteAttachedTaskRefusedBeforeNativeTransport(t *testing.T) {
	f, _, _, p, _, _ := finiteAuthorityFixture(t, false)
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	ctx, e := f.runtime.prepareFiniteCaller(context.Background(), "committee", "lead", cfg)
	if e != nil {
		t.Fatal(e)
	}
	f.agent.getTasks = map[string]*Task{"task": {ID: "task", Title: "approved", Description: "approved", ProjectRoot: p.Repository, ScopePath: p.Repository, ContextAttachments: []byte(`[{"type":"file","path":"fixture"}]`)}}
	f.runtime.Efforts.Tasks = f.agent
	hits := 0
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.WriteHeader(201) }))
	defer native.Close()
	client := &AgentManagerClient{httpClient: native.Client(), testBaseURL: native.URL}
	if _, e = client.CreateRun(ctx, &CreateRunRequest{TaskID: "task", ProfileRef: &ProfileRef{ProfileKey: "qualified-profile"}, IdempotencyKey: "attached"}); e == nil || hits != 0 {
		t.Fatal("incomplete attached task view signed or sent")
	}
}

type finiteSignerFault struct {
	delegate EffortSigner
	fault    func()
}

func (s finiteSignerFault) SignEffort(ctx context.Context, client string, p effortauthority.Proof) (effortauthority.Proof, error) {
	proof, e := s.delegate.SignEffort(ctx, client, p)
	if e == nil {
		s.fault()
	}
	return proof, e
}

func TestFiniteDurableIntentSaveFailureStopsActualExecutorBeforeRunHTTP(t *testing.T) {
	f, a, _, p, _, _ := finiteAuthorityFixture(t, false)
	var queue *TeamExecutionContext
	posts := 0
	taskCreates := 0
	task := &Task{ID: "disposable-task", Title: "fixture", Description: "fixture", ScopePath: p.Repository, ProjectRoot: p.Repository}
	blocked := filepath.Join(t.TempDir(), "blocked")
	if e := os.WriteFile(blocked, []byte("not a directory"), 0600); e != nil {
		t.Fatal(e)
	}
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tasks" && r.Method == http.MethodPost {
			taskCreates++
			queue.mu.Lock()
			queue.persistDir = blocked
			queue.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CreateTaskResponse{Task: task})
			return
		}
		if r.URL.Path == "/api/v1/tasks/disposable-task" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CreateTaskResponse{Task: task})
			return
		}
		if r.URL.Path == "/api/v1/runs" && r.Method == http.MethodPost {
			posts++
		}
		w.WriteHeader(404)
	}))
	defer native.Close()
	client := &AgentManagerClient{httpClient: native.Client(), testBaseURL: native.URL, efforts: f.runtime.Efforts}
	f.runtime.Efforts.Tasks = client
	f.runtime.Executor.agentClient = client
	f.runtime.Executor.vrooliRoot = p.Repository
	queues := NewTeamExecutionStore(f.teams, f.runtime.Executor, t.TempDir(), client)
	queue = queues.GetOrCreate("committee")
	f.runtime.Queue = queues
	f.runtime.Executor.teamExecStore = queues
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	queues.Shutdown()
	record, _ := a.Store.Get(context.Background(), p.ID)
	if taskCreates != 1 || posts != 0 || len(record.Reservations) != 1 || len(queue.running) != 1 || queue.running["lead"].State != ObligationDispatchUncertain {
		t.Fatal("durable-intent failure crossed native boundary or released obligation")
	}
}
