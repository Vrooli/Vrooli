package agentspend

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	"treasury/internal/authorization"
	"treasury/internal/identity"
	"treasury/internal/settlement"
)

type connectHandler struct {
	service     *authorization.Service
	settlements *settlement.Service
}

func NewConnectHandler(service *authorization.Service, settlements ...*settlement.Service) authorizationconnect.AgentSpendHandler {
	handler := &connectHandler{service: service}
	if len(settlements) > 0 {
		handler.settlements = settlements[0]
	}
	return handler
}

func (h *connectHandler) ProposeCharge(ctx context.Context, request *connect.Request[authorizationv1.ProposeChargeRequest]) (*connect.Response[authorizationv1.ProposeChargeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	msg := request.Msg
	result, err := h.service.Propose(ctx, authorization.ProposeInput{ID: msg.GetId(), IdempotencyKey: msg.GetIdempotencyKey(), MandateID: msg.GetMandateId(), IdentityToken: request.Header().Get(identity.HeaderAgentIdentityToken), AmountMinor: msg.GetAmountMinor(), Currency: msg.GetCurrency(), Counterparty: msg.GetCounterparty()})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.ProposeChargeResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) GetAuthorization(ctx context.Context, request *connect.Request[authorizationv1.GetAuthorizationRequest]) (*connect.Response[authorizationv1.GetAuthorizationResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	result, err := h.service.Get(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.GetAuthorizationResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) ListMandates(context.Context, *connect.Request[authorizationv1.ListMandatesRequest]) (*connect.Response[authorizationv1.ListMandatesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("AgentSpend.ListMandates lands with the governed agent CLI surface"))
}

func (h *connectHandler) ReportOutcome(ctx context.Context, request *connect.Request[authorizationv1.ReportOutcomeRequest]) (*connect.Response[authorizationv1.ReportOutcomeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.settlements == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("settlement service is unavailable"))
	}
	msg := request.Msg
	if strings.TrimSpace(msg.GetOutcome()) != "" || strings.TrimSpace(msg.GetRailReference()) != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("caller-asserted outcome and rail_reference are forbidden; Treasury reads both from the registered rail"))
	}
	charge, err := h.settlements.Settle(ctx, settlement.SettleInput{ID: msg.GetSettlementId(), AuthorizationID: msg.GetAuthorizationId(), InstrumentID: msg.GetInstrumentId(), IdempotencyKey: msg.GetIdempotencyKey(), IdentityToken: request.Header().Get(identity.HeaderAgentIdentityToken)})
	if err != nil {
		return nil, mapSettlementError(err)
	}
	auth, err := h.service.Get(ctx, charge.AuthorizationID)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.ReportOutcomeResponse{Authorization: toProto(auth), Settlement: settlementToProto(charge)}), nil
}

func mapSettlementError(err error) error {
	switch {
	case errors.Is(err, settlement.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, settlement.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func settlementToProto(value settlement.Record) *settlementv1.Charge {
	outcome := settlementv1.ChargeOutcome_CHARGE_OUTCOME_UNSPECIFIED
	switch value.Outcome {
	case settlement.OutcomeReady:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_READY
	case settlement.OutcomeCalling:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_CALLING
	case settlement.OutcomeSettled:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_SETTLED
	case settlement.OutcomeFailed:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_FAILED
	case settlement.OutcomeUnknown:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_UNKNOWN
	}
	result := &settlementv1.Charge{Id: value.ID, AuthorizationId: value.AuthorizationID, MandateId: value.MandateID, InstrumentId: value.InstrumentID, Rail: value.Rail, IdempotencyKey: value.IdempotencyKey, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Outcome: outcome, ExternalId: value.ExternalID, ReceiptReference: value.ReceiptReference, Basis: value.Basis, Detail: value.Detail, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), RetainUntil: timestamppb.New(value.RetainUntil)}
	if !value.OccurredAt.IsZero() {
		result.OccurredAt = timestamppb.New(value.OccurredAt)
	}
	if value.Outcome == settlement.OutcomeSettled && !value.OccurredAt.IsZero() {
		result.SettledAt = timestamppb.New(value.OccurredAt)
	}
	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, authorization.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, authorization.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProto(value authorization.Record) *authorizationv1.AuthorizationRecord {
	verdict := authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_UNSPECIFIED
	switch value.Verdict {
	case authorization.VerdictRefused:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED
	case authorization.VerdictPending:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING
	case authorization.VerdictApproved:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED
	case authorization.VerdictReleased:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_RELEASED
	case authorization.VerdictSettled:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED
	}
	return &authorizationv1.AuthorizationRecord{Id: value.ID, IdempotencyKey: value.IdempotencyKey, MandateId: value.MandateID, BudgetId: value.BudgetID, RequestingAgent: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Verdict: verdict, ViolatedConstraint: value.ViolatedConstraint, Remediation: value.Remediation, HoldMinor: value.HoldMinor, CreatedAt: timestamppb.New(value.CreatedAt), ExpiresAt: timestamppb.New(value.ExpiresAt)}
}
