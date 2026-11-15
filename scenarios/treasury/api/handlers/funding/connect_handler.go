package funding

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
	"treasury/internal/identity"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
	"treasury/internal/receiptfunding"
	"treasury/internal/walletops"
)

type FundingIntents interface {
	CreateFundingIntent(context.Context, string, *fundingv1.CreateFundingIntentRequest) (*fundingv1.FundingIntent, error)
	GetFundingIntent(context.Context, string, string) (*fundingv1.FundingIntent, error)
	ConfirmFundingTransfer(context.Context, string, *fundingv1.ConfirmFundingTransferRequest) (*fundingv1.FundingIntent, error)
	ListFundingIntents(context.Context, string, string) ([]*fundingv1.FundingIntent, error)
}

type connectHandler struct {
	authorizer operatorauth.Authorizer
	intents    FundingIntents
	walletOps  *walletops.Service
	receipts   *receiptfunding.Service
}

func NewConnectHandler(authorizer operatorauth.Authorizer, intents FundingIntents, walletOperations *walletops.Service, receipts *receiptfunding.Service) fundingconnect.FundingAdminServiceHandler {
	return &connectHandler{authorizer: authorizer, intents: intents, walletOps: walletOperations, receipts: receipts}
}

type agentWalletHandler struct {
	verifier identity.Verifier
	wallets  *walletops.Service
}

func NewAgentWalletHandler(verifier identity.Verifier, wallets *walletops.Service) fundingconnect.FundingAgentServiceHandler {
	return &agentWalletHandler{verifier: verifier, wallets: wallets}
}

func (h *connectHandler) authorize(ctx context.Context, request interface{ Header() http.Header }) (operatorauth.Identity, error) {
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

func (h *connectHandler) CreateFundingIntent(ctx context.Context, request *connect.Request[fundingv1.CreateFundingIntentRequest]) (*connect.Response[fundingv1.FundingIntentResponse], error) {
	identity, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.intents == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("funding intent service unavailable"))
	}
	value, err := h.intents.CreateFundingIntent(ctx, identity.Subject, request.Msg)
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingv1.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) GetFundingIntent(ctx context.Context, request *connect.Request[fundingv1.FundingIntentRequest]) (*connect.Response[fundingv1.FundingIntentResponse], error) {
	identity, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.intents == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("funding intent service unavailable"))
	}
	value, err := h.intents.GetFundingIntent(ctx, identity.Subject, request.Msg.GetId())
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingv1.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) ConfirmFundingTransfer(ctx context.Context, request *connect.Request[fundingv1.ConfirmFundingTransferRequest]) (*connect.Response[fundingv1.FundingIntentResponse], error) {
	identity, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.intents == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("funding intent service unavailable"))
	}
	value, err := h.intents.ConfirmFundingTransfer(ctx, identity.Subject, request.Msg)
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingv1.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) ListFundingIntents(ctx context.Context, request *connect.Request[fundingv1.ListFundingIntentsRequest]) (*connect.Response[fundingv1.ListFundingIntentsResponse], error) {
	identity, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.intents == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("funding intent service unavailable"))
	}
	values, err := h.intents.ListFundingIntents(ctx, identity.Subject, request.Msg.GetBudgetId())
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingv1.ListFundingIntentsResponse{Intents: values}), nil
}

func Module(authorizer operatorauth.Authorizer, intents FundingIntents, walletOperations *walletops.Service, receipts *receiptfunding.Service, verifier identity.Verifier) module.Module {
	adminPath, adminHandler := fundingconnect.NewFundingAdminServiceHandler(NewConnectHandler(authorizer, intents, walletOperations, receipts))
	agentPath, agentHandler := fundingconnect.NewFundingAgentServiceHandler(NewAgentWalletHandler(verifier, walletOperations))
	return module.Module{Name: "fundingadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: adminPath, Handler: adminHandler})
		connectx.RegisterServices(router, connectx.ServiceMount{Path: agentPath, Handler: agentHandler})
	}, Endpoints: append(append([]module.EndpointDescriptor(nil), Endpoints...), AgentEndpoints...)}
}
