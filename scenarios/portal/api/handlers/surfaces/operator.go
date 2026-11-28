package surfaces

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	accounts "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"net/http"
	"portal/internal/module"
	"time"
)

type OperatorHandler struct {
	resolve func(context.Context, string) (string, error)
	client  connect.HTTPClient
}

var OperatorEndpoints = []module.EndpointDescriptor{
	{ID: "operator_login", Path: surfacesv1connect.OperatorSessionServiceLoginProcedure, Method: "POST", Summary: "Operator login", Category: "identity"},
	{ID: "operator_register", Path: surfacesv1connect.OperatorSessionServiceRegisterProcedure, Method: "POST", Summary: "Operator register", Category: "identity"},
	{ID: "operator_refresh", Path: surfacesv1connect.OperatorSessionServiceRefreshProcedure, Method: "POST", Summary: "Operator refresh", Category: "identity"},
	{ID: "operator_logout", Path: surfacesv1connect.OperatorSessionServiceLogoutProcedure, Method: "POST", Summary: "Operator logout", Category: "identity"},
}

func OperatorModule() module.Module {
	return operatorModule(&OperatorHandler{resolve: discovery.ResolveScenarioURLDefault, client: &http.Client{Timeout: 10 * time.Second}})
}
func operatorModule(h *OperatorHandler) module.Module {
	path, handler := surfacesv1connect.NewOperatorSessionServiceHandler(h, connect.WithReadMaxBytes(16*1024))
	private := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler.ServeHTTP(w, r)
	})
	return module.Module{Name: "operator-sessions", Endpoints: OperatorEndpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: private}) }}
}
func operatorForward[Req, Resp any](ctx context.Context, h *OperatorHandler, request *Req, call func(context.Context, accounts_v1connect.AccountsServiceClient, *connect.Request[Req]) (*connect.Response[Resp], error)) (*connect.Response[Resp], error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	base, err := h.resolve(ctx, "scenario-authenticator")
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("account service unavailable"))
	}
	client := accounts_v1connect.NewAccountsServiceClient(h.client, base, connect.WithReadMaxBytes(64*1024))
	result, err := call(ctx, client, connect.NewRequest(request))
	if err != nil {
		code := connect.CodeOf(err)
		if code != connect.CodeUnauthenticated && code != connect.CodeInvalidArgument && code != connect.CodeAlreadyExists && code != connect.CodeResourceExhausted {
			code = connect.CodeUnavailable
		}
		return nil, connect.NewError(code, errors.New("account request could not be completed"))
	}
	return connect.NewResponse(result.Msg), nil
}
func (h *OperatorHandler) Login(ctx context.Context, r *connect.Request[accounts.LoginRequest]) (*connect.Response[accounts.LoginResponse], error) {
	return operatorForward(ctx, h, r.Msg, func(ctx context.Context, c accounts_v1connect.AccountsServiceClient, r *connect.Request[accounts.LoginRequest]) (*connect.Response[accounts.LoginResponse], error) {
		return c.Login(ctx, r)
	})
}
func (h *OperatorHandler) Register(ctx context.Context, r *connect.Request[accounts.RegisterRequest]) (*connect.Response[accounts.RegisterResponse], error) {
	return operatorForward(ctx, h, r.Msg, func(ctx context.Context, c accounts_v1connect.AccountsServiceClient, r *connect.Request[accounts.RegisterRequest]) (*connect.Response[accounts.RegisterResponse], error) {
		return c.Register(ctx, r)
	})
}
func (h *OperatorHandler) Refresh(ctx context.Context, r *connect.Request[accounts.RefreshRequest]) (*connect.Response[accounts.RefreshResponse], error) {
	return operatorForward(ctx, h, r.Msg, func(ctx context.Context, c accounts_v1connect.AccountsServiceClient, r *connect.Request[accounts.RefreshRequest]) (*connect.Response[accounts.RefreshResponse], error) {
		return c.Refresh(ctx, r)
	})
}
func (h *OperatorHandler) Logout(ctx context.Context, r *connect.Request[accounts.LogoutRequest]) (*connect.Response[accounts.LogoutResponse], error) {
	return operatorForward(ctx, h, r.Msg, func(ctx context.Context, c accounts_v1connect.AccountsServiceClient, r *connect.Request[accounts.LogoutRequest]) (*connect.Response[accounts.LogoutResponse], error) {
		return c.Logout(ctx, r)
	})
}
