package expiry

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"compute-manager/internal/provider"
)

type DB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Sweeper struct {
	DB       DB
	Provider provider.Provider
	Finalize func(context.Context, string, float64) error
	Now      func() time.Time
}

// Run destroys only expired, locally-running instances and settles their
// reservations through the same billing boundary used by the request path.
func (s Sweeper) Run(ctx context.Context) error {
	if s.DB == nil || s.Provider == nil {
		return fmt.Errorf("expiry dependencies are unavailable")
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,provider_instance_id,reservation_id,created_at FROM instances WHERE state='running' AND expires_at<>'' AND expires_at<=?`, now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	defer rows.Close()
	type candidate struct{ id, providerID, reservationID, created string }
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.providerID, &item.reservationID, &item.created); err != nil {
			_ = rows.Close()
			return err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range candidates {
		if err := s.Provider.Destroy(ctx, item.providerID); err != nil {
			return fmt.Errorf("destroy expired instance %s: %w", item.id, err)
		}
		if s.Finalize != nil && item.reservationID != "" {
			created, parseErr := time.Parse(time.RFC3339Nano, item.created)
			if parseErr != nil {
				return fmt.Errorf("parse creation time for %s: %w", item.id, parseErr)
			}
			amount := now().UTC().Sub(created).Minutes()
			if amount < 1 {
				amount = 1
			}
			if err := s.Finalize(ctx, item.reservationID, amount); err != nil {
				return fmt.Errorf("settle expired instance %s: %w", item.id, err)
			}
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE instances SET state='destroyed',destroyed_at=? WHERE id=?`, now().UTC().Format(time.RFC3339Nano), item.id); err != nil {
			return err
		}
	}
	return nil
}
