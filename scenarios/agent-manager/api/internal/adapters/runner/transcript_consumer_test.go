package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type testSequencedSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
	seq    int64
}

func TestConsumeNativeGoalSurvivesTurnEndButNotNewWork(t *testing.T) {
	for _, suffix := range []string{"turn\n", "user\n", "user_metadata\n", "active\n", "failure\n"} {
		t.Run(strings.TrimSpace(suffix), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "transcript")
			if err := os.WriteFile(path, []byte("goal\n"+suffix), 0600); err != nil {
				t.Fatal(err)
			}
			_, terminal, err := Consume(t.Context(), ConsumeArgs{RunID: uuid.New(), Transcript: path, ParseFn: func(id uuid.UUID, line string) TranscriptParseResult {
				switch strings.TrimSpace(line) {
				case "goal":
					return TranscriptParseResult{Goal: &GoalMarker{Objective: "finish", Status: GoalStatusComplete}}
				case "turn":
					return TranscriptParseResult{Terminal: &TranscriptTerminal{Success: true}}
				case "user":
					return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewMessageEvent(id, "user", "next work")}}
				case "user_metadata":
					return TranscriptParseResult{UserTurnStarted: true}
				case "active":
					return TranscriptParseResult{Goal: &GoalMarker{Objective: "next", Status: GoalStatusActive}}
				case "failure":
					return TranscriptParseResult{Terminal: &TranscriptTerminal{Success: false}}
				}
				return TranscriptParseResult{}
			}})
			if err != nil {
				t.Fatal(err)
			}
			switch suffix {
			case "turn\n":
				if terminal == nil || terminal.TerminalReason != "goal_complete" {
					t.Fatal("accepted goal overwritten by turn boundary", terminal)
				}
			case "failure\n":
				if terminal == nil || terminal.Success {
					t.Fatal("later failure hidden", terminal)
				}
			default:
				if terminal != nil {
					t.Fatal("earlier goal closed new work", terminal)
				}
			}
		})
	}
}

func (s *testSequencedSink) Emit(event *domain.RunEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	event.Sequence = s.seq
	s.events = append(s.events, event)
	return nil
}

func (s *testSequencedSink) Close() error { return nil }
func (s *testSequencedSink) LastSequence() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}
func (s *testSequencedSink) eventCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}
func (s *testSequencedSink) event(index int) *domain.RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events[index]
}

