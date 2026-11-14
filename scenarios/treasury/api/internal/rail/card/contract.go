// Package card defines the provider-neutral contract for issuing a payment
// card whose provider-enforced scope is derived from a Treasury mandate.
package card

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid scoped card request")
	ErrNotFound = errors.New("scoped card issuer not found")
)

// Scope is the complete authority that must be carried onto an issued card.
// It deliberately contains no provider-specific identifier or enum.
type Scope struct {
	MandateReference string
	AmountMinor      int64
	Currency         string
	Counterparty     string
	ExpiresAt        time.Time
}

type IssueCommand struct {
	InstrumentID   string
	IdempotencyKey string
	Credential     string
	Scope          Scope
}

// Issued contains the provider reference and the short-lived credential
// envelope that the caller must immediately place in its credential authority.
// Implementations must never log Credential.
type Issued struct {
	ExternalID string
	Credential string
	Scope      Scope
}

type InspectQuery struct {
	ExternalID string
	Credential string
}

// Issuer is the stable, provider-neutral card issuance seam. A second adapter
// can satisfy it without changing Treasury's instrument domain.
type Issuer interface {
	Name() string
	Issue(context.Context, IssueCommand) (Issued, error)
	Inspect(context.Context, InspectQuery) (Issued, error)
}

func ValidateScope(scope Scope) error {
	scope.MandateReference = strings.TrimSpace(scope.MandateReference)
	scope.Currency = strings.ToUpper(strings.TrimSpace(scope.Currency))
	scope.Counterparty = strings.ToLower(strings.TrimSpace(scope.Counterparty))
	switch {
	case scope.MandateReference == "":
		return fmt.Errorf("%w: mandate_reference is required", ErrInvalid)
	case scope.AmountMinor <= 0:
		return fmt.Errorf("%w: amount_minor must be positive", ErrInvalid)
	case len(scope.Currency) != 3:
		return fmt.Errorf("%w: currency must be a three-letter code", ErrInvalid)
	case scope.Counterparty == "":
		return fmt.Errorf("%w: counterparty is required", ErrInvalid)
	case scope.ExpiresAt.IsZero():
		return fmt.Errorf("%w: expires_at is required", ErrInvalid)
	default:
		return nil
	}
}

func EqualScope(left, right Scope) bool {
	return strings.TrimSpace(left.MandateReference) == strings.TrimSpace(right.MandateReference) &&
		left.AmountMinor == right.AmountMinor &&
		strings.EqualFold(strings.TrimSpace(left.Currency), strings.TrimSpace(right.Currency)) &&
		strings.EqualFold(strings.TrimSpace(left.Counterparty), strings.TrimSpace(right.Counterparty)) &&
		left.ExpiresAt.Equal(right.ExpiresAt)
}

type Registry struct{ issuers map[string]Issuer }

func NewRegistry(issuers ...Issuer) (*Registry, error) {
	registry := &Registry{issuers: make(map[string]Issuer, len(issuers))}
	for _, issuer := range issuers {
		if issuer == nil {
			return nil, fmt.Errorf("%w: nil issuer", ErrInvalid)
		}
		name := strings.ToLower(strings.TrimSpace(issuer.Name()))
		if name == "" {
			return nil, fmt.Errorf("%w: issuer name is required", ErrInvalid)
		}
		if _, exists := registry.issuers[name]; exists {
			return nil, fmt.Errorf("%w: duplicate issuer %q", ErrInvalid, name)
		}
		registry.issuers[name] = guardedIssuer{name: name, inner: issuer}
	}
	return registry, nil
}

func (r *Registry) Get(name string) (Issuer, error) {
	if r == nil {
		return nil, ErrNotFound
	}
	issuer, ok := r.issuers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, ErrNotFound
	}
	return issuer, nil
}

func (r *Registry) Has(name string) bool {
	_, err := r.Get(name)
	return err == nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.issuers))
	for name := range r.issuers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type guardedIssuer struct {
	name  string
	inner Issuer
}

func (i guardedIssuer) Name() string { return i.name }

func (i guardedIssuer) Issue(ctx context.Context, command IssueCommand) (Issued, error) {
	if strings.TrimSpace(command.InstrumentID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.Credential) == "" {
		return Issued{}, fmt.Errorf("%w: instrument_id, idempotency_key, and credential are required", ErrInvalid)
	}
	if err := ValidateScope(command.Scope); err != nil {
		return Issued{}, err
	}
	issued, err := i.inner.Issue(ctx, command)
	if err != nil {
		return Issued{}, err
	}
	if strings.TrimSpace(issued.ExternalID) == "" || strings.TrimSpace(issued.Credential) == "" || !EqualScope(command.Scope, issued.Scope) {
		return Issued{}, fmt.Errorf("%w: issuer returned an incomplete or widened scope", ErrInvalid)
	}
	return issued, nil
}

func (i guardedIssuer) Inspect(ctx context.Context, query InspectQuery) (Issued, error) {
	if strings.TrimSpace(query.ExternalID) == "" || strings.TrimSpace(query.Credential) == "" {
		return Issued{}, fmt.Errorf("%w: external_id and credential are required", ErrInvalid)
	}
	issued, err := i.inner.Inspect(ctx, query)
	if err != nil {
		return Issued{}, err
	}
	if strings.TrimSpace(issued.ExternalID) == "" {
		return Issued{}, fmt.Errorf("%w: issuer inspection omitted external_id", ErrInvalid)
	}
	return issued, nil
}

var _ Issuer = guardedIssuer{}
