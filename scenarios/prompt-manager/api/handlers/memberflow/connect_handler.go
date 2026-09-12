package memberflow

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	memberflowv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"

	"prompt-manager/handlers/transportbridge"
	graphdomain "prompt-manager/internal/graph"
	domain "prompt-manager/internal/memberflow"
)

type connectHandler struct {
	memberflowconnect.UnimplementedMemberflowServiceHandler
	legacy *domain.Handlers
	graph  *graphdomain.Handlers
}

func NewConnectMount(legacy *domain.Handlers, graph *graphdomain.Handlers) (string, http.Handler) {
	return memberflowconnect.NewMemberflowServiceHandler(&connectHandler{legacy: legacy, graph: graph})
}

func (h *connectHandler) GetMemberTopics(ctx context.Context, req *connect.Request[memberflowv1.MemberRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invoke(ctx, req.Header(), h.legacy.GetMember, http.MethodGet, memberPath(req.Msg.GetTeamId(), req.Msg.GetAgentId()), nil, memberVars(req.Msg.GetTeamId(), req.Msg.GetAgentId()))
}

func (h *connectHandler) UpdateMemberTopics(ctx context.Context, req *connect.Request[memberflowv1.UpdateMemberTopicsRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invoke(ctx, req.Header(), h.legacy.PutMember, http.MethodPut, memberPath(req.Msg.GetTeamId(), req.Msg.GetAgentId()), transportbridge.ValueBody(req.Msg.GetTopics()), memberVars(req.Msg.GetTeamId(), req.Msg.GetAgentId()))
}

func (h *connectHandler) GetTeamTopics(ctx context.Context, req *connect.Request[memberflowv1.TeamRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invoke(ctx, req.Header(), h.legacy.GetTeam, http.MethodGet, "/teams/"+url.PathEscape(req.Msg.GetTeamId())+"/topics", nil, map[string]string{"id": req.Msg.GetTeamId()})
}

func (h *connectHandler) GetTopicGraph(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetGraph, "/topics/graph")
}

func (h *connectHandler) GetRules(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetRules, "/topics/rules")
}

func (h *connectHandler) GetObjectives(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetObjectives, "/objectives")
}

func (h *connectHandler) GetOrientationCost(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetOrientationCost, "/orientation-cost")
}

func (h *connectHandler) GetInstruments(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetInstruments, "/instruments")
}

func (h *connectHandler) GetDrainStatus(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetDrainStatus, "/topics/drain-status")
}

func (h *connectHandler) GetOperatingModels(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.GetOperatingModels, "/operating-models")
}

func (h *connectHandler) ValidateOperatingModels(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.ValidateOperatingModelsHandler, "/operating-models/validate")
}

func (h *connectHandler) DiffOperatingModels(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.DiffOperatingModelsHandler, "/operating-models/diff")
}

func (h *connectHandler) GetOperatingModelCoverage(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.legacy.CoverageOperatingModelsHandler, "/operating-models/coverage")
}

func (h *connectHandler) GetOperatingMap(ctx context.Context, req *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invokeSimple(ctx, req.Header(), h.graph.GetOperatingMap, "/operating-models/map")
}

func invokeSimple(ctx context.Context, headers http.Header, handler http.HandlerFunc, path string) (*connect.Response[memberflowv1.JsonResponse], error) {
	return invoke(ctx, headers, handler, http.MethodGet, path, nil, nil)
}

func invoke(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, path string, body any, vars map[string]string) (*connect.Response[memberflowv1.JsonResponse], error) {
	result, err := transportbridge.Invoke(ctx, headers, handler, method, path, body, vars)
	if err != nil {
		return nil, err
	}
	value, err := transportbridge.DecodeValue(result.Body)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&memberflowv1.JsonResponse{Data: value}), nil
}

func memberPath(teamID, agentID string) string {
	return "/teams/" + url.PathEscape(teamID) + "/members/" + url.PathEscape(agentID) + "/topics"
}

func memberVars(teamID, agentID string) map[string]string {
	return map[string]string{"id": teamID, "agentId": agentID}
}
