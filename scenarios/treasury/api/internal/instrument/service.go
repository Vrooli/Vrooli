// Package instrument owns credential references and mandate-derived scope.
package instrument

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"treasury/internal/mandate"
	"treasury/internal/rail/card"
)

var (
	ErrInvalid  = errors.New("invalid instrument")
	ErrNotFound = errors.New("instrument not found")
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
	if isCard {
		if s.resolver == nil {
			return Instrument{}, fmt.Errorf("%w: card issuance requires credential resolution", ErrInvalid)
		}
		writer, ok := s.resolver.(CredentialWriter)
		if !ok {
			return Instrument{}, fmt.Errorf("%w: card issuance requires a writable credential authority", ErrInvalid)
		}
		providerCredential, resolveErr := s.resolver.Resolve(ctx, in.CredentialReference, "value")
		if resolveErr != nil {
			return Instrument{}, fmt.Errorf("resolve card provider credential: %w", resolveErr)
		}
		issuer, issuerErr := s.cardIssuers.Get(in.Rail)
		if issuerErr != nil {
			return Instrument{}, fmt.Errorf("load card issuer: %w", issuerErr)
		}
		scope := card.Scope{MandateReference: grant.ID, AmountMinor: grant.CapMinor, Currency: grant.Currency, Counterparty: in.Counterparty, ExpiresAt: grant.ExpiresAt}
		issued, issueErr := issuer.Issue(ctx, card.IssueCommand{InstrumentID: in.ID, IdempotencyKey: in.ID, Credential: providerCredential, Scope: scope})
		if issueErr != nil {
			return Instrument{}, fmt.Errorf("issue mandate-scoped card: %w", issueErr)
		}
		issuedReference := cardCredentialReference(in.ID)
		if storeErr := writer.Store(ctx, issuedReference, "value", issued.Credential); storeErr != nil {
			return Instrument{}, fmt.Errorf("store issued card credential: %w", storeErr)
		}
		in.CredentialReference = issuedReference
	}
	value := Instrument{ID: in.ID, BookID: grant.BookID, MandateID: grant.ID, Rail: in.Rail, CredentialReference: in.CredentialReference, CapMinor: grant.CapMinor, Currency: grant.Currency, Counterparty: in.Counterparty, ExpiresAt: grant.ExpiresAt, CreatedAt: s.clock.Now().UTC()}
	return s.repository.Create(ctx, value)
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
