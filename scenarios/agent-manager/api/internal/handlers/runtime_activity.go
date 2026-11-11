package handlers

import (
	"context"
	"errors"
	"net/http"

	"agent-manager/internal/runtimeactivity"
	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RuntimeActivityReader is ordinary telemetry, independent of maintenance gates.
type RuntimeActivityReader interface {
	Read(context.Context) runtimeactivity.Snapshot
}

func WithRuntimeActivityReader(reader RuntimeActivityReader) HandlerOption {
	return func(h *Handler) { h.runtimeActivity = reader }
}
func (h *Handler) runtimeActivityView(ctx context.Context) (*apipb.GetRuntimeActivityResponse, error) {
	if h == nil || h.runtimeActivity == nil {
		return nil, errors.New("physical runtime activity observer unavailable")
	}
	snapshot := h.runtimeActivity.Read(ctx)
	response := &apipb.GetRuntimeActivityResponse{Complete: snapshot.Complete, ExecutingRuns: snapshot.ExecutingRuns, ObservedExecutingRuns: snapshot.ObservedExecutingRuns, LogicalQueuedRuns: snapshot.LogicalQueuedRuns, UnknownReasons: snapshot.Unknown, Reason: snapshot.Reason()}
	if !snapshot.ObservedAt.IsZero() {
		response.ObservedAt = timestamppb.New(snapshot.ObservedAt)
	}
	return response, nil
}
func (h *Handler) GetRuntimeActivity(w http.ResponseWriter, r *http.Request) {
	response, err := h.runtimeActivityView(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeProtoJSON(w, http.StatusOK, response)
}
func (h *AgentManagerConnectHandler) GetRuntimeActivity(ctx context.Context, _ *connect.Request[apipb.GetRuntimeActivityRequest]) (*connect.Response[apipb.GetRuntimeActivityResponse], error) {
	response, err := h.h.runtimeActivityView(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(response), nil
}
