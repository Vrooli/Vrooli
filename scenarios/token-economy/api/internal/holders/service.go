package holders

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type HistoryEvent struct {
	ID                      string
	TokenTypeID             string
	Amount                  int64
	Kind                    string
	Reason                  string
	CauseReference          string
	ActorIdentity           string
	ActorKind               string
	ActorVerificationStatus string
	ActorRunID              string
	CreatedAt               time.Time
}

type Balance struct {
	TokenTypeID string
	Amount      int64
}

type History struct {
	Events   []HistoryEvent
	Balances []Balance
}

type HistoryReader interface {
	Read(context.Context, string, string) (History, error)
}

type View struct {
	Holder  Holder
	History History
}

type Service interface {
	Add(context.Context, AddInput) (Holder, error)
	Get(context.Context, string) (Holder, error)
	List(context.Context) ([]Holder, error)
	View(context.Context, string) (View, error)
}

type AddInput struct {
	DisplayName          string
	AuthenticatorSubject string
	IdempotencyKey       string
}

type service struct {
	holders Repository
	history HistoryReader
	clock   schedule.Clock
}

func NewService(holders Repository, history HistoryReader) Service {
	return NewServiceWithClock(holders, history, schedule.System())
}

func NewServiceWithClock(holders Repository, history HistoryReader, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{holders: holders, history: history, clock: clock}
}

func (s *service) Add(ctx context.Context, input AddInput) (Holder, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.AuthenticatorSubject = strings.TrimSpace(input.AuthenticatorSubject)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.DisplayName == "" || input.AuthenticatorSubject == "" || input.IdempotencyKey == "" {
		return Holder{}, fmt.Errorf("%w: display name, authenticator subject, and idempotency key are required", ErrInvalidHolder)
	}
	holder := Holder{ID: uuid.NewString(), DisplayName: input.DisplayName, AuthenticatorSubject: input.AuthenticatorSubject, CreatedAt: s.clock.Now().UTC()}
	return s.holders.CreateIdempotent(ctx, holder, input.IdempotencyKey)
}

func (s *service) Get(ctx context.Context, id string) (Holder, error) {
	return s.holders.Get(ctx, id)
}

func (s *service) List(ctx context.Context) ([]Holder, error) {
	return s.holders.List(ctx)
}

func (s *service) View(ctx context.Context, authenticatedSubject string) (View, error) {
	authenticatedSubject = strings.TrimSpace(authenticatedSubject)
	if authenticatedSubject == "" {
		return View{}, fmt.Errorf("%w: authenticated subject is required", ErrInvalidHolder)
	}
	holder, err := s.holders.GetBySubject(ctx, authenticatedSubject)
	if err != nil {
		return View{}, err
	}
	if s.history == nil {
		return View{}, errorsUnavailable("holder history")
	}
	history, err := s.history.Read(ctx, holder.ID, authenticatedSubject)
	if err != nil {
		return View{}, err
	}
	return View{Holder: holder, History: history}, nil
}

func errorsUnavailable(name string) error { return fmt.Errorf("%s unavailable", name) }
