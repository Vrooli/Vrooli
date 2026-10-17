package interactive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const timeoutRolloutOpenTurn = `{"type":"session_meta","payload":{"id":"codex-session"}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"t1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":15211,"cached_input_tokens":10880,"output_tokens":205}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":30666,"cached_input_tokens":21760,"output_tokens":415}}}}
`

const timeoutRolloutAbort = `{"type":"event_msg","payload":{"type":"turn_aborted","reason":"interrupted"}}
`

func newTimeoutCoordinator(sessions *fakeSessions, sink runner.EventSink, grace time.Duration) *Coordinator {
	return NewCoordinator(CoordinatorDeps{
		Tailer:                NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond)),
		Sessions:              sessions,
		Runs:                  &fakeRunStore{},
		NewSink:               func(uuid.UUID) runner.EventSink { return sink },
		Debounce:              80 * time.Millisecond,
		ActivityPoll:          10 * time.Millisecond,
		SessionPoll:           15 * time.Millisecond,
		SessionReattachWindow: time.Second,
		Heartbeat:             -1,
		TimeoutDrainGrace:     grace,
	})
}

func authoritativeUsage(sink *collectSink) []*domain.UsageEventData {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	var out []*domain.UsageEventData
	for _, event := range sink.events {
		if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
			out = append(out, usage)
		}
	}
	return out
}

// A goal run cut off by its timeout used to leave the harness working in its
// session and the run's usage unknown. At the deadline the coordinator now
// interrupts the harness and keeps reading, so codex's own turn_aborted record
// — and the usage receipt its parser derives from it — is stored, while the
// run is still reported as a timeout.
func TestCoordinatorTimeoutInterruptsHarnessAndKeepsItsClosingReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte(timeoutRolloutOpenTurn), 0o644); err != nil {
		t.Fatal(err)
	}
	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	run.WebConsoleSessionID = "sess-1"
	run.SessionID = "codex-session"
	sessions := newFakeSessions("sess-1")
	sessions.onInterrupt = func() {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		if _, err := f.WriteString(timeoutRolloutAbort); err != nil {
			t.Error(err)
		}
	}
	sink := &collectSink{}
	coord := newTimeoutCoordinator(sessions, sink, 3*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	terminal, err := coord.TailToCompletion(ctx, run)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("tail error = %v, want the run's timeout", err)
	}
	if terminal != nil {
		t.Fatalf("terminal = %+v, want none: the abort answers the timeout interrupt", terminal)
	}
	if !slices.Contains(sessions.calls, "interrupt") {
		t.Fatalf("harness was never interrupted: %v", sessions.calls)
	}
	receipts := authoritativeUsage(sink)
	if len(receipts) != 1 {
		t.Fatalf("authoritative receipts = %d, want the abort's one receipt", len(receipts))
	}
	got := receipts[0]
	if total := got.InputTokens + got.OutputTokens + got.CacheReadTokens + got.CacheCreationTokens; total != 31081 {
		t.Fatalf("receipt tokens = %d, want the whole turn's 31081", total)
	}
	if err := coord.Finalize(ctx, run, terminal, err); err != nil {
		t.Fatal(err)
	}
	if run.Status != domain.RunStatusFailed || run.StopReason != domain.RunStopReasonTimeout {
		t.Fatalf("run = %s/%s, want failed/timeout", run.Status, run.StopReason)
	}
}

// Shutdown is not a timeout: the tail stops at once and the harness is left
// alone for restart recovery.
func TestCoordinatorCancellationDoesNotInterruptHarness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte(timeoutRolloutOpenTurn), 0o644); err != nil {
		t.Fatal(err)
	}
	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	run.WebConsoleSessionID = "sess-1"
	sessions := newFakeSessions("sess-1")
	coord := newTimeoutCoordinator(sessions, &collectSink{}, 3*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := coord.TailToCompletion(ctx, run)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("tail error = %v, want cancellation", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cancellation waited %v for a drain it does not owe", elapsed)
	}
	if slices.Contains(sessions.calls, "interrupt") {
		t.Fatalf("shutdown interrupted the harness: %v", sessions.calls)
	}
}

// With the drain disabled the tail ends exactly at the deadline, as before.
func TestCoordinatorTimeoutWithoutDrainEndsAtDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(timeoutRolloutOpenTurn)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	run.WebConsoleSessionID = "sess-1"
	sessions := newFakeSessions("sess-1")
	coord := newTimeoutCoordinator(sessions, &collectSink{}, -1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := coord.TailToCompletion(ctx, run)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("tail error = %v, want the timeout", err)
	}
	if slices.Contains(sessions.calls, "interrupt") {
		t.Fatalf("disabled drain still interrupted: %v", sessions.calls)
	}
}
