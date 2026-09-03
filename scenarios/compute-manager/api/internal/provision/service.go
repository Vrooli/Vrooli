// Package provision coordinates intent, credit and provider operations.
package provision

import (
	"context"
	"fmt"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
)

type Service struct {
	Intents  intent.Service
	Meter    meter.Service
	Provider provider.Provider
	Window   time.Duration
}

func (s Service) Create(ctx context.Context, req intent.Request, amount float64) (intent.Record, provider.Instance, error) {
	record, err := s.Intents.CreateIntent(ctx, req)
	if err != nil {
		return intent.Record{}, provider.Instance{}, err
	}
	if record.State != intent.StateOpen {
		return record, provider.Instance{}, nil
	}
	reservation, err := s.Meter.Reserve(ctx, req.RequestedBy, amount, s.Window)
	if err != nil {
		record.State = intent.StateRefused
		record.ResolvedAt = time.Now().UTC()
		_ = s.Intents.Store.Update(ctx, record)
		return record, provider.Instance{}, fmt.Errorf("capacity refused: %w", err)
	}
	record.ReservationID = reservation.ID
	if err := s.Intents.Store.Update(ctx, record); err != nil {
		_ = s.Meter.Release(ctx, reservation.ID)
		return record, provider.Instance{}, err
	}
	created, err := s.Provider.Create(ctx, req.Spec)
	if err != nil {
		_ = s.Meter.Release(ctx, reservation.ID)
		return record, provider.Instance{}, err
	}
	record.State = intent.StateFulfilled
	record.InstanceID = created.ID
	record.ResolvedAt = time.Now().UTC()
	if err := s.Intents.Store.Update(ctx, record); err != nil {
		return record, created, err
	}
	return record, created, nil
}