func TestConsumeReplayFromCursor(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "transcript.ndjson")
	lines := "first\nsecond\nthird\n"
	if err := os.WriteFile(transcript, []byte(lines), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runID := uuid.New()
	sink := &testSequencedSink{}
	var cursor int64
	parseFn := func(runID uuid.UUID, line string) TranscriptParseResult {
		return TranscriptParseResult{
			Events: []*domain.RunEvent{domain.NewLogEvent(runID, "info", line)},
		}
	}

	startAt := int64(len("first\n"))
	gotCursor, _, err := Consume(context.Background(), ConsumeArgs{
		RunID:      runID,
		Transcript: transcript,
		StartAt:    startAt,
		ParseFn:    parseFn,
		EventSink:  sink,
		OnAdvance: func(nextCursor, _ int64) error {
			cursor = nextCursor
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if gotCursor != int64(len(lines)) || cursor != int64(len(lines)) {
		t.Fatalf("cursor = %d/%d, want %d", gotCursor, cursor, len(lines))
	}
	if sink.eventCount() != 2 {
		t.Fatalf("events = %d, want 2", sink.eventCount())
	}
}

func TestConsumeTreatsCompletedNativeGoalAsTerminalSuccess(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte("goal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runID := uuid.New()
	var terminal *TranscriptTerminal
	_, got, err := Consume(context.Background(), ConsumeArgs{
		RunID:      runID,
		Transcript: transcript,
		ParseFn: func(uuid.UUID, string) TranscriptParseResult {
			return TranscriptParseResult{Goal: &GoalMarker{Objective: "finish", Status: GoalStatusComplete}}
		},
		OnTerminal: func(value *TranscriptTerminal) error {
			terminal = value
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if got == nil || !got.Success || terminal == nil || !terminal.Success {
		t.Fatalf("completed native goal did not produce terminal success: got=%+v callback=%+v", got, terminal)
	}
}

// TestConsumeMapsGoalStatusToStopClass proves each harness goal status produces
// the typed terminal class Swarm reads: a complete/blocked goal is a verdict, a
// usage/budget limit is an interruption, and no non-complete state is reported
// as success.
func TestConsumeMapsGoalStatusToStopClass(t *testing.T) {
	cases := []struct {
		status     GoalStatus
		wantOK     bool
		wantClass  domain.RunTerminalClass
		wantReason domain.RunStopReason
	}{
		{GoalStatusComplete, true, domain.RunTerminalClassVerdict, domain.RunStopReasonComplete},
		{GoalStatusBlocked, true, domain.RunTerminalClassVerdict, domain.RunStopReasonBlocked},
		{GoalStatusUsageLimited, false, domain.RunTerminalClassInterruption, domain.RunStopReasonUsageWindow},
		{GoalStatusBudgetLimited, false, domain.RunTerminalClassInterruption, domain.RunStopReasonUsageWindow},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			dir := t.TempDir()
			transcript := filepath.Join(dir, "transcript.ndjson")
			if err := os.WriteFile(transcript, []byte("goal\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			_, got, err := Consume(context.Background(), ConsumeArgs{
				RunID:      uuid.New(),
				Transcript: transcript,
				ParseFn: func(uuid.UUID, string) TranscriptParseResult {
					return TranscriptParseResult{Goal: &GoalMarker{Objective: "finish", Status: tc.status}}
				},
			})
			if err != nil {
				t.Fatalf("Consume: %v", err)
			}
			if got == nil {
				t.Fatal("no terminal produced")
			}
			if got.Success != tc.wantOK {
				t.Fatalf("success = %v, want %v", got.Success, tc.wantOK)
			}
			if got.TerminalClass != tc.wantClass {
				t.Fatalf("terminal class = %q, want %q", got.TerminalClass, tc.wantClass)
			}
			if got.StopReason != tc.wantReason {
				t.Fatalf("stop reason = %q, want %q", got.StopReason, tc.wantReason)
			}
		})
	}
}

// TestConsumeLiveReassemblesPartialLine guards the interactive-tail contract:
// when an external writer flushes a line in two fragments across poll cycles
// (a half-written JSON line at EOF), the live tailer must reassemble the whole
// line, never emit the trailing fragment alone. Regression for the bufio
// consume-and-lose bug where the pre-newline fragment was dropped.
func TestConsumeLiveReassemblesPartialLine(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	sink := &testSequencedSink{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = Consume(ctx, ConsumeArgs{
			RunID:        uuid.New(),
			Transcript:   transcript,
			Live:         true,
			PollInterval: 10 * time.Millisecond,
			ParseFn: func(runID uuid.UUID, line string) TranscriptParseResult {
				return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewLogEvent(runID, "info", line)}}
			},
			EventSink: sink,
		})
	}()

	f, err := os.OpenFile(transcript, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	// Fragment 1 without newline: the tailer must hold it, not emit it.
	if _, err := f.WriteString(`{"part":`); err != nil {
		t.Fatalf("write frag1: %v", err)
	}
	_ = f.Sync()
	time.Sleep(40 * time.Millisecond) // let a poll hit EOF mid-line
	// Fragment 2 completes the line.
	if _, err := f.WriteString("\"whole\"}\n"); err != nil {
		t.Fatalf("write frag2: %v", err)
	}
	_ = f.Sync()
	_ = f.Close()

	deadline := time.Now().Add(2 * time.Second)
	for sink.eventCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if sink.eventCount() != 1 {
		t.Fatalf("events = %d, want 1 reassembled line", sink.eventCount())
	}
	got := sink.event(0).Data.(*domain.LogEventData).Message
	if want := `{"part":"whole"}` + "\n"; got != want {
		t.Fatalf("line = %q, want %q (partial fragment dropped?)", got, want)
	}
}

func TestConsumeLiveTailsNewBytes(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runID := uuid.New()
	sink := &testSequencedSink{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = Consume(ctx, ConsumeArgs{
			RunID:        runID,
			Transcript:   transcript,
			Live:         true,
			PollInterval: 10 * time.Millisecond,
			ParseFn: func(runID uuid.UUID, line string) TranscriptParseResult {
				return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewLogEvent(runID, "info", line)}}
			},
			EventSink: sink,
		})
	}()

	time.Sleep(20 * time.Millisecond)
	f, err := os.OpenFile(transcript, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, err := f.WriteString("hello\n"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	_ = f.Close()

	deadline := time.Now().Add(2 * time.Second)
	for sink.eventCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if sink.eventCount() == 0 {
		t.Fatal("expected tailed event")
	}
	cancel()
	<-done
}
