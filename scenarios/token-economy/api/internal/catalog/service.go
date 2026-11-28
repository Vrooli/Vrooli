package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type ApprovalPosture string

const (
	ApprovalPostureImmediate        ApprovalPosture = "immediate"
	ApprovalPostureRequiresApproval ApprovalPosture = "requires_approval"
)

type Availability struct {
	AvailableFrom     *time.Time
	AvailableUntil    *time.Time
	RemainingQuantity *int64
}

type Entry struct {
	ID              string
	TokenTypeID     string
	Title           string
	Description     string
	CostAmount      int64
	Availability    Availability
	ApprovalPosture ApprovalPosture
	Retired         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	RetiredAt       *time.Time
}

type Input struct {
	ID              string
	TokenTypeID     string
	Title           string
	Description     string
	CostAmount      int64
	Availability    Availability
	ApprovalPosture ApprovalPosture
}

var (
	ErrEntryNotFound     = errors.New("catalog entry not found")
	ErrEntryUnavailable  = errors.New("catalog entry unavailable")
	ErrTokenTypeNotFound = errors.New("token type not found")
	ErrTokenTypeRetired  = errors.New("token type is retired")
)

type InvalidCatalogError struct{ Reason string }

func (e *InvalidCatalogError) Error() string { return "invalid catalog entry: " + e.Reason }

type UnavailableCatalogError struct{ Reason string }

func (e *UnavailableCatalogError) Error() string {
	return fmt.Sprintf("%s: %s", ErrEntryUnavailable, e.Reason)
}

func (e *UnavailableCatalogError) Unwrap() error { return ErrEntryUnavailable }

type TokenTypeState struct {
	ID      string
	Retired bool
}

type TokenTypeReader interface {
	GetTokenType(context.Context, string) (TokenTypeState, error)
}

type TokenTypeReaderFunc func(context.Context, string) (TokenTypeState, error)

func (f TokenTypeReaderFunc) GetTokenType(ctx context.Context, id string) (TokenTypeState, error) {
	return f(ctx, id)
}

type Service interface {
	Create(context.Context, Input, string) (Entry, error)
	Update(context.Context, Input, string) (Entry, error)
	Get(context.Context, string) (Entry, error)
	List(context.Context, bool) ([]Entry, error)
	ListAvailable(context.Context) ([]Entry, error)
	RequireAvailable(context.Context, string) (Entry, error)
	Retire(context.Context, string, string) (Entry, error)
}

type service struct {
	repository Repository
	tokenTypes TokenTypeReader
	clock      schedule.Clock
}

func NewService(repository Repository, tokenTypes TokenTypeReader, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repository: repository, tokenTypes: tokenTypes, clock: clock}
}

func (s *service) Create(ctx context.Context, input Input, idempotencyKey string) (Entry, error) {
	normalizeInput(&input)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if err := validateInput(input, idempotencyKey); err != nil {
		return Entry{}, err
	}
	if err := s.validateTokenType(ctx, input.TokenTypeID); err != nil {
		return Entry{}, err
	}
	now := s.clock.Now().UTC()
	return s.repository.Create(ctx, entryFromInput(input, now), idempotencyKey)
}

func (s *service) Update(ctx context.Context, input Input, idempotencyKey string) (Entry, error) {
	normalizeInput(&input)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if input.ID == "" {
		return Entry{}, &InvalidCatalogError{Reason: "id is required"}
	}
	if err := validateInput(input, idempotencyKey); err != nil {
		return Entry{}, err
	}
	if err := s.validateTokenType(ctx, input.TokenTypeID); err != nil {
		return Entry{}, err
	}
	existing, err := s.repository.Get(ctx, input.ID)
	if err != nil {
		return Entry{}, err
	}
	if existing.Retired {
		return Entry{}, &UnavailableCatalogError{Reason: "retired entries cannot be updated"}
	}
	updated := entryFromInput(input, existing.CreatedAt)
	updated.UpdatedAt = s.clock.Now().UTC()
	return s.repository.Update(ctx, updated, idempotencyKey)
}

func (s *service) Get(ctx context.Context, id string) (Entry, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Entry{}, &InvalidCatalogError{Reason: "id is required"}
	}
	return s.repository.Get(ctx, id)
}

