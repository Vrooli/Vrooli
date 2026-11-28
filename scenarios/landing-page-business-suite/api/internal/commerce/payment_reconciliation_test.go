package commerce

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type memoryPaymentReconciliationState struct {
	mu           sync.Mutex
	owner        string
	generation   int64
	expiresAt    time.Time
	cursor       PaymentReconciliationCursor
	hasCursor    bool
	failAdvances int
}

func (s *memoryPaymentReconciliationState) Claim(_ context.Context, owner string, now, expires time.Time) (PaymentReconciliationLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != "" && s.expiresAt.After(now) {
		return PaymentReconciliationLease{}, ErrReconciliationLeaseHeld
	}
	s.generation++
	s.owner, s.expiresAt = owner, expires
	return PaymentReconciliationLease{Owner: owner, Generation: s.generation, ExpiresAt: expires, Cursor: s.cursor, HasCursor: s.hasCursor}, nil
}

func (s *memoryPaymentReconciliationState) Renew(_ context.Context, lease PaymentReconciliationLease, now, expires time.Time) (PaymentReconciliationLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != lease.Owner || s.generation != lease.Generation || !s.expiresAt.After(now) {
		return PaymentReconciliationLease{}, ErrReconciliationLeaseLost
	}
	s.expiresAt = expires
	lease.ExpiresAt = expires
	lease.Cursor, lease.HasCursor = s.cursor, s.hasCursor
	return lease, nil
}

func (s *memoryPaymentReconciliationState) Advance(_ context.Context, lease PaymentReconciliationLease, cursor PaymentReconciliationCursor, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failAdvances > 0 {
		s.failAdvances--
		return errors.New("injected checkpoint failure")
	}
	if s.owner != lease.Owner || s.generation != lease.Generation || !s.expiresAt.After(now) {
		return ErrReconciliationLeaseLost
	}
	if s.hasCursor && cursor.Compare(s.cursor) < 0 {
		return ErrInvalidReconciliationEvent
	}
	s.cursor, s.hasCursor = cursor, true
	return nil
}

func (s *memoryPaymentReconciliationState) Release(_ context.Context, lease PaymentReconciliationLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != lease.Owner || s.generation != lease.Generation {
		return ErrReconciliationLeaseLost
	}
	s.owner, s.expiresAt = "", time.Time{}
	return nil
}

type fixedPaymentReconciliationSource struct {
	mu     sync.Mutex
	events []PaymentReconciliationEvent
	block  chan struct{}
}

type cancelingPaymentReconciliationSource struct {
	cancel context.CancelFunc
	calls  int
}

func (s *cancelingPaymentReconciliationSource) ListAfter(context.Context, PaymentReconciliationCursor, bool, int) ([]PaymentReconciliationEvent, error) {
	s.calls++
	s.cancel()
	return nil, nil
}

