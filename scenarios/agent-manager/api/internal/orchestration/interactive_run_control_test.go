package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

// recordingSessions is a webconsole.SessionController that records the lifecycle
// calls (and SendPrompt source) the interactive Stop/Continue paths make, and
// lets a session be marked gone or GetSession forced to error.
type recordingSessions struct {
	mu          sync.Mutex
	calls       []string
	promptSrc   []string
	promptText  []string
	gone        map[string]bool
	getNotFound bool
	onCreate    func()
}

func newRecordingSessions() *recordingSessions {
	return &recordingSessions{gone: map[string]bool{}}
}

func (r *recordingSessions) record(c string) {
	r.mu.Lock()
	r.calls = append(r.calls, c)
	r.mu.Unlock()
}

func (r *recordingSessions) callLog() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func (r *recordingSessions) CreateSession(context.Context, webconsole.CreateSessionParams) (string, error) {
	r.record("create")
	if r.onCreate != nil {
		r.onCreate()
	}
	return "created", nil
}

func (r *recordingSessions) GetSession(_ context.Context, id string) (webconsole.SessionInfo, error) {
	r.mu.Lock()
	notFound := r.getNotFound || r.gone[id]
	r.mu.Unlock()
	if notFound {
		return webconsole.SessionInfo{}, webconsole.ErrSessionNotFound
	}
	return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager}, nil
}

func (r *recordingSessions) DeleteSession(_ context.Context, id string) error {
	r.record("delete")
	r.mu.Lock()
	r.gone[id] = true
	r.mu.Unlock()
	return nil
}

func (r *recordingSessions) SendText(context.Context, string, string, string) error {
	r.record("sendtext")
	return nil
}

func (r *recordingSessions) SendPrompt(_ context.Context, _, prompt, source string) error {
	r.record("sendprompt")
	r.mu.Lock()
	r.promptSrc = append(r.promptSrc, source)
	r.promptText = append(r.promptText, prompt)
	r.mu.Unlock()
	return nil
}

func (r *recordingSessions) Interrupt(context.Context, string, string) error {
	r.record("interrupt")
	return nil
}

func (r *recordingSessions) Screen(context.Context, string, bool) (string, error) { return "", nil }

