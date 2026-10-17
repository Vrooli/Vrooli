package mints

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SupplyPolicy string

const (
	SupplyPolicyUnbounded SupplyPolicy = "unbounded"
	SupplyPolicyCapped    SupplyPolicy = "capped"
	SupplyPolicyFixed     SupplyPolicy = "fixed"
)

type MinterAuthority struct {
	TokenTypeID string
	Subject     string
}

type TokenType struct {
	ID           string
	Name         string
	Symbol       string
	Color        string
	SupplyPolicy SupplyPolicy
	CapAmount    int64
	MintedAmount int64
	Authority    MinterAuthority
	Retired      bool
	CreatedAt    time.Time
	RetiredAt    *time.Time
}

type CreateInput struct {
	Name          string
	Symbol        string
	Color         string
	SupplyPolicy  SupplyPolicy
	CapAmount     int64
	MinterSubject string
}

var (
	ErrTokenTypeNotFound = errors.New("token type not found")
	ErrTokenTypeRetired  = errors.New("token type is retired")
)

type InvalidTokenTypeError struct{ Reason string }

func (e *InvalidTokenTypeError) Error() string { return "invalid token type: " + e.Reason }

type SupplyCapExceededError struct {
	TokenTypeID     string
	Cap             int64
	AttemptedAmount int64
	CurrentAmount   int64
}

func (e *SupplyCapExceededError) Error() string {
	return fmt.Sprintf("token type %q supply cap %d exceeded by attempted mint amount %d (current supply %d)", e.TokenTypeID, e.Cap, e.AttemptedAmount, e.CurrentAmount)
}

type Service interface {
	Create(context.Context, CreateInput) (TokenType, error)
	Get(context.Context, string) (TokenType, error)
	List(context.Context, bool) ([]TokenType, error)
	Retire(context.Context, string) (TokenType, error)
	Mint(context.Context, string, int64) (TokenType, error)
}

type service struct {
	repo  Repository
	clock schedule.Clock
}

func NewService(repo Repository, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repo: repo, clock: clock}
}

func (s *service) Create(ctx context.Context, in CreateInput) (TokenType, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Symbol = strings.TrimSpace(in.Symbol)
	in.Color = strings.TrimSpace(in.Color)
	in.MinterSubject = strings.TrimSpace(in.MinterSubject)
	if err := validateCreate(in); err != nil {
		return TokenType{}, err
	}
	now := s.clock.Now().UTC()
	tokenType := TokenType{
		ID:           uuid.NewString(),
		Name:         in.Name,
		Symbol:       in.Symbol,
		Color:        in.Color,
		SupplyPolicy: in.SupplyPolicy,
		CapAmount:    in.CapAmount,
		Authority: MinterAuthority{
			Subject: in.MinterSubject,
		},
		CreatedAt: now,
	}
	tokenType.Authority.TokenTypeID = tokenType.ID
	return s.repo.Create(ctx, tokenType)
}

func validateCreate(in CreateInput) error {
	switch {
	case in.Name == "":
		return &InvalidTokenTypeError{Reason: "name is required"}
	case in.Symbol == "":
		return &InvalidTokenTypeError{Reason: "symbol is required"}
	case in.Color == "":
		return &InvalidTokenTypeError{Reason: "color is required"}
	case in.MinterSubject == "":
		return &InvalidTokenTypeError{Reason: "exactly one named minter authority is required"}
	}
	switch in.SupplyPolicy {
	case SupplyPolicyUnbounded:
		if in.CapAmount != 0 {
			return &InvalidTokenTypeError{Reason: "unbounded supply cannot declare a cap"}
		}
	case SupplyPolicyCapped, SupplyPolicyFixed:
		if in.CapAmount <= 0 {
			return &InvalidTokenTypeError{Reason: "capped and fixed supply require a positive cap"}
		}
	default:
		return &InvalidTokenTypeError{Reason: "supply policy must be unbounded, capped, or fixed"}
	}
	return nil
}

func (s *service) Get(ctx context.Context, id string) (TokenType, error) {
	if strings.TrimSpace(id) == "" {
		return TokenType{}, &InvalidTokenTypeError{Reason: "token type id is required"}
	}
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, includeRetired bool) ([]TokenType, error) {
	return s.repo.List(ctx, includeRetired)
}

func (s *service) Retire(ctx context.Context, id string) (TokenType, error) {
	if strings.TrimSpace(id) == "" {
		return TokenType{}, &InvalidTokenTypeError{Reason: "token type id is required"}
	}
	return s.repo.Retire(ctx, id, s.clock.Now().UTC())
}

func (s *service) Mint(ctx context.Context, id string, amount int64) (TokenType, error) {
	if strings.TrimSpace(id) == "" {
		return TokenType{}, &InvalidTokenTypeError{Reason: "token type id is required"}
	}
	if amount <= 0 {
		return TokenType{}, &InvalidTokenTypeError{Reason: "mint amount must be positive"}
	}
	tokenType, err := s.repo.Get(ctx, id)
	if err != nil {
		return TokenType{}, err
	}
	if tokenType.Retired {
		return TokenType{}, ErrTokenTypeRetired
	}
	if amount > math.MaxInt64-tokenType.MintedAmount {
		return TokenType{}, &InvalidTokenTypeError{Reason: "mint amount overflows supply"}
	}
	if attempted := tokenType.MintedAmount + amount; tokenType.SupplyPolicy != SupplyPolicyUnbounded && attempted > tokenType.CapAmount {
		return TokenType{}, &SupplyCapExceededError{
			TokenTypeID: id, Cap: tokenType.CapAmount, AttemptedAmount: amount, CurrentAmount: tokenType.MintedAmount,
		}
	}
	return s.repo.Mint(ctx, id, amount)
}
