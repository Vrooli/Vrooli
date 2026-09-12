package treasuryadmin_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	"google.golang.org/protobuf/types/known/timestamppb"

	"treasury/handlers/treasuryadmin"
	"treasury/internal/authorization"
	"treasury/internal/operatorauth"
	"treasury/internal/settlement"
)

type operatorSettlement struct{ input settlement.SettleInput }

func (s *operatorSettlement) SettleOperator(_ context.Context, input settlement.SettleInput, subject string) (settlement.Record, error) {
	s.input = input
	if input.Attestation != nil {
		input.Attestation.ActorIdentity = subject
	}
	now := input.Attestation.OccurredAt
	return settlement.Record{ID: input.ID, AuthorizationID: input.AuthorizationID, MandateID: "mandate-1", InstrumentID: input.InstrumentID, Rail: "manual", IdempotencyKey: input.IdempotencyKey, AmountMinor: 250, Currency: "USD", Counterparty: "vendor.example", Outcome: settlement.OutcomeSettled, ExternalID: input.Attestation.ExternalReference, ReceiptReference: input.Attestation.ReceiptReference, Basis: "operator_attestation", OccurredAt: now, CreatedAt: now, UpdatedAt: now, RetainUntil: now.Add(settlement.RetentionWindow)}, nil
}

type authorizationReader struct{ value authorization.Record }

func (r authorizationReader) Get(context.Context, string) (authorization.Record, error) {
	return r.value, nil
}

// [REQ:TRS-P0-011] Manual evidence enters only through the operator realm; the
// server supplies the actor identity instead of trusting an asserted subject.
func TestManualSettlementBindsAuthenticatedOperator(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	settlements := &operatorSettlement{}
	reader := authorizationReader{value: authorization.Record{ID: "auth-1", MandateID: "mandate-1", RequestingAgent: "agent:1", Verdict: authorization.VerdictSettled, AmountMinor: 250, Currency: "USD", Counterparty: "vendor.example", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}}
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, settlements, reader))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorizationv1.ReportManualOutcomeRequest{AuthorizationId: "auth-1", SettlementId: "settlement-1", InstrumentId: "instrument-1", IdempotencyKey: "settle-key", Attestation: &settlementv1.ManualAttestation{ExternalReference: "bank-1", ReceiptReference: "receipt-1", OccurredAt: timestamppb.New(now)}})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")

	response, err := client.ReportManualOutcome(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, settlementv1.ChargeOutcome_CHARGE_OUTCOME_SETTLED, response.Msg.GetSettlement().GetOutcome())
	require.Equal(t, "local-operator", settlements.input.Attestation.ActorIdentity)
	require.Equal(t, "bank-1", settlements.input.Attestation.ExternalReference)
}
