// Package mandate owns signed, scoped, capped, and expiring authority grants.
package mandate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"treasury/internal/mandate/flow"
)

var (
	ErrInvalid  = errors.New("invalid mandate")
	ErrInactive = errors.New("mandate is not live")
)

type ValidationError struct {
	Constraint string
}

func (e *ValidationError) Error() string { return "mandate constraint violated: " + e.Constraint }
func (e *ValidationError) Unwrap() error { return ErrInvalid }

type Mandate struct {
	ID                    string
	IdempotencyKey        string
	BookID                string
	BudgetID              string
	Authorizer            string
	CapMinor              int64
	Currency              string
	AllowedCounterparties []string
	DeniedCounterparties  []string
	RequiredEvidence      []string
	ExpiresAt             time.Time
	IssuedAt              time.Time
	Signature             []byte
	Status                flow.MandateStatus
}

type IssueInput struct {
	ID                    string
	IdempotencyKey        string
	BookID                string
	BudgetID              string
	Authorizer            string
	CapMinor              int64
	Currency              string
	AllowedCounterparties []string
	DeniedCounterparties  []string
	RequiredEvidence      []string
	ExpiresAt             time.Time
}

func (m Mandate) AllowsCounterparty(counterparty string) bool {
	counterparty = strings.ToLower(strings.TrimSpace(counterparty))
	if counterparty == "" || slices.Contains(m.DeniedCounterparties, counterparty) {
		return false
	}
	return len(m.AllowedCounterparties) == 0 || slices.Contains(m.AllowedCounterparties, counterparty)
}

type Signer interface {
	Sign(context.Context, []byte) ([]byte, error)
}

type Service interface {
	Issue(context.Context, IssueInput) (Mandate, error)
	Get(context.Context, string) (Mandate, error)
	RequireLive(context.Context, string) (Mandate, error)
}

type service struct {
	repository Repository
	clock      schedule.Clock
	signer     Signer
}

func NewService(repository Repository, clock schedule.Clock, signer Signer) Service {
	return &service{repository: repository, clock: clock, signer: signer}
}

