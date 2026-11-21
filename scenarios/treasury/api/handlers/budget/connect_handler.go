package budget

import (
	"context"
	"errors"
	"net/http"
	"time"

	domain "treasury/internal/budget"
	"treasury/internal/module"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	budgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget/budget_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PolicyService interface {
	SetCaps(context.Context, domain.Budget) (domain.Budget, error)
	SetGating(context.Context, string, bool) (domain.Budget, error)
	SetFrozen(context.Context, string, bool) (domain.Budget, error)
	SetScopeFrozen(context.Context, domain.FreezeScope, string, bool) (domain.FreezeControl, error)
	ScenarioFreezeStatus(context.Context) (domain.FreezeControl, error)
}

type connectHandler struct {
	authorizer operatorauth.Authorizer
	budgets    PolicyService
}

func (h *connectHandler) authorize(ctx context.Context, request interface{ Header() http.Header }) error {
	if h.authorizer == nil {
		return connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	_, err := h.authorizer.Authorize(ctx, request.Header())
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

func (h *connectHandler) SetBudgetCaps(ctx context.Context, request *connect.Request[budgetv1.SetBudgetCapsRequest]) (*connect.Response[budgetv1.SetBudgetCapsResponse], error) {
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
	return connect.NewResponse(&budgetv1.SetBudgetCapsResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) SetGating(ctx context.Context, request *connect.Request[budgetv1.SetGatingRequest]) (*connect.Response[budgetv1.SetGatingResponse], error) {
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
	return connect.NewResponse(&budgetv1.SetGatingResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) FreezeBudget(ctx context.Context, request *connect.Request[budgetv1.FreezeBudgetRequest]) (*connect.Response[budgetv1.FreezeBudgetResponse], error) {
	return h.setFrozen(ctx, request, true)
}

func (h *connectHandler) UnfreezeBudget(ctx context.Context, request *connect.Request[budgetv1.UnfreezeBudgetRequest]) (*connect.Response[budgetv1.UnfreezeBudgetResponse], error) {
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
	return connect.NewResponse(&budgetv1.UnfreezeBudgetResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) setFrozen(ctx context.Context, request *connect.Request[budgetv1.FreezeBudgetRequest], frozen bool) (*connect.Response[budgetv1.FreezeBudgetResponse], error) {
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
	return connect.NewResponse(&budgetv1.FreezeBudgetResponse{Budget: budgetToProto(value)}), nil
}

func (h *connectHandler) FreezeBook(ctx context.Context, request *connect.Request[budgetv1.FreezeBookRequest]) (*connect.Response[budgetv1.FreezeBookResponse], error) {
	control, err := h.setScopeFrozen(ctx, request, domain.FreezeScopeBook, request.Msg.GetBookId(), true)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&budgetv1.FreezeBookResponse{BookId: control.ScopeID, Frozen: control.Frozen, UpdatedAt: timestamppb.New(control.UpdatedAt)}), nil
}

func (h *connectHandler) UnfreezeBook(ctx context.Context, request *connect.Request[budgetv1.UnfreezeBookRequest]) (*connect.Response[budgetv1.UnfreezeBookResponse], error) {
	control, err := h.setScopeFrozen(ctx, request, domain.FreezeScopeBook, request.Msg.GetBookId(), false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&budgetv1.UnfreezeBookResponse{BookId: control.ScopeID, Frozen: control.Frozen, UpdatedAt: timestamppb.New(control.UpdatedAt)}), nil
}

func (h *connectHandler) FreezeAll(ctx context.Context, request *connect.Request[budgetv1.FreezeAllRequest]) (*connect.Response[budgetv1.FreezeAllResponse], error) {
	control, err := h.setScenarioFrozen(ctx, request, true)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&budgetv1.FreezeAllResponse{Frozen: control.Frozen, UpdatedAt: timestamppb.New(control.UpdatedAt)}), nil
}

func (h *connectHandler) UnfreezeAll(ctx context.Context, request *connect.Request[budgetv1.UnfreezeAllRequest]) (*connect.Response[budgetv1.UnfreezeAllResponse], error) {
	control, err := h.setScenarioFrozen(ctx, request, false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&budgetv1.UnfreezeAllResponse{Frozen: control.Frozen, UpdatedAt: timestamppb.New(control.UpdatedAt)}), nil
}

func (h *connectHandler) GetFreezeStatus(ctx context.Context, request *connect.Request[budgetv1.GetFreezeStatusRequest]) (*connect.Response[budgetv1.GetFreezeStatusResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.budgets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	control, err := h.budgets.ScenarioFreezeStatus(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &budgetv1.GetFreezeStatusResponse{Frozen: control.Frozen}
	if !control.UpdatedAt.IsZero() {
		response.UpdatedAt = timestamppb.New(control.UpdatedAt)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) setScopeFrozen(ctx context.Context, request interface{ Header() http.Header }, scope domain.FreezeScope, id string, frozen bool) (domain.FreezeControl, error) {
	if err := h.authorize(ctx, request); err != nil {
		return domain.FreezeControl{}, err
	}
	if h.budgets == nil {
		return domain.FreezeControl{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	control, err := h.budgets.SetScopeFrozen(ctx, scope, id, frozen)
	if err != nil {
		return domain.FreezeControl{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return control, nil
}

func (h *connectHandler) setScenarioFrozen(ctx context.Context, request interface{ Header() http.Header }, frozen bool) (domain.FreezeControl, error) {
	if err := h.authorize(ctx, request); err != nil {
		return domain.FreezeControl{}, err
	}
	if h.budgets == nil {
		return domain.FreezeControl{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("budget service unavailable"))
	}
	control, err := h.budgets.SetScopeFrozen(ctx, domain.FreezeScopeScenario, "*", frozen)
	if err != nil {
		return domain.FreezeControl{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return control, nil
}

func budgetFromProto(value *budgetv1.Budget) domain.Budget {
	return domain.Budget{ID: value.GetId(), BookID: value.GetBookId(), TotalCapMinor: value.GetTotalCapMinor(), PeriodicCapMinor: value.GetPeriodicCapMinor(), PerTransactionCapMinor: value.GetPerTransactionCapMinor(), Period: time.Duration(value.GetPeriodSeconds()) * time.Second, Currency: value.GetCurrency(), AllowedCounterparties: value.GetAllowedCounterparties(), DeniedCounterparties: value.GetDeniedCounterparties()}
}

func budgetToProto(value domain.Budget) *budgetv1.Budget {
	return &budgetv1.Budget{Id: value.ID, BookId: value.BookID, TotalCapMinor: value.TotalCapMinor, PeriodicCapMinor: value.PeriodicCapMinor, PerTransactionCapMinor: value.PerTransactionCapMinor, PeriodSeconds: int64(value.Period / time.Second), Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties, DeniedCounterparties: value.DeniedCounterparties, RequiresApproval: value.RequiresApproval, Frozen: value.Frozen}
}

func Module(authorizer operatorauth.Authorizer, budgets PolicyService) module.Module {
	path, handler := budgetconnect.NewBudgetAdminServiceHandler(NewConnectHandler(authorizer, budgets))
	return module.Module{Name: "budgetadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func NewConnectHandler(authorizer operatorauth.Authorizer, budgets PolicyService) budgetconnect.BudgetAdminServiceHandler {
	return &connectHandler{authorizer: authorizer, budgets: budgets}
}
