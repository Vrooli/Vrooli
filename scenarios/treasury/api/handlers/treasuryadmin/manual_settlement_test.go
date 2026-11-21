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

type operatorSettlement struct {
	input     settlement.SettleInput
	resolveID string
	resolve   settlement.Record
}

func (s *operatorSettlement) SettleOperator(_ context.Context, input settlement.SettleInput, subject string) (settlement.Record, error) {
	s.input = input
	if input.Attestation != nil {
		input.Attestation.ActorIdentity = subject
	}
	now := input.Attestation.OccurredAt
	return settlement.Record{ID: input.ID, AuthorizationID: input.AuthorizationID, MandateID: "mandate-1", InstrumentID: input.InstrumentID, Rail: "manual", IdempotencyKey: input.IdempotencyKey, AmountMinor: 250, Currency: "USD", Counterparty: "vendor.example", Outcome: settlement.OutcomeSettled, ExternalID: input.Attestation.ExternalReference, ReceiptReference: input.Attestation.ReceiptReference, Basis: "operator_attestation", OccurredAt: now, CreatedAt: now, UpdatedAt: now, RetainUntil: now.Add(settlement.RetentionWindow)}, nil
}

func (s *operatorSettlement) ResolveUnknown(_ context.Context, id string) (settlement.Record, error) {
	s.resolveID = id
	return s.resolve, nil
}

type authorizationReader struct{ value authorization.Record }

func (r authorizationReader) Get(context.Context, string) (authorization.Record, error) {
	return r.value, nil
}

func TestResolveUnknownSettlementIsOperatorOnlyAndReturnsUpdatedState(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	settlements := &operatorSettlement{resolve: settlement.Record{
		ID: "settlement-1", AuthorizationID: "auth-1", MandateID: "mandate-1", InstrumentID: "instrument-1",
		Rail: "x402", IdempotencyKey: "settle-key", Outcome: settlement.OutcomeUnknown,
		ExternalID: "sha256:payload-digest", ReceiptReference: "synthetic-signed-authorization",
		CreatedAt: now, UpdatedAt: now, RetainUntil: now.Add(settlement.RetentionWindow),
	}}
	reader := authorizationReader{value: authorization.Record{ID: "auth-1", MandateID: "mandate-1", RequestingAgent: "agent:1", Verdict: authorization.VerdictApproved, AmountMinor: 250, HoldMinor: 250, Currency: "USD", Counterparty: "vendor.example", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}}
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, settlements, reader))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)

	agentRequest := connect.NewRequest(&authorizationv1.ResolveUnknownSettlementRequest{SettlementId: "settlement-1", ConfirmPriorProcessStopped: true})
	agentRequest.Header().Set(operatorauth.HeaderAgentToken, "valid-agent-token")
	_, err = client.ResolveUnknownSettlement(context.Background(), agentRequest)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Empty(t, settlements.resolveID, "agent realm must never initiate external outcome replay")

	missingConfirmation := connect.NewRequest(&authorizationv1.ResolveUnknownSettlementRequest{SettlementId: "settlement-1"})
	missingConfirmation.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	_, err = client.ResolveUnknownSettlement(context.Background(), missingConfirmation)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Empty(t, settlements.resolveID, "operator must attest that the prior process stopped")

	request := connect.NewRequest(&authorizationv1.ResolveUnknownSettlementRequest{SettlementId: "settlement-1", ConfirmPriorProcessStopped: true})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	response, err := client.ResolveUnknownSettlement(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "settlement-1", settlements.resolveID)
	require.Equal(t, settlementv1.ChargeOutcome_CHARGE_OUTCOME_UNKNOWN, response.Msg.GetSettlement().GetOutcome())
	require.Empty(t, response.Msg.GetSettlement().GetReceiptReference(), "pending x402 replay references stay inside Treasury")
	require.Equal(t, authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED, response.Msg.GetAuthorization().GetVerdict())
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
