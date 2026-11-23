package event_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
)

type historyQueryGate struct {
	sqlcompat.DB
	started chan struct{}
	release chan struct{}
	once    sync.Once
	err     error
}

func (g *historyQueryGate) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	if strings.Contains(query, "FROM run_events WHERE run_id = ?") {
		g.once.Do(func() {
			close(g.started)
			<-g.release
		})
		if g.err != nil {
			return g.err
		}
	}
	return g.DB.SelectContext(ctx, dest, query, args...)
}

func TestSQLiteStreamReleasesBufferedLiveEventsWhenHistoryQueryFails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()
	gate := &historyQueryGate{
		DB:      db,
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("injected history query failure"),
	}
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(gate.release) }) })
	store := event.NewSQLiteStore(gate, newTestLogger())
	stream, err := store.Stream(ctx, runID, event.StreamOptions{FromSequence: 0, BufferSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-gate.started:
	case <-ctx.Done():
		t.Fatal("stream history query did not reach the controlled handoff")
	}

	live := domain.NewLogEvent(runID, "info", "appended while history query was loading")
	if err := store.Append(ctx, runID, live); err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(gate.release) })
	if got := awaitStreamEvent(t, ctx, stream); got.ID != live.ID {
		t.Fatalf("event after history query failure = %s, want buffered live event %s", got.ID, live.ID)
	}
}

func TestSQLiteStreamOrdersHistoryBeforeConcurrentAppendWithoutDuplicates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()
	stored := domain.NewLogEvent(runID, "info", "stored before subscription")
	if err := event.NewSQLiteStore(db, newTestLogger()).Append(ctx, runID, stored); err != nil {
		t.Fatal(err)
	}
	gate := &historyQueryGate{DB: db, started: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(gate.release) }) })
	store := event.NewSQLiteStore(gate, newTestLogger())
	stream, err := store.Stream(ctx, runID, event.StreamOptions{FromSequence: 0, BufferSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-gate.started:
	case <-ctx.Done():
		t.Fatal("stream history query did not reach the controlled handoff")
	}

	concurrent := domain.NewLogEvent(runID, "info", "appended during history load")
	if err := store.Append(ctx, runID, concurrent); err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(gate.release) })

	if got := awaitStreamEvent(t, ctx, stream); got.ID != stored.ID {
		t.Fatalf("first event = %s, want historical event %s", got.ID, stored.ID)
	}
	if got := awaitStreamEvent(t, ctx, stream); got.ID != concurrent.ID {
		t.Fatalf("second event = %s, want concurrent append %s", got.ID, concurrent.ID)
	}
	select {
	case got, open := <-stream:
		if open && got != nil {
			t.Fatalf("concurrent append was delivered more than once: %+v", got)
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSQLiteStreamAppliesFromSequenceToLiveAppends(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()
	store := event.NewSQLiteStore(db, newTestLogger())
	if err := store.Append(ctx, runID, domain.NewLogEvent(runID, "info", "sequence zero")); err != nil {
		t.Fatal(err)
	}
	stream, err := store.Stream(ctx, runID, event.StreamOptions{FromSequence: 3, BufferSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	for sequence := int64(1); sequence < 3; sequence++ {
		if err := store.Append(ctx, runID, domain.NewLogEvent(runID, "info", "below requested cursor")); err != nil {
			t.Fatal(err)
		}
	}
	wanted := domain.NewLogEvent(runID, "info", "at requested cursor")
	if err := store.Append(ctx, runID, wanted); err != nil {
		t.Fatal(err)
	}
	if got := awaitStreamEvent(t, ctx, stream); got.ID != wanted.ID || got.Sequence != 3 {
		t.Fatalf("first live event = (%s, %d), want requested event (%s, 3)", got.ID, got.Sequence, wanted.ID)
	}
}
