package monetization

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidUsage = errors.New("monetization usage is invalid")
	ErrNilOutbox    = errors.New("monetization outbox is nil")
)

const (
	outboxDefaultMaxRetries = 8
	outboxDefaultMaxDelay   = 5 * time.Minute
	outboxDurationShiftBits = 62
)

// Usage is the immutable event submitted to the billing authority. OperationID
// is the idempotency key and must be preserved by every transport adapter.
type Usage struct {
	OperationID  string
	UserIdentity string
	BundleKey    string
	AppKey       string
	MeterKey     string
	Units        int64
	OccurredAt   time.Time
	Metadata     map[string]string
}

// OutboxRecord is the durable delivery state for one usage event.
type OutboxRecord struct {
	Usage         Usage
	Attempts      int
	NextAttemptAt time.Time
	LastError     string
	DeliveredAt   time.Time
}

func (r OutboxRecord) Delivered() bool { return !r.DeliveredAt.IsZero() }

// OutboxStore is implemented by a scenario's durable database adapter. Append
// must be idempotent on Usage.OperationID. Pending must return records whose
// NextAttemptAt is at or before now, and MarkDelivered/MarkRetry must be
// durable before the worker acknowledges the item.
type OutboxStore interface {
	Append(context.Context, Usage) (inserted bool, err error)
	Pending(context.Context, int, time.Time) ([]OutboxRecord, error)
	MarkDelivered(context.Context, string, time.Time) error
	MarkRetry(context.Context, string, time.Time, string) error
}

// PendingCounter is an optional durable-store capability used by account
// surfaces to display queued Class B usage. Implementations must count stored
// rows, not an in-memory worker queue.
type PendingCounter interface {
	PendingCount(context.Context, string) (int, error)
}

// UsageTransport sends one event to the trusted billing authority. The
// authority must deduplicate OperationID so a crash between send and
// MarkDelivered cannot double-charge a customer.
type UsageTransport interface {
	Report(context.Context, Usage) error
}

