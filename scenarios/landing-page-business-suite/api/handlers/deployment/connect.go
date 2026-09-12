package deployment

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

// ConnectHandler exposes deployment readiness through the generated contract.
// It deliberately delegates to CheckReadiness so the REST compatibility route
// cannot drift from the typed transport.
type ConnectHandler struct{ deps Dependencies }

func NewConnectHandler(deps Dependencies) *ConnectHandler { return &ConnectHandler{deps: deps} }

func (h *ConnectHandler) CheckReadiness(ctx context.Context, request *connect.Request[lpbsv1.CheckDeploymentReadinessRequest]) (*connect.Response[lpbsv1.CheckDeploymentReadinessResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("readiness request is required"))
	}
	response := CheckReadiness(ctx, h.deps, Request{
		AppKey:        request.Msg.GetAppKey(),
		RemoteProfile: request.Msg.GetRemoteProfile(),
		Channel:       request.Msg.GetChannel(),
	})
	return connect.NewResponse(protoResponse(response)), nil
}

func protoResponse(response Response) *lpbsv1.CheckDeploymentReadinessResponse {
	result := &lpbsv1.CheckDeploymentReadinessResponse{Ready: response.Ready, Error: response.Error}
	for _, gate := range response.Gates {
		result.Gates = append(result.Gates, &lpbsv1.DeploymentReadinessGate{
			Name: gate.Name, Ready: gate.Ready, Message: gate.Message,
		})
	}
	return result
}

// RegisterConnectRoutes mounts the typed deployment readiness endpoint with
// the same admin-or-service authorization policy as the legacy REST route.
func RegisterConnectRoutes(router *mux.Router, deps Dependencies, requireAdminOrService func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewDeploymentServiceHandler(NewConnectHandler(deps))
	router.Handle(lpbsconnect.DeploymentServiceCheckReadinessProcedure, requireAdminOrService(generated.ServeHTTP)).Methods(http.MethodPost)
}

var _ lpbsconnect.DeploymentServiceHandler = (*ConnectHandler)(nil)
