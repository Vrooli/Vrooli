package wiring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/modules"
	"agent-manager/internal/runmodel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

type terminalAcceptorFunc func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error)

func (f terminalAcceptorFunc) AcceptTerminalEventIntent(c context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
	return f(c, s, id)
}

type terminalBoundaryFunc func(context.Context, runmodel.TerminalReceiptScope, func() error) error

func (f terminalBoundaryFunc) WithTerminalDispatch(c context.Context, s runmodel.TerminalReceiptScope, fn func() error) error {
	return f(c, s, fn)
}

var terminalScope = runmodel.TerminalReceiptScope{SourceInstance: "test-installation", SourceEpoch: "epoch-1"}

func terminalConfig(f terminalAcceptorFunc) *TerminalEventsConfig {
	return &TerminalEventsConfig{Scope: terminalScope, Acceptor: f, Boundary: terminalBoundaryFunc(func(ctx context.Context, s runmodel.TerminalReceiptScope, fn func() error) error {
		if s != terminalScope {
			return errors.New("source denied")
		}
		return fn()
	}), BatchSize: 2, Interval: time.Hour}
}
func terminalDB(t *testing.T, path string) *database.DB {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+path+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err = coredb.EnsureSchemas(t.Context(), db, modules.AllSchemas()...); err != nil {
		db.Close()
		t.Fatal(err)
	}
	wrapped := database.NewDB(db, logrus.New())
	t.Cleanup(func() { _ = wrapped.Close() })
	return wrapped
}
func terminalRun(t *testing.T, r *database.Repositories, status domain.RunStatus) *domain.Run {
	t.Helper()
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
	if err := r.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	run.Status = status
	at := time.Now().UTC()
	run.EndedAt = &at
	if err := r.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	return run
}
func terminalCount(t *testing.T, db *database.DB, table string) int {
	t.Helper()
	var n int
	if err := db.Get(&n, "SELECT COUNT(*) FROM "+table); err != nil {
		t.Fatal(err)
	}
	return n
}
func terminalAck(_ context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
	_, b, err := eventbus.RetainedDomainEventEnvelope(s, id)
	if err != nil {
		return eventbus.DomainEventAcknowledgment{}, err
	}
	h := sha256.Sum256(b)
	return eventbus.DomainEventAcknowledgment{Schema: eventbus.DomainEventAcknowledgmentSchema, EventID: id, CanonicalSHA256: hex.EncodeToString(h[:]), StoreID: 1}, nil
}
func terminalCompose(t *testing.T, db *database.DB, cfg *TerminalEventsConfig) (*database.Repositories, *TerminalEvents) {
	t.Helper()
	r, w, err := NewTerminalPersistence(db, logrus.New(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Stop)
	return r, w
}
func waitTerminal(t *testing.T, w *TerminalEvents, fn func(TerminalEventsStatus) bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for {
		if fn(w.Status()) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("terminal lifecycle timeout: %+v", w.Status())
		case <-time.After(time.Millisecond):
		}
	}
}
func TestTerminalCompositionDefaultOff(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	r, w := terminalCompose(t, db, nil)
	if w != nil || w.Status().State != "disabled" {
		t.Fatal("default enabled")
	}
	if err := w.Start(nil); err != nil {
		t.Fatal(err)
	}
	w.Stop()
	for _, table := range []string{"run_terminal_receipts", "run_terminal_receipt_source", "run_terminal_event_acknowledgments"} {
		if _, err := db.Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	terminalRun(t, r, domain.RunStatusComplete)
}
func TestTerminalCompositionRejectsIncompleteOptions(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	cases := []func(*TerminalEventsConfig){func(c *TerminalEventsConfig) { c.Scope.SourceEpoch = "" }, func(c *TerminalEventsConfig) { c.Acceptor = nil }, func(c *TerminalEventsConfig) { c.Boundary = nil }, func(c *TerminalEventsConfig) { c.BatchSize = 0 }, func(c *TerminalEventsConfig) { c.BatchSize = 101 }, func(c *TerminalEventsConfig) { c.Interval = time.Nanosecond }, func(c *TerminalEventsConfig) { c.Interval = 2 * time.Hour }}
	for i, change := range cases {
		cfg := terminalConfig(terminalAck)
		change(cfg)
		if _, _, err := NewTerminalPersistence(db, logrus.New(), cfg); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
	if terminalCount(t, db, "run_terminal_receipt_source") != 0 {
		t.Fatal("configuration wrote source binding")
	}
}
func TestTerminalCompositionCaptureStartStopAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "am.sqlite")
	db := terminalDB(t, path)
	var calls atomic.Int64
	cfg := terminalConfig(func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		calls.Add(1)
		return terminalAck(ctx, s, id)
	})
	r, w := terminalCompose(t, db, cfg)
	terminalRun(t, r, domain.RunStatusUnknown)
	if w.Status().State != "not_started" || calls.Load() != 0 || terminalCount(t, db, "run_terminal_receipts") != 1 {
		t.Fatal("constructor starts or omits capture")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Start(t.Context()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	waitTerminal(t, w, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	w.Stop()
	if calls.Load() != 1 || terminalCount(t, db, "run_terminal_event_acknowledgments") != 1 {
		t.Fatal("duplicate dispatch or missing ack")
	}
	if err := w.Start(t.Context()); err == nil {
		t.Fatal("stopped lifecycle restarted")
	}
	_ = db.Close()
	db = terminalDB(t, path)
	_, w = terminalCompose(t, db, cfg)
	if err := w.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitTerminal(t, w, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	w.Stop()
	if calls.Load() != 1 || terminalCount(t, db, "run_terminal_receipts") != 1 {
		t.Fatal("reopen recreated intent")
	}
}
func TestTerminalCompositionCurrentAuthorityAndSourceFence(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	var calls atomic.Int64
	var revoked atomic.Bool
	revoked.Store(true)
	cfg := terminalConfig(func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		calls.Add(1)
		return terminalAck(ctx, s, id)
	})
	cfg.Boundary = terminalBoundaryFunc(func(ctx context.Context, s runmodel.TerminalReceiptScope, fn func() error) error {
		if revoked.Load() {
			return errors.New("fixture denied")
		}
		return fn()
	})
	r, w := terminalCompose(t, db, cfg)
	terminalRun(t, r, domain.RunStatusComplete)
	clock := schedule.NewFake(time.Now())
	w.clock = clock
	if err := w.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitTerminal(t, w, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	if calls.Load() != 0 || w.Status().State != "held" {
		t.Fatal("denied authority dispatched")
	}
	revoked.Store(false)
	clock.Tick()
	waitTerminal(t, w, func(s TerminalEventsStatus) bool { return s.Attempts == 2 })
	w.Stop()
	if calls.Load() != 1 {
		t.Fatal("retry did not use current authority")
	}
	cfg.Scope.SourceEpoch = "replacement"
	if _, _, err := NewTerminalPersistence(db, logrus.New(), cfg); err == nil {
		t.Fatal("mismatched immutable source accepted at composition")
	}
	if calls.Load() != 1 {
		t.Fatal("source preflight invoked transport")
	}

}
func TestTerminalCompositionStopCancelsAndJoinsPendingDispatch(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	release := make(chan struct{})
	cfg := terminalConfig(func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		<-release
		return eventbus.DomainEventAcknowledgment{}, ctx.Err()
	})
	r, w := terminalCompose(t, db, cfg)
	terminalRun(t, r, domain.RunStatusCancelled)
	if err := w.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker not started")
	}
	done := make(chan struct{})
	go func() { w.Stop(); close(done) }()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("cancel not propagated")
	}
	select {
	case <-done:
		t.Fatal("Stop returned before acceptor joined")
	default:
	}
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not join")
	}
	if terminalCount(t, db, "run_terminal_event_acknowledgments") != 0 {
		t.Fatal("cancelled send retained acknowledgment")
	}
	_, retry := terminalCompose(t, db, terminalConfig(terminalAck))
	retry.dispatch(t.Context())
	if terminalCount(t, db, "run_terminal_event_acknowledgments") != 1 {
		t.Fatal("pending intent not replayed")
	}
}
func TestTerminalCompositionStopBeforeStartAndCancelledStart(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	_, w := terminalCompose(t, db, terminalConfig(terminalAck))
	w.Stop()
	w.Stop()
	if err := w.Start(t.Context()); err == nil {
		t.Fatal("Start after Stop succeeded")
	}
	_, w = terminalCompose(t, db, terminalConfig(terminalAck))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := w.Start(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if w.done != nil || w.Status().Running {
		t.Fatal("cancelled startup created worker")
	}
}
func TestTerminalCompositionFiniteBatchAndFailedCallbackHold(t *testing.T) {
	db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
	cfg := terminalConfig(terminalAck)
	cfg.BatchSize = 1
	r, w := terminalCompose(t, db, cfg)
	terminalRun(t, r, domain.RunStatusFailed)
	terminalRun(t, r, domain.RunStatusComplete)
	w.dispatch(t.Context())
	if terminalCount(t, db, "run_terminal_event_acknowledgments") != 1 {
		t.Fatal("batch not finite")
	}
	w.dispatch(t.Context())
	if terminalCount(t, db, "run_terminal_event_acknowledgments") != 2 {
		t.Fatal("next pass omitted pending")
	}
	cfg.Acceptor = terminalAcceptorFunc(func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error) {
		return eventbus.DomainEventAcknowledgment{}, errors.New("private error details")
	})
	cfg.Boundary = terminalBoundaryFunc(func(ctx context.Context, s runmodel.TerminalReceiptScope, fn func() error) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 15*time.Second {
			t.Error("unbounded attempt")
		}
		_ = fn()
		return nil
	})
	r, w = terminalCompose(t, db, cfg)
	terminalRun(t, r, domain.RunStatusUnknown)
	w.dispatch(t.Context())
	if w.Status().State != "held" {
		t.Fatal("boundary swallowed failure")
	}
}

