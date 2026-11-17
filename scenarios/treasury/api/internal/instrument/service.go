// Package instrument owns credential references and mandate-derived scope.
package instrument

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"treasury/internal/mandate"
	"treasury/internal/rail/card"

	"github.com/vrooli/api-core/schedule"
)

var (
	ErrInvalid                 = errors.New("invalid instrument")
	ErrNotFound                = errors.New("instrument not found")
	ErrCardIssueInProgress     = errors.New("card issuance is already in progress")
	ErrCardIssueReconciliation = errors.New("card issuance is outside provider idempotency retention; operator reconciliation is required")
)

const (
	cardIssueIdempotencyRetention = 30 * 24 * time.Hour
	cardIssueInFlightLease        = 5 * time.Minute
)

type Instrument struct {
	ID, BookID, MandateID, Rail, CredentialReference string
	CapMinor                                         int64
	Currency, Counterparty                           string
	ExpiresAt, CreatedAt                             time.Time
}

type RegisterInput struct {
	ID, MandateID, Rail, CredentialReference, Counterparty string
}

type Mandates interface {
	RequireLive(context.Context, string) (mandate.Mandate, error)
}

type Rails interface{ Has(string) bool }

type CredentialResolver interface {
	Resolve(context.Context, string, string) (string, error)
}

type CredentialWriter interface {
	Store(context.Context, string, string, string) error
}

type CardIssuers interface {
	Get(string) (card.Issuer, error)
	Has(string) bool
}

