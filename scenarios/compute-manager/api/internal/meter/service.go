// Package meter is the only compute-manager package allowed to coordinate
// business-suite credit reservations.
package meter

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInsufficientCredits = errors.New("insufficient credits")

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
