package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"
)

func TestInteractivePromptTransportUnknownRetainsIdempotencyReservation(t *testing.T) {
	ctx := context.Background()
	db, repos, _ := testutil.SetupTestDBWithRepos(t)
	sessions := newRecordingSessions()
	sessions.promptErr = errors.New("submit result lost after paste")
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithIdempotency(repos.Idempotency))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "provider-session", "wc-1")
	req := ContinueRunRequest{RunID: run.ID, Message: "D5", IdempotencyKey: "same-handoff"}
	if _, err := svc.ContinueRun(ctx, req); err == nil || !strings.Contains(err.Error(), "outcome unknown") || domain.IsPreEffectRefusal(err) {
		t.Fatalf("partial transport did not surface unknown (or claimed no effects): %v", err)
	}
	updatedRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || updatedRun.InteractiveInvocationStartedAt == nil {
		t.Fatalf("ambiguous prompt outcome lost its pre-effect invocation fence: run=%+v err=%v", updatedRun, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE idempotency_records SET expires_at = '2000-01-01 00:00:00' WHERE key = ?`, req.IdempotencyKey); err != nil {
		t.Fatalf("expire ambiguous receipt fixture: %v", err)
	}
	if removed, err := repos.Idempotency.CleanupExpired(ctx); err != nil || removed != 0 {
		t.Fatalf("expired ambiguous receipt was cleaned up: removed %d, err %v", removed, err)
	}
	if _, err := svc.ContinueRun(ctx, req); err == nil {
		t.Fatal("duplicate handoff was resent after the ambiguous receipt expired")
	}
	if calls := sessions.callLog(); len(calls) != 1 || calls[0] != "sendprompt" {
		t.Fatalf("duplicate handoff effects: %v", calls)
	}
}

func TestContinueRunRefusesConfiguredRepositoryWithoutContinuationReceipts(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithIdempotency(legacyOnlyIdempotency{}))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "provider-session", "wc-no-typed-receipt")
	_, err := svc.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: "must refuse", IdempotencyKey: "legacy-repository"})
	if err == nil || !strings.Contains(err.Error(), "does not support durable continuation receipts") {
		t.Fatalf("configured legacy repository was not explicitly refused: %v", err)
	}
	if calls := sessions.callLog(); len(calls) != 0 {
		t.Fatalf("configured legacy repository allowed continuation effects: %v", calls)
	}
}

func TestRunningSessionContinuationRecordsDirectiveAsUserMessage(t *testing.T) {
	ctx := context.Background()
	db, repos, eventStore := testutil.SetupTestDBWithRepos(t)
	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithEvents(eventStore), WithIdempotency(repos.Idempotency))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "provider-session", "wc-1")

	req := ContinueRunRequest{RunID: run.ID, Message: "D2: run J02 only", IdempotencyKey: "handoff-d2"}
	if _, err := svc.ContinueRun(ctx, req); err != nil {
		t.Fatal(err)
	}

	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	var recorded bool
	for _, ev := range events {
		if msg, ok := ev.Data.(*domain.MessageEventData); ok && msg.Role == "user" && msg.Content == "D2: run J02 only" {
			recorded = true
		}
	}
	if !recorded {
		t.Fatalf("directive not recorded as a user message: %+v", events)
	}
	receipt, err := repos.Idempotency.Check(ctx, "handoff-d2")
	if err != nil || receipt == nil || receipt.Status != domain.IdempotencyStatusComplete {
		t.Fatalf("receipt = %+v err=%v, want a completed continuation receipt", receipt, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE idempotency_records SET expires_at = '2000-01-01 00:00:00' WHERE key = ?`, req.IdempotencyKey); err != nil {
		t.Fatalf("expire completed receipt fixture: %v", err)
	}
	if _, err := svc.ContinueRun(ctx, req); err != nil {
		t.Fatalf("replay completed continuation after expiry: %v", err)
	}
	if calls := sessions.callLog(); len(calls) != 1 || calls[0] != "sendprompt" {
		t.Fatalf("expired completed receipt caused another delivery: %v", calls)
	}
}

