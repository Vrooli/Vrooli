// Package rail defines the vendor-neutral value-movement adapter contract.
package rail

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid rail request")
	ErrNotFound = errors.New("rail adapter not found")
)

type Outcome string

const (
	OutcomeSettled Outcome = "settled"
	OutcomeFailed  Outcome = "failed"
	OutcomeUnknown Outcome = "unknown"
)

// Attestation is rail-neutral evidence that value moved outside Treasury.
// The manual adapter requires it; automated adapters normally ignore it.
type Attestation struct {
	ActorIdentity     string
	ExternalReference string
	ReceiptReference  string
	OccurredAt        time.Time
}

// SettleCommand is the only execution shape accepted by a rail. In
// particular, MandateReference is required even for an operator-reported
// payment, so the manual path is not privileged.
type SettleCommand struct {
	SettlementID     string
	AuthorizationID  string
	MandateReference string
	InstrumentID     string
	IdempotencyKey   string
	AmountMinor      int64
	Currency         string
	Counterparty     string
	Credential       string
	Attestation      *Attestation
}

type Query struct {
	SettlementID     string
	MandateReference string
	ExternalID       string
	IdempotencyKey   string
}

// Result is deliberately adapter-neutral. Settlement, evidence, and ledger
// consume this same envelope for manual and automated rails.
type Result struct {
	Outcome          Outcome
	ExternalID       string
	ReceiptReference string
	Basis            string
	OccurredAt       time.Time
	Detail           string
}

type Adapter interface {
	Name() string
	Settle(context.Context, SettleCommand) (Result, error)
	Query(context.Context, Query) (Result, error)
}

func ValidateSettle(command SettleCommand) error {
	command.SettlementID = strings.TrimSpace(command.SettlementID)
	command.AuthorizationID = strings.TrimSpace(command.AuthorizationID)
	command.MandateReference = strings.TrimSpace(command.MandateReference)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.Currency = strings.ToUpper(strings.TrimSpace(command.Currency))
	command.Counterparty = strings.ToLower(strings.TrimSpace(command.Counterparty))
	switch {
	case command.SettlementID == "":
		return fmt.Errorf("%w: settlement_id is required", ErrInvalid)
	case command.AuthorizationID == "":
		return fmt.Errorf("%w: authorization_id is required", ErrInvalid)
	case command.MandateReference == "":
		return fmt.Errorf("%w: mandate_reference is required", ErrInvalid)
	case command.IdempotencyKey == "":
		return fmt.Errorf("%w: idempotency_key is required", ErrInvalid)
	case command.AmountMinor <= 0:
		return fmt.Errorf("%w: amount_minor must be positive", ErrInvalid)
	case command.Currency == "":
		return fmt.Errorf("%w: currency is required", ErrInvalid)
	case command.Counterparty == "":
		return fmt.Errorf("%w: counterparty is required", ErrInvalid)
	default:
		return nil
	}
}

type Registry struct{ adapters map[string]Adapter }

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	registry := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("%w: nil adapter", ErrInvalid)
		}
		name := strings.ToLower(strings.TrimSpace(adapter.Name()))
		if name == "" {
			return nil, fmt.Errorf("%w: adapter name is required", ErrInvalid)
		}
		if _, exists := registry.adapters[name]; exists {
			return nil, fmt.Errorf("%w: duplicate adapter %q", ErrInvalid, name)
		}
		registry.adapters[name] = guardedAdapter{name: name, inner: adapter}
	}
	return registry, nil
}

func (r *Registry) Get(name string) (Adapter, error) {
	if r == nil {
		return nil, ErrNotFound
	}
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, ErrNotFound
	}
	return adapter, nil
}

func (r *Registry) Has(name string) bool {
	_, err := r.Get(name)
	return err == nil
}

func (r *Registry) Adapters() []Adapter {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	values := make([]Adapter, 0, len(names))
	for _, name := range names {
		values = append(values, r.adapters[name])
	}
	return values
}

type guardedAdapter struct {
	name  string
	inner Adapter
}

func (a guardedAdapter) Name() string { return a.name }

func (a guardedAdapter) Settle(ctx context.Context, command SettleCommand) (Result, error) {
	if err := ValidateSettle(command); err != nil {
		return Result{}, err
	}
	return a.inner.Settle(ctx, command)
}

func (a guardedAdapter) Query(ctx context.Context, query Query) (Result, error) {
	if strings.TrimSpace(query.SettlementID) == "" || strings.TrimSpace(query.MandateReference) == "" || (strings.TrimSpace(query.ExternalID) == "" && strings.TrimSpace(query.IdempotencyKey) == "") {
		return Result{}, fmt.Errorf("%w: settlement_id, mandate_reference, and an external_id or idempotency_key are required for query", ErrInvalid)
	}
	return a.inner.Query(ctx, query)
}