func (s *service) List(ctx context.Context, includeRetired bool) ([]Entry, error) {
	return s.repository.List(ctx, includeRetired)
}

func (s *service) ListAvailable(ctx context.Context) ([]Entry, error) {
	entries, err := s.repository.List(ctx, false)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	available := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if availabilityError(entry, now) == nil {
			available = append(available, entry)
		}
	}
	return available, nil
}

func (s *service) RequireAvailable(ctx context.Context, id string) (Entry, error) {
	entry, err := s.Get(ctx, id)
	if err != nil {
		return Entry{}, err
	}
	if err := availabilityError(entry, s.clock.Now().UTC()); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func (s *service) Retire(ctx context.Context, id, idempotencyKey string) (Entry, error) {
	id = strings.TrimSpace(id)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if id == "" || idempotencyKey == "" {
		return Entry{}, &InvalidCatalogError{Reason: "id and idempotency key are required"}
	}
	return s.repository.Retire(ctx, id, s.clock.Now().UTC(), idempotencyKey)
}

func (s *service) validateTokenType(ctx context.Context, id string) error {
	if s.tokenTypes == nil {
		return errors.New("catalog token-type reader unavailable")
	}
	state, err := s.tokenTypes.GetTokenType(ctx, id)
	if err != nil {
		return err
	}
	if state.Retired {
		return ErrTokenTypeRetired
	}
	return nil
}

func normalizeInput(input *Input) {
	input.ID = strings.TrimSpace(input.ID)
	input.TokenTypeID = strings.TrimSpace(input.TokenTypeID)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Availability.AvailableFrom != nil {
		value := input.Availability.AvailableFrom.UTC()
		input.Availability.AvailableFrom = &value
	}
	if input.Availability.AvailableUntil != nil {
		value := input.Availability.AvailableUntil.UTC()
		input.Availability.AvailableUntil = &value
	}
}

func validateInput(input Input, idempotencyKey string) error {
	switch {
	case input.TokenTypeID == "":
		return &InvalidCatalogError{Reason: "token type id is required"}
	case input.Title == "":
		return &InvalidCatalogError{Reason: "title is required"}
	case input.CostAmount <= 0:
		return &InvalidCatalogError{Reason: "cost amount must be positive"}
	case idempotencyKey == "":
		return &InvalidCatalogError{Reason: "idempotency key is required"}
	case input.ApprovalPosture != ApprovalPostureImmediate && input.ApprovalPosture != ApprovalPostureRequiresApproval:
		return &InvalidCatalogError{Reason: "approval posture must be immediate or requires_approval"}
	case input.Availability.AvailableFrom != nil && input.Availability.AvailableUntil != nil &&
		!input.Availability.AvailableUntil.After(*input.Availability.AvailableFrom):
		return &InvalidCatalogError{Reason: "available_until must be after available_from"}
	case input.Availability.RemainingQuantity != nil && *input.Availability.RemainingQuantity < 0:
		return &InvalidCatalogError{Reason: "remaining quantity cannot be negative"}
	default:
		return nil
	}
}

func entryFromInput(input Input, createdAt time.Time) Entry {
	return Entry{
		ID: input.ID, TokenTypeID: input.TokenTypeID, Title: input.Title,
		Description: input.Description, CostAmount: input.CostAmount,
		Availability: input.Availability, ApprovalPosture: input.ApprovalPosture,
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func availabilityError(entry Entry, now time.Time) error {
	switch {
	case entry.Retired:
		return &UnavailableCatalogError{Reason: "entry is retired"}
	case entry.Availability.AvailableFrom != nil && now.Before(*entry.Availability.AvailableFrom):
		return &UnavailableCatalogError{Reason: "availability window has not started"}
	case entry.Availability.AvailableUntil != nil && !now.Before(*entry.Availability.AvailableUntil):
		return &UnavailableCatalogError{Reason: "availability window has ended"}
	case entry.Availability.RemainingQuantity != nil && *entry.Availability.RemainingQuantity == 0:
		return &UnavailableCatalogError{Reason: "entry is out of stock"}
	default:
		return nil
	}
}