type CardIssueAttempt struct {
	Fingerprint string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CardIssueJournal stores only a non-reversible request fingerprint and
// lifecycle timestamps. Provider credentials and issued card material stay in
// Credential Authority.
type CardIssueJournal interface {
	GetCardIssueAttempt(context.Context, string) (CardIssueAttempt, error)
	BeginCardIssue(context.Context, string, string, time.Time) (CardIssueAttempt, error)
	SetCardIssueAttemptStatus(context.Context, string, string, string, time.Time) error
}

type ScopedCredential struct {
	Instrument Instrument
	Value      string
}

type Service struct {
	repository  Repository
	mandates    Mandates
	rails       Rails
	resolver    CredentialResolver
	cardIssuers CardIssuers
	clock       schedule.Clock
}

// StoreSettlementRecoveryInput uses the same Credential Authority resolver
// that owns provider credentials. No signed authorization enters app storage.
func (s *Service) StoreSettlementRecoveryInput(ctx context.Context, id, payload string) (string, error) {
	writer, ok := s.resolver.(CredentialWriter)
	if !ok {
		return "", fmt.Errorf("credential resolver cannot store recovery inputs")
	}
	identity := settlementRecoveryIdentity(id)
	if err := writer.Store(ctx, identity, "signed-payment", payload); err != nil {
		return "", err
	}
	return identity, nil
}

func (s *Service) ResolveSettlementRecoveryInput(ctx context.Context, reference string) (string, error) {
	return s.resolver.Resolve(ctx, reference, "signed-payment")
}

func (s *Service) DeleteSettlementRecoveryInput(ctx context.Context, reference string) error {
	deleter, ok := s.resolver.(interface {
		Delete(context.Context, string, string) error
	})
	if !ok {
		return fmt.Errorf("credential resolver cannot delete recovery inputs")
	}
	return deleter.Delete(ctx, reference, "signed-payment")
}

// DeleteUnreferencedSettlementRecoveryInput removes the deterministic CA
// identity for a call that crashed after credential provisioning but before
// SQLite committed its reference. The x402 adapter has not dispatched payment
// at this cut because reference persistence is a required precondition.
func (s *Service) DeleteUnreferencedSettlementRecoveryInput(ctx context.Context, settlementID string) error {
	deleter, ok := s.resolver.(interface {
		Delete(context.Context, string, string) error
	})
	if !ok {
		return fmt.Errorf("credential resolver cannot delete recovery inputs")
	}
	return deleter.Delete(ctx, settlementRecoveryIdentity(settlementID), "signed-payment")
}

func NewService(repository Repository, mandates Mandates, rails Rails, resolver CredentialResolver, clock schedule.Clock) *Service {
	return &Service{repository: repository, mandates: mandates, rails: rails, resolver: resolver, clock: clock}
}

func NewServiceWithCardIssuers(repository Repository, mandates Mandates, rails Rails, resolver CredentialResolver, clock schedule.Clock, issuers CardIssuers) *Service {
	return &Service{repository: repository, mandates: mandates, rails: rails, resolver: resolver, cardIssuers: issuers, clock: clock}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Instrument, error) {
	if s.repository == nil || s.mandates == nil || s.rails == nil || s.clock == nil {
		return Instrument{}, fmt.Errorf("%w: repository, mandates, rails, and clock are required", ErrInvalid)
	}
	in.ID = strings.TrimSpace(in.ID)
	in.MandateID = strings.TrimSpace(in.MandateID)
	in.Rail = strings.ToLower(strings.TrimSpace(in.Rail))
	in.CredentialReference = strings.TrimSpace(in.CredentialReference)
	in.Counterparty = strings.ToLower(strings.TrimSpace(in.Counterparty))
	if in.Rail == "manual" && in.CredentialReference == "" {
		in.CredentialReference = "manual/operator-attestation"
	}
	if in.ID == "" || in.MandateID == "" || in.Rail == "" || in.Counterparty == "" || in.Rail != "manual" && in.CredentialReference == "" {
		return Instrument{}, fmt.Errorf("%w: id, mandate_id, rail, counterparty, and an automated-rail credential_reference are required", ErrInvalid)
	}
	if in.CredentialReference != "" && (strings.ContainsAny(in.CredentialReference, "=\r\n\t ") || !strings.Contains(in.CredentialReference, "/")) {
		return Instrument{}, fmt.Errorf("%w: credential_reference must be a namespaced logical identity", ErrInvalid)
	}
	isCard := s.cardIssuers != nil && s.cardIssuers.Has(in.Rail)
	if !s.rails.Has(in.Rail) && !isCard {
		return Instrument{}, fmt.Errorf("%w: rail is not registered", ErrInvalid)
	}
	grant, err := s.mandates.RequireLive(ctx, in.MandateID)
	if err != nil {
		return Instrument{}, fmt.Errorf("load live mandate: %w", err)
	}
	if !grant.AllowsCounterparty(in.Counterparty) {
		return Instrument{}, fmt.Errorf("%w: counterparty is outside mandate scope", ErrInvalid)
	}
	scope := card.Scope{MandateReference: grant.ID, AmountMinor: grant.CapMinor, Currency: grant.Currency, Counterparty: in.Counterparty, ExpiresAt: grant.ExpiresAt}
	if existing, getErr := s.repository.Get(ctx, in.ID); getErr == nil {
		if existing.MandateID != grant.ID || existing.BookID != grant.BookID || existing.Rail != in.Rail || existing.CapMinor != grant.CapMinor || existing.Currency != grant.Currency || existing.Counterparty != in.Counterparty || !existing.ExpiresAt.Equal(grant.ExpiresAt) {
			return Instrument{}, fmt.Errorf("%w: instrument id is already bound to a different mandate scope", ErrInvalid)
		}
		if isCard {
			journal, ok := s.repository.(CardIssueJournal)
			if ok {
				if attempt, attemptErr := journal.GetCardIssueAttempt(ctx, in.ID); attemptErr == nil {
					if attempt.Fingerprint != cardIssueFingerprint(in, scope) {
						return Instrument{}, fmt.Errorf("%w: instrument id is already bound to a different card provider identity", ErrInvalid)
					}
					if attempt.Status != "complete" {
						if err := journal.SetCardIssueAttemptStatus(context.WithoutCancel(ctx), in.ID, attempt.Fingerprint, "complete", s.clock.Now().UTC()); err != nil {
							return Instrument{}, fmt.Errorf("complete recovered card issuance record: %w", err)
						}
					}
				} else if !errors.Is(attemptErr, ErrNotFound) {
					return Instrument{}, attemptErr
				}
			}
		}
		return existing, nil
	} else if !errors.Is(getErr, ErrNotFound) {
		return Instrument{}, getErr
	}
	if isCard {
		journal, ok := s.repository.(CardIssueJournal)
		if !ok {
			return Instrument{}, fmt.Errorf("%w: card issuance requires a durable issuance journal", ErrInvalid)
		}
		if s.resolver == nil {
			return Instrument{}, fmt.Errorf("%w: card issuance requires credential resolution", ErrInvalid)
		}
		writer, ok := s.resolver.(CredentialWriter)
		if !ok {
			return Instrument{}, fmt.Errorf("%w: card issuance requires a writable credential authority", ErrInvalid)
		}
		issuer, issuerErr := s.cardIssuers.Get(in.Rail)
		if issuerErr != nil {
			return Instrument{}, fmt.Errorf("load card issuer: %w", issuerErr)
		}
		fingerprint := cardIssueFingerprint(in, scope)
		attempt, beginErr := journal.BeginCardIssue(ctx, in.ID, fingerprint, s.clock.Now().UTC())
		if beginErr != nil {
			return Instrument{}, beginErr
		}
		if attempt.Status == "complete" {
			return Instrument{}, fmt.Errorf("%w: completed card issuance has no instrument row; operator reconciliation is required", ErrInvalid)
		}
		markUnknown := func(cause error) error {
			if err := journal.SetCardIssueAttemptStatus(context.WithoutCancel(ctx), in.ID, fingerprint, "unknown", s.clock.Now().UTC()); err != nil {
				return fmt.Errorf("%v; retain card issuance uncertainty: %w", cause, err)
			}
			return cause
		}
		providerCredential, resolveErr := s.resolver.Resolve(ctx, in.CredentialReference, "value")
		if resolveErr != nil {
			return Instrument{}, fmt.Errorf("resolve card provider credential: %w", markUnknown(resolveErr))
		}
		issued, issueErr := issuer.Issue(ctx, card.IssueCommand{InstrumentID: in.ID, IdempotencyKey: in.ID, Credential: providerCredential, Scope: scope})
		if issueErr != nil {
			return Instrument{}, fmt.Errorf("issue mandate-scoped card: %w", markUnknown(issueErr))
		}
		issuedReference := cardCredentialReference(in.ID)
		if storeErr := writer.Store(ctx, issuedReference, "value", issued.Credential); storeErr != nil {
			return Instrument{}, fmt.Errorf("store issued card credential: %w", markUnknown(storeErr))
		}
		in.CredentialReference = issuedReference
		value := Instrument{ID: in.ID, BookID: grant.BookID, MandateID: grant.ID, Rail: in.Rail, CredentialReference: in.CredentialReference, CapMinor: grant.CapMinor, Currency: grant.Currency, Counterparty: in.Counterparty, ExpiresAt: grant.ExpiresAt, CreatedAt: s.clock.Now().UTC()}
		created, createErr := s.repository.Create(ctx, value)
		if createErr != nil {
			return Instrument{}, fmt.Errorf("create instrument after card issuance: %w", markUnknown(createErr))
		}
		if err := journal.SetCardIssueAttemptStatus(context.WithoutCancel(ctx), in.ID, fingerprint, "complete", s.clock.Now().UTC()); err != nil {
			return created, fmt.Errorf("instrument created but card issuance journal completion failed: %w", err)
		}
		return created, nil
	}
	value := Instrument{ID: in.ID, BookID: grant.BookID, MandateID: grant.ID, Rail: in.Rail, CredentialReference: in.CredentialReference, CapMinor: grant.CapMinor, Currency: grant.Currency, Counterparty: in.Counterparty, ExpiresAt: grant.ExpiresAt, CreatedAt: s.clock.Now().UTC()}
	return s.repository.Create(ctx, value)
}

func cardIssueFingerprint(input RegisterInput, scope card.Scope) string {
	encoded, _ := json.Marshal(struct {
		InstrumentID, MandateID, Rail, ProviderCredentialReference string
		Scope                                                      card.Scope
	}{input.ID, input.MandateID, input.Rail, input.CredentialReference, scope})
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func cardCredentialReference(instrumentID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(instrumentID)))
	return fmt.Sprintf("vrooli/treasury/instruments/%x", sum[:16])
}

