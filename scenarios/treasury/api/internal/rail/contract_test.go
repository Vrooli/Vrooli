package rail_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"treasury/internal/rail"
	"treasury/internal/rail/manual"
)

// [REQ:TRS-P0-001] [REQ:TRS-P0-007] The registry is the conformance
// boundary: every adapter rejects execution without human-issued authority.
func TestEveryRegisteredAdapterRejectsMissingMandateReference(t *testing.T) {
	registry, err := rail.NewRegistry(manual.New(), automatedFixture{})
	require.NoError(t, err)
	require.NotEmpty(t, registry.Adapters(), "the table must fail loudly if registration disappears")
	for _, adapter := range registry.Adapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			_, err := adapter.Settle(context.Background(), validCommand())
			require.ErrorIs(t, err, rail.ErrInvalid)
			require.ErrorContains(t, err, "mandate_reference")
		})
	}
}

// [REQ:TRS-P0-007] Settlement receives the same result type from manual and
// automated rails, so evidence and emission have no adapter-specific branch.
func TestManualAndAutomatedRailsReturnTheSameDownstreamEnvelope(t *testing.T) {
	registry, err := rail.NewRegistry(manual.New(), automatedFixture{})
	require.NoError(t, err)
	command := validCommand()
	command.MandateReference = "mandate-1"
	command.Attestation = &rail.Attestation{ActorIdentity: "operator:1", ExternalReference: "bank-1", ReceiptReference: "receipt-1", OccurredAt: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)}
	for _, name := range []string{"manual", "automated-fixture"} {
		adapter, getErr := registry.Get(name)
		require.NoError(t, getErr)
		result, settleErr := adapter.Settle(context.Background(), command)
		require.NoError(t, settleErr)
		require.Equal(t, rail.OutcomeSettled, result.Outcome)
		require.NotEmpty(t, result.ExternalID)
		require.NotEmpty(t, result.ReceiptReference)
		require.NotEmpty(t, result.Basis)
		require.False(t, result.OccurredAt.IsZero())
	}
}

func validCommand() rail.SettleCommand {
	return rail.SettleCommand{SettlementID: "settlement-1", AuthorizationID: "auth-1", IdempotencyKey: "idem-1", AmountMinor: 100, Currency: "USD", Counterparty: "vendor.example"}
}

type automatedFixture struct{}

func (automatedFixture) Name() string { return "automated-fixture" }
func (automatedFixture) Settle(_ context.Context, command rail.SettleCommand) (rail.Result, error) {
	if err := rail.ValidateSettle(command); err != nil {
		return rail.Result{}, err
	}
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: "processor-1", ReceiptReference: "processor-receipt-1", Basis: "processor_confirmation", OccurredAt: time.Date(2026, 8, 18, 12, 0, 1, 0, time.UTC)}, nil
}

func (automatedFixture) QueryOutcome(context.Context, rail.Query) (rail.Result, error) {
	return rail.Result{}, nil
}