func (s *fixedPaymentReconciliationSource) ListAfter(ctx context.Context, cursor PaymentReconciliationCursor, hasCursor bool, limit int) ([]PaymentReconciliationEvent, error) {
	if s.block != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.block:
		}
		s.block = nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PaymentReconciliationEvent, 0, limit)
	for _, event := range s.events {
		if !hasCursor || event.Cursor.Compare(cursor) > 0 {
			out = append(out, event)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

type idempotentPaymentReconciliationProcessor struct {
	mu         sync.Mutex
	seen       map[string]bool
	applyCount map[string]int
	firstError error
}

func (p *idempotentPaymentReconciliationProcessor) Apply(_ context.Context, event PaymentReconciliationEvent) (PaymentReconciliationOutcome, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := event.IdempotencyKey()
	if p.firstError != nil {
		err := p.firstError
		p.firstError = nil
		return "", err
	}
	if p.seen[key] {
		p.applyCount[key]++
		return PaymentReconciliationReplay, nil
	}
	p.seen[key] = true
	p.applyCount[key]++
	return PaymentReconciliationApplied, nil
}

func TestPaymentReconciliationRestartReplaysEffectAndResumesDurableCursor(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	state := &memoryPaymentReconciliationState{failAdvances: 1}
	source := &fixedPaymentReconciliationSource{events: reconciliationEvents(now, "evt_a", "evt_b")}
	processor := &idempotentPaymentReconciliationProcessor{seen: map[string]bool{}, applyCount: map[string]int{}}
	runner := PaymentReconciliationRunner{State: state, Source: source, Processor: processor, Now: func() time.Time { return now }}
	if _, err := runner.RunOnce(ctx); err == nil {
		t.Fatal("injected checkpoint write failure was hidden")
	}
	report, err := runner.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.EventsRead != 2 || report.Replayed != 1 || report.Applied != 1 || !report.HasCursor || report.Cursor.EventID != "evt_b" {
		t.Fatalf("restart report = %+v", report)
	}
	if processor.applyCount["stripe:test:evt_a"] != 2 || processor.applyCount["stripe:test:evt_b"] != 1 {
		t.Fatalf("idempotent application counts = %+v", processor.applyCount)
	}
}

func TestPaymentReconciliationRunOnceStopsAtConfiguredBatch(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	state := &memoryPaymentReconciliationState{}
	source := &fixedPaymentReconciliationSource{events: reconciliationEvents(now, "evt_a", "evt_b", "evt_c")}
	processor := &idempotentPaymentReconciliationProcessor{seen: map[string]bool{}, applyCount: map[string]int{}}
	runner := PaymentReconciliationRunner{State: state, Source: source, Processor: processor, Now: func() time.Time { return now }, BatchSize: 2}
	first, err := runner.RunOnce(ctx)
	if err != nil || first.EventsRead != 2 || first.Cursor.EventID != "evt_b" {
		t.Fatalf("first bounded run = %+v, %v", first, err)
	}
	second, err := runner.RunOnce(ctx)
	if err != nil || second.EventsRead != 1 || second.Cursor.EventID != "evt_c" {
		t.Fatalf("second bounded run = %+v, %v", second, err)
	}
}

func TestPaymentReconciliationLeaseExcludesConcurrentRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	state := &memoryPaymentReconciliationState{}
	entered, release := make(chan struct{}), make(chan struct{})
	source := &fixedPaymentReconciliationSource{events: reconciliationEvents(now, "evt_a"), block: release}
	processor := &idempotentPaymentReconciliationProcessor{seen: map[string]bool{}, applyCount: map[string]int{}}
	runner := PaymentReconciliationRunner{State: state, Source: source, Processor: processor, Now: func() time.Time { return now }, NewOwner: func() (string, error) { return "first-owner", nil }}
	firstDone := make(chan error, 1)
	go func() { _, err := runner.RunOnce(ctx); firstDone <- err }()
	for i := 0; i < 100; i++ {
		state.mu.Lock()
		claimed := state.owner != ""
		state.mu.Unlock()
		if claimed {
			close(entered)
			break
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first runner did not claim durable lease")
	}
	second := runner
	second.NewOwner = func() (string, error) { return "second-owner", nil }
	if _, err := second.RunOnce(ctx); !errors.Is(err, ErrReconciliationLeaseHeld) {
		t.Fatalf("second runner error = %v, want lease held", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestPaymentReconciliationScheduleRunsImmediatelyAndStopsOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	source := &cancelingPaymentReconciliationSource{cancel: cancel}
	runner := PaymentReconciliationRunner{
		State:     &memoryPaymentReconciliationState{},
		Source:    source,
		Processor: &idempotentPaymentReconciliationProcessor{seen: map[string]bool{}, applyCount: map[string]int{}},
		Now:       func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) },
	}
	if err := runner.Run(ctx, time.Millisecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("scheduled runner stop = %v, want context canceled", err)
	}
	if source.calls != 1 {
		t.Fatalf("scheduled runner performed %d initial source passes, want one immediate pass", source.calls)
	}
}

func TestPaymentReconciliationRejectsOutOfOrderAndConflictingEvents(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	state := &memoryPaymentReconciliationState{}
	processor := &idempotentPaymentReconciliationProcessor{seen: map[string]bool{}, applyCount: map[string]int{}}
	runner := PaymentReconciliationRunner{State: state, Processor: processor, Now: func() time.Time { return now }}
	runner.Source = &fixedPaymentReconciliationSource{events: reconciliationEvents(now, "evt_b", "evt_a")}
	if _, err := runner.RunOnce(context.Background()); !errors.Is(err, ErrInvalidReconciliationEvent) {
		t.Fatalf("out-of-order source error = %v", err)
	}
	if len(processor.seen) != 1 {
		t.Fatalf("processor applied out-of-order event: %+v", processor.seen)
	}
	runner.Source = &fixedPaymentReconciliationSource{events: reconciliationEvents(now, "evt_c")}
	processor.firstError = errors.New("conflicting event receipt")
	if _, err := runner.RunOnce(context.Background()); err == nil {
		t.Fatal("conflicting provider receipt was accepted")
	}
}

func TestSQLPaymentReconciliationStoreFencesLeaseAndAdvancesCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewSQLPaymentReconciliationStateStore(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payment_reconciliation_checkpoint(scope_id) VALUES($1) ON CONFLICT(scope_id) DO NOTHING`)).
		WithArgs(paymentReconciliationScope).WillReturnResult(sqlmock.NewResult(0, 0))
	claimSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET lease_owner=$2, lease_generation=lease_generation+1,
		    lease_expires_at=CURRENT_TIMESTAMP + ($3 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND (lease_expires_at IS NULL OR lease_expires_at <= CURRENT_TIMESTAMP)
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`
	mock.ExpectQuery(regexp.QuoteMeta(claimSQL)).WithArgs(paymentReconciliationScope, "owner-a", int64(time.Minute.Microseconds())).
		WillReturnRows(sqlmock.NewRows([]string{"lease_generation", "lease_expires_at", "cursor_created_at", "cursor_event_id"}).AddRow(int64(4), expires, nil, nil))
	lease, err := store.Claim(context.Background(), "owner-a", now, expires)
	if err != nil || lease.Generation != 4 || lease.HasCursor {
		t.Fatalf("claim = %+v, %v", lease, err)
	}

	renewSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET lease_expires_at=CURRENT_TIMESTAMP + ($4 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3 AND lease_expires_at > CURRENT_TIMESTAMP
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`
	nextExpiry := expires.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(renewSQL)).WithArgs(paymentReconciliationScope, "owner-a", int64(4), int64((2 * time.Minute).Microseconds())).
		WillReturnRows(sqlmock.NewRows([]string{"lease_generation", "lease_expires_at", "cursor_created_at", "cursor_event_id"}).AddRow(int64(4), nextExpiry, nil, nil))
	lease, err = store.Renew(context.Background(), lease, now, nextExpiry)
	if err != nil || !lease.ExpiresAt.After(now) {
		t.Fatalf("renew = %+v, %v", lease, err)
	}

	advanceSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET cursor_created_at=$4, cursor_event_id=$5, updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3 AND lease_expires_at > CURRENT_TIMESTAMP
		  AND (cursor_created_at IS NULL OR cursor_created_at < $4 OR (cursor_created_at=$4 AND cursor_event_id <= $5))
	`
	cursor := PaymentReconciliationCursor{CreatedAt: now, EventID: "evt_a"}
	mock.ExpectExec(regexp.QuoteMeta(advanceSQL)).WithArgs(paymentReconciliationScope, "owner-a", int64(4), now, "evt_a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.Advance(context.Background(), lease, cursor, now); err != nil {
		t.Fatal(err)
	}
	releaseSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET lease_owner='', lease_expires_at=NULL, updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3
	`
	mock.ExpectExec(regexp.QuoteMeta(releaseSQL)).WithArgs(paymentReconciliationScope, "owner-a", int64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.Release(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLPaymentReconciliationStoreRejectsHeldAndStaleLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewSQLPaymentReconciliationStateStore(db)
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payment_reconciliation_checkpoint(scope_id) VALUES($1) ON CONFLICT(scope_id) DO NOTHING`)).WithArgs(paymentReconciliationScope).WillReturnResult(sqlmock.NewResult(0, 0))
	claimSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET lease_owner=$2, lease_generation=lease_generation+1,
		    lease_expires_at=CURRENT_TIMESTAMP + ($3 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND (lease_expires_at IS NULL OR lease_expires_at <= CURRENT_TIMESTAMP)
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`
	mock.ExpectQuery(regexp.QuoteMeta(claimSQL)).WithArgs(paymentReconciliationScope, "busy-owner", int64(time.Minute.Microseconds())).
		WillReturnRows(sqlmock.NewRows([]string{"lease_generation", "lease_expires_at", "cursor_created_at", "cursor_event_id"}))
	if _, err := store.Claim(context.Background(), "busy-owner", now, expires); !errors.Is(err, ErrReconciliationLeaseHeld) {
		t.Fatalf("held claim error = %v", err)
	}
	lease := PaymentReconciliationLease{Owner: "stale-owner", Generation: 7}
	renewSQL := `
		UPDATE payment_reconciliation_checkpoint
		SET lease_expires_at=CURRENT_TIMESTAMP + ($4 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3 AND lease_expires_at > CURRENT_TIMESTAMP
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`
	mock.ExpectQuery(regexp.QuoteMeta(renewSQL)).WithArgs(paymentReconciliationScope, "stale-owner", int64(7), int64(time.Minute.Microseconds())).
		WillReturnRows(sqlmock.NewRows([]string{"lease_generation", "lease_expires_at", "cursor_created_at", "cursor_event_id"}))
	if _, err := store.Renew(context.Background(), lease, now, expires); !errors.Is(err, ErrReconciliationLeaseLost) {
		t.Fatalf("stale renew error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFinancialSchemaDeclaresDurablePaymentReconciliationCheckpoint(t *testing.T) {
	if !strings.Contains(FinancialSchema(), "CREATE TABLE IF NOT EXISTS payment_reconciliation_checkpoint") || !strings.Contains(FinancialSchema(), "lease_generation") || !strings.Contains(FinancialSchema(), "cursor_event_id") {
		t.Fatal("financial schema does not declare the durable fenced reconciliation checkpoint")
	}
}

func reconciliationEvents(at time.Time, ids ...string) []PaymentReconciliationEvent {
	events := make([]PaymentReconciliationEvent, len(ids))
	for i, id := range ids {
		events[i] = PaymentReconciliationEvent{Cursor: PaymentReconciliationCursor{CreatedAt: at, EventID: id}, Type: "invoice.paid", ObjectID: "in_test", Payload: []byte(`{"id":"in_test"}`)}
	}
	return events
}
