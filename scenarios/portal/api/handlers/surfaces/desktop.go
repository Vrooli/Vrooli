package surfaces

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	"portal/internal/module"
)

var DesktopEndpoints = []module.EndpointDescriptor{
	{ID: "desktop_reconcile_open", Path: surfacesv1connect.DesktopSessionServiceReconcileOpenProcedure, Method: "POST", Summary: "Reconcile uncertain desktop admission", Category: "surfaces"},
	{ID: "desktop_list_admissions", Path: surfacesv1connect.DesktopSessionServiceListAdmissionsProcedure, Method: "POST", Summary: "List operator desktop admission history", Category: "surfaces"},
	{ID: "desktop_read_cleanup", Path: surfacesv1connect.DesktopSessionServiceReadCleanupProcedure, Method: "POST", Summary: "Read desktop cleanup evidence", Category: "surfaces"},
	{ID: "desktop_run_flow", Path: surfacesv1connect.DesktopSessionServiceRunFlowProcedure, Method: "POST", Summary: "Run an admitted desktop flow", Category: "surfaces"},
	{ID: "desktop_run_saved_flow", Path: surfacesv1connect.DesktopSessionServiceRunSavedFlowProcedure, Method: "POST", Summary: "RunSavedFlow for the admitted desktop", Category: "surfaces"},
	{ID: "desktop_get_saved_flow", Path: surfacesv1connect.DesktopSessionServiceGetSavedFlowProcedure, Method: "POST", Summary: "GetSavedFlow for the admitted desktop", Category: "surfaces"},
	{ID: "desktop_promote_flow", Path: surfacesv1connect.DesktopSessionServicePromoteFlowProcedure, Method: "POST", Summary: "PromoteFlow for the admitted desktop", Category: "surfaces"},
	{ID: "desktop_resolve", Path: surfacesv1connect.DesktopSessionServiceResolveProcedure, Method: "POST", Summary: "Resolve fields in an authorized desktop window", Category: "surfaces"},
	{ID: "desktop_applications", Path: surfacesv1connect.DesktopSessionServiceApplicationsProcedure, Method: "POST", Summary: "Discover authorized desktop applications", Category: "surfaces"},
	{ID: "desktop_open", Path: surfacesv1connect.DesktopSessionServiceOpenProcedure, Method: "POST", Summary: "Open an authorized desktop session", Category: "surfaces"},
	{ID: "desktop_observe", Path: surfacesv1connect.DesktopSessionServiceObserveProcedure, Method: "POST", Summary: "Observe an authorized desktop session", Category: "surfaces"},
	{ID: "desktop_act", Path: surfacesv1connect.DesktopSessionServiceActProcedure, Method: "POST", Summary: "Act in an authorized desktop session", Category: "surfaces"},
	{ID: "desktop_stop", Path: surfacesv1connect.DesktopSessionServiceStopProcedure, Method: "POST", Summary: "Stop an authorized desktop session", Category: "surfaces"},
}

type DesktopHandler struct {
	owner desktopv1connect.DesktopAccountServiceClient
}

func DesktopModule(socket string) module.Module {
	h := &DesktopHandler{}
	if strings.TrimSpace(socket) == "" {
		socket = defaultDesktopOwnerSocket()
	}
	if filepath.IsAbs(socket) {
		transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}, IdleConnTimeout: 15 * time.Second, MaxIdleConns: 4}
		h.owner = desktopv1connect.NewDesktopAccountServiceClient(&http.Client{Transport: transport, Timeout: 40 * time.Second}, "http://desktop-owner", connect.WithReadMaxBytes(33*1024*1024))
	}
	path, handler := surfacesv1connect.NewDesktopSessionServiceHandler(h, connect.WithReadMaxBytes(128*1024))
	private := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler.ServeHTTP(w, r)
	})
	return module.Module{Name: "desktop-sessions", Endpoints: DesktopEndpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: private}) }}
}

