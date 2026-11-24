package commerce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrReconciliationLeaseHeld    = errors.New("payment reconciliation lease is held")
	ErrReconciliationLeaseLost    = errors.New("payment reconciliation lease was lost")
	ErrInvalidReconciliationEvent = errors.New("invalid or out-of-order payment reconciliation event")
)

const (
	DefaultPaymentReconciliationInterval = time.Minute
	DefaultPaymentReconciliationLease    = 30 * time.Second
	DefaultPaymentReconciliationBatch    = 100
)

// PaymentReconciliationCursor is a strict, stable ordering key. Stripe event
// creation timestamps can collide, so EventID is the deterministic tiebreaker.
type PaymentReconciliationCursor struct {
	CreatedAt time.Time
	EventID   string
}

func (c PaymentReconciliationCursor) Compare(other PaymentReconciliationCursor) int {
	createdAt := c.CreatedAt.UTC().Truncate(time.Microsecond)
	otherCreatedAt := other.CreatedAt.UTC().Truncate(time.Microsecond)
	if createdAt.Before(otherCreatedAt) {
		return -1
	}
	if createdAt.After(otherCreatedAt) {
		return 1
	}
	switch {
	case c.EventID < other.EventID:
		return -1
	case c.EventID > other.EventID:
		return 1
	default:
		return 0
	}
}

func (c PaymentReconciliationCursor) Valid() bool {
	return !c.CreatedAt.IsZero() && c.EventID != ""
}

// PaymentReconciliationEvent carries only the provider's stable event identity
// and the exact payload needed by the owner processor. The event payload stays
// in memory and must never be placed in lease, cursor, or alert metadata.
type PaymentReconciliationEvent struct {
	Cursor   PaymentReconciliationCursor
	LiveMode bool
	Type     string
	ObjectID string
	Payload  []byte
}

func (e PaymentReconciliationEvent) IdempotencyKey() string {
	mode := "test"
	if e.LiveMode {
		mode = "live"
	}
	return "stripe:" + mode + ":" + e.Cursor.EventID
}

type PaymentReconciliationLease struct {
	Owner      string
	Generation int64
	ExpiresAt  time.Time
	Cursor     PaymentReconciliationCursor
	HasCursor  bool
}

// PaymentReconciliationStateStore is the durable fenced lease and checkpoint
// boundary. Implementations must condition every mutation on owner+generation.
type PaymentReconciliationStateStore interface {
	Claim(context.Context, string, time.Time, time.Time) (PaymentReconciliationLease, error)
	Renew(context.Context, PaymentReconciliationLease, time.Time, time.Time) (PaymentReconciliationLease, error)
	Advance(context.Context, PaymentReconciliationLease, PaymentReconciliationCursor, time.Time) error
	Release(context.Context, PaymentReconciliationLease) error
}

type PaymentReconciliationSource interface {
	ListAfter(context.Context, PaymentReconciliationCursor, bool, int) ([]PaymentReconciliationEvent, error)
}

// PaymentReconciliationProcessor applies one owner-authorized event and must
// deduplicate by Event.IdempotencyKey(). The runner checkpoints only after the
// transition succeeds. If it dies between apply and checkpoint, the same event
// is retried and the owner processor returns a typed replay/no-op result.
type PaymentReconciliationProcessor interface {
	Apply(context.Context, PaymentReconciliationEvent) (PaymentReconciliationOutcome, error)
}

type PaymentReconciliationOutcome string

const (
	PaymentReconciliationApplied  PaymentReconciliationOutcome = "applied"
	PaymentReconciliationReplay   PaymentReconciliationOutcome = "replay"
	PaymentReconciliationNoop     PaymentReconciliationOutcome = "noop"
	PaymentReconciliationConflict PaymentReconciliationOutcome = "conflict"
)

type PaymentReconciliationReport struct {
	EventsRead      int
	Applied         int
	Replayed        int
	Noop            int
	Cursor          PaymentReconciliationCursor
	HasCursor       bool
	LeaseGeneration int64
}

type PaymentReconciliationRunner struct {
	State         PaymentReconciliationStateStore
	Source        PaymentReconciliationSource
	Processor     PaymentReconciliationProcessor
	Now           func() time.Time
	NewOwner      func() (string, error)
	LeaseDuration time.Duration
	BatchSize     int
	OnError       func(error)
	OnSuccess     func(PaymentReconciliationReport)
}

