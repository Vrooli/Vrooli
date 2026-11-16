package intent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"compute-manager/internal/clock"
	"compute-manager/internal/provider"
	"github.com/google/uuid"
)

type State string

const (
	// StateReserving is retained for durable rows written by older releases.
	StateReserving State = "reserving"
	StateOpen      State = "open"
	StateFulfilled State = "fulfilled"
	StateRefused   State = "refused"
	// StateAbandoned is a legacy terminal state; new failures use StateRefused.
	StateAbandoned State = "abandoned"
)

var ErrInvalidTransition = errors.New("invalid intent state transition")

func CanTransition(from, to State) bool {
	if from == StateOpen {
		return to == StateFulfilled || to == StateRefused
	}
	return from == StateReserving &&
		(to == StateFulfilled || to == StateRefused || to == StateAbandoned)
}

func validState(state State) bool {
	return state == StateReserving || state == StateOpen || state == StateFulfilled || state == StateRefused || state == StateAbandoned
}

type Record struct {
	ID             string
	IdempotencyKey string
	RequestedBy    string
	Provider       string
	Spec           map[string]any
	ReservationID  string
	State          State
	InstanceID     string
	CreatedAt      time.Time
	ResolvedAt     time.Time
}

type Store interface {
	Create(context.Context, Record) (Record, error)
	GetByIDOrKey(context.Context, string) (Record, error)
	SaveOpen(context.Context, Record) error
	Transition(context.Context, State, Record) error
}

type OpenStore interface {
	ListOpen(context.Context) ([]Record, error)
}

type Service struct {
	Store    Store
	Provider provider.Provider
	Now      func() time.Time
}

type Request struct {
	IdempotencyKey string
	RequestedBy    string
	Provider       string
	Spec           provider.Spec
}

// Create records the intent before invoking the provider. A lost provider
// response intentionally leaves the intent open for reconciliation.
func (s Service) Create(ctx context.Context, req Request) (Record, provider.Instance, error) {
	record, err := s.CreateIntent(ctx, req)
	if err != nil {
		return Record{}, provider.Instance{}, err
	}
	if record.State != StateOpen {
		return record, provider.Instance{}, nil
	}
	req.Spec.Tags = withIntentTag(req.Spec.Tags, record.ID)
	return s.Fulfill(ctx, record, req.Spec)
}

// RecoverOpen matches provider resources carrying an intent tag to durable
// open intents left behind by a lost create response. It never creates or
// destroys a provider resource.
func (s Service) RecoverOpen(ctx context.Context) ([]Record, error) {
	store, ok := s.Store.(OpenStore)
	if !ok || s.Provider == nil {
		return nil, nil
	}
	open, err := store.ListOpen(ctx)
	if err != nil {
		return nil, err
	}
	observed, err := s.Provider.List(ctx)
	if err != nil {
		return nil, err
	}
	byIntent := make(map[string]provider.Instance)
	for _, item := range observed {
		if id := item.Tags["vrooli-intent-id"]; id != "" {
			byIntent[id] = item
		}
	}
	recovered := make([]Record, 0)
	now := clock.System{}.Now
	if s.Now != nil {
		now = s.Now
	}
	for _, record := range open {
		item, found := byIntent[record.ID]
		if !found {
			continue
		}
		record.InstanceID = item.ID
		record.ResolvedAt = now().UTC()
		if err := s.Transition(ctx, &record, StateFulfilled); err != nil {
			return recovered, err
		}
		recovered = append(recovered, record)
	}
	return recovered, nil
}

func withIntentTag(tags map[string]string, intentID string) map[string]string {
	result := make(map[string]string, len(tags)+1)
	for key, value := range tags {
		result[key] = value
	}
	result["vrooli-intent-id"] = intentID
	return result
}

// CreateIntent durably records the requested operation before any external
// reservation or provider call.
func (s Service) CreateIntent(ctx context.Context, req Request) (Record, error) {
	if req.IdempotencyKey == "" {
		return Record{}, fmt.Errorf("idempotency key is required")
	}
	if existing, err := s.Store.GetByIDOrKey(ctx, req.IdempotencyKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Record{}, err
	}
	now := clock.System{}.Now
	if s.Now != nil {
		now = s.Now
	}
	record, err := s.Store.Create(ctx, Record{IdempotencyKey: req.IdempotencyKey, RequestedBy: req.RequestedBy, Provider: req.Provider, Spec: map[string]any{"region": req.Spec.Region, "size": req.Spec.Size, "image": req.Spec.Image}, CreatedAt: now().UTC(), State: StateOpen})
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

// Fulfill calls the provider only after CreateIntent has succeeded.
func (s Service) Fulfill(ctx context.Context, record Record, spec provider.Spec) (Record, provider.Instance, error) {
	instance, err := s.Provider.Create(ctx, spec)
	if err != nil {
		return record, provider.Instance{}, err
	}
	record.InstanceID = instance.ID
	now := clock.System{}.Now
	if s.Now != nil {
		now = s.Now
	}
	record.ResolvedAt = now().UTC()
	if err := s.Transition(ctx, &record, StateFulfilled); err != nil {
		return record, instance, err
	}
	return record, instance, nil
}

// Transition is the sole domain owner for persisted intent state changes.
func (s Service) Transition(ctx context.Context, record *Record, to State) error {
	if record == nil || !CanTransition(record.State, to) {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, stateOf(record), to)
	}
	from := record.State
	record.State = to
	if err := s.Store.Transition(ctx, from, *record); err != nil {
		record.State = from
		return err
	}
	return nil
}

