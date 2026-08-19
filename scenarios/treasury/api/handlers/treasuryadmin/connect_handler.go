package treasuryadmin

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	approvalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	instrumentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/approval"
	"treasury/internal/instrument"
	"treasury/internal/operatorauth"
)

type Approvals interface {
	Resolve(context.Context, string, approval.Status, string) (approval.Request, error)
}

type Instruments interface {
	Register(context.Context, instrument.RegisterInput) (instrument.Instrument, error)
}

type connectHandler struct {
	authorizer  operatorauth.Authorizer
	approvals   Approvals
	instruments Instruments
}

func NewConnectHandler(authorizer operatorauth.Authorizer, services ...any) authorizationconnect.TreasuryAdminHandler {
	handler := &connectHandler{authorizer: authorizer}
	for _, service := range services {
		switch value := service.(type) {
		case Approvals:
			handler.approvals = value
		case Instruments:
			handler.instruments = value
		}
	}
	return handler
}

func (h *connectHandler) authorize(ctx context.Context, headers mapHeader) error {
	if h.authorizer == nil {
		return connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	_, err := h.authorizer.Authorize(ctx, headers.Header())
	switch {
	case err == nil:
		return nil
	case errors.Is(err, operatorauth.ErrRequired):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operatorauth.ErrDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
}

type mapHeader interface{ Header() http.Header }

func authorizeThenUnimplemented[T any](h *connectHandler, ctx context.Context, request interface {
	Header() http.Header
}, operation string,
) (*connect.Response[T], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New(operation+" is declared structurally and lands with its owning domain phase"))
}

func (h *connectHandler) CreateMandate(ctx context.Context, request *connect.Request[authorizationv1.CreateMandateRequest]) (*connect.Response[authorizationv1.CreateMandateResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.CreateMandateResponse](h, ctx, request, "TreasuryAdmin.CreateMandate")
}

func (h *connectHandler) RevokeMandate(ctx context.Context, request *connect.Request[authorizationv1.RevokeMandateRequest]) (*connect.Response[authorizationv1.RevokeMandateResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.RevokeMandateResponse](h, ctx, request, "TreasuryAdmin.RevokeMandate")
}

func (h *connectHandler) SetBudgetCaps(ctx context.Context, request *connect.Request[authorizationv1.SetBudgetCapsRequest]) (*connect.Response[authorizationv1.SetBudgetCapsResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.SetBudgetCapsResponse](h, ctx, request, "TreasuryAdmin.SetBudgetCaps")
}

func (h *connectHandler) SetGating(ctx context.Context, request *connect.Request[authorizationv1.SetGatingRequest]) (*connect.Response[authorizationv1.SetGatingResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.SetGatingResponse](h, ctx, request, "TreasuryAdmin.SetGating")
}

func (h *connectHandler) ResolveApproval(ctx context.Context, request *connect.Request[authorizationv1.ResolveApprovalRequest]) (*connect.Response[authorizationv1.ResolveApprovalResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.approvals == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("approval service unavailable"))
	}
	resolution := approval.Status("")
	switch request.Msg.GetResolution() {
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_APPROVED:
		resolution = approval.StatusApproved
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED:
		resolution = approval.StatusDeclined
	default:
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
	status := approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED
	if resolved.Status == approval.StatusApproved {
		status = approvalv1.ApprovalStatus_APPROVAL_STATUS_APPROVED
	}
	msg := &approvalv1.ApprovalRequest{Id: resolved.ID, AuthorizationId: resolved.AuthorizationID, MandateId: resolved.MandateID, RequestingAgent: resolved.RequestingAgent, AmountMinor: resolved.AmountMinor, Currency: resolved.Currency, Counterparty: resolved.Counterparty, Status: status, ResolverIdentity: resolved.ResolverIdentity, CreatedAt: timestamppb.New(resolved.CreatedAt)}
	if !resolved.ResolvedAt.IsZero() {
		msg.ResolvedAt = timestamppb.New(resolved.ResolvedAt)
	}
	return connect.NewResponse(&authorizationv1.ResolveApprovalResponse{Approval: msg}), nil
}

func (h *connectHandler) FreezeBudget(ctx context.Context, request *connect.Request[authorizationv1.FreezeBudgetRequest]) (*connect.Response[authorizationv1.FreezeBudgetResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.FreezeBudgetResponse](h, ctx, request, "TreasuryAdmin.FreezeBudget")
}

func (h *connectHandler) UnfreezeBudget(ctx context.Context, request *connect.Request[authorizationv1.UnfreezeBudgetRequest]) (*connect.Response[authorizationv1.UnfreezeBudgetResponse], error) {
	return authorizeThenUnimplemented[authorizationv1.UnfreezeBudgetResponse](h, ctx, request, "TreasuryAdmin.UnfreezeBudget")
}

func (h *connectHandler) RegisterInstrument(ctx context.Context, request *connect.Request[authorizationv1.RegisterInstrumentRequest]) (*connect.Response[authorizationv1.RegisterInstrumentResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.instruments == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("instrument service unavailable"))
	}
	input := request.Msg.GetInstrument()
	if input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instrument is required"))
	}
	created, err := h.instruments.Register(ctx, instrument.RegisterInput{ID: input.GetId(), MandateID: input.GetMandateId(), Rail: input.GetRail(), CredentialReference: input.GetCredentialReference(), Counterparty: input.GetCounterparty()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// CredentialReference is intentionally absent from every response. It is
	// an internal locator into Secrets Manager, not operator-facing data.
	msg := &instrumentv1.Instrument{Id: created.ID, BookId: created.BookID, MandateId: created.MandateID, Rail: created.Rail, CapMinor: created.CapMinor, Currency: created.Currency, Counterparty: created.Counterparty, ExpiresAt: timestamppb.New(created.ExpiresAt)}
	return connect.NewResponse(&authorizationv1.RegisterInstrumentResponse{Instrument: msg}), nil
}
