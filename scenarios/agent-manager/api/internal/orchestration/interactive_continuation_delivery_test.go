package orchestration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
)

type nativeContinuationReceiptFixture struct {
	*recordingSessions
	thread, session         string
	receipt                 webconsole.PromptSubmission
	err                     error
	beforeReceipt           func()
	nativeCalls             int
	activeTurn              string
	owner                   string
	replaceAfterObservation bool
}

func (s *nativeContinuationReceiptFixture) GetSession(ctx context.Context, id string) (webconsole.SessionInfo, error) {
	info, err := s.recordingSessions.GetSession(ctx, id)
	info.LaunchMode, info.ControlMode = "codex_app_server", "native_capable"
	info.NativeOwner, info.NativeThreadID = s.owner, s.thread
	return info, err
}
func (s *nativeContinuationReceiptFixture) ObserveNativeAgent(context.Context, string, string) (webconsole.NativeAgentObservation, error) {
	observation := webconsole.NativeAgentObservation{State: webconsole.NativeAgentBusy, ActiveTurnID: s.activeTurn, Generation: s.owner}
	if s.replaceAfterObservation {
		s.owner = "replacement-owner"
	}
	return observation, nil
}
func (s *nativeContinuationReceiptFixture) SendPromptForNativeOwner(ctx context.Context, session, thread, owner, prompt, source string) (webconsole.PromptSubmission, error) {
	if owner != s.owner {
		return webconsole.PromptSubmission{}, webconsole.ErrNativePromptRefused
	}
	return s.SendPromptForNativeSession(ctx, session, thread, prompt, source)
}

func (s *nativeContinuationReceiptFixture) SendPromptForNativeSession(ctx context.Context, session, thread, prompt, source string) (webconsole.PromptSubmission, error) {
	if session != s.session || thread != s.thread {
		return webconsole.PromptSubmission{}, errors.New("wrong receiving session/thread")
	}
	s.nativeCalls++
	if s.beforeReceipt != nil {
		s.beforeReceipt()
	}
	return s.receipt, s.err
}

func TestNativeContinuationCommitsRunningOnlyAfterVerifiedDelivery(t *testing.T) {
	for _, tc := range []struct {
		name     string
		verified bool
		err      error
	}{
		{name: "verified", verified: true},
		{name: "unverified"},
		{name: "receiver generation unavailable", err: webconsole.ErrNativeAgentUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			registry := runner.NewRegistry()
			if err := registry.Register(mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)); err != nil {
				t.Fatal(err)
			}
			sessions := &nativeContinuationReceiptFixture{recordingSessions: newRecordingSessions(), owner: "fixture-owner", session: "web-fixture", thread: "thread-fixture", receipt: webconsole.PromptSubmission{Verified: tc.verified}, err: tc.err}
			svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithInteractiveSessions(sessions), WithRunners(registry), WithRunStateRoot(t.TempDir()))
			task := interactiveTestTask(t, svc)
			run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusComplete, sessions.thread, sessions.session)
			run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
			run.Result = &domain.RunResult{Success: true, FinalOutput: "prior outcome"}
			run.TranscriptPath = filepath.Join(t.TempDir(), "transcript.jsonl")
			if err := os.WriteFile(run.TranscriptPath, []byte("message:still observing\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := repos.Runs.Update(ctx, run); err != nil {
				t.Fatal(err)
			}
			run, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			// ContinueRun/WakeRun establish this exact task admission before
			// calling the shared delivery boundary exercised below.
			if err := svc.ensureCurrentTaskRun(ctx, run); err != nil {
				t.Fatal(err)
			}
			sessions.beforeReceipt = func() {
				pending, err := repos.Runs.Get(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if pending.Status != domain.RunStatusComplete || pending.InteractiveInvocationStartedAt != nil || pending.Result == nil || pending.Result.FinalOutput != "prior outcome" {
					t.Fatalf("unverified delivery was projected as running: %+v", pending)
				}
			}
			t.Cleanup(func() { svc.interactiveDrivers.cancelAndWait(run.ID) })
			_, err = svc.continueInteractiveRunWithVerifiedSession(ctx, run, "follow up", false, interactive.TailContext{RunDir: t.TempDir()}, "fixture-owner")
			stored, readErr := repos.Runs.Get(ctx, run.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.verified {
				if err != nil || stored.Status != domain.RunStatusRunning || stored.InteractiveInvocationStartedAt == nil || stored.Result != nil {
					t.Fatalf("verified receipt did not admit new invocation: run=%+v err=%v", stored, err)
				}
			} else if err == nil || stored.Status != domain.RunStatusComplete || stored.InteractiveInvocationStartedAt != nil || stored.Result == nil || stored.Result.FinalOutput != "prior outcome" {
				t.Fatalf("failed delivery changed prior invocation: run=%+v err=%v", stored, err)
			}
			if sessions.nativeCalls != 1 || len(sessions.callLog()) != 0 {
				t.Fatalf("delivery fell through to legacy input: native=%d legacy=%v", sessions.nativeCalls, sessions.callLog())
			}
		})
	}
}