func TestTerminalCompositionRejectsRepeatedAndLateAuthorityCallbacks(t *testing.T) {
	for _, mode := range []string{"missing", "repeated", "late"} {
		t.Run(mode, func(t *testing.T) {
			db := terminalDB(t, filepath.Join(t.TempDir(), "am.sqlite"))
			var calls atomic.Int64
			cfg := terminalConfig(func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
				calls.Add(1)
				return terminalAck(ctx, s, id)
			})
			var delayed func() error
			cfg.Boundary = terminalBoundaryFunc(func(ctx context.Context, scope runmodel.TerminalReceiptScope, fn func() error) error {
				switch mode {
				case "missing":
					return nil
				case "repeated":
					_ = fn()
					_ = fn()
					return nil
				default:
					delayed = fn
					return nil
				}
			})
			repos, worker := terminalCompose(t, db, cfg)
			terminalRun(t, repos, domain.RunStatusComplete)
			worker.dispatch(t.Context())
			if worker.Status().State != "held" {
				t.Fatal("invalid callback contract reported success")
			}
			if delayed != nil && delayed() == nil {
				t.Fatal("late callback accepted")
			}
			want := int64(0)
			if mode == "repeated" {
				want = 1
			}
			if calls.Load() != want {
				t.Fatalf("acceptor calls=%d want %d", calls.Load(), want)
			}
		})
	}
}
