// Package manual implements operator-attested settlement through the same
// rail contract used by automated adapters.
package manual

import (
	"context"
	"fmt"
	"strings"

	"treasury/internal/rail"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (*Adapter) Name() string { return "manual" }

func (*Adapter) Settle(_ context.Context, command rail.SettleCommand) (rail.Result, error) {
	if err := rail.ValidateSettle(command); err != nil {
		return rail.Result{}, err
	}
	claim := command.Attestation
	if claim == nil || strings.TrimSpace(claim.ActorIdentity) == "" || strings.TrimSpace(claim.ExternalReference) == "" || strings.TrimSpace(claim.ReceiptReference) == "" || claim.OccurredAt.IsZero() {
		return rail.Result{}, fmt.Errorf("%w: manual rail requires a complete operator attestation", rail.ErrInvalid)
	}
	return rail.Result{
		Outcome:          rail.OutcomeSettled,
		ExternalID:       strings.TrimSpace(claim.ExternalReference),
		ReceiptReference: strings.TrimSpace(claim.ReceiptReference),
		Basis:            "operator_attestation",
		OccurredAt:       claim.OccurredAt.UTC(),
		Detail:           "operator reported an externally settled payment",
	}, nil
}

func (*Adapter) Query(_ context.Context, query rail.Query) (rail.Result, error) {
	if strings.TrimSpace(query.SettlementID) == "" || strings.TrimSpace(query.MandateReference) == "" || (strings.TrimSpace(query.ExternalID) == "" && strings.TrimSpace(query.IdempotencyKey) == "") {
		return rail.Result{}, fmt.Errorf("%w: complete manual query is required", rail.ErrInvalid)
	}
	return rail.Result{}, fmt.Errorf("%w: manual settlements are terminal at operator attestation and never enter unknown", rail.ErrInvalid)
}

var _ rail.Adapter = (*Adapter)(nil)