func interactiveTestTask(t *testing.T, svc *Orchestrator) *domain.Task {
	t.Helper()
	task, err := svc.CreateTask(context.Background(), &domain.Task{Title: "interactive", ScopePath: "src/"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return task
}

func persistInteractiveRun(t *testing.T, runs repository.RunRepository, taskID uuid.UUID, status domain.RunStatus, sessionID, wcSession string) *domain.Run {
	t.Helper()
	now := time.Now()
	id := uuid.New()
	run := &domain.Run{
		ID:                  id,
		TaskID:              taskID,
		Tag:                 id.String(),
		RunMode:             domain.RunModeInPlace,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: wcSession,
		SessionID:           sessionID,
		Status:              status,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
		ResolvedConfig:      &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if status.IsTerminal() {
		run.Phase = domain.RunPhaseCompleted
		run.EndedAt = &now
	}
	if err := runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

// TestExecuteInteractiveRun_ProtectedBackstop verifies the execution-path backstop
// (in addition to the CreateRun-time gate): a run that reaches executeInteractiveRun
// as protected (sandboxed) is failed with the actionable gate error rather than
// launching a session.
func TestExecuteInteractiveRun_ProtectedBackstop(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithRunStateRoot(t.TempDir()))
	task := interactiveTestTask(t, svc)

	now := time.Now()
	id := uuid.New()
	run := &domain.Run{
		ID:             id,
		TaskID:         task.ID,
		Tag:            id.String(),
		RunMode:        domain.RunModeSandboxed, // protected
		ExecutionMode:  domain.ExecutionModeInteractive,
		Status:         domain.RunStatusStarting,
		Phase:          domain.RunPhaseExecuting,
		ApprovalState:  domain.ApprovalStateNone,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	svc.executeInteractiveRun(ctx, run, task, "do the task", nil)

	got, err := repos.Runs.Get(ctx, id)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if !strings.Contains(strings.ToLower(got.ErrorMsg), "protected") {
		t.Errorf("error should explain the protected-run gate, got %q", got.ErrorMsg)
	}
	for _, c := range sessions.callLog() {
		if c == "create" {
			t.Fatal("a protected run must never create a web-console session")
		}
	}
}

// TestExecuteInteractiveRun_TrackingPersistsSandboxAttribution drives the
// actual interactive coordinator through a recorded Codex terminal. It proves
// Tracking uses the sandbox workdir, then applies the sandbox-owned attribution
// facts to the same persisted run as codec-pipe finalization.
func TestExecuteInteractiveRun_TrackingPersistsSandboxAttribution(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	root := t.TempDir()
	sessions := newRecordingSessions()
	provider := mocks.NewFakeSandboxProvider()
	provider.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{
		Success: true, Applied: 2, TotalSizeBytes: 3072,
		DiffPath: "/api/v1/sandboxes/tracking/diff", CommitHash: "tracking-commit", AppliedAt: time.Now(),
	}
	registry := runner.NewRegistry()
	if err := registry.Register(core.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithInteractiveSessions(sessions), WithSandbox(provider), WithRunners(registry), WithRunStateRoot(root))
	task := interactiveTestTask(t, svc)

	now := time.Now()
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "interactive-tracking", RunMode: domain.RunModeSandboxed,
		ExecutionMode: domain.ExecutionModeInteractive, Status: domain.RunStatusStarting, Phase: domain.RunPhaseExecuting,
		ApprovalState:  domain.ApprovalStateNone,
		SandboxConfig:  &domain.SandboxConfig{Mode: domain.SandboxModeTracking, AutoApply: func() *bool { value := true; return &value }()},
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	runDir, err := runstate.RunDir(root, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := os.ReadFile(filepath.Join("..", "adapters", "runner", "codecs", "testdata", "codex_rollout_trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sessions.onCreate = func() {
		rollout := filepath.Join(runDir, "codex", "sessions", "2026", "07", "26", "rollout-tracking.jsonl")
		if err := os.MkdirAll(filepath.Dir(rollout), 0o755); err != nil {
			t.Errorf("mkdir rollout: %v", err)
			return
		}
		if err := os.WriteFile(rollout, trace, 0o600); err != nil {
			t.Errorf("write rollout: %v", err)
		}
	}

	svc.executeInteractiveRun(ctx, run, task, "perform tracked work", nil)
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want complete (%s)", got.Status, got.ErrorMsg)
	}
	if got.SandboxID == nil {
		t.Fatal("tracking interactive run did not persist sandbox id")
	}
	if got.ChangedFiles != 2 || got.TotalSizeBytes != 3072 || got.DiffPath != "/api/v1/sandboxes/tracking/diff" || got.CommitHash != "tracking-commit" {
		t.Fatalf("tracking attribution = files=%d bytes=%d diff=%q commit=%q", got.ChangedFiles, got.TotalSizeBytes, got.DiffPath, got.CommitHash)
	}
	if provider.ApplyAtRunEndCallCount() != 1 {
		t.Fatalf("apply calls = %d, want 1", provider.ApplyAtRunEndCallCount())
	}
	created := provider.CreateRequests()
	if len(created) != 1 || created[0].Behavior == nil || created[0].Behavior.Mode != domain.SandboxModeTracking {
		t.Fatalf("sandbox create did not use tracking mode: %+v", created)
	}
}

// TestStopInteractiveRun_EscalationLadderAndSingleFinalize drives Stop through the
// live-driver path: the coordinator is cancelled and drained, the session is torn
// down interrupt-then-delete, and the run is finalized Cancelled exactly once.
func TestStopInteractiveRun_EscalationLadderAndSingleFinalize(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithInteractiveSessions(sessions), WithEvents(eventStore), WithRunStateRoot(t.TempDir()))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "agent-sess", "wc-1")

	// Register a live driver that mimics a coordinator: it exits when cancelled.
	driverCtx, driver := svc.interactiveDrivers.register(context.Background(), run.ID)
	exited := make(chan struct{})
	go func() {
		<-driverCtx.Done()
		svc.interactiveDrivers.finish(run.ID, driver)
		close(exited)
	}()

	if err := svc.StopRun(ctx, run.ID); err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("live coordinator was not cancelled+drained by Stop")
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusCancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}
	// Soft-then-hard escalation: interrupt precedes delete.
	calls := sessions.callLog()
	if len(calls) != 2 || calls[0] != "interrupt" || calls[1] != "delete" {
		t.Fatalf("expected [interrupt delete], got %v", calls)
	}
	if svc.interactiveDrivers.has(run.ID) {
		t.Error("driver should be deregistered after Stop")
	}
}

// TestStopInteractiveRun_NaturalCompletionRaceNoDoubleFinalize verifies that when
// the coordinator finalizes the run (natural completion) during the Stop
// hand-off, Stop yields — it does not overwrite Complete with Cancelled.
func TestStopInteractiveRun_NaturalCompletionRaceNoDoubleFinalize(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithRunStateRoot(t.TempDir()))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "agent-sess", "wc-2")

	// The driver simulates the coordinator winning the race: on cancellation it
	// finalizes the run Complete (bypassing the state machine, as the real
	// coordinator does) before signalling done.
	driverCtx, driver := svc.interactiveDrivers.register(context.Background(), run.ID)
	go func() {
		<-driverCtx.Done()
		cur, _ := repos.Runs.Get(context.Background(), run.ID)
		now := time.Now()
		cur.Status = domain.RunStatusComplete
		cur.Phase = domain.RunPhaseCompleted
		cur.EndedAt = &now
		_ = repos.Runs.Update(context.Background(), cur)
		svc.interactiveDrivers.finish(run.ID, driver)
	}()

	if err := svc.StopRun(ctx, run.ID); err != nil {
		t.Fatalf("StopRun: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want complete (Stop must not overwrite a natural completion)", got.Status)
	}
}

