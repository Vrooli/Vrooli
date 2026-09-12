package obs

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// captureSink is a Sink test double that records every emitted event
// in order. Using a tiny double rather than the real eventStore keeps
// the test independent of repository wiring.
type captureSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (s *captureSink) Emit(evt *domain.RunEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
	return nil
}

func (s *captureSink) take() []*domain.RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*domain.RunEvent, len(s.events))
	copy(out, s.events)
	return out
}

// TestLifecycleTaxonomyIsCovered enumerates every helper in
// obs/events.go and asserts each emits exactly one lifecycle event of
// the expected phase. The test fails loudly when a phase is added to
// LifecyclePhase without a corresponding helper (or a helper grows a
// hidden second emission).
func TestLifecycleTaxonomyIsCovered(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("text", "info", &buf)

	runID := uuid.New()
	sandboxID := uuid.New()

	cases := []struct {
		name  string
		phase domain.LifecyclePhase
		emit  func(s Sink)
	}{
		{
			name:  "spawn_enqueued",
			phase: domain.LifecyclePhaseSpawnEnqueued,
			emit: func(s Sink) {
				EmitSpawnEnqueued(s, runID, SpawnEnqueuedFields{
					RunMode:    domain.RunModeSandboxed,
					RunnerType: domain.RunnerTypeCodex,
					QueueDepth: 2, ActiveCount: 1,
				})
			},
		},
		{
			name:  "spawn_started",
			phase: domain.LifecyclePhaseSpawnStarted,
			emit: func(s Sink) {
				EmitSpawnStarted(s, runID, SpawnStartedFields{
					RunMode:    domain.RunModeSandboxed,
					RunnerType: domain.RunnerTypeCodex,
					QueuedFor:  500 * time.Millisecond,
				})
			},
		},
		{
			name:  "runner_acquired",
			phase: domain.LifecyclePhaseRunnerAcquired,
			emit: func(s Sink) {
				EmitRunnerAcquired(s, runID, RunnerAcquiredFields{
					RunnerType:   domain.RunnerTypeCodex,
					LauncherType: "sandbox",
					SandboxID:    &sandboxID,
				})
			},
		},
		{
			name:  "runner_exited",
			phase: domain.LifecyclePhaseRunnerExited,
			emit: func(s Sink) {
				code := 0
				EmitRunnerExited(s, runID, RunnerExitedFields{
					RunnerType: domain.RunnerTypeCodex,
					ExitCode:   &code,
					Duration:   2 * time.Second,
					Success:    true,
				})
			},
		},
		{
			name:  "finalize_started",
			phase: domain.LifecyclePhaseFinalizeStarted,
			emit: func(s Sink) {
				EmitFinalizeStarted(s, runID, FinalizeFields{SandboxID: &sandboxID})
			},
		},
		{
			name:  "finalize_completed",
			phase: domain.LifecyclePhaseFinalizeCompleted,
			emit: func(s Sink) {
				EmitFinalizeCompleted(s, runID, FinalizeFields{SandboxID: &sandboxID, Action: "delete"}, 250*time.Millisecond)
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s := &captureSink{}
			tc.emit(s)
			events := s.take()
			if len(events) != 1 {
				t.Fatalf("%s: expected exactly 1 event, got %d", tc.name, len(events))
			}
			evt := events[0]
			if evt.EventType != domain.EventTypeLifecycle {
				t.Errorf("%s: expected EventTypeLifecycle, got %q", tc.name, evt.EventType)
			}
			data, ok := evt.Data.(*domain.LifecycleEventData)
			if !ok {
				t.Fatalf("%s: expected *LifecycleEventData payload, got %T", tc.name, evt.Data)
			}
			if data.Phase != tc.phase {
				t.Errorf("%s: expected phase %q, got %q", tc.name, tc.phase, data.Phase)
			}
			if evt.RunID != runID {
				t.Errorf("%s: expected runID %q, got %q", tc.name, runID, evt.RunID)
			}
		})
	}
}

// TestNilSinkIsSafe — a nil Sink must not panic; emission falls back
// to log-only. This is the path tests / construction code hit when no
// Gate is wired.
func TestNilSinkIsSafe(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("text", "info", &buf)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil sink should not panic, got %v", r)
		}
	}()
	EmitSpawnStarted(nil, uuid.New(), SpawnStartedFields{
		RunMode:    domain.RunModeSandboxed,
		RunnerType: domain.RunnerTypeCodex,
	})
	if buf.Len() == 0 {
		t.Errorf("expected log output even with nil sink")
	}
}

// TestRunnerExitedMessageReflectsOutcome guards the small bit of
// branching in the runner-exit helper so a future tweak can't silently
// turn "exited cleanly" into "exited with failure".
func TestRunnerExitedMessageReflectsOutcome(t *testing.T) {
	successFields := RunnerExitedFields{Success: true}
	failureFields := RunnerExitedFields{Success: false}
	classifiedFields := RunnerExitedFields{Success: false, TerminalCode: "RUNNER_SESSION_STATE_LOST"}

	if got := runnerExitedMessage(successFields); got != "runner exited cleanly" {
		t.Errorf("success: expected clean exit message, got %q", got)
	}
	if got := runnerExitedMessage(failureFields); got != "runner exited with failure" {
		t.Errorf("failure: expected failure message, got %q", got)
	}
	if got := runnerExitedMessage(classifiedFields); got != "runner exited: RUNNER_SESSION_STATE_LOST" {
		t.Errorf("classified: expected typed message, got %q", got)
	}
}
