package agentspend

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
	"treasury/internal/settlement"
)

type MandateLister interface {
	List(context.Context) ([]mandate.Mandate, error)
}

type HeadroomReader interface {
	Headroom(context.Context, string) (budget.Headroom, error)
}

type connectHandler struct {
	service     *authorization.Service
	settlements *settlement.Service
	mandates    MandateLister
	verifier    identity.Verifier
	headroom    HeadroomReader
}

func NewConnectHandler(services ...any) authorizationconnect.AgentSpendHandler {
	handler := &connectHandler{}
	for _, service := range services {
		switch value := service.(type) {
		case *authorization.Service:
			handler.service = value
		case *settlement.Service:
			handler.settlements = value
		case MandateLister:
			handler.mandates = value
		case identity.Verifier:
			handler.verifier = value
		case HeadroomReader:
			handler.headroom = value
		}
	}
	return handler
}

func (h *connectHandler) GetBudgetHeadroom(ctx context.Context, request *connect.Request[authorizationv1.GetBudgetHeadroomRequest]) (*connect.Response[authorizationv1.GetBudgetHeadroomResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.headroom == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("headroom reporting is unavailable"))
	}
	if _, err := h.verifier.Verify(ctx, identityTokenFromHeaders(request.Header())); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	value, err := h.headroom.Headroom(ctx, request.Msg.GetBudgetId())
	if err != nil {
		if errors.Is(err, budget.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&authorizationv1.GetBudgetHeadroomResponse{Headroom: &budgetv1.Headroom{
		BudgetId: value.BudgetID, BookId: value.BookID, Currency: value.Currency,
		TotalCapMinor: value.TotalCapMinor, TotalUsedMinor: value.TotalUsedMinor, TotalRemainingMinor: value.TotalRemainingMinor,
		PeriodicCapMinor: value.PeriodicCapMinor, PeriodUsedMinor: value.PeriodUsedMinor, PeriodRemainingMinor: value.PeriodRemainingMinor,
		PerTransactionCapMinor: value.PerTransactionCapMinor, AvailableMinor: value.AvailableMinor,
		PeriodStartedAt: timestamppb.New(value.PeriodStartedAt), ComputedAt: timestamppb.New(value.ComputedAt),
	}}), nil
}

func (h *connectHandler) ProposeCharge(ctx context.Context, request *connect.Request[authorizationv1.ProposeChargeRequest]) (*connect.Response[authorizationv1.ProposeChargeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service is unavailable"))
	}
	msg := request.Msg
	result, err := h.service.Propose(ctx, authorization.ProposeInput{ID: msg.GetId(), IdempotencyKey: msg.GetIdempotencyKey(), MandateID: msg.GetMandateId(), IdentityToken: identityTokenFromHeaders(request.Header()), AmountMinor: msg.GetAmountMinor(), Currency: msg.GetCurrency(), Counterparty: msg.GetCounterparty()})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.ProposeChargeResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) GetAuthorization(ctx context.Context, request *connect.Request[authorizationv1.GetAuthorizationRequest]) (*connect.Response[authorizationv1.GetAuthorizationResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service is unavailable"))
	}
	result, err := h.service.Get(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.GetAuthorizationResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) ListMandates(ctx context.Context, request *connect.Request[authorizationv1.ListMandatesRequest]) (*connect.Response[authorizationv1.ListMandatesResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.mandates == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("mandate listing is unavailable"))
	}
	if _, err := h.verifier.Verify(ctx, identityTokenFromHeaders(request.Header())); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	values, err := h.mandates.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &authorizationv1.ListMandatesResponse{Mandates: make([]*mandatev1.Mandate, 0, len(values))}
	for _, value := range values {
		response.Mandates = append(response.Mandates, mandateToProto(value))
	}
	return connect.NewResponse(response), nil
}

func mandateToProto(value mandate.Mandate) *mandatev1.Mandate {
	status := mandatev1.MandateStatus_MANDATE_STATUS_UNSPECIFIED
	switch value.Status {
	case mandateflow.MandateDraft:
		status = mandatev1.MandateStatus_MANDATE_STATUS_DRAFT
	case mandateflow.MandateLive:
		status = mandatev1.MandateStatus_MANDATE_STATUS_LIVE
	case mandateflow.MandateExhausted:
		status = mandatev1.MandateStatus_MANDATE_STATUS_EXHAUSTED
	case mandateflow.MandateExpired:
		status = mandatev1.MandateStatus_MANDATE_STATUS_EXPIRED
	case mandateflow.MandateRevoked:
		status = mandatev1.MandateStatus_MANDATE_STATUS_REVOKED
	}
	result := &mandatev1.Mandate{Id: value.ID, IdempotencyKey: value.IdempotencyKey, BookId: value.BookID, BudgetId: value.BudgetID, Authorizer: value.Authorizer, CapMinor: value.CapMinor, Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties, DeniedCounterparties: value.DeniedCounterparties, RequiredEvidence: value.RequiredEvidence, ExpiresAt: timestamppb.New(value.ExpiresAt), IssuedAt: timestamppb.New(value.IssuedAt), Signature: value.Signature, Status: status, RecurrenceSeconds: int64(value.RecurrenceInterval / time.Second)}
	if !value.NextChargeAt.IsZero() {
		result.NextChargeAt = timestamppb.New(value.NextChargeAt)
	}
	if !value.CancelledAt.IsZero() {
		result.CancelledAt = timestamppb.New(value.CancelledAt)
	}
	return result
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
	charge, err := h.settlements.Settle(ctx, settlement.SettleInput{ID: msg.GetSettlementId(), AuthorizationID: msg.GetAuthorizationId(), InstrumentID: msg.GetInstrumentId(), IdempotencyKey: msg.GetIdempotencyKey(), IdentityToken: identityTokenFromHeaders(request.Header())})
	if err != nil {
		return nil, mapSettlementError(err)
	}
	auth, err := h.service.Get(ctx, charge.AuthorizationID)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.ReportOutcomeResponse{Authorization: toProto(auth), Settlement: settlementToProto(charge)}), nil
}

func identityTokenFromHeaders(headers http.Header) string {
	if token := strings.TrimSpace(headers.Get(identity.HeaderAgentIdentityToken)); token != "" {
		return token
	}
	parts := strings.Fields(strings.TrimSpace(headers.Get("Authorization")))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
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
	return &authorizationv1.AuthorizationRecord{Id: value.ID, IdempotencyKey: value.IdempotencyKey, BookId: value.BookID, MandateId: value.MandateID, BudgetId: value.BudgetID, RequestingAgent: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Verdict: verdict, ViolatedConstraint: value.ViolatedConstraint, Remediation: value.Remediation, HoldMinor: value.HoldMinor, CreatedAt: timestamppb.New(value.CreatedAt), ExpiresAt: timestamppb.New(value.ExpiresAt)}
}
