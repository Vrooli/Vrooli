package grants

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type GrantStatus string

const (
	GrantStatusDraft     GrantStatus = "draft"
	GrantStatusLive      GrantStatus = "live"
	GrantStatusExhausted GrantStatus = "exhausted"
	GrantStatusExpired   GrantStatus = "expired"
	GrantStatusRevoked   GrantStatus = "revoked"
)

type Grant struct {
	ID                   string
	TokenTypeID          string
	GrantSourceID        string
	Authorizer           string
	HolderID             string
	AmountMinor          int64
	AllowedCatalogScopes []string
	DeniedCatalogScopes  []string
	ExpiresAt            time.Time
	IssuedAt             time.Time
	Status               GrantStatus
	IdempotencyKey       string
	RequiredEvidence     []string
	RecurrenceSeconds    int64
	NextIssueAt          *time.Time
	CancelledAt          *time.Time
	Rules                []GrantRule
}

type CreateInput struct {
	TokenTypeID          string
	GrantSourceID        string
	Authorizer           string
	HolderID             string
	AmountMinor          int64
	AllowedCatalogScopes []string
	DeniedCatalogScopes  []string
	ExpiresAt            time.Time
	IdempotencyKey       string
	RequiredEvidence     []string
	Rules                []GrantRule
	ActorIdentity        string
}

var (
	ErrGrantNotFound     = errors.New("grant not found")
	ErrGrantRefused      = errors.New("grant rule refused redemption")
	ErrTokenTypeNotFound = errors.New("token type not found")
	ErrTokenTypeRetired  = errors.New("token type is retired")
)

type InvalidGrantError struct{ Reason string }

func (e *InvalidGrantError) Error() string { return "invalid grant: " + e.Reason }

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
	Create(context.Context, CreateInput) (Grant, error)
	Get(context.Context, string) (Grant, error)
	List(context.Context, string, string, bool) ([]Grant, error)
	Revoke(context.Context, RevokeInput) (Grant, error)
	EvaluateRedemption(context.Context, string, EvaluationRequest) (Decision, error)
}

type RevokeInput struct {
	ID             string
	Reason         string
	IdempotencyKey string
}

type service struct {
	repository Repository
	tokenTypes TokenTypeReader
	evaluator  Evaluator
	clock      schedule.Clock
}

func NewService(repository Repository, tokenTypes TokenTypeReader, evaluator Evaluator, clock schedule.Clock) Service {
	if evaluator == nil {
		evaluator = NewRuleEvaluator()
	}
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repository: repository, tokenTypes: tokenTypes, evaluator: evaluator, clock: clock}
}

func (s *service) Create(ctx context.Context, input CreateInput) (Grant, error) {
	normalizeInput(&input)
	now := s.clock.Now().UTC()
	if err := validateCreate(input, now); err != nil {
		return Grant{}, err
	}
	tokenType, err := s.tokenTypes.GetTokenType(ctx, input.TokenTypeID)
	if err != nil {
		return Grant{}, err
	}
	if tokenType.Retired {
		return Grant{}, ErrTokenTypeRetired
	}
	grant := Grant{
		ID: uuid.NewString(), TokenTypeID: input.TokenTypeID, GrantSourceID: input.GrantSourceID,
		Authorizer: input.Authorizer, HolderID: input.HolderID, AmountMinor: input.AmountMinor,
		AllowedCatalogScopes: input.AllowedCatalogScopes, DeniedCatalogScopes: input.DeniedCatalogScopes,
		ExpiresAt: input.ExpiresAt.UTC(), IssuedAt: now, Status: GrantStatusLive,
		IdempotencyKey: input.IdempotencyKey, RequiredEvidence: input.RequiredEvidence,
	}
	grant.Rules = buildRules(grant, input.Rules)
	credit := Credit{
		ID: uuid.NewString(), TokenTypeID: grant.TokenTypeID, HolderID: grant.HolderID,
		Amount:         grant.AmountMinor,
		CauseReference: "grant:" + grant.ID, ActorIdentity: input.ActorIdentity,
		CreatedAt: grant.IssuedAt,
	}
	return s.repository.Create(ctx, grant, credit)
}