// Outbox coordinates durable enqueue and retryable delivery. It deliberately
// does not contain SQL or billing policy; scenarios own only the storage and
// transport adapters for their declared meter.
type Outbox struct {
	Store      OutboxStore
	Transport  UsageTransport
	Now        func() time.Time
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// NewOutbox creates an outbox with conservative retry defaults.
func NewOutbox(store OutboxStore, transport UsageTransport) *Outbox {
	return &Outbox{
		Store:      store,
		Transport:  transport,
		Now:        time.Now,
		MaxRetries: outboxDefaultMaxRetries,
		BaseDelay:  time.Second,
		MaxDelay:   outboxDefaultMaxDelay,
	}
}

// NewOperationID generates a non-secret, collision-resistant idempotency key.
func NewOperationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate operation id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

// Enqueue durably records one usage event. Re-enqueuing the same operation is
// a successful no-op when the store reports inserted=false.
func (o *Outbox) Enqueue(ctx context.Context, usage Usage) error {
	if o == nil || o.Store == nil {
		return ErrNilOutbox
	}
	if err := validateUsage(usage); err != nil {
		return err
	}
	_, err := o.Store.Append(ctx, cloneUsage(usage))
	return err
}

// Drain attempts up to limit ready records. It returns the number delivered
// and the first transport or persistence error, if any. A failed record is
// left durable with an exponential retry time; later records still run.
func (o *Outbox) Drain(ctx context.Context, limit int) (delivered int, firstErr error) {
	if o == nil || o.Store == nil || o.Transport == nil {
		return 0, ErrNilOutbox
	}
	if limit <= 0 {
		return 0, nil
	}
	now := o.now()
	records, err := o.Store.Pending(ctx, limit, now)
	if err != nil {
		return 0, err
	}
	for _, record := range records {
		if record.Delivered() {
			continue
		}
		if err := o.Transport.Report(ctx, cloneUsage(record.Usage)); err != nil {
			retryAt := now.Add(o.retryDelay(record.Attempts + 1))
			markErr := o.Store.MarkRetry(ctx, record.Usage.OperationID, retryAt, err.Error())
			if firstErr == nil {
				firstErr = err
			}
			if markErr != nil && firstErr == nil {
				firstErr = markErr
			}
			continue
		}
		if err := o.Store.MarkDelivered(ctx, record.Usage.OperationID, now); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		delivered++
	}
	return delivered, firstErr
}

// PendingCount reports durable undelivered usage for one identity. A store
// that does not expose a count returns ErrPendingCountUnsupported rather than
// pretending the queue is empty.
func (o *Outbox) PendingCount(ctx context.Context, userIdentity string) (int, error) {
	if o == nil || o.Store == nil {
		return 0, ErrNilOutbox
	}
	counter, ok := o.Store.(PendingCounter)
	if !ok {
		return 0, ErrPendingCountUnsupported
	}
	return counter.PendingCount(ctx, userIdentity)
}

func (o *Outbox) now() time.Time {
	if o.Now == nil {
		return time.Now()
	}
	return o.Now()
}

func (o *Outbox) retryDelay(attempt int) time.Duration {
	base := o.BaseDelay
	if base <= 0 {
		base = time.Second
	}
	delay := base
	for i := 1; i < attempt; i++ {
		if delay > time.Duration(1<<outboxDurationShiftBits)/2 {
			break
		}
		delay *= 2
	}
	if o.MaxDelay > 0 && delay > o.MaxDelay {
		return o.MaxDelay
	}
	return delay
}

func validateUsage(usage Usage) error {
	if strings.TrimSpace(usage.OperationID) == "" ||
		strings.TrimSpace(usage.UserIdentity) == "" ||
		strings.TrimSpace(usage.BundleKey) == "" ||
		strings.TrimSpace(usage.AppKey) == "" ||
		strings.TrimSpace(usage.MeterKey) == "" ||
		usage.Units < 0 || usage.OccurredAt.IsZero() {
		return ErrInvalidUsage
	}
	return nil
}

func cloneUsage(usage Usage) Usage {
	usage.Metadata = cloneMetadata(usage.Metadata)
	return usage
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

// MemoryOutboxStore is a deterministic test double and a reference for
// scenario database adapters. Production scenarios should use a durable
// unique key on operation_id rather than this in-memory implementation.
type MemoryOutboxStore struct {
	mu      sync.Mutex
	records map[string]OutboxRecord
}

func NewMemoryOutboxStore() *MemoryOutboxStore {
	return &MemoryOutboxStore{records: make(map[string]OutboxRecord)}
}

func (s *MemoryOutboxStore) Append(_ context.Context, usage Usage) (bool, error) {
	if s == nil {
		return false, ErrNilOutbox
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[usage.OperationID]; exists {
		return false, nil
	}
	s.records[usage.OperationID] = OutboxRecord{Usage: cloneUsage(usage)}
	return true, nil
}

func (s *MemoryOutboxStore) Pending(_ context.Context, limit int, now time.Time) ([]OutboxRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]OutboxRecord, 0, limit)
	for _, record := range s.records {
		if record.Delivered() || record.NextAttemptAt.After(now) {
			continue
		}
		result = append(result, cloneRecord(record))
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (s *MemoryOutboxStore) MarkDelivered(_ context.Context, operationID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[operationID]
	if !exists {
		return fmt.Errorf("operation %q not found", operationID)
	}
	record.DeliveredAt = at
	record.LastError = ""
	s.records[operationID] = record
	return nil
}

func (s *MemoryOutboxStore) MarkRetry(_ context.Context, operationID string, next time.Time, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[operationID]
	if !exists {
		return fmt.Errorf("operation %q not found", operationID)
	}
	record.Attempts++
	record.NextAttemptAt = next
	record.LastError = reason
	s.records[operationID] = record
	return nil
}

func cloneRecord(record OutboxRecord) OutboxRecord {
	record.Usage = cloneUsage(record.Usage)
	return record
}

var ErrPendingCountUnsupported = errors.New("monetization outbox pending count is unsupported")

// PendingCount returns the number of undelivered records for an identity in
// the deterministic test store.
func (s *MemoryOutboxStore) PendingCount(_ context.Context, userIdentity string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, record := range s.records {
		if record.Delivered() || (strings.TrimSpace(userIdentity) != "" && record.Usage.UserIdentity != userIdentity) {
			continue
		}
		count++
	}
	return count, nil
}
