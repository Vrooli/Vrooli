package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/runmodel"
	"agent-manager/internal/wiring"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
)

type terminalServerAcceptor func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error)

func (f terminalServerAcceptor) AcceptTerminalEventIntent(c context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
	return f(c, s, id)
}

type terminalServerBoundary struct{}

func (terminalServerBoundary) WithTerminalDispatch(c context.Context, s runmodel.TerminalReceiptScope, fn func() error) error {
	return fn()
}

func TestTerminalServerLifecycleDefaultDisabled(t *testing.T) {
	s := &Server{}
	if err := s.startTerminalEvents(nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalServerCleanupJoinsBeforeStorageClose(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	release := make(chan struct{})
	dbReadable := make(chan error, 1)
	cfg := &wiring.TerminalEventsConfig{Scope: runmodel.TerminalReceiptScope{SourceInstance: "test-installation", SourceEpoch: "epoch-1"}, Boundary: terminalServerBoundary{}, BatchSize: 1, Interval: time.Hour,
		Acceptor: terminalServerAcceptor(func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
			close(entered)
			<-ctx.Done()
			close(cancelled)
			<-release
			var count int
			dbReadable <- db.Get(&count, "SELECT COUNT(*) FROM run_terminal_receipts")
			return eventbus.DomainEventAcknowledgment{}, errors.New("uncertain after cancellation")
		})}
	repos, worker, err := wiring.NewTerminalPersistence(db, logrus.New(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(worker.Stop)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
	if err = repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusComplete
	if err = repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, terminalEvents: worker}
	if err = s.startTerminalEvents(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("no startup dispatch")
	}
	done := make(chan error, 1)
	go func() { done <- s.Cleanup() }()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not cancel")
	}
	select {
	case <-done:
		t.Fatal("cleanup returned before dispatch joined")
	default:
	}
	close(release)
	if err = <-dbReadable; err != nil {
		t.Fatalf("DB closed while acceptor in flight: %v", err)
	}
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not join")
	}
	var afterClose int
	if err = db.Get(&afterClose, "SELECT 1"); err == nil {
		t.Fatal("cleanup did not close DB")
	}
}

var _ database.TerminalEventAcceptor = terminalServerAcceptor(nil)
