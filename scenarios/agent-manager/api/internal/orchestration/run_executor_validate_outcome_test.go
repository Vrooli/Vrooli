// Tests for the validateRunOutcome categorizer added 2026-04-28 to demote
// silent-launch failures from masquerading as clean successes. See
// docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md, Phase D.

package orchestration

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// filterableEventStore lets validateRunOutcome's countMessageEvents see
// configurable response shapes without touching the SQLite backend.
type filterableEventStore struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (f *filterableEventStore) Append(_ context.Context, _ uuid.UUID, evts ...*domain.RunEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, evts...)
	return nil
}

func (f *filterableEventStore) Get(_ context.Context, _ uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := map[domain.RunEventType]bool{}
	for _, t := range opts.EventTypes {
		want[t] = true
	}
	var out []*domain.RunEvent
	for _, e := range f.events {
		if len(want) > 0 && !want[e.EventType] {
			continue
		}
		out = append(out, e)
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	return out, nil
}

func (f *filterableEventStore) Stream(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error) {
	ch := make(chan *domain.RunEvent)
	close(ch)
	return ch, nil
}
func (f *filterableEventStore) Count(context.Context, uuid.UUID) (int64, error) { return 0, nil }
func (f *filterableEventStore) Delete(context.Context, uuid.UUID) error         { return nil }

func newSilentLaunchExecutor(t *testing.T, mode domain.SandboxMode, runMode domain.RunMode) (*RunExecutor, *filterableEventStore) {
	t.Helper()
	store := &filterableEventStore{}
	run := &domain.Run{
		ID:      uuid.New(),
		RunMode: runMode,
		ResolvedConfig: &domain.RunConfig{
			SandboxConfig: &domain.SandboxConfig{Mode: mode},
		},
	}
	return &RunExecutor{
		run:    run,
		events: store,
	}, store
}

// TestValidateRunOutcome_DemotesSilentLaunchFailure: protected sandbox +
// 100ms wall + 0 message events + Success=true → demoted to FAILED with
// SANDBOX_LAUNCH_FAILED. This is the swarm-manager 134ms-no-output bug.
func TestValidateRunOutcome_DemotesSilentLaunchFailure(t *testing.T) {
	exec, _ := newSilentLaunchExecutor(t, domain.SandboxModeProtected, domain.RunModeSandboxed)
	exec.result = &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}

	exec.validateRunOutcome(context.Background())

	if exec.result.Success {
		t.Error("Success should be flipped to false on silent launch failure")
	}
	if exec.execErr == nil {
		t.Fatal("execErr should be set so classifyOutcome routes to FAILED")
	}
	var sbxErr *domain.SandboxError
	if !asSandboxError(exec.execErr, &sbxErr) {
		t.Fatalf("execErr type = %T; want *domain.SandboxError", exec.execErr)
	}
	if sbxErr.Code() != domain.ErrCodeSandboxLaunchFailed {
		t.Errorf("Code = %s; want %s", sbxErr.Code(), domain.ErrCodeSandboxLaunchFailed)
	}
}

// TestValidateRunOutcome_KeepsHonestSuccess: protected sandbox + 30s +
// 1+ message events + Success=true → preserved as success. Real claude
// runs always emit at least one assistant turn.
func TestValidateRunOutcome_KeepsHonestSuccess(t *testing.T) {
	exec, store := newSilentLaunchExecutor(t, domain.SandboxModeProtected, domain.RunModeSandboxed)
	exec.result = &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 30 * time.Second,
	}
	// Seed one message event so the categorizer's "no-output" rule misses.
	_ = store.Append(context.Background(), exec.run.ID,
		domain.NewMessageEvent(exec.run.ID, "assistant", "hello"),
	)

	exec.validateRunOutcome(context.Background())

	if !exec.result.Success {
		t.Error("Success should remain true for an honest run")
	}
	if exec.execErr != nil {
		t.Errorf("execErr should be nil; got %v", exec.execErr)
	}
}

// TestValidateRunOutcome_KeepsFastSuccessWithMessages: even a sub-2s
// runtime is fine if the run produced output (e.g. cached / preflight
// runs). The categorizer requires both signals.
func TestValidateRunOutcome_KeepsFastSuccessWithMessages(t *testing.T) {
	exec, store := newSilentLaunchExecutor(t, domain.SandboxModeProtected, domain.RunModeSandboxed)
	exec.result = &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 500 * time.Millisecond,
	}
	_ = store.Append(context.Background(), exec.run.ID,
		domain.NewMessageEvent(exec.run.ID, "assistant", "hi"),
	)

	exec.validateRunOutcome(context.Background())

	if !exec.result.Success {
		t.Error("fast run with message events must NOT be demoted")
	}
}

// TestValidateRunOutcome_IgnoresInPlace: in-place runs don't traverse
// bwrap, so the launch-failure shape doesn't apply. Categorizer must
// be a no-op.
func TestValidateRunOutcome_IgnoresInPlace(t *testing.T) {
	exec, _ := newSilentLaunchExecutor(t, domain.SandboxModeUnspecified, domain.RunModeInPlace)
	exec.result = &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 50 * time.Millisecond,
	}

	exec.validateRunOutcome(context.Background())

	if !exec.result.Success {
		t.Error("in-place runs must not be demoted by validateRunOutcome")
	}
	if exec.execErr != nil {
		t.Errorf("execErr should be nil; got %v", exec.execErr)
	}
}

// TestValidateRunOutcome_IgnoresFailedResult: validateRunOutcome only
// touches successful results — failures already route through
// handleFailure.
func TestValidateRunOutcome_IgnoresFailedResult(t *testing.T) {
	exec, _ := newSilentLaunchExecutor(t, domain.SandboxModeProtected, domain.RunModeSandboxed)
	exec.result = &runner.ExecuteResult{
		Success:      false,
		ExitCode:     1,
		Duration:     100 * time.Millisecond,
		ErrorMessage: "boom",
	}

	exec.validateRunOutcome(context.Background())

	if exec.result.ErrorMessage != "boom" {
		t.Errorf("ErrorMessage was rewritten to %q; want preserved", exec.result.ErrorMessage)
	}
}

// asSandboxError mirrors errors.As but returns a bool to keep the
// assertion readable above.
func asSandboxError(err error, target **domain.SandboxError) bool {
	if err == nil {
		return false
	}
	se, ok := err.(*domain.SandboxError)
	if !ok {
		return false
	}
	*target = se
	return true
}