func TestRunningSessionContinuationNotSubmittedIsAnErrorWithoutReceipt(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	sessions := newRecordingSessions()
	sessions.promptResult = &webconsole.PromptSubmission{EnterPresses: 3}
	sessions.promptErr = webconsole.ErrPromptNotSubmitted
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithEvents(eventStore), WithIdempotency(repos.Idempotency))
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "provider-session", "wc-stuck")
	req := ContinueRunRequest{RunID: run.ID, Message: "D1 is recorded in E17.md", IdempotencyKey: "handoff-d1"}

	_, err := svc.ContinueRun(ctx, req)
	if err == nil || !strings.Contains(err.Error(), "not submitted") || !strings.Contains(err.Error(), "wc-stuck") {
		t.Fatalf("err = %v, want a not-submitted error naming the session", err)
	}
	if domain.IsPreEffectRefusal(err) {
		t.Fatal("a pasted-but-unsent directive was reported as refused before effects")
	}
	receipt, err := repos.Idempotency.Check(ctx, "handoff-d1")
	if err != nil || receipt == nil || receipt.Status == domain.IdempotencyStatusComplete {
		t.Fatalf("receipt = %+v err=%v, want the reservation held, not completed", receipt, err)
	}
	if _, err := svc.ContinueRun(ctx, req); err == nil {
		t.Fatal("same-key retry was accepted while the directive sits in the composer")
	}
	if calls := sessions.callLog(); len(calls) != 1 {
		t.Fatalf("retry pasted again: %v", calls)
	}
	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if msg, ok := ev.Data.(*domain.MessageEventData); ok && msg.Role == "user" {
			t.Fatalf("unsent directive recorded as a delivered user message: %+v", msg)
		}
	}
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

func TestInitialPromptWithUntilDeliversEngineContract(t *testing.T) {
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{Until: "all phases complete"}}
	got := initialPromptWithUntil(run, "implement the plan")
	if !strings.Contains(got, "implement the plan") || !strings.Contains(got, "all phases complete") {
		t.Fatalf("prompt=%q; expected task and until contract", got)
	}
}

func TestObservedGoalMatchesLaunchNormalizedUntilWithoutWhitespaceCollapse(t *testing.T) {
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{Until: "finish  every\r\nphase\tcarefully"}}
	for _, tc := range []struct {
		name, objective string
		want            bool
	}{
		{name: "normalized launch value", objective: "finish  every phase\tcarefully", want: true},
		{name: "raw configured value", objective: "finish  every\r\nphase\tcarefully", want: true},
		{name: "newline suffix", objective: "finish  every phase\tcarefully\nattached prompt", want: true},
		{name: "space collapse rejected", objective: "finish every phase\tcarefully"},
		{name: "tab replacement rejected", objective: "finish  every phase carefully"},
		{name: "term replacement rejected", objective: "finish  every different\tcarefully"},
		{name: "delimiter required", objective: "finish  every phase\tcarefully-extra"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := observedGoalMatchesRun(run, tc.objective); got != tc.want {
				t.Fatalf("observedGoalMatchesRun(%q) = %v, want %v", tc.objective, got, tc.want)
			}
		})
	}
}

type rejectingRunUpdateRepository struct {
	repository.RunRepository
	err error
}

func (r rejectingRunUpdateRepository) Update(context.Context, *domain.Run) error { return r.err }

func TestStopInteractiveRunRevocationFailurePreventsSessionRetirement(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	sessions := newRecordingSessions()
	taskSvc := New(repos.Profiles, repos.Tasks, repos.Runs)
	task := interactiveTestTask(t, taskSvc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "agent-sess", "wc-revoke-fail")
	run.IdentityTokenHash = "test-child-hash"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("identity revocation store unavailable")
	svc := New(repos.Profiles, repos.Tasks, rejectingRunUpdateRepository{RunRepository: repos.Runs, err: failure}, WithInteractiveSessions(sessions))
	err := svc.StopRun(ctx, run.ID)
	if !errors.Is(err, failure) {
		t.Fatalf("StopRun error = %v, want revocation failure", err)
	}
	if got := sessions.callLog(); len(got) != 0 {
		t.Fatalf("session was touched before durable revocation: %v", got)
	}
	stored, getErr := repos.Runs.Get(ctx, run.ID)
	if getErr != nil || stored.Status != domain.RunStatusRunning || stored.IdentityTokenRevokedAt != nil {
		t.Fatalf("run state after failed revocation = %+v, err=%v", stored, getErr)
	}
}

func TestResolvedRunModeRejectsProtectedInPlace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode      domain.RunMode
		cfg       *domain.SandboxConfig
		wantError bool
	}{
		{"protected in place", domain.RunModeInPlace, &domain.SandboxConfig{Mode: domain.SandboxModeProtected}, true},
		{"default protected in place", domain.RunModeInPlace, &domain.SandboxConfig{}, true},
		{"protected sandbox", domain.RunModeSandboxed, &domain.SandboxConfig{Mode: domain.SandboxModeProtected}, false},
		{"off in place", domain.RunModeInPlace, &domain.SandboxConfig{Mode: domain.SandboxModeOff}, false},
		{"tracking in place", domain.RunModeInPlace, &domain.SandboxConfig{Mode: domain.SandboxModeTracking}, false},
		{"no config", domain.RunModeInPlace, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateResolvedRunMode(tc.mode, tc.cfg)
			if (err != nil) != tc.wantError {
				t.Fatalf("validateResolvedRunMode() = %v; want error %v", err, tc.wantError)
			}
		})
	}
}
