package telemetry

import (
	"connectrpc.com/connect"
	"context"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry/telemetry_v1connect"
	"program-runtime/internal/module"
	internaltelemetry "program-runtime/internal/telemetry"
)

type handler struct {
	telemetryconnect.UnimplementedTelemetryServiceHandler
	store *internaltelemetry.Store
}

func Module(store *internaltelemetry.Store) module.Module {
	return module.Module{Name: "telemetry", Mount: func(r *mux.Router) {
		path, h := telemetryconnect.NewTelemetryServiceHandler(&handler{store: store})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}
func (h *handler) ListEvents(_ context.Context, req *connect.Request[telemetryv1.ListEventsRequest]) (*connect.Response[telemetryv1.ListEventsResponse], error) {
	return connect.NewResponse(&telemetryv1.ListEventsResponse{Events: h.store.List(req.Msg.SessionId, req.Msg.Kind)}), nil
}