// ResolveForUse revalidates the live mandate before resolving credential
// material. The value exists only in the returned in-memory scope and is never
// passed to persistence or transport DTOs.
func (s *Service) ResolveForUse(ctx context.Context, id string) (ScopedCredential, error) {
	if s.repository == nil || s.mandates == nil {
		return ScopedCredential{}, fmt.Errorf("%w: repository and mandates are required", ErrInvalid)
	}
	value, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return ScopedCredential{}, err
	}
	grant, err := s.mandates.RequireLive(ctx, value.MandateID)
	if err != nil {
		return ScopedCredential{}, fmt.Errorf("revalidate instrument mandate: %w", err)
	}
	if value.BookID != grant.BookID || value.CapMinor != grant.CapMinor || value.Currency != grant.Currency || !value.ExpiresAt.Equal(grant.ExpiresAt) || !grant.AllowsCounterparty(value.Counterparty) {
		return ScopedCredential{}, fmt.Errorf("%w: persisted scope no longer matches its mandate", ErrInvalid)
	}
	if value.Rail == "manual" {
		return ScopedCredential{Instrument: value}, nil
	}
	if s.resolver == nil {
		return ScopedCredential{}, fmt.Errorf("%w: automated rail credential resolver is required", ErrInvalid)
	}
	secret, err := s.resolver.Resolve(ctx, value.CredentialReference, "value")
	if err != nil {
		return ScopedCredential{}, fmt.Errorf("resolve instrument credential: %w", err)
	}
	if secret == "" {
		return ScopedCredential{}, fmt.Errorf("%w: credential is not configured", ErrInvalid)
	}
	return ScopedCredential{Instrument: value, Value: secret}, nil
}