// Run executes one reconciliation immediately, then repeats at a bounded
// cadence until shutdown. Competing replicas rely on the durable lease; a held
// lease is normal and does not emit an alert. Other failures are observable via
// OnError and retried on the next cadence without advancing the cursor.
func (r PaymentReconciliationRunner) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = DefaultPaymentReconciliationInterval
	}
	if interval < time.Second {
		interval = time.Second
	}
	if interval > time.Hour {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		report, runErr := r.RunOnce(ctx)
		if runErr != nil && !errors.Is(runErr, ErrReconciliationLeaseHeld) && !errors.Is(runErr, context.Canceled) && !errors.Is(runErr, context.DeadlineExceeded) {
			if r.OnError != nil {
				r.OnError(runErr)
			}
		} else if runErr == nil && r.OnSuccess != nil {
			r.OnSuccess(report)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r PaymentReconciliationRunner) RunOnce(ctx context.Context) (report PaymentReconciliationReport, err error) {
	if r.State == nil || r.Source == nil || r.Processor == nil {
		return report, errors.New("payment reconciliation runner dependencies are incomplete")
	}
	now := r.Now
	if now == nil {
		now = time.Now
	}
	newOwner := r.NewOwner
	if newOwner == nil {
		newOwner = newReconciliationOwner
	}
	leaseDuration := r.LeaseDuration
	if leaseDuration <= 0 {
		leaseDuration = DefaultPaymentReconciliationLease
	}
	batchSize := r.BatchSize
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = DefaultPaymentReconciliationBatch
	}
	owner, err := newOwner()
	if err != nil {
		return report, fmt.Errorf("create payment reconciliation owner: %w", err)
	}
	claimedAt := now().UTC()
	lease, err := r.State.Claim(ctx, owner, claimedAt, claimedAt.Add(leaseDuration))
	if err != nil {
		return report, err
	}
	defer func() {
		releaseErr := r.State.Release(context.WithoutCancel(ctx), lease)
		if err == nil && releaseErr != nil {
			err = fmt.Errorf("release payment reconciliation lease: %w", releaseErr)
		}
	}()
	report.Cursor, report.HasCursor, report.LeaseGeneration = lease.Cursor, lease.HasCursor, lease.Generation
	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		nowAt := now().UTC()
		lease, err = r.State.Renew(ctx, lease, nowAt, nowAt.Add(leaseDuration))
		if err != nil {
			return report, err
		}
		cursor := report.Cursor
		if !report.HasCursor {
			cursor = PaymentReconciliationCursor{}
		}
		events, listErr := r.Source.ListAfter(ctx, cursor, report.HasCursor, batchSize)
		if listErr != nil {
			return report, fmt.Errorf("list Stripe reconciliation events: %w", listErr)
		}
		if len(events) > batchSize {
			return report, fmt.Errorf("payment reconciliation source exceeded batch bound %d", batchSize)
		}
		if len(events) == 0 {
			return report, nil
		}
		previous, hasPrevious := report.Cursor, report.HasCursor
		for _, event := range events {
			if !event.Cursor.Valid() || event.Type == "" || event.ObjectID == "" || len(event.Payload) == 0 || hasPrevious && event.Cursor.Compare(previous) <= 0 {
				return report, ErrInvalidReconciliationEvent
			}
			outcome, applyErr := r.Processor.Apply(ctx, event)
			if applyErr != nil {
				return report, fmt.Errorf("apply Stripe reconciliation event %q: %w", event.Cursor.EventID, applyErr)
			}
			switch outcome {
			case PaymentReconciliationApplied:
				report.Applied++
			case PaymentReconciliationReplay:
				report.Replayed++
			case PaymentReconciliationNoop:
				report.Noop++
			case PaymentReconciliationConflict:
				return report, fmt.Errorf("Stripe reconciliation event %q conflicts with its prior receipt", event.Cursor.EventID)
			default:
				return report, fmt.Errorf("Stripe reconciliation event %q returned invalid outcome %q", event.Cursor.EventID, outcome)
			}
			nowAt = now().UTC()
			lease, err = r.State.Renew(ctx, lease, nowAt, nowAt.Add(leaseDuration))
			if err != nil {
				return report, err
			}
			if err = r.State.Advance(ctx, lease, event.Cursor, nowAt); err != nil {
				return report, err
			}
			report.Cursor, report.HasCursor = event.Cursor, true
			report.EventsRead++
			previous, hasPrevious = event.Cursor, true
		}
		// RunOnce consumes at most one provider batch. The durable cursor lets
		// the next cadence resume, bounding database/provider work and preventing
		// a recovered historical backlog from monopolizing a process forever.
		return report, nil
	}
}

func newReconciliationOwner() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
