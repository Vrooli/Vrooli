package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/rolepolicy"
	"github.com/google/uuid"
)

type browserPaneProcessRunner struct {
	*runner.MockRunner
	pid atomic.Int32
}

type browserPaneRoleResolver struct{}

func (browserPaneRoleResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: "mock-model"}, nil
}

func browserPaneRolePolicyOption(t *testing.T) Option {
	t.Helper()
	path := filepath.Join(t.TempDir(), "role-policy.json")
	content := `{"schemaVersion":1,"metadata":{"catalogId":"pane-test","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"claude-code","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	return WithRolePolicyState(state, browserPaneRoleResolver{})
}

func (r *browserPaneProcessRunner) PID(uuid.UUID) int { return int(r.pid.Load()) }

func TestBrowserPaneControlYieldsAndReturnsOnlyTheExactInteractiveRun(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	sessions := newRecordingSessions()
	baseRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	baseRunner.StopFunc = func(context.Context, uuid.UUID) error { return nil }
	processRunner := &browserPaneProcessRunner{MockRunner: baseRunner}
	runners := runner.NewRegistry()
	if err := runners.Register(processRunner); err != nil {
		t.Fatal(err)
	}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInteractiveSessions(sessions), WithIdempotency(repos.Idempotency), WithRunners(runners), WithRunStateRoot(t.TempDir()), browserPaneRolePolicyOption(t))
	awaits := NewAwaitRegistry(svc, NewBrowserPaneControlWaiter())
	t.Cleanup(awaits.Stop)
	svc.SetAwaitRegistry(awaits)
	task := interactiveTestTask(t, svc)
	run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "exact-thread", "exact-wc-session")
	run.OwnerSubject = "human-owner"
	run.ResolvedConfig.Features.BrowserCapability = &browsercapability.Capability{Enabled: true, GrantReceipt: "persisted-grant", Permissions: browsercapability.PermissionProfileSpec{ProfileID: "persisted-profile", View: true, Control: true}}
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	binding := BrowserPaneControlBinding{RunID: run.ID, OwnerSubject: "human-owner", WebConsoleSessionID: "exact-wc-session", ThreadID: "exact-thread", BASSessionID: "exact-bas", GrantReceipt: "persisted-grant", ProfileID: "persisted-profile"}

	yield, err := svc.YieldBrowserPaneControl(ctx, binding)
	if err != nil || yield.Receipt == "" || !yield.ExpiresAt.After(time.Now()) || yield.ExpiresAt.After(time.Now().Add(BrowserPaneControlLeaseTTL+time.Second)) {
		t.Fatalf("yield=%+v err=%v", yield, err)
	}
	parked, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Producer != ProducerBrowserPaneControl {
		t.Fatalf("agent was not durably yielded: run=%+v err=%v", parked, err)
	}
	if _, err := svc.YieldBrowserPaneControl(ctx, func() BrowserPaneControlBinding { b := binding; b.BASSessionID = "competing-bas"; return b }()); err == nil {
		t.Fatal("a competing BAS session joined an existing run yield")
	}
	if err := svc.ReturnBrowserPaneControl(ctx, binding, "wrong-receipt"); err == nil {
		t.Fatal("wrong receipt returned the run")
	}
	if err := svc.ReturnBrowserPaneControl(ctx, binding, yield.Receipt); err != nil {
		t.Fatalf("exact hand-back failed: %v", err)
	}
	running, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || running.Status != domain.RunStatusRunning || running.ID != run.ID || running.SessionID != binding.ThreadID || running.WebConsoleSessionID != binding.WebConsoleSessionID {
		t.Fatalf("hand-back resumed the wrong owner: run=%+v err=%v", running, err)
	}
	if len(sessions.callLog()) == 0 {
		t.Fatal("same retained interactive session was not continued after hand-back")
	}
	if err := svc.ReturnBrowserPaneControl(ctx, binding, yield.Receipt); err != nil {
		t.Fatalf("lost return response was not idempotent: %v", err)
	}
	if !strings.Contains(strings.Join(sessions.promptText, "\n"), "human released browser control") {
		t.Fatalf("run did not receive a typed hand-back reason: %v", sessions.promptText)
	}
}
