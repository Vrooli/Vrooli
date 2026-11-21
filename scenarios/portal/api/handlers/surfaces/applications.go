package surfaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"portal/internal/module"
	"portal/internal/surfaces"
)

var ApplicationEndpoints = []module.EndpointDescriptor{
	{ID: "applications_list", Path: rpc.ApplicationCatalogServiceListApplicationsProcedure, Method: "POST", Summary: "List Vrooli applications", Description: "Project control-plane runtime inventory without assuming TV qualification.", Category: "surfaces"},
	{ID: "applications_resolve", Path: rpc.ApplicationCatalogServiceResolveApplicationProcedure, Method: "POST", Summary: "Resolve a running application", Description: "Resolve an existing application browser URL; application access remains with its owner.", Category: "surfaces"},
	{ID: "applications_lifecycle", Path: rpc.ApplicationLifecycleServiceActProcedure, Method: "POST", Summary: "Run an operator-authorized application action", Description: "Run an explicit lifecycle action through the control plane.", Category: "surfaces"},
	{ID: "application_icon", Path: "/api/v1/applications/{scenario}/icon", Method: "GET", Summary: "Read applied application artwork", Description: "Read only the bounded applied public PNG for an existing scenario.", Category: "surfaces", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Browser-native PNG artwork is consumed directly by image clients."}},
	{ID: "application_proxy_target", Path: "/api/v1/applications/{scenario}/proxy-target", Method: "GET", Summary: "Resolve an application gateway target", Description: "Discover numeric ports for a known running application; never starts it.", Category: "surfaces", RESTException: &module.RESTException{Reason: module.RESTReasonThirdPartyShape, Note: "The shared application gateway consumes this JSON target contract."}},
}

type applicationHandler struct{ catalog *surfaces.Applications }
type applicationLifecycleHandler struct {
	catalog  *surfaces.Applications
	verifier authn.TokenVerifier
}

func ApplicationsModule(catalog *surfaces.Applications) module.Module {
	path, handler := rpc.NewApplicationCatalogServiceHandler(&applicationHandler{catalog: catalog}, connect.WithReadMaxBytes(16<<10))
	lifecyclePath, lifecycle := rpc.NewApplicationLifecycleServiceHandler(&applicationLifecycleHandler{catalog: catalog, verifier: authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"))})}, connect.WithReadMaxBytes(16<<10))
	return module.Module{Name: "applications", Endpoints: ApplicationEndpoints, Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: discovery.ExternalHostMiddleware(handler)})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: lifecyclePath, Handler: lifecycle})
		r.HandleFunc("/api/v1/applications/{scenario}/proxy-target", func(w http.ResponseWriter, req *http.Request) {
			value, err := catalog.ProxyTarget(req.Context(), mux.Vars(req)["scenario"])
			if err != nil {
				http.Error(w, "Application gateway unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(value)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/applications/{scenario}/icon", func(w http.ResponseWriter, req *http.Request) {
			if catalog.Branding == nil {
				http.NotFound(w, req)
				return
			}
			data, digest, err := catalog.Branding.Icon(req.Context(), mux.Vars(req)["scenario"])
			if err != nil {
				http.NotFound(w, req)
				return
			}
			etag := `"` + digest + `"`
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "private, max-age=300")
			w.Header().Set("ETag", etag)
			if req.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			_, _ = w.Write(data)
		}).Methods(http.MethodGet)
	}}
}

func (h *applicationLifecycleHandler) Act(ctx context.Context, req *connect.Request[wire.ApplicationLifecycleRequest]) (*connect.Response[wire.ApplicationLifecycleResponse], error) {
	values := req.Header().Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") || len(values[0]) > 16384 || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operator authentication required"))
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operator authentication required"))
	}
	principal, err := h.verifier.Verify(ctx, token)
	if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !principal.ExpiresAt.After(time.Now()) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operator authentication required"))
	}
	action := ""
	switch req.Msg.Action {
	case wire.ApplicationLifecycleAction_APPLICATION_LIFECYCLE_ACTION_START:
		action = "start"
	case wire.ApplicationLifecycleAction_APPLICATION_LIFECYCLE_ACTION_RESTART:
		action = "restart"
	case wire.ApplicationLifecycleAction_APPLICATION_LIFECYCLE_ACTION_STOP:
		action = "stop"
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid application action"))
	}
	ctx = identity.WithPrincipal(ctx, principal)
	if _, err := authn.RequireOperation(ctx, "vrooli:write", "scenario."+action, req.Msg.Scenario); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("operator lacks lifecycle authority"))
	}
	if h.catalog == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("application lifecycle unavailable"))
	}
	app, err := h.catalog.Act(ctx, req.Msg.Scenario, action)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(err.Error()))
	}
	verified := action == "stop" && (app.RuntimeState == "available" || app.RuntimeState == "stopped") || action != "stop" && app.RuntimeState == "running"
	outcome := "unverified"
	if verified { outcome = "completed" }
	response := connect.NewResponse(&wire.ApplicationLifecycleResponse{Application: applicationProto(app), Outcome: outcome, ObservedAt: timestamppb.Now()})
	response.Header().Set("Cache-Control", "no-store")
	return response, nil
}

func applicationProto(app surfaces.Application) *wire.Application {
	return &wire.Application{Scenario: app.Scenario, Label: app.Label, Description: app.Description, RuntimeState: app.RuntimeState, TvQualification: "unverified", IconPath: app.IconPath}
}

func (h *applicationHandler) ListApplications(ctx context.Context, _ *connect.Request[wire.ListApplicationsRequest]) (*connect.Response[wire.ListApplicationsResponse], error) {
	apps, err := h.catalog.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	response := &wire.ListApplicationsResponse{ObservedAt: timestamppb.New(time.Now().UTC())}
	for _, app := range apps {
		response.Applications = append(response.Applications, applicationProto(app))
	}
	return connect.NewResponse(response), nil
}

func (h *applicationHandler) ResolveApplication(ctx context.Context, req *connect.Request[wire.ResolveApplicationRequest]) (*connect.Response[wire.ResolveApplicationResponse], error) {
	var app surfaces.Application
	var address string
	var err error
	if req.Msg.UseGateway {
		app, address, err = h.catalog.ResolveGateway(ctx, req.Msg.Scenario)
	} else {
		app, address, err = h.catalog.Resolve(ctx, req.Msg.Scenario, discovery.ExternalHostFromContext(ctx))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&wire.ResolveApplicationResponse{Application: applicationProto(app), Url: address, ObservedAt: timestamppb.New(time.Now().UTC())}), nil
}
