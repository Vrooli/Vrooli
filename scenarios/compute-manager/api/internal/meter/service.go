// Package meter is the only compute-manager package allowed to coordinate
// business-suite credit reservations.
package meter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrCeilingExceeded     = errors.New("tenant compute ceiling exceeded")
)

type Reservation struct {
	ID        string
	ExpiresAt time.Time
}

type Credits interface {
	ReserveCredits(ctx context.Context, user, limit string, amount float64, window time.Duration) (Reservation, error)
	ReleaseReservation(ctx context.Context, id string) error
	FinalizeReservation(ctx context.Context, id string, amount float64) error
}

type Service struct {
	Credits Credits
	DB      interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	TenantCeilingMinutes int64
	Now                  func() time.Time
}

// AcquireCeilingHold atomically claims room under the tenant ceiling before
// an external credit reservation is attempted. The conditional INSERT is a
// single database write, so separate API processes cannot both pass a
// read-then-write check. Expired holds stop counting automatically.
func (s Service) AcquireCeilingHold(ctx context.Context, tenant string, amount float64, window time.Duration) (string, error) {
	if s.DB == nil || s.TenantCeilingMinutes <= 0 || tenant == "" {
		return "", nil
	}
	requested := int64(amount)
	if requested < 1 {
		requested = 1
	}
	holdID := uuid.NewString()
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	current := now().UTC()
	expires := current.Add(window).Format(time.RFC3339Nano)
	result, err := s.DB.ExecContext(ctx, `INSERT INTO tenant_ceiling_holds (id,tenant,quantity,expires_at) SELECT ?,?,?,? WHERE (? + COALESCE((SELECT SUM(quantity) FROM tenant_ceiling_holds WHERE tenant=? AND expires_at>?),0)) <= ?`, holdID, tenant, requested, expires, requested, tenant, current.Format(time.RFC3339Nano), s.TenantCeilingMinutes)
	if err != nil {
		return "", fmt.Errorf("acquire tenant ceiling hold: %w", err)
	}
	if result == nil {
		return "", fmt.Errorf("acquire tenant ceiling hold: database returned no result")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("check tenant ceiling hold: %w", err)
	}
	if rows == 0 {
		return "", &CeilingError{Used: 0, Requested: requested, Limit: s.TenantCeilingMinutes}
	}
	return holdID, nil
}

func (s Service) ReleaseCeilingHold(ctx context.Context, id string) error {
	if id == "" || s.DB == nil {
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM tenant_ceiling_holds WHERE id=?`, id)
	return err
}

type CeilingError struct {
	Used, Requested, Limit int64
}

func (e *CeilingError) Error() string {
	return fmt.Sprintf("%s: tenant usage %d + requested %d exceeds ceiling %d; upgrade required", ErrCeilingExceeded, e.Used, e.Requested, e.Limit)
}

func (e *CeilingError) Unwrap() error { return ErrCeilingExceeded }

func (s Service) CheckCeiling(ctx context.Context, user string, amount float64) error {
	if s.DB == nil || s.TenantCeilingMinutes <= 0 || user == "" {
		return nil
	}
	var used int64
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(r.quantity),0) FROM reservations r JOIN instance_intents i ON i.id=r.intent_id WHERE i.requested_by=? AND r.state='held'`, user).Scan(&used); err != nil {
		return fmt.Errorf("read tenant compute usage: %w", err)
	}
	requested := int64(amount)
	if requested < 1 {
		requested = 1
	}
	if used+requested > s.TenantCeilingMinutes {
		return &CeilingError{Used: used, Requested: requested, Limit: s.TenantCeilingMinutes}
	}
	return nil
}

func (s Service) Reserve(ctx context.Context, user string, amount float64, window time.Duration) (Reservation, error) {
	if s.Credits == nil {
		return Reservation{}, fmt.Errorf("meter credits service is unavailable")
	}
	r, err := s.Credits.ReserveCredits(ctx, user, "compute_minutes", amount, window)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) {
			return Reservation{}, fmt.Errorf("%w: upgrade required", err)
		}
		return Reservation{}, err
	}
	return r, nil
}

func (s Service) Release(ctx context.Context, id string) error {
	if s.Credits == nil {
		return fmt.Errorf("meter credits service is unavailable")
	}
	return s.Credits.ReleaseReservation(ctx, id)
}

func (s Service) Finalize(ctx context.Context, id string, amount float64) error {
	if s.Credits == nil {
		return fmt.Errorf("meter credits service is unavailable")
	}
	return s.Credits.FinalizeReservation(ctx, id, amount)
}
