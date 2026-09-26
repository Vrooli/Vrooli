// Tests for ValidateRunOutcome — silent-launch failure detection.

package phases

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// filterableEventStore lets countMessageEvents see configurable response
// shapes without touching the SQLite backend.
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

func newSilentLaunchFixture(mode domain.SandboxMode, runMode domain.RunMode) (*domain.Run, *filterableEventStore, Deps) {
	store := &filterableEventStore{}
	run := &domain.Run{
		ID:      uuid.New(),
		RunMode: runMode,
		ResolvedConfig: &domain.RunConfig{
			SandboxConfig: &domain.SandboxConfig{Mode: mode},
		},
	}
	return run, store, Deps{Events: store, Levers: config.DefaultLevers()}
}

// TestValidateRunOutcome_DemotesSilentLaunchFailure: protected sandbox +
// 100ms wall + 0 message events + Success=true → demoted to FAILED.
func TestValidateRunOutcome_DemotesSilentLaunchFailure(t *testing.T) {
	run, _, deps := newSilentLaunchFixture(domain.SandboxModeProtected, domain.RunModeSandboxed)
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}

	out := ValidateRunOutcome(context.Background(), ValidateOutcomeInput{
		Deps:   deps,
		Run:    run,
		Result: result,
	})

	if out.Result.Success {
		t.Error("Success should be flipped to false on silent launch failure")
	}
	if out.ExecErr == nil {
		t.Fatal("execErr should be set so classifyOutcome routes to FAILED")
	}
	sbxErr, ok := out.ExecErr.(*domain.SandboxError)
	if !ok {
		t.Fatalf("execErr type = %T; want *domain.SandboxError", out.ExecErr)
	}
	if sbxErr.Code() != domain.ErrCodeSandboxLaunchFailed {
		t.Errorf("Code = %s; want %s", sbxErr.Code(), domain.ErrCodeSandboxLaunchFailed)
	}
}

func TestValidateRunOutcome_KeepsHonestSuccess(t *testing.T) {
	run, store, deps := newSilentLaunchFixture(domain.SandboxModeProtected, domain.RunModeSandboxed)
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 30 * time.Second,
	}
	_ = store.Append(context.Background(), run.ID,
		domain.NewMessageEvent(run.ID, "assistant", "hello"),
	)

	out := ValidateRunOutcome(context.Background(), ValidateOutcomeInput{
		Deps:   deps,
		Run:    run,
		Result: result,
	})

	if !out.Result.Success {
		t.Error("Success should remain true for an honest run")
	}
	if out.ExecErr != nil {
		t.Errorf("execErr should be nil; got %v", out.ExecErr)
	}
}

func TestValidateRunOutcome_KeepsFastSuccessWithMessages(t *testing.T) {
	run, store, deps := newSilentLaunchFixture(domain.SandboxModeProtected, domain.RunModeSandboxed)
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 500 * time.Millisecond,
	}
	_ = store.Append(context.Background(), run.ID,
		domain.NewMessageEvent(run.ID, "assistant", "hi"),
	)

	out := ValidateRunOutcome(context.Background(), ValidateOutcomeInput{
		Deps:   deps,
		Run:    run,
		Result: result,
	})

	if !out.Result.Success {
		t.Error("fast run with message events must NOT be demoted")
	}
}

func TestValidateRunOutcome_IgnoresInPlace(t *testing.T) {
	run, _, deps := newSilentLaunchFixture(domain.SandboxModeUnspecified, domain.RunModeInPlace)
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 50 * time.Millisecond,
	}

	out := ValidateRunOutcome(context.Background(), ValidateOutcomeInput{
		Deps:   deps,
		Run:    run,
		Result: result,
	})

	if !out.Result.Success {
		t.Error("in-place runs must not be demoted by ValidateRunOutcome")
	}
	if out.ExecErr != nil {
		t.Errorf("execErr should be nil; got %v", out.ExecErr)
	}
}

func TestValidateRunOutcome_IgnoresFailedResult(t *testing.T) {
	run, _, deps := newSilentLaunchFixture(domain.SandboxModeProtected, domain.RunModeSandboxed)
	result := &runner.ExecuteResult{
		Success:      false,
		ExitCode:     1,
		Duration:     100 * time.Millisecond,
		ErrorMessage: "boom",
	}

	out := ValidateRunOutcome(context.Background(), ValidateOutcomeInput{
		Deps:   deps,
		Run:    run,
		Result: result,
	})

	if out.Result.ErrorMessage != "boom" {
		t.Errorf("ErrorMessage was rewritten to %q; want preserved", out.Result.ErrorMessage)
	}
}
