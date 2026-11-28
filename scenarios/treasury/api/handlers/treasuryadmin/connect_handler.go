package treasuryadmin

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/settlement"

	"connectrpc.com/connect"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Approvals interface {
	List(context.Context, approval.Status, ...string) ([]approval.Request, error)
	Resolve(context.Context, string, approval.Status, string) (approval.Request, error)
}

type Settlements interface {
	SettleOperator(context.Context, settlement.SettleInput, string) (settlement.Record, error)
	ResolveUnknown(context.Context, string) (settlement.Record, error)
}

type Authorizations interface {
	Get(context.Context, string) (authorization.Record, error)
}

type connectHandler struct {
	authorizer     operatorauth.Authorizer
	approvals      Approvals
	settlements    Settlements
	authorizations Authorizations
}

func NewConnectHandler(authorizer operatorauth.Authorizer, services ...any) authorizationconnect.TreasuryAdminHandler {
	handler := &connectHandler{authorizer: authorizer}
	for _, service := range services {
		switch value := service.(type) {
		case Approvals:
			handler.approvals = value
		case Settlements:
			handler.settlements = value
		case Authorizations:
			handler.authorizations = value
		}
	}
	return handler
}

func (h *connectHandler) operatorIdentity(ctx context.Context, headers mapHeader) (operatorauth.Identity, error) {
	if h.authorizer == nil {
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	identity, err := h.authorizer.Authorize(ctx, headers.Header())
	switch {
	case err == nil:
		return identity, nil
	case errors.Is(err, operatorauth.ErrRequired):
		return operatorauth.Identity{}, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operatorauth.ErrDenied):
		return operatorauth.Identity{}, connect.NewError(connect.CodePermissionDenied, err)
	default:
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
}

func (h *connectHandler) authorize(ctx context.Context, headers mapHeader) error {
	_, err := h.operatorIdentity(ctx, headers)
	return err
}

type mapHeader interface{ Header() http.Header }

func (h *connectHandler) ListApprovals(ctx context.Context, request *connect.Request[authorizationv1.ListApprovalsRequest]) (*connect.Response[authorizationv1.ListApprovalsResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.approvals == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("approval service unavailable"))
	}
	status, err := approvalStatusFromProto(request.Msg.GetStatus(), true)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	values, err := h.approvals.List(ctx, status, request.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &authorizationv1.ListApprovalsResponse{Approvals: make([]*sharedv1.ApprovalRequest, 0, len(values))}
	for _, value := range values {
		response.Approvals = append(response.Approvals, approvalToProto(value))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ResolveApproval(ctx context.Context, request *connect.Request[authorizationv1.ResolveApprovalRequest]) (*connect.Response[authorizationv1.ResolveApprovalResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.approvals == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("approval service unavailable"))
	}
	resolution, err := approvalStatusFromProto(request.Msg.GetResolution(), false)
	if err != nil || resolution != approval.StatusApproved && resolution != approval.StatusDeclined {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("approved or declined resolution is required"))
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	resolved, err := h.approvals.Resolve(ctx, request.Msg.GetApprovalId(), resolution, identity.Subject)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.ResolveApprovalResponse{Approval: approvalToProto(resolved)}), nil
}

func approvalStatusFromProto(status sharedv1.ApprovalStatus, allowUnspecified bool) (approval.Status, error) {
	switch status {
	case sharedv1.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED:
		if allowUnspecified {
			return "", nil
		}
	case sharedv1.ApprovalStatus_APPROVAL_STATUS_QUEUED:
		return approval.StatusQueued, nil
	case sharedv1.ApprovalStatus_APPROVAL_STATUS_APPROVED:
		return approval.StatusApproved, nil
	case sharedv1.ApprovalStatus_APPROVAL_STATUS_DECLINED:
		return approval.StatusDeclined, nil
	case sharedv1.ApprovalStatus_APPROVAL_STATUS_EXPIRED:
		return approval.StatusExpired, nil
	}
	return "", errors.New("unsupported approval status")
}

func approvalToProto(value approval.Request) *sharedv1.ApprovalRequest {
	status := sharedv1.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED
	switch value.Status {
	case approval.StatusQueued:
		status = sharedv1.ApprovalStatus_APPROVAL_STATUS_QUEUED
	case approval.StatusApproved:
		status = sharedv1.ApprovalStatus_APPROVAL_STATUS_APPROVED
	case approval.StatusDeclined:
		status = sharedv1.ApprovalStatus_APPROVAL_STATUS_DECLINED
	case approval.StatusExpired:
		status = sharedv1.ApprovalStatus_APPROVAL_STATUS_EXPIRED
	}
	msg := &sharedv1.ApprovalRequest{Id: value.ID, AuthorizationId: value.AuthorizationID, BookId: value.BookID, MandateId: value.MandateID, RequestingAgent: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Status: status, ResolverIdentity: value.ResolverIdentity, CreatedAt: timestamppb.New(value.CreatedAt), ExpiresAt: timestamppb.New(value.ExpiresAt)}
	if !value.ResolvedAt.IsZero() {
		msg.ResolvedAt = timestamppb.New(value.ResolvedAt)
	}
	return msg
}

func (h *connectHandler) ReportManualOutcome(ctx context.Context, request *connect.Request[authorizationv1.ReportManualOutcomeRequest]) (*connect.Response[authorizationv1.ReportManualOutcomeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.settlements == nil || h.authorizations == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("settlement services unavailable"))
	}
	input := request.Msg
	attestation := input.GetAttestation()
	if attestation == nil || attestation.GetOccurredAt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("complete manual attestation is required"))
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	charge, err := h.settlements.SettleOperator(ctx, settlement.SettleInput{
		ID: input.GetSettlementId(), AuthorizationID: input.GetAuthorizationId(), InstrumentID: input.GetInstrumentId(), IdempotencyKey: input.GetIdempotencyKey(),
		Attestation: &rail.Attestation{ExternalReference: attestation.GetExternalReference(), ReceiptReference: attestation.GetReceiptReference(), OccurredAt: attestation.GetOccurredAt().AsTime()},
	}, identity.Subject)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	auth, err := h.authorizations.Get(ctx, charge.AuthorizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&authorizationv1.ReportManualOutcomeResponse{Authorization: authorizationToProto(auth), Settlement: settlementToProto(charge)}), nil
}

