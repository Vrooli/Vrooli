package treasuryadmin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/settlement"

	"connectrpc.com/connect"
	approvalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	instrumentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Approvals interface {
	List(context.Context, approval.Status, ...string) ([]approval.Request, error)
	Resolve(context.Context, string, approval.Status, string) (approval.Request, error)
}

type Instruments interface {
	Register(context.Context, instrument.RegisterInput) (instrument.Instrument, error)
}

type Settlements interface {
	SettleOperator(context.Context, settlement.SettleInput, string) (settlement.Record, error)
}

type Authorizations interface {
	Get(context.Context, string) (authorization.Record, error)
}

type Books interface {
	Create(context.Context, book.CreateInput) (book.Book, error)
	Get(context.Context, string) (book.Book, error)
}

type Budgets interface {
	SetCaps(context.Context, budget.Budget) (budget.Budget, error)
	SetGating(context.Context, string, bool) (budget.Budget, error)
	SetFrozen(context.Context, string, bool) (budget.Budget, error)
	SetScopeFrozen(context.Context, budget.FreezeScope, string, bool) (budget.FreezeControl, error)
	ScenarioFreezeStatus(context.Context) (budget.FreezeControl, error)
}

type Mandates interface {
	Issue(context.Context, mandate.IssueInput) (mandate.Mandate, error)
	Revoke(context.Context, string) (mandate.Mandate, error)
	CancelStanding(context.Context, string) (mandate.Mandate, error)
	List(context.Context) ([]mandate.Mandate, error)
}

type connectHandler struct {
	authorizer     operatorauth.Authorizer
	approvals      Approvals
	instruments    Instruments
	settlements    Settlements
	authorizations Authorizations
	books          Books
	budgets        Budgets
	mandates       Mandates
}

func NewConnectHandler(authorizer operatorauth.Authorizer, services ...any) authorizationconnect.TreasuryAdminHandler {
	handler := &connectHandler{authorizer: authorizer}
	for _, service := range services {
		switch value := service.(type) {
		case Approvals:
			handler.approvals = value
		case Instruments:
			handler.instruments = value
		case Settlements:
			handler.settlements = value
		case Authorizations:
			handler.authorizations = value
		case Books:
			handler.books = value
		case Budgets:
			handler.budgets = value
		case Mandates:
			handler.mandates = value
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

func (h *connectHandler) CreateBook(ctx context.Context, request *connect.Request[authorizationv1.CreateBookRequest]) (*connect.Response[authorizationv1.CreateBookResponse], error) {
	identity, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.books == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("book service unavailable"))
	}
	input := request.Msg.GetBook()
	if input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("book is required"))
	}
	created, err := h.books.Create(ctx, book.CreateInput{ID: input.GetId(), Name: input.GetName(), BeneficiaryIdentity: identity.Subject})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.CreateBookResponse{Book: bookToProto(created)}), nil
}

func (h *connectHandler) GetBook(ctx context.Context, request *connect.Request[authorizationv1.GetBookRequest]) (*connect.Response[authorizationv1.GetBookResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.books == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("book service unavailable"))
	}
	value, err := h.books.Get(ctx, request.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.GetBookResponse{Book: bookToProto(value)}), nil
}

