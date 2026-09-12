package aisearch

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	aisearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch"
	aisearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch/aisearch_v1connect"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/aisearch"
)

type connectHandler struct {
	aisearchconnect.UnimplementedAISearchServiceHandler
	legacy *domain.Handlers
}

// NewConnectMount exposes semantic search and index reconciliation through the
// generated contract while reusing the established domain behavior.
func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return aisearchconnect.NewAISearchServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) SearchSkills(ctx context.Context, req *connect.Request[aisearchv1.SearchSkillsRequest]) (*connect.Response[aisearchv1.SearchSkillsResponse], error) {
	return invokeAI(ctx, req.Header(), h.legacy.Search, http.MethodPost, "/search/ai", req.Msg, &aisearchv1.SearchSkillsResponse{})
}

func (h *connectHandler) SearchAgents(ctx context.Context, req *connect.Request[aisearchv1.SearchAgentsRequest]) (*connect.Response[aisearchv1.SearchAgentsResponse], error) {
	return invokeAI(ctx, req.Header(), h.legacy.SearchAgents, http.MethodPost, "/search/agents/ai", req.Msg, &aisearchv1.SearchAgentsResponse{})
}

func (h *connectHandler) SearchActions(ctx context.Context, req *connect.Request[aisearchv1.SearchActionsRequest]) (*connect.Response[aisearchv1.SearchActionsResponse], error) {
	return invokeAI(ctx, req.Header(), h.legacy.SearchActions, http.MethodPost, "/search/actions/ai", req.Msg, &aisearchv1.SearchActionsResponse{})
}

func (h *connectHandler) SearchTeams(ctx context.Context, req *connect.Request[aisearchv1.SearchTeamsRequest]) (*connect.Response[aisearchv1.SearchTeamsResponse], error) {
	return invokeAI(ctx, req.Header(), h.legacy.SearchTeams, http.MethodPost, "/search/teams/ai", req.Msg, &aisearchv1.SearchTeamsResponse{})
}

func (h *connectHandler) GetStatus(ctx context.Context, req *connect.Request[aisearchv1.GetStatusRequest]) (*connect.Response[aisearchv1.GetStatusResponse], error) {
	return invokeAI(ctx, req.Header(), h.legacy.Status, http.MethodGet, "/search/ai/status", nil, &aisearchv1.GetStatusResponse{})
}

func (h *connectHandler) Reconcile(ctx context.Context, req *connect.Request[aisearchv1.ReconcileRequest]) (*connect.Response[aisearchv1.ReconcileResponse], error) {
	q := url.Values{}
	if req.Msg.GetCollection() != "" && req.Msg.GetCollection() != "all" {
		q.Set("collection", req.Msg.GetCollection())
	}
	if req.Msg.GetDryRun() {
		q.Set("dry_run", "true")
	}
	path := "/search/ai/reconcile"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Reconcile, http.MethodPost, path, nil, nil)
	if err != nil {
		return nil, err
	}
	out := &aisearchv1.ReconcileResponse{DryRun: req.Msg.GetDryRun()}
	if req.Msg.GetDryRun() {
		if err := transportbridge.Decode(result.Body, out); err != nil {
			return nil, err
		}
	} else {
		out.Status = &aisearchv1.ReconcileStatus{}
		if err := transportbridge.Decode(result.Body, out.Status); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetReconcileStatus(ctx context.Context, req *connect.Request[aisearchv1.GetReconcileStatusRequest]) (*connect.Response[aisearchv1.ReconcileStatus], error) {
	return invokeAI(ctx, req.Header(), h.legacy.ReconcileStatus, http.MethodGet, "/search/ai/reconcile/status", nil, &aisearchv1.ReconcileStatus{})
}

func (h *connectHandler) CancelReconcile(ctx context.Context, req *connect.Request[aisearchv1.CancelReconcileRequest]) (*connect.Response[aisearchv1.ReconcileStatus], error) {
	return invokeAI(ctx, req.Header(), h.legacy.CancelReconcile, http.MethodPost, "/search/ai/reconcile/cancel", nil, &aisearchv1.ReconcileStatus{})
}

func invokeAI[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, path string, input proto.Message, out *T) (*connect.Response[T], error) {
	return transportbridge.InvokeProto(ctx, headers, handler, method, path, input, out)
}