func (h *connectHandler) ResolveUnknownSettlement(ctx context.Context, request *connect.Request[authorizationv1.ResolveUnknownSettlementRequest]) (*connect.Response[authorizationv1.ResolveUnknownSettlementResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetSettlementId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("settlement_id is required"))
	}
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if !request.Msg.GetConfirmPriorProcessStopped() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("confirm_prior_process_stopped is required before reconciliation"))
	}
	if h.settlements == nil || h.authorizations == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("settlement services unavailable"))
	}
	charge, err := h.settlements.ResolveUnknown(ctx, request.Msg.GetSettlementId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	auth, err := h.authorizations.Get(ctx, charge.AuthorizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&authorizationv1.ResolveUnknownSettlementResponse{Authorization: authorizationToProto(auth), Settlement: settlementToProto(charge)}), nil
}

func authorizationToProto(value authorization.Record) *authorizationv1.AuthorizationRecord {
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
	receiptReference := value.ReceiptReference
	if value.Rail == "x402" && value.Outcome == settlement.OutcomeUnknown {
		receiptReference = ""
	}
	result := &settlementv1.Charge{Id: value.ID, AuthorizationId: value.AuthorizationID, MandateId: value.MandateID, InstrumentId: value.InstrumentID, Rail: value.Rail, IdempotencyKey: value.IdempotencyKey, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Outcome: outcome, ExternalId: value.ExternalID, ReceiptReference: receiptReference, Basis: value.Basis, Detail: value.Detail, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), RetainUntil: timestamppb.New(value.RetainUntil)}
	if !value.OccurredAt.IsZero() {
		result.OccurredAt = timestamppb.New(value.OccurredAt)
	}
	if value.Outcome == settlement.OutcomeSettled && !value.OccurredAt.IsZero() {
		result.SettledAt = timestamppb.New(value.OccurredAt)
	}
	return result
}
