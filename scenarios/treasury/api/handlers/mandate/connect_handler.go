package mandate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	domain "treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
	"treasury/internal/walletops"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	mandateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate/mandate_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Issue(context.Context, domain.IssueInput) (domain.Mandate, error)
	Revoke(context.Context, string) (domain.Mandate, error)
	CancelStanding(context.Context, string) (domain.Mandate, error)
	List(context.Context) ([]domain.Mandate, error)
}

type SessionRevoker interface {
	RevokeMandateSession(context.Context, string, string) error
}

type connectHandler struct {
	authorizer operatorauth.Authorizer
	mandates   Service
	sessions   SessionRevoker
}

func NewConnectHandler(authorizer operatorauth.Authorizer, mandates Service, sessions SessionRevoker) mandateconnect.MandateAdminServiceHandler {
	return &connectHandler{authorizer: authorizer, mandates: mandates, sessions: sessions}
}

func (h *connectHandler) operator(ctx context.Context, request interface{ Header() http.Header }) (operatorauth.Identity, error) {
	if h.authorizer == nil {
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
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

func (h *connectHandler) CreateMandate(ctx context.Context, request *connect.Request[mandatev1.CreateMandateRequest]) (*connect.Response[mandatev1.CreateMandateResponse], error) {
	identity, err := h.operator(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	input := request.Msg.GetMandate()
	if input == nil || input.GetExpiresAt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mandate with expiry is required"))
	}
	issue := domain.IssueInput{AgentSubject: input.GetAgentSubject(), AgentRunID: input.GetAgentRunId(), ID: input.GetId(), IdempotencyKey: input.GetIdempotencyKey(), BookID: input.GetBookId(), BudgetID: input.GetBudgetId(), Authorizer: identity.Subject, CapMinor: input.GetCapMinor(), Currency: input.GetCurrency(), AllowedCounterparties: input.GetAllowedCounterparties(), DeniedCounterparties: input.GetDeniedCounterparties(), RequiredEvidence: input.GetRequiredEvidence(), ExpiresAt: input.GetExpiresAt().AsTime(), RecurrenceInterval: time.Duration(input.GetRecurrenceSeconds()) * time.Second}
	if input.GetNextChargeAt() != nil {
		issue.NextChargeAt = input.GetNextChargeAt().AsTime()
	}
	created, err := h.mandates.Issue(ctx, issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&mandatev1.CreateMandateResponse{Mandate: toProto(created)}), nil
}

func (h *connectHandler) RevokeMandate(ctx context.Context, request *connect.Request[mandatev1.RevokeMandateRequest]) (*connect.Response[mandatev1.RevokeMandateResponse], error) {
	identity, err := h.operator(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	value, err := h.mandates.Revoke(ctx, request.Msg.GetMandateId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.sessions != nil {
		if revokeErr := h.sessions.RevokeMandateSession(ctx, identity.Subject, request.Msg.GetMandateId()); revokeErr != nil && !errors.Is(revokeErr, walletops.ErrNotFound) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("revoke mandate session key: %w", revokeErr))
		}
	}
	return connect.NewResponse(&mandatev1.RevokeMandateResponse{Mandate: toProto(value)}), nil
}

func (h *connectHandler) RevokeMandateSession(ctx context.Context, request *connect.Request[mandatev1.RevokeMandateSessionRequest]) (*connect.Response[mandatev1.RevokeMandateSessionResponse], error) {
	identity, err := h.operator(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate session lifecycle unavailable"))
	}
	if request == nil || request.Msg == nil || request.Msg.GetMandateId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mandate ID is required"))
	}
	if err := h.sessions.RevokeMandateSession(ctx, identity.Subject, request.Msg.GetMandateId()); err != nil {
		switch {
		case errors.Is(err, walletops.ErrNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, walletops.ErrInvalid):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, walletops.ErrConflict):
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&mandatev1.RevokeMandateSessionResponse{Revoked: true}), nil
}

func (h *connectHandler) CancelStandingMandate(ctx context.Context, request *connect.Request[mandatev1.CancelStandingMandateRequest]) (*connect.Response[mandatev1.CancelStandingMandateResponse], error) {
	if _, err := h.operator(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	value, err := h.mandates.CancelStanding(ctx, request.Msg.GetMandateId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&mandatev1.CancelStandingMandateResponse{Mandate: toProto(value)}), nil
}

func (h *connectHandler) ListMandates(ctx context.Context, request *connect.Request[mandatev1.ListMandatesRequest]) (*connect.Response[mandatev1.ListMandatesResponse], error) {
	if _, err := h.operator(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	values, err := h.mandates.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &mandatev1.ListMandatesResponse{Mandates: make([]*mandatev1.Mandate, 0, len(values))}
	for _, value := range values {
		response.Mandates = append(response.Mandates, toProto(value))
	}
	return connect.NewResponse(response), nil
}

func toProto(value domain.Mandate) *mandatev1.Mandate {
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
	result := &mandatev1.Mandate{AgentSubject: value.AgentSubject, AgentRunId: value.AgentRunID, Id: value.ID, IdempotencyKey: value.IdempotencyKey, BookId: value.BookID, BudgetId: value.BudgetID, Authorizer: value.Authorizer, CapMinor: value.CapMinor, Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties, DeniedCounterparties: value.DeniedCounterparties, RequiredEvidence: value.RequiredEvidence, ExpiresAt: timestamppb.New(value.ExpiresAt), IssuedAt: timestamppb.New(value.IssuedAt), Signature: value.Signature, Status: status, RecurrenceSeconds: int64(value.RecurrenceInterval / time.Second)}
	if !value.NextChargeAt.IsZero() {
		result.NextChargeAt = timestamppb.New(value.NextChargeAt)
	}
	if !value.CancelledAt.IsZero() {
		result.CancelledAt = timestamppb.New(value.CancelledAt)
	}
	return result
}

func Module(authorizer operatorauth.Authorizer, mandates Service, sessions SessionRevoker) module.Module {
	path, handler := mandateconnect.NewMandateAdminServiceHandler(NewConnectHandler(authorizer, mandates, sessions))
	return module.Module{Name: "mandateadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