// TestStopInteractiveRun_NoLiveDriver covers stopping an interactive run with no
// live coordinator (e.g. recovered but not reattached): the session is still torn
// down and the run finalized Cancelled.
func TestStopInteractiveRun_NoLiveDriver(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithRunStateRoot(t.TempDir()))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "agent-sess", "wc-3")

	if err := svc.StopRun(ctx, run.ID); err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	got, _ := repos.Runs.Get(ctx, run.ID)
	if got.Status != domain.RunStatusCancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}
	calls := sessions.callLog()
	if len(calls) != 2 || calls[0] != "interrupt" || calls[1] != "delete" {
		t.Fatalf("expected [interrupt delete], got %v", calls)
	}
}

// TestContinueInteractiveRun_RoutesToSendPromptNeverRespawn verifies interactive
// Continue types the follow-up into the live session with the run-scoped source
// attribution, reactivates the run, and reattaches a driver — never respawning a
// process (no new session is created).
func TestContinueInteractiveRun_RoutesToSendPromptNeverRespawn(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// A replay runner exposes the transcript parser the reattach tailer needs.
	registry := runner.NewRegistry()
	if err := registry.Register(mocks.NewTranscriptReplayRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithInteractiveSessions(sessions), WithEvents(eventStore), WithRunners(registry))
	task := interactiveTestTask(t, svc)

	// A completed interactive run whose transcript exists (non-terminal, so the
	// reattached tailer blocks polling for the new turn rather than finalizing).
	transcript := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(transcript, []byte("{\"type\":\"noise\"}\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusComplete, "agent-sess", "wc-4")
	run.TranscriptPath = transcript
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("update run: %v", err)
	}
	// Ensure any reattached driver is stopped at test end.
	t.Cleanup(func() { svc.interactiveDrivers.cancelAndWait(run.ID) })

	out, err := svc.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: "please continue the task"})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}
	if out.Status != domain.RunStatusRunning {
		t.Fatalf("status = %s, want running", out.Status)
	}

	calls := sessions.callLog()
	sawPrompt := false
	for _, c := range calls {
		if c == "sendprompt" {
			sawPrompt = true
		}
		if c == "create" {
			t.Fatal("interactive Continue must never create a new session (no respawn)")
		}
	}
	if !sawPrompt {
		t.Fatalf("expected sendprompt, calls=%v", calls)
	}
	sessions.mu.Lock()
	src := append([]string(nil), sessions.promptSrc...)
	text := append([]string(nil), sessions.promptText...)
	sessions.mu.Unlock()
	wantSrc := "agent-manager:run-" + run.ID.String()
	if len(src) != 1 || src[0] != wantSrc {
		t.Errorf("prompt source = %v, want [%s]", src, wantSrc)
	}
	if len(text) != 1 || text[0] != "please continue the task" {
		t.Errorf("prompt text = %v", text)
	}
	if !svc.interactiveDrivers.has(run.ID) {
		t.Error("a live driver should be registered to drive the continuation turn")
	}
}

// TestContinueInteractiveRun_SessionGone rejects a continuation when the live
// web-console session has been torn down, with an actionable error.
func TestContinueInteractiveRun_SessionGone(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sessions := newRecordingSessions()
	sessions.getNotFound = true
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithInteractiveSessions(sessions), WithEvents(eventStore))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusComplete, "agent-sess", "wc-dead")

	_, err := svc.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: "keep going"})
	if err == nil {
		t.Fatal("expected error when the live session is gone")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no longer exists") {
		t.Errorf("error should explain the session is gone, got %q", err.Error())
	}
	// The run must be left in its terminal state, not flipped to running.
	got, _ := repos.Runs.Get(ctx, run.ID)
	if got.Status != domain.RunStatusComplete {
		t.Errorf("run status = %s, want complete (unchanged)", got.Status)
	}
	if sawSendPrompt(sessions) {
		t.Error("no prompt should be sent when the session is gone")
	}
}

func sawSendPrompt(s *recordingSessions) bool {
	for _, c := range s.callLog() {
		if c == "sendprompt" {
			return true
		}
	}
	return false
}

func TestInteractiveInitialPrompt(t *testing.T) {
	cases := []struct {
		name   string
		system string
		user   string
		want   string
	}{
		{"user-only (no attachments)", "", "do the task", "do the task"},
		{"system + user combined", "instructions", "context data", "instructions\n\ncontext data"},
		{"system-only", "instructions", "", "instructions"},
		{"trims surrounding whitespace", "  sys  ", "  usr  ", "sys\n\nusr"},
		{"both empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := interactiveInitialPrompt(tc.system, tc.user); got != tc.want {
				t.Errorf("interactiveInitialPrompt(%q,%q) = %q, want %q", tc.system, tc.user, got, tc.want)
			}
		})
	}
}
