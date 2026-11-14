package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type persistenceSinkFunc func(*domain.RunEvent) error

func (f persistenceSinkFunc) Emit(event *domain.RunEvent) error { return f(event) }
func (persistenceSinkFunc) Close() error                        { return nil }

func TestConsumeRetriesOnlyFailedPersistenceStage(t *testing.T) {
	for _, failAt := range []string{"session", "label", "goal", "event-2", "cursor"} {
		t.Run(failAt, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "stdout")
			if err := os.WriteFile(path, []byte("record\n"), 0600); err != nil {
				t.Fatal(err)
			}
			calls := map[string]int{}
			stage := func(name string) error {
				calls[name]++
				if name == failAt && calls[name] == 1 {
					return &domain.DatabaseError{IsTransient: true, Cause: errors.New("busy")}
				}
				return nil
			}
			parsed, accounted := 0, 0
			cursor, _, err := Consume(t.Context(), ConsumeArgs{RunID: uuid.New(), Transcript: path, Live: true, PollInterval: time.Millisecond,
				ParseFn: func(id uuid.UUID, _ string) TranscriptParseResult {
					parsed++
					return TranscriptParseResult{SessionID: "native", Label: "work", Goal: &GoalMarker{Status: GoalStatusActive}, Events: []*domain.RunEvent{domain.NewLogEvent(id, "info", "event-1"), domain.NewLogEvent(id, "info", "event-2")}}
				},
				OnSessionID: func(string) error { return stage("session") }, OnLabel: func(string, domain.RunLabelSource) error { return stage("label") },
				OnGoalStatus: func(GoalMarker) error { return stage("goal") },
				EventSink:    persistenceSinkFunc(func(e *domain.RunEvent) error { return stage(e.Data.(*domain.LogEventData).Message) }),
				OnAdvance:    func(int64, int64) error { return stage("cursor") }, OnEvents: func([]*domain.RunEvent) { accounted++ },
				StopAfterTranscriptResult: func(TranscriptParseResult) bool { return true },
			})
			if err != nil || cursor != int64(len("record\n")) || parsed != 1 || accounted != 1 {
				t.Fatalf("cursor=%d parsed=%d accounted=%d err=%v", cursor, parsed, accounted, err)
			}
			for name, n := range calls {
				want := 1
				if name == failAt {
					want = 2
				}
				if n != want {
					t.Errorf("%s called %d want%d", name, n, want)
				}
			}
		})
	}
}

func TestConsumeCancelledPartialLineResumesWithoutReparseOrDuplicateEffects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(path, []byte("record\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	state := &TranscriptConsumeState{}
	parsed, accounted, advances := 0, 0, 0
	calls := map[string]int{}
	args := ConsumeArgs{RunID: uuid.New(), Transcript: path, Live: true, State: state, PollInterval: time.Second,
		ParseFn: func(id uuid.UUID, _ string) TranscriptParseResult {
			parsed++
			return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewLogEvent(id, "info", "one"), domain.NewLogEvent(id, "info", "two")}}
		},
		EventSink: persistenceSinkFunc(func(e *domain.RunEvent) error {
			name := e.Data.(*domain.LogEventData).Message
			calls[name]++
			if name == "two" && calls[name] == 1 {
				cancel()
				return &domain.DatabaseError{IsTransient: true, Cause: errors.New("busy")}
			}
			return nil
		}),
		OnAdvance: func(int64, int64) error { advances++; return nil }, OnEvents: func([]*domain.RunEvent) { accounted++ },
	}
	cursor, _, err := Consume(ctx, args)
	if !errors.Is(err, context.Canceled) || cursor != 0 || accounted != 1 || advances != 0 {
		t.Fatalf("failed line committed: cursor%d accounted%d advances%d err%v", cursor, accounted, advances, err)
	}
	args.Live = false
	args.StartAt = cursor
	cursor, _, err = Consume(t.Context(), args)
	if err != nil || cursor != 7 || parsed != 1 || accounted != 1 || advances != 1 || calls["one"] != 1 || calls["two"] != 2 {
		t.Fatalf("bad resume: cursor%d parsed%d accounted%d advances%d calls%v err%v", cursor, parsed, accounted, advances, calls, err)
	}
}

func TestConsumePermanentFailureReturnsCommittedCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	failure := domain.NewStateError("Run", "superseded", "publish", "new lifecycle")
	accounted := 0
	cursor, _, err := Consume(t.Context(), ConsumeArgs{RunID: uuid.New(), Transcript: path, Live: true,
		ParseFn: func(id uuid.UUID, line string) TranscriptParseResult {
			return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewLogEvent(id, "info", line)}}
		},
		EventSink: persistenceSinkFunc(func(e *domain.RunEvent) error {
			if e.Data.(*domain.LogEventData).Message == "second\n" {
				return failure
			}
			return nil
		}), OnEvents: func([]*domain.RunEvent) { accounted++ },
	})
	if !errors.Is(err, failure) || cursor != 6 || accounted != 2 {
		t.Fatalf("permanent failure cursor%d accounted%d err%v", cursor, accounted, err)
	}
}

func TestConsumeTerminalCallbackFailureDoesNotRetryOrCommit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(path, []byte("terminal\n"), 0600); err != nil {
		t.Fatal(err)
	}
	failure := domain.NewStateError("Run", "superseded", "terminal", "new lifecycle")
	calls := 0
	cursor, _, err := Consume(t.Context(), ConsumeArgs{RunID: uuid.New(), Transcript: path, Live: true, ParseFn: func(uuid.UUID, string) TranscriptParseResult {
		return TranscriptParseResult{Terminal: &TranscriptTerminal{Success: true}}
	}, OnTerminal: func(*TranscriptTerminal) error { calls++; return failure }})
	if !errors.Is(err, failure) || cursor != 0 || calls != 1 {
		t.Fatalf("terminal failure cursor%d calls%d err%v", cursor, calls, err)
	}
}

func TestConsumePendingLineCannotDisappearDuringDrainHandoff(t *testing.T) {
	for _, replacement := range []string{"", "partial"} {
		t.Run(replacement, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "stdout")
			if err := os.WriteFile(path, []byte("record\n"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			state := &TranscriptConsumeState{}
			args := ConsumeArgs{RunID: uuid.New(), Transcript: path, Live: true, State: state, ParseFn: func(id uuid.UUID, _ string) TranscriptParseResult {
				return TranscriptParseResult{Events: []*domain.RunEvent{domain.NewLogEvent(id, "info", "work")}}
			}, EventSink: persistenceSinkFunc(func(*domain.RunEvent) error { cancel(); return &domain.DatabaseError{IsTransient: true} })}
			if cursor, _, err := Consume(ctx, args); cursor != 0 || !errors.Is(err, context.Canceled) {
				t.Fatalf("initial cancellation cursor%d err%v", cursor, err)
			}
			if err := os.WriteFile(path, []byte(replacement), 0600); err != nil {
				t.Fatal(err)
			}
			args.Live = false
			if cursor, _, err := Consume(t.Context(), args); cursor != 0 || err == nil {
				t.Fatalf("missing pending source accepted cursor%d err%v", cursor, err)
			}
		})
	}
}

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