func (h *connectHandler) CreateMandate(ctx context.Context, request *connect.Request[authorizationv1.CreateMandateRequest]) (*connect.Response[authorizationv1.CreateMandateResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	input := request.Msg.GetMandate()
	if input == nil || input.GetExpiresAt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mandate with expiry is required"))
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	issue := mandate.IssueInput{ID: input.GetId(), IdempotencyKey: input.GetIdempotencyKey(), BookID: input.GetBookId(), BudgetID: input.GetBudgetId(), Authorizer: identity.Subject, CapMinor: input.GetCapMinor(), Currency: input.GetCurrency(), AllowedCounterparties: input.GetAllowedCounterparties(), DeniedCounterparties: input.GetDeniedCounterparties(), RequiredEvidence: input.GetRequiredEvidence(), ExpiresAt: input.GetExpiresAt().AsTime(), RecurrenceInterval: time.Duration(input.GetRecurrenceSeconds()) * time.Second}
	if input.GetNextChargeAt() != nil {
		issue.NextChargeAt = input.GetNextChargeAt().AsTime()
	}
	created, err := h.mandates.Issue(ctx, issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.CreateMandateResponse{Mandate: mandateToProto(created)}), nil
}

func (h *connectHandler) CancelStandingMandate(ctx context.Context, request *connect.Request[authorizationv1.CancelStandingMandateRequest]) (*connect.Response[authorizationv1.CancelStandingMandateResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	value, err := h.mandates.CancelStanding(ctx, request.Msg.GetMandateId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.CancelStandingMandateResponse{Mandate: mandateToProto(value)}), nil
}

func (h *connectHandler) ListMandates(ctx context.Context, request *connect.Request[authorizationv1.ListMandatesRequest]) (*connect.Response[authorizationv1.ListMandatesResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
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

func (h *connectHandler) RevokeMandate(ctx context.Context, request *connect.Request[authorizationv1.RevokeMandateRequest]) (*connect.Response[authorizationv1.RevokeMandateResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.mandates == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mandate service unavailable"))
	}
	value, err := h.mandates.Revoke(ctx, request.Msg.GetMandateId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.RevokeMandateResponse{Mandate: mandateToProto(value)}), nil
}

func (h *connectHandler) SetBudgetCaps(ctx context.Context, request *connect.Request[authorizationv1.SetBudgetCapsRequest]) (*connect.Response[authorizationv1.SetBudgetCapsResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	input := request.Msg.GetBudget()
	if input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("budget is required"))
	}
	value, err := h.budgets.SetCaps(ctx, budgetFromProto(input))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.SetBudgetCapsResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) SetGating(ctx context.Context, request *connect.Request[authorizationv1.SetGatingRequest]) (*connect.Response[authorizationv1.SetGatingResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	value, err := h.budgets.SetGating(ctx, request.Msg.GetBudgetId(), request.Msg.GetRequiresApproval())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.SetGatingResponse{Budget: budgetToProto(value)}), nil
}

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
	response := &authorizationv1.ListApprovalsResponse{Approvals: make([]*approvalv1.ApprovalRequest, 0, len(values))}
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

func approvalStatusFromProto(status approvalv1.ApprovalStatus, allowUnspecified bool) (approval.Status, error) {
	switch status {
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED:
		if allowUnspecified {
			return "", nil
		}
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_QUEUED:
		return approval.StatusQueued, nil
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_APPROVED:
		return approval.StatusApproved, nil
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED:
		return approval.StatusDeclined, nil
	case approvalv1.ApprovalStatus_APPROVAL_STATUS_EXPIRED:
		return approval.StatusExpired, nil
	}
	return "", errors.New("unsupported approval status")
}

func approvalToProto(value approval.Request) *approvalv1.ApprovalRequest {
	status := approvalv1.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED
	switch value.Status {
	case approval.StatusQueued:
		status = approvalv1.ApprovalStatus_APPROVAL_STATUS_QUEUED
	case approval.StatusApproved:
		status = approvalv1.ApprovalStatus_APPROVAL_STATUS_APPROVED
	case approval.StatusDeclined:
		status = approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED
	case approval.StatusExpired:
		status = approvalv1.ApprovalStatus_APPROVAL_STATUS_EXPIRED
	}
	msg := &approvalv1.ApprovalRequest{Id: value.ID, AuthorizationId: value.AuthorizationID, BookId: value.BookID, MandateId: value.MandateID, RequestingAgent: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Status: status, ResolverIdentity: value.ResolverIdentity, CreatedAt: timestamppb.New(value.CreatedAt), ExpiresAt: timestamppb.New(value.ExpiresAt)}
	if !value.ResolvedAt.IsZero() {
		msg.ResolvedAt = timestamppb.New(value.ResolvedAt)
	}
	return msg
}

func (h *connectHandler) FreezeBudget(ctx context.Context, request *connect.Request[authorizationv1.FreezeBudgetRequest]) (*connect.Response[authorizationv1.FreezeBudgetResponse], error) {
	return h.setBudgetFrozen(ctx, request, true)
}

func (h *connectHandler) UnfreezeBudget(ctx context.Context, request *connect.Request[authorizationv1.UnfreezeBudgetRequest]) (*connect.Response[authorizationv1.UnfreezeBudgetResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	value, err := h.budgets.SetFrozen(ctx, request.Msg.GetBudgetId(), false)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.UnfreezeBudgetResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) setBudgetFrozen(ctx context.Context, request *connect.Request[authorizationv1.FreezeBudgetRequest], frozen bool) (*connect.Response[authorizationv1.FreezeBudgetResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	value, err := h.budgets.SetFrozen(ctx, request.Msg.GetBudgetId(), frozen)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authorizationv1.FreezeBudgetResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) FreezeBook(ctx context.Context, request *connect.Request[authorizationv1.FreezeBookRequest]) (*connect.Response[authorizationv1.FreezeBookResponse], error) {
	value, err := h.setScopeFrozen(ctx, request, budget.FreezeScopeBook, request.Msg.GetBookId(), true)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&authorizationv1.FreezeBookResponse{BookId: value.ScopeID, Frozen: value.Frozen, UpdatedAt: timestamppb.New(value.UpdatedAt)}), nil
}

func (h *connectHandler) UnfreezeBook(ctx context.Context, request *connect.Request[authorizationv1.UnfreezeBookRequest]) (*connect.Response[authorizationv1.UnfreezeBookResponse], error) {
	value, err := h.setScopeFrozen(ctx, request, budget.FreezeScopeBook, request.Msg.GetBookId(), false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&authorizationv1.UnfreezeBookResponse{BookId: value.ScopeID, Frozen: value.Frozen, UpdatedAt: timestamppb.New(value.UpdatedAt)}), nil
}

func (h *connectHandler) FreezeAll(ctx context.Context, request *connect.Request[authorizationv1.FreezeAllRequest]) (*connect.Response[authorizationv1.FreezeAllResponse], error) {
	value, err := h.setScopeFrozen(ctx, request, budget.FreezeScopeScenario, "*", true)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&authorizationv1.FreezeAllResponse{Frozen: value.Frozen, UpdatedAt: timestamppb.New(value.UpdatedAt)}), nil
}

func (h *connectHandler) GetFreezeStatus(ctx context.Context, request *connect.Request[authorizationv1.GetFreezeStatusRequest]) (*connect.Response[authorizationv1.GetFreezeStatusResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	value, err := h.budgets.ScenarioFreezeStatus(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &authorizationv1.GetFreezeStatusResponse{Frozen: value.Frozen}
	if !value.UpdatedAt.IsZero() {
		response.UpdatedAt = timestamppb.New(value.UpdatedAt)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) UnfreezeAll(ctx context.Context, request *connect.Request[authorizationv1.UnfreezeAllRequest]) (*connect.Response[authorizationv1.UnfreezeAllResponse], error) {
	value, err := h.setScopeFrozen(ctx, request, budget.FreezeScopeScenario, "*", false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&authorizationv1.UnfreezeAllResponse{Frozen: value.Frozen, UpdatedAt: timestamppb.New(value.UpdatedAt)}), nil
}

func (h *connectHandler) setScopeFrozen(ctx context.Context, request mapHeader, scope budget.FreezeScope, id string, frozen bool) (budget.FreezeControl, error) {
	if err := h.authorize(ctx, request); err != nil {
		return budget.FreezeControl{}, err
	}
	if h.budgets == nil {
		return budget.FreezeControl{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	value, err := h.budgets.SetScopeFrozen(ctx, scope, id, frozen)
	if err != nil {
		return budget.FreezeControl{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return value, nil
}

func bookToProto(value book.Book) *bookv1.Book {
	return &bookv1.Book{Id: value.ID, Name: value.Name, BeneficiaryIdentity: value.BeneficiaryIdentity, CreatedAt: timestamppb.New(value.CreatedAt)}
}

func budgetFromProto(value *budgetv1.Budget) budget.Budget {
	return budget.Budget{ID: value.GetId(), BookID: value.GetBookId(), TotalCapMinor: value.GetTotalCapMinor(), PeriodicCapMinor: value.GetPeriodicCapMinor(), PerTransactionCapMinor: value.GetPerTransactionCapMinor(), Period: time.Duration(value.GetPeriodSeconds()) * time.Second, Currency: value.GetCurrency(), AllowedCounterparties: value.GetAllowedCounterparties(), DeniedCounterparties: value.GetDeniedCounterparties()}
}

func budgetToProto(value budget.Budget) *budgetv1.Budget {
	return &budgetv1.Budget{Id: value.ID, BookId: value.BookID, TotalCapMinor: value.TotalCapMinor, PeriodicCapMinor: value.PeriodicCapMinor, PerTransactionCapMinor: value.PerTransactionCapMinor, PeriodSeconds: int64(value.Period / time.Second), Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties, DeniedCounterparties: value.DeniedCounterparties, RequiresApproval: value.RequiresApproval, Frozen: value.Frozen}
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
	result := &settlementv1.Charge{Id: value.ID, AuthorizationId: value.AuthorizationID, MandateId: value.MandateID, InstrumentId: value.InstrumentID, Rail: value.Rail, IdempotencyKey: value.IdempotencyKey, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Outcome: outcome, ExternalId: value.ExternalID, ReceiptReference: value.ReceiptReference, Basis: value.Basis, Detail: value.Detail, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), RetainUntil: timestamppb.New(value.RetainUntil)}
	if !value.OccurredAt.IsZero() {
		result.OccurredAt = timestamppb.New(value.OccurredAt)
	}
	if value.Outcome == settlement.OutcomeSettled && !value.OccurredAt.IsZero() {
		result.SettledAt = timestamppb.New(value.OccurredAt)
	}
	return result
}
