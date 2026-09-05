// Package provision coordinates intent, credit and provider operations.
package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"compute-manager/internal/clock"
	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
)

type Service struct {
	Intents   intent.Service
	Meter     meter.Service
	Provider  provider.Provider
	Providers *provider.Registry
	Window    time.Duration
	DB        interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	Now func() time.Time
	Mu  *sync.Mutex
}

// RenewReservations advances each held reservation before releasing its
// predecessor. A transient release failure therefore leaves two holds rather
// than leaving a running instance unreserved.
func (s Service) RenewReservations(ctx context.Context) error {
	if s.DB == nil || s.Window <= 0 {
		return nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,i.tenant,i.reservation_id,r.quantity FROM instances i JOIN reservations r ON r.id=i.reservation_id WHERE i.state='running' AND r.state='held'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type candidate struct {
		instanceID, tenant, predecessor string
		amount                          int64
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.instanceID, &item.tenant, &item.predecessor, &item.amount); err != nil {
			return err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range candidates {
		amount := float64(item.amount)
		if amount < 1 {
			amount = 1
		}
		next, err := s.Meter.Reserve(ctx, item.tenant, amount, s.Window)
		if err != nil {
			return fmt.Errorf("renew reservation for %s: %w", item.instanceID, err)
		}
		now := clock.System{}.Now
		if s.Now != nil {
			now = s.Now
		}
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO reservations (id,instance_id,supersedes,meter_key,state,held_at,quantity) VALUES (?,?,?,?,?,?,?)`, next.ID, item.instanceID, item.predecessor, "compute_minutes", "held", now().UTC().Format(time.RFC3339Nano), int64(amount)); err != nil {
			_ = s.Meter.Release(ctx, next.ID)
			return fmt.Errorf("record renewed reservation for %s: %w", item.instanceID, err)
		}
		if err := s.Meter.Release(ctx, item.predecessor); err != nil {
			return fmt.Errorf("release predecessor reservation for %s: %w", item.instanceID, err)
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=? AND state='held'`, now().UTC().Format(time.RFC3339Nano), item.predecessor); err != nil {
			return fmt.Errorf("close predecessor reservation for %s: %w", item.instanceID, err)
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE instances SET reservation_id=? WHERE id=?`, next.ID, item.instanceID); err != nil {
			return fmt.Errorf("link renewed reservation for %s: %w", item.instanceID, err)
		}
	}
	return nil
}

func (s Service) ReleaseReservation(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return s.Meter.Release(ctx, id)
}

func (s Service) FinalizeReservation(ctx context.Context, id string, amount float64) error {
	if id == "" {
		return nil
	}
	return s.Meter.Finalize(ctx, id, amount)
}

func (s Service) Create(ctx context.Context, req intent.Request, amount float64) (intent.Record, provider.Instance, error) {
	return s.create(ctx, req, amount, s.Provider)
}

func (s Service) CreateWithProvider(ctx context.Context, req intent.Request, amount float64, selected provider.Provider) (intent.Record, provider.Instance, error) {
	return s.create(ctx, req, amount, selected)
}

func (s Service) create(ctx context.Context, req intent.Request, amount float64, selected provider.Provider) (intent.Record, provider.Instance, error) {
	if selected == nil {
		return intent.Record{}, provider.Instance{}, fmt.Errorf("provider is unavailable")
	}
	if s.Mu != nil {
		s.Mu.Lock()
		defer s.Mu.Unlock()
	}
	record, err := s.Intents.CreateIntent(ctx, req)
	if err != nil {
		return intent.Record{}, provider.Instance{}, err
	}
	if record.State != intent.StateOpen {
		return record, provider.Instance{}, nil
	}
	ceilingHold, err := s.Meter.AcquireCeilingHold(ctx, req.RequestedBy, amount, s.Window)
	if err != nil {
		record.State = intent.StateRefused
		record.ResolvedAt = clock.System{}.Now().UTC()
		_ = s.Intents.Store.Update(ctx, record)
		return record, provider.Instance{}, err
	}
	defer func() { _ = s.Meter.ReleaseCeilingHold(ctx, ceilingHold) }()
	reservation, err := s.Meter.Reserve(ctx, req.RequestedBy, amount, s.Window)
	if err != nil {
		record.State = intent.StateRefused
		record.ResolvedAt = clock.System{}.Now().UTC()
		_ = s.Intents.Store.Update(ctx, record)
		return record, provider.Instance{}, fmt.Errorf("capacity refused: %w", err)
	}
	record.ReservationID = reservation.ID
	if s.DB != nil {
		now := time.Now
		if s.Now != nil {
			now = s.Now
		}
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO reservations (id,intent_id,meter_key,state,held_at,quantity) VALUES (?,?,?,?,?,?)`, reservation.ID, record.ID, "compute_minutes", "held", now().UTC().Format(time.RFC3339Nano), int64(amount)); err != nil {
			_ = s.Meter.Release(ctx, reservation.ID)
			return record, provider.Instance{}, fmt.Errorf("record reservation: %w", err)
		}
	}
	if err := s.Intents.Store.Update(ctx, record); err != nil {
		_ = s.Meter.Release(ctx, reservation.ID)
		if s.DB != nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=?`, clock.System{}.Now().UTC().Format(time.RFC3339Nano), reservation.ID)
		}
		return record, provider.Instance{}, err
	}
	req.Spec.Tags = withIntentTag(req.Spec.Tags, record.ID)
	created, err := selected.Create(ctx, req.Spec)
	if err != nil {
		if errors.Is(err, provider.ErrCreateResponseLost) {
			return record, provider.Instance{}, err
		}
		_ = s.Meter.Release(ctx, reservation.ID)
		if s.DB != nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=?`, clock.System{}.Now().UTC().Format(time.RFC3339Nano), reservation.ID)
		}
		record.State = intent.StateRefused
		record.ResolvedAt = time.Now().UTC()
		_ = s.Intents.Store.Update(ctx, record)
		return record, provider.Instance{}, err
	}
	record.State = intent.StateFulfilled
	record.InstanceID = created.ID
	record.ResolvedAt = clock.System{}.Now().UTC()
	if err := s.Intents.Store.Update(ctx, record); err != nil {
		return record, created, err
	}
	if s.DB != nil {
		_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET instance_id=? WHERE id=?`, created.ID, reservation.ID)
	}
	return record, created, nil
}

func withIntentTag(tags map[string]string, intentID string) map[string]string {
	result := make(map[string]string, len(tags)+1)
	for key, value := range tags {
		result[key] = value
	}
	result["vrooli-intent-id"] = intentID
	return result
}