// defaultDesktopOwnerSocket keeps the local companion usable when the
// lifecycle manager does not inject an override. XDG_RUNTIME_DIR can point at
// the supervisor's uid in a managed process, so prefer the process uid when
// the advertised socket is absent; callers can still supply an explicit path.
func defaultDesktopOwnerSocket() string {
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" || !fileExists(filepath.Join(runtimeDir, "vrooli-desktop-owner", "desktop-owner.sock")) {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(runtimeDir, "vrooli-desktop-owner", "desktop-owner.sock")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Only the caller's bearer reaches destination admission. Cookies, actor
// headers, test-mode flags and arbitrary request metadata are not forwarded.
// Never retry a native operation after an ambiguous transport error.
func desktopForward[Request, Response any](ctx context.Context, r *connect.Request[Request], call func(context.Context, *connect.Request[Request]) (*connect.Response[Response], error)) (*connect.Response[Response], error) {
	return desktopForwardWithin(ctx, r, call, 10*time.Second)
}

func desktopForwardWithin[Request, Response any](ctx context.Context, r *connect.Request[Request], call func(context.Context, *connect.Request[Request]) (*connect.Response[Response], error), limit time.Duration) (*connect.Response[Response], error) {
	authorization := r.Header().Get("Authorization")
	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || token == "" || len(token) > 16*1024 || strings.TrimSpace(token) != token {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operator authentication required"))
	}
	if call == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("desktop owner unavailable"))
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	request := connect.NewRequest(r.Msg)
	request.Header().Set("Authorization", authorization)
	result, err := call(ctx, request)
	if err != nil {
		code := connect.CodeOf(err)
		if code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied && code != connect.CodeInvalidArgument {
			code = connect.CodeUnavailable
		}
		return nil, connect.NewError(code, errors.New("desktop operation refused or incomplete; do not blindly retry input"))
	}
	return connect.NewResponse(result.Msg), nil
}

func (h *DesktopHandler) Open(ctx context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenResponse], error)
	if h.owner != nil {
		call = h.owner.Open
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) Observe(ctx context.Context, r *connect.Request[desktopv1.OwnerObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error)
	if h.owner != nil {
		call = h.owner.Observe
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) Act(ctx context.Context, r *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error)
	if h.owner != nil {
		call = h.owner.Act
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) Stop(ctx context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.StopResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.StopResponse], error)
	if h.owner != nil {
		call = h.owner.Stop
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) Applications(ctx context.Context, r *connect.Request[desktopv1.OwnerApplicationsRequest]) (*connect.Response[desktopv1.ApplicationsResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerApplicationsRequest]) (*connect.Response[desktopv1.ApplicationsResponse], error)
	if h.owner != nil {
		call = h.owner.Applications
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) Resolve(ctx context.Context, r *connect.Request[desktopv1.OwnerResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error)
	if h.owner != nil {
		call = h.owner.Resolve
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) RunFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerRunFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error)
	if h.owner != nil {
		call = h.owner.RunFlow
	}
	return desktopForwardWithin(ctx, r, call, 35*time.Second)
}

func (h *DesktopHandler) PromoteFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerPromoteFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerPromoteFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error)
	if h.owner != nil {
		call = h.owner.PromoteFlow
	}
	return desktopForwardWithin(ctx, r, call, 10*time.Second)
}

func (h *DesktopHandler) GetSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error)
	if h.owner != nil {
		call = h.owner.GetSavedFlow
	}
	return desktopForwardWithin(ctx, r, call, 10*time.Second)
}

func (h *DesktopHandler) RunSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunSavedFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerRunSavedFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error)
	if h.owner != nil {
		call = h.owner.RunSavedFlow
	}
	return desktopForwardWithin(ctx, r, call, 35*time.Second)
}

func (h *DesktopHandler) ReadCleanup(ctx context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.OwnerCleanupResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.OwnerCleanupResponse], error)
	if h.owner != nil {
		call = h.owner.ReadCleanup
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) ListAdmissions(ctx context.Context, r *connect.Request[desktopv1.OwnerListAdmissionsRequest]) (*connect.Response[desktopv1.OwnerListAdmissionsResponse], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerListAdmissionsRequest]) (*connect.Response[desktopv1.OwnerListAdmissionsResponse], error)
	if h.owner != nil {
		call = h.owner.ListAdmissions
	}
	return desktopForward(ctx, r, call)
}

func (h *DesktopHandler) ReconcileOpen(ctx context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenDisposition], error) {
	var call func(context.Context, *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenDisposition], error)
	if h.owner != nil {
		call = h.owner.ReconcileOpen
	}
	return desktopForward(ctx, r, call)
}