func normalizeInput(input *CreateInput) {
	input.TokenTypeID = strings.TrimSpace(input.TokenTypeID)
	input.GrantSourceID = strings.TrimSpace(input.GrantSourceID)
	input.Authorizer = strings.TrimSpace(input.Authorizer)
	input.HolderID = strings.TrimSpace(input.HolderID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.AllowedCatalogScopes = normalizeStrings(input.AllowedCatalogScopes)
	input.DeniedCatalogScopes = normalizeStrings(input.DeniedCatalogScopes)
	input.RequiredEvidence = normalizeStrings(input.RequiredEvidence)
	input.ActorIdentity = strings.TrimSpace(input.ActorIdentity)
	if input.ActorIdentity == "" {
		input.ActorIdentity = input.Authorizer
	}
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateCreate(input CreateInput, now time.Time) error {
	switch {
	case input.TokenTypeID == "":
		return &InvalidGrantError{Reason: "token type id is required"}
	case input.GrantSourceID == "":
		return &InvalidGrantError{Reason: "grant source id is required"}
	case input.Authorizer == "":
		return &InvalidGrantError{Reason: "authorizer is required"}
	case input.HolderID == "":
		return &InvalidGrantError{Reason: "holder id is required"}
	case input.AmountMinor <= 0:
		return &InvalidGrantError{Reason: "amount_minor must be positive"}
	case input.ExpiresAt.IsZero():
		return &InvalidGrantError{Reason: "expires_at is required"}
	case !input.ExpiresAt.After(now):
		return &InvalidGrantError{Reason: "expires_at must be in the future"}
	case input.IdempotencyKey == "":
		return &InvalidGrantError{Reason: "idempotency key is required"}
	}
	for _, rule := range input.Rules {
		if strings.HasPrefix(strings.TrimSpace(rule.ID), "system:") {
			return &InvalidGrantError{Reason: "rule ids beginning with system: are reserved"}
		}
		if rule.AmountLimit < 0 {
			return &InvalidGrantError{Reason: "rule amount limit cannot be negative"}
		}
		switch rule.Condition {
		case RuleConditionCatalogScopeAllowed, RuleConditionCatalogScopeDenied,
			RuleConditionBeforeExpiry, RuleConditionRequiredEvidence, RuleConditionSufficientBalance:
		default:
			return &InvalidGrantError{Reason: fmt.Sprintf("unknown closed rule condition %q", rule.Condition)}
		}
	}
	return nil
}

func buildRules(grant Grant, declared []GrantRule) []GrantRule {
	rules := make([]GrantRule, 0, len(declared)+5)
	for _, rule := range declared {
		rule.ID = strings.TrimSpace(rule.ID)
		if rule.ID == "" {
			rule.ID = uuid.NewString()
		}
		rule.Operands = normalizeStrings(rule.Operands)
		rules = append(rules, rule)
	}
	if len(grant.AllowedCatalogScopes) > 0 {
		rules = append(rules, GrantRule{ID: "system:allowed_catalog_scopes", Condition: RuleConditionCatalogScopeAllowed, Operands: grant.AllowedCatalogScopes})
	}
	if len(grant.DeniedCatalogScopes) > 0 {
		rules = append(rules, GrantRule{ID: "system:denied_catalog_scopes", Condition: RuleConditionCatalogScopeDenied, Operands: grant.DeniedCatalogScopes})
	}
	rules = append(rules, GrantRule{ID: "system:expires_at", Condition: RuleConditionBeforeExpiry})
	if len(grant.RequiredEvidence) > 0 {
		rules = append(rules, GrantRule{ID: "system:required_evidence", Condition: RuleConditionRequiredEvidence, Operands: grant.RequiredEvidence})
	}
	rules = append(rules, GrantRule{ID: "system:available_balance", Condition: RuleConditionSufficientBalance, AmountLimit: grant.AmountMinor})
	return rules
}

func (s *service) Get(ctx context.Context, id string) (Grant, error) {
	if strings.TrimSpace(id) == "" {
		return Grant{}, &InvalidGrantError{Reason: "grant id is required"}
	}
	return s.repository.Get(ctx, id)
}

func (s *service) List(ctx context.Context, holderID, tokenTypeID string, includeInactive bool) ([]Grant, error) {
	return s.repository.List(ctx, strings.TrimSpace(holderID), strings.TrimSpace(tokenTypeID), includeInactive)
}

func (s *service) Revoke(ctx context.Context, input RevokeInput) (Grant, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ID == "" || input.Reason == "" || input.IdempotencyKey == "" {
		return Grant{}, &InvalidGrantError{Reason: "grant id, reason, and idempotency key are required"}
	}
	return s.repository.Revoke(ctx, input.ID, input.Reason, input.IdempotencyKey, s.clock.Now().UTC())
}

func (s *service) EvaluateRedemption(ctx context.Context, id string, request EvaluationRequest) (Decision, error) {
	grant, err := s.Get(ctx, id)
	if err != nil {
		return Decision{}, err
	}
	if request.Now.IsZero() {
		request.Now = s.clock.Now().UTC()
	}
	decision, err := s.evaluator.Evaluate(ctx, grant, request)
	if err != nil {
		return Decision{}, fmt.Errorf("evaluate grant %q: %w", id, err)
	}
	return decision, nil
}
