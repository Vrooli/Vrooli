package contextcapture

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture/contextcapture_v1connect"
	domain "portal/internal/contextcapture"
	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "context_render", Path: rpc.ContextCaptureServiceRenderProcedure, Method: "POST", Summary: "Render selected context with annotations", Category: "context"},
	{ID: "context_cancel_import", Path: rpc.ContextCaptureServiceCancelImportProcedure, Method: "POST", Summary: "Cancel context import and remove pixels", Category: "context"},
	{ID: "context_reconcile_import", Path: rpc.ContextCaptureServiceReconcileImportProcedure, Method: "POST", Summary: "Resolve context import intent", Category: "context"},
	{ID: "context_import", Path: rpc.ContextCaptureServiceImportProcedure, Method: "POST", Summary: "Import owned context", Category: "context"},
	{ID: "context_read", Path: rpc.ContextCaptureServiceReadProcedure, Method: "POST", Summary: "Read owned context", Category: "context"},
	{ID: "context_delete", Path: rpc.ContextCaptureServiceDeleteProcedure, Method: "POST", Summary: "Delete owned context", Category: "context"},
}

func Schema() string { return domain.Schema() }
func Module(h *Handler) module.Module {
	path, handler := rpc.NewContextCaptureServiceHandler(h, connect.WithReadMaxBytes(45*1024*1024))
	private := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler.ServeHTTP(w, r)
	})
	return module.Module{Name: "contextcapture", Endpoints: Endpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: private}) }}
}
