package journal

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type ReverseInput struct {
	OriginalEventID string
	Reason          string
	IdempotencyKey  string
	ActorIdentity   string
}

type InvalidReversalError struct{ Reason string }

func (e *InvalidReversalError) Error() string { return "invalid reversal: " + e.Reason }

// Service is read-only at the domain boundary. Event admission goes through
// the separately injected Appender seam used by owning mutation domains.
type Service interface {
	Events(context.Context, string, string) ([]Event, error)
	Balance(context.Context, string, string) (Balance, error)
	RebuildProjections(context.Context) error
	Reverse(context.Context, ReverseInput) (Event, error)
}

type readProjector interface {
	Reader
	Projector
	Reverse(context.Context, Reversal) (Event, error)
}

type service struct {
	repository readProjector
	clock      schedule.Clock
}

func NewService(repository readProjector, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repository: repository, clock: clock}
}

func (s *service) Events(ctx context.Context, holderID, tokenTypeID string) ([]Event, error) {
	return s.repository.Read(ctx, holderID, tokenTypeID)
}

func (s *service) Balance(ctx context.Context, holderID, tokenTypeID string) (Balance, error) {
	return s.repository.BalanceAt(ctx, holderID, tokenTypeID)
}

func (s *service) RebuildProjections(ctx context.Context) error {
	return s.repository.Rebuild(ctx)
}

func (s *service) Reverse(ctx context.Context, input ReverseInput) (Event, error) {
	input.OriginalEventID = strings.TrimSpace(input.OriginalEventID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.ActorIdentity = strings.TrimSpace(input.ActorIdentity)
	switch {
	case input.OriginalEventID == "":
		return Event{}, &InvalidReversalError{Reason: "original event id is required"}
	case input.Reason == "":
		return Event{}, &InvalidReversalError{Reason: "reason is required"}
	case input.IdempotencyKey == "":
		return Event{}, &InvalidReversalError{Reason: "idempotency key is required"}
	}
	value, err := s.repository.Reverse(ctx, Reversal{
		ID: uuid.NewString(), OriginalEventID: input.OriginalEventID, Reason: input.Reason,
		IdempotencyKey: input.IdempotencyKey, ActorIdentity: input.ActorIdentity,
		CreatedAt: s.clock.Now().UTC(),
	})
	if errors.Is(err, ErrEventNotFound) {
		return Event{}, ErrEventNotFound
	}
	return value, err
}