func stateOf(record *Record) State {
	if record == nil {
		return ""
	}
	return record.State
}

// SaveOpen persists reservation metadata without allowing an arbitrary state change.
func (s Service) SaveOpen(ctx context.Context, record Record) error {
	if record.State != StateOpen {
		return fmt.Errorf("%w: open metadata save has state %q", ErrInvalidTransition, record.State)
	}
	return s.Store.SaveOpen(ctx, record)
}

type SQLStore struct {
	DB interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	Now func() time.Time
}

func (s SQLStore) ListOpen(ctx context.Context) ([]Record, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,idempotency_key,requested_by,provider,spec_json,reservation_id,state,instance_id,created_at,resolved_at FROM instance_intents WHERE state IN (?,?) ORDER BY created_at`, StateOpen, StateReserving)
	if err != nil {
		return nil, fmt.Errorf("list open intents: %w", err)
	}
	defer rows.Close()
	result := make([]Record, 0)
	for rows.Next() {
		var record Record
		var spec, created, resolved string
		if err := rows.Scan(&record.ID, &record.IdempotencyKey, &record.RequestedBy, &record.Provider, &spec, &record.ReservationID, &record.State, &record.InstanceID, &created, &resolved); err != nil {
			return nil, fmt.Errorf("scan open intent: %w", err)
		}
		if err := json.Unmarshal([]byte(spec), &record.Spec); err != nil {
			return nil, fmt.Errorf("decode open intent spec: %w", err)
		}
		if record.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, fmt.Errorf("parse open intent created_at: %w", err)
		}
		if resolved != "" {
			if record.ResolvedAt, err = time.Parse(time.RFC3339Nano, resolved); err != nil {
				return nil, fmt.Errorf("parse open intent resolved_at: %w", err)
			}
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate open intents: %w", err)
	}
	return result, nil
}

var ErrNotFound = errors.New("intent not found")

func (s SQLStore) Create(ctx context.Context, record Record) (Record, error) {
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		now := s.Now
		if now == nil {
			now = clock.System{}.Now
		}
		record.CreatedAt = now().UTC()
	}
	if record.State == "" {
		record.State = StateOpen
	}
	if record.State != StateOpen {
		return Record{}, fmt.Errorf("%w: create state %q", ErrInvalidTransition, record.State)
	}
	spec, err := json.Marshal(record.Spec)
	if err != nil {
		return Record{}, fmt.Errorf("marshal intent spec: %w", err)
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO instance_intents (id,idempotency_key,requested_by,provider,spec_json,reservation_id,state,instance_id,created_at,resolved_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, record.ID, record.IdempotencyKey, record.RequestedBy, record.Provider, string(spec), record.ReservationID, record.State, record.InstanceID, record.CreatedAt.UTC().Format(time.RFC3339Nano), formatTime(record.ResolvedAt))
	if err != nil {
		return Record{}, fmt.Errorf("create intent: %w", err)
	}
	return record, nil
}

func (s SQLStore) GetByIDOrKey(ctx context.Context, value string) (Record, error) {
	var r Record
	var spec, created, resolved string
	err := s.DB.QueryRowContext(ctx, `SELECT id,idempotency_key,requested_by,provider,spec_json,reservation_id,state,instance_id,created_at,resolved_at FROM instance_intents WHERE id=? OR idempotency_key=? LIMIT 1`, value, value).Scan(&r.ID, &r.IdempotencyKey, &r.RequestedBy, &r.Provider, &spec, &r.ReservationID, &r.State, &r.InstanceID, &created, &resolved)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("get intent: %w", err)
	}
	if !validState(r.State) {
		return Record{}, fmt.Errorf("invalid persisted intent state %q", r.State)
	}
	if r.Spec = map[string]any{}; spec != "" {
		if err := json.Unmarshal([]byte(spec), &r.Spec); err != nil {
			return Record{}, fmt.Errorf("decode intent spec: %w", err)
		}
	}
	if r.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Record{}, fmt.Errorf("parse intent created_at: %w", err)
	}
	if resolved != "" {
		r.ResolvedAt, err = time.Parse(time.RFC3339Nano, resolved)
		if err != nil {
			return Record{}, fmt.Errorf("parse intent resolved_at: %w", err)
		}
	}
	return r, nil
}

func (s SQLStore) SaveOpen(ctx context.Context, r Record) error {
	if r.State != StateOpen {
		return fmt.Errorf("%w: save-open state %q", ErrInvalidTransition, r.State)
	}
	spec, err := json.Marshal(r.Spec)
	if err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE instance_intents SET reservation_id=?,spec_json=? WHERE id=? AND state=?`, r.ReservationID, string(spec), r.ID, StateOpen)
	if err != nil {
		return fmt.Errorf("save open intent: %w", err)
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		return fmt.Errorf("save open intent: %w", ErrNotFound)
	}
	return nil
}

func (s SQLStore) Transition(ctx context.Context, from State, r Record) error {
	if !CanTransition(from, r.State) {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, from, r.State)
	}
	spec, err := json.Marshal(r.Spec)
	if err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE instance_intents SET reservation_id=?,state=?,instance_id=?,resolved_at=?,spec_json=? WHERE id=? AND state=?`, r.ReservationID, r.State, r.InstanceID, formatTime(r.ResolvedAt), string(spec), r.ID, from)
	if err != nil {
		return fmt.Errorf("transition intent: %w", err)
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		return fmt.Errorf("transition intent: %w", ErrNotFound)
	}
	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