func TestNativeContinuationBusyRequiresExactActiveTurn(t *testing.T) {
	for _, turn := range []string{"", "active-turn"} {
		sessions := &nativeContinuationReceiptFixture{recordingSessions: newRecordingSessions(), activeTurn: turn, owner: "fixture-owner", session: "web-fixture", thread: "thread-fixture"}
		svc := New(nil, nil, nil, WithInteractiveSessions(sessions))
		run := &domain.Run{WebConsoleSessionID: "web-fixture", SessionID: "thread-fixture", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
		err := svc.verifyInteractiveSession(t.Context(), run)
		if (err == nil) != (turn != "") {
			t.Fatalf("busy turn=%q verification=%v", turn, err)
		}
		if sessions.nativeCalls != 0 || len(sessions.callLog()) != 0 {
			t.Fatal("verification submitted input")
		}
	}
}

func TestNativeContinuationPinsOwnerAcrossVerificationAndDelivery(t *testing.T) {
	sessions := &nativeContinuationReceiptFixture{recordingSessions: newRecordingSessions(), activeTurn: "active-turn", owner: "original-owner", session: "web-fixture", thread: "thread-fixture", replaceAfterObservation: true}
	svc := New(nil, nil, nil, WithInteractiveSessions(sessions))
	run := &domain.Run{WebConsoleSessionID: sessions.session, SessionID: sessions.thread, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Until: "retain exact native goal"}}
	owner, err := svc.verifyInteractiveSessionOwner(t.Context(), run)
	if err != nil || owner != "original-owner" {
		t.Fatalf("initial binding = %q %v", owner, err)
	}
	if _, err := svc.sendBoundInteractivePromptForOwner(t.Context(), run, "continue", owner); !errors.Is(err, webconsole.ErrNativePromptRefused) {
		t.Fatalf("replacement owner accepted prompt: %v", err)
	}
	if handled, err := svc.setCodexNativeGoalStatusForOwner(t.Context(), run, "active", run.ResolvedConfig.Until, owner); !handled || err == nil {
		t.Fatal("replacement owner accepted goal reinstall")
	}
	if sessions.nativeCalls != 0 || len(sessions.callLog()) != 0 {
		t.Fatal("replacement owner received input")
	}
}

func TestLegacyContinuationPreservesProviderSpecificSessionContract(t *testing.T) {
	sessions := newRecordingSessions()
	sessions.nativeErr = errors.New("Codex liveness is unavailable for this provider")
	svc := New(nil, nil, nil, WithInteractiveSessions(sessions))
	run := &domain.Run{WebConsoleSessionID: "legacy-web-session", SessionID: "legacy-conversation", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}}
	if err := svc.verifyInteractiveSession(t.Context(), run); err != nil {
		t.Fatalf("legacy provider acquired a Codex dependency: %v", err)
	}
	if sessions.nativeReads != 0 || len(sessions.callLog()) != 0 {
		t.Fatal("legacy verification invoked native control or input")
	}
}

func TestNativeGoalReinstallRefusesMissingCapabilityOwner(t *testing.T) {
	sessions := newRecordingSessions()
	svc := New(nil, nil, nil, WithInteractiveSessions(sessions))
	run := &domain.Run{WebConsoleSessionID: "web-fixture", SessionID: "thread-fixture", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Until: "retain the goal"}}
	if err := svc.reinstallNativeGoal(t.Context(), run, "original-owner"); err == nil {
		t.Fatal("missing capability owner was silently accepted")
	}
	if len(sessions.callLog()) != 0 {
		t.Fatal("missing capability owner caused provider input")
	}
}
