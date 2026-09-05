package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type testSequencedSink struct {
	events []*domain.RunEvent
	seq    int64
}

func (s *testSequencedSink) Emit(event *domain.RunEvent) error {
	s.seq++
	event.Sequence = s.seq
	s.events = append(s.events, event)
	return nil
}

func (s *testSequencedSink) Close() error        { return nil }
func (s *testSequencedSink) LastSequence() int64 { return s.seq }

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
	if len(sink.events) != 2 {
		t.Fatalf("events = %d, want 2", len(sink.events))
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
	for len(sink.events) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1 reassembled line", len(sink.events))
	}
	got := sink.events[0].Data.(*domain.LogEventData).Message
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
	for len(sink.events) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(sink.events) == 0 {
		t.Fatal("expected tailed event")
	}
	cancel()
	<-done
}