func (s *service) Issue(ctx context.Context, in IssueInput) (Mandate, error) {
	if s.repository == nil || s.clock == nil || s.signer == nil {
		return Mandate{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	if key := strings.TrimSpace(in.IdempotencyKey); key != "" {
		existing, err := s.repository.GetByIdempotencyKey(ctx, key)
		if err == nil {
			return s.stateAt(existing, s.clock.Now()), nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Mandate{}, err
		}
	}
	now := s.clock.Now().UTC()
	value, err := validateIssue(in, now)
	if err != nil {
		return Mandate{}, err
	}
	next, err := flow.TransitionMandate(flow.InitialMandateState(), flow.MandateIssue)
	if err != nil {
		return Mandate{}, err
	}
	value.Status = next.Status
	payload, err := signingPayload(value)
	if err != nil {
		return Mandate{}, fmt.Errorf("build signing payload: %w", err)
	}
	value.Signature, err = s.signer.Sign(ctx, payload)
	if err != nil {
		return Mandate{}, fmt.Errorf("sign mandate: %w", err)
	}
	if len(value.Signature) == 0 {
		return Mandate{}, fmt.Errorf("%w: signer returned an empty signature", ErrInvalid)
	}
	return s.repository.Create(ctx, value)
}

func (s *service) Get(ctx context.Context, id string) (Mandate, error) {
	if s.repository == nil || s.clock == nil {
		return Mandate{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	value, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Mandate{}, err
	}
	return s.stateAt(value, s.clock.Now()), nil
}

// RequireLive is the fail-closed read boundary used by charge consumers. It
// derives expiry from the injected clock, so a grant stops authorizing work at
// its timestamp even when no sweep job or operator action has run.
func (s *service) RequireLive(ctx context.Context, id string) (Mandate, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return Mandate{}, err
	}
	if value.Status != flow.MandateLive {
		return Mandate{}, fmt.Errorf("%w: status=%s", ErrInactive, value.Status)
	}
	return value, nil
}

func (s *service) stateAt(value Mandate, now time.Time) Mandate {
	if value.Status == flow.MandateLive && !now.Before(value.ExpiresAt) {
		next, err := flow.TransitionMandate(flow.MandateState{Status: value.Status}, flow.MandateReachExpiry)
		if err == nil {
			value.Status = next.Status
		}
	}
	return value
}

func validateIssue(in IssueInput, now time.Time) (Mandate, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.BookID = strings.TrimSpace(in.BookID)
	in.BudgetID = strings.TrimSpace(in.BudgetID)
	in.Authorizer = strings.TrimSpace(in.Authorizer)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	scopeAllow := normalize(in.AllowedCounterparties)
	scopeDeny := normalize(in.DeniedCounterparties)
	switch {
	case in.ID == "":
		return Mandate{}, &ValidationError{Constraint: "id is required"}
	case in.IdempotencyKey == "":
		return Mandate{}, &ValidationError{Constraint: "idempotency_key is required"}
	case in.BookID == "":
		return Mandate{}, &ValidationError{Constraint: "book_id is required"}
	case in.BudgetID == "":
		return Mandate{}, &ValidationError{Constraint: "budget_id is required"}
	case in.Authorizer == "":
		return Mandate{}, &ValidationError{Constraint: "authorizer is required"}
	case in.CapMinor <= 0:
		return Mandate{}, &ValidationError{Constraint: "cap_minor must be positive"}
	case in.Currency == "":
		return Mandate{}, &ValidationError{Constraint: "currency is required"}
	case len(scopeAllow) == 0 && len(scopeDeny) == 0:
		return Mandate{}, &ValidationError{Constraint: "counterparty_scope is required"}
	case in.ExpiresAt.IsZero():
		return Mandate{}, &ValidationError{Constraint: "expires_at is required"}
	case !now.Before(in.ExpiresAt):
		return Mandate{}, &ValidationError{Constraint: "expires_at must be in the future"}
	}
	return Mandate{
		ID:                    in.ID,
		IdempotencyKey:        in.IdempotencyKey,
		BookID:                in.BookID,
		BudgetID:              in.BudgetID,
		Authorizer:            in.Authorizer,
		CapMinor:              in.CapMinor,
		Currency:              in.Currency,
		AllowedCounterparties: scopeAllow,
		DeniedCounterparties:  scopeDeny,
		RequiredEvidence:      normalize(in.RequiredEvidence),
		ExpiresAt:             in.ExpiresAt.UTC(),
		IssuedAt:              now.UTC(),
	}, nil
}

func normalize(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func signingPayload(value Mandate) ([]byte, error) {
	type unsignedMandate struct {
		ID                    string    `json:"id"`
		IdempotencyKey        string    `json:"idempotency_key"`
		BookID                string    `json:"book_id"`
		BudgetID              string    `json:"budget_id"`
		Authorizer            string    `json:"authorizer"`
		CapMinor              int64     `json:"cap_minor"`
		Currency              string    `json:"currency"`
		AllowedCounterparties []string  `json:"allowed_counterparties"`
		DeniedCounterparties  []string  `json:"denied_counterparties"`
		RequiredEvidence      []string  `json:"required_evidence"`
		ExpiresAt             time.Time `json:"expires_at"`
		IssuedAt              time.Time `json:"issued_at"`
		Status                string    `json:"status"`
	}
	return json.Marshal(unsignedMandate{
		ID: value.ID, IdempotencyKey: value.IdempotencyKey, BookID: value.BookID,
		BudgetID: value.BudgetID, Authorizer: value.Authorizer, CapMinor: value.CapMinor,
		Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties,
		DeniedCounterparties: value.DeniedCounterparties, RequiredEvidence: value.RequiredEvidence,
		ExpiresAt: value.ExpiresAt, IssuedAt: value.IssuedAt, Status: string(value.Status),
	})
}

var _ Service = (*service)(nil)
