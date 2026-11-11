package event_test

import (
	"agent-manager/internal/adapters/runner"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// A parser resumed mid-file can derive a different payload for a line already
// stored. A transcript event keeps its first record; any other event is refused.
func TestReplayKeepsStoredTranscriptEvidence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID, id := uuid.New(), uuid.NewSHA1(uuid.NameSpaceOID, []byte("line"))
	make := func(input int, keep bool) *domain.RunEvent {
		return &domain.RunEvent{ID: id, RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now().UTC(), KeepStoredOnReplay: keep, Data: &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, InputTokens: input}}
	}
	if err := store.Append(ctx, runID, make(100, true)); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, runID, make(0, false)); err == nil {
		t.Fatal("a changed payload without the transcript mark must be refused")
	}
	replay := make(0, true)
	if err := store.Append(ctx, runID, replay); err != nil || !replay.PersistenceReplay {
		t.Fatalf("transcript replay: %v %+v", err, replay)
	}
	events, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(events) != 1 {
		t.Fatalf("events: %d %v", len(events), err)
	}
	if usage, ok := events[0].Data.(*domain.UsageEventData); !ok || usage.InputTokens != 100 {
		t.Fatalf("stored evidence changed: %+v", events[0].Data)
	}
}

type transcriptSink func(*domain.RunEvent) error

func (f transcriptSink) Emit(e *domain.RunEvent) error { return f(e) }
func (transcriptSink) Close() error                    { return nil }

func TestLiveTranscriptSurvivesSQLiteContentionAndReplay(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	db.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	writer, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err = writer.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")
	reader, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reader.ExecContext(ctx, "PRAGMA busy_timeout=1"); err != nil {
		t.Fatal(err)
	}
	reader.Close()
	store := event.NewSQLiteStore(db, newTestLogger())
	runID := uuid.New()
	path := filepath.Join(t.TempDir(), "stdout")
	if err = os.WriteFile(path, []byte("record\n"), 0600); err != nil {
		t.Fatal(err)
	}
	failed := make(chan error, 1)
	type outcome struct {
		cursor int64
		err    error
	}
	done := make(chan outcome, 1)
	parse := func(id uuid.UUID, _ string) runner.TranscriptParseResult {
		return runner.TranscriptParseResult{Timestamp: time.Unix(123, 0).UTC(), Events: []*domain.RunEvent{domain.NewLogEvent(id, "info", "actual transcript")}}
	}
	go func() {
		cursor, _, err := runner.Consume(ctx, runner.ConsumeArgs{RunID: runID, Transcript: path, Live: true, PollInterval: time.Millisecond, ParseFn: parse, EventSink: transcriptSink(func(e *domain.RunEvent) error {
			err := store.Append(ctx, runID, e)
			if err != nil {
				select {
				case failed <- err:
				default:
				}
			}
			return err
		}), StopAfterTranscriptResult: func(runner.TranscriptParseResult) bool { return true }})
		done <- outcome{cursor, err}
	}()
	select {
	case err := <-failed:
		var typed *domain.DatabaseError
		if !errors.As(err, &typed) || !typed.IsTransient {
			t.Fatalf("contention not retryable: %T %v", err, err)
		}
	case <-ctx.Done():
		t.Fatal("contention was not observed")
	}
	if _, err = writer.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		if result.err != nil || result.cursor != 7 {
			t.Fatalf("tail stopped: %+v", result)
		}
	case <-ctx.Done():
		t.Fatal("tail did not recover after writer released")
	}
	// Replaying from the committed source occurrence must not create a second row.
	if _, _, err = runner.Consume(ctx, runner.ConsumeArgs{RunID: runID, Transcript: path, ParseFn: parse, EventSink: transcriptSink(func(e *domain.RunEvent) error { return store.Append(ctx, runID, e) })}); err != nil {
		t.Fatal(err)
	}
	if count, err := store.Count(ctx, runID); err != nil || count != 1 {
		t.Fatalf("replay count=%d err=%v", count, err)
	}
}
