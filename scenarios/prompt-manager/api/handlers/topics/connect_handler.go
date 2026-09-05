package topics

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	topicsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/topics"
	topicsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/topics/topics_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/topics"
)

type connectHandler struct {
	topicsconnect.UnimplementedTopicsServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return topicsconnect.NewTopicsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListTopics(ctx context.Context, req *connect.Request[topicsv1.ListTopicsRequest]) (*connect.Response[topicsv1.ListTopicsResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.List, http.MethodGet, "/topics", nil, nil, "topics", &topicsv1.ListTopicsResponse{})
}

func (h *connectHandler) ListTopicTree(ctx context.Context, req *connect.Request[topicsv1.ListTopicTreeRequest]) (*connect.Response[topicsv1.ListTopicsResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.List, http.MethodGet, "/topics", nil, nil, "topics", &topicsv1.ListTopicsResponse{})
}

func (h *connectHandler) GetTopic(ctx context.Context, req *connect.Request[topicsv1.GetTopicRequest]) (*connect.Response[topicsv1.Topic], error) {
	return topicCall(ctx, req.Header(), h.legacy.Get, http.MethodGet, req.Msg.GetId(), "", nil, &topicsv1.Topic{})
}

func (h *connectHandler) CreateTopic(ctx context.Context, req *connect.Request[topicsv1.CreateTopicRequest]) (*connect.Response[topicsv1.Topic], error) {
	body, err := transportbridge.ProtoBody(req.Msg.GetTopic())
	if err != nil {
		return nil, err
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/topics", body, nil, &topicsv1.Topic{})
}

func (h *connectHandler) UpdateTopic(ctx context.Context, req *connect.Request[topicsv1.UpdateTopicRequest]) (*connect.Response[topicsv1.Topic], error) {
	body, err := transportbridge.MaskedBody(req.Msg.GetTopic(), req.Msg.GetUpdateMask().GetPaths())
	if err != nil {
		return nil, err
	}
	return topicCall(ctx, req.Header(), h.legacy.Update, http.MethodPut, req.Msg.GetId(), "", body, &topicsv1.Topic{})
}

func (h *connectHandler) DeleteTopic(ctx context.Context, req *connect.Request[topicsv1.DeleteTopicRequest]) (*connect.Response[topicsv1.DeleteTopicResponse], error) {
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Delete, http.MethodDelete, "/topics/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&topicsv1.DeleteTopicResponse{}), nil
}

func (h *connectHandler) GetAccumulatedSkills(ctx context.Context, req *connect.Request[topicsv1.GetAccumulatedSkillsRequest]) (*connect.Response[topicsv1.AccumulatedSkillsResponse], error) {
	return topicCall(ctx, req.Header(), h.legacy.AccumulatedSkills, http.MethodGet, req.Msg.GetId(), "/skills", nil, &topicsv1.AccumulatedSkillsResponse{})
}

func (h *connectHandler) MatchTopics(ctx context.Context, req *connect.Request[topicsv1.MatchTopicsRequest]) (*connect.Response[topicsv1.MatchTopicsResponse], error) {
	body := map[string]any{"queries": req.Msg.GetQueries(), "limit": req.Msg.GetLimit()}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Match, http.MethodPost, "/topics/match", body, nil, &topicsv1.MatchTopicsResponse{})
}

func topicCall[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, id, suffix string, body any, out *T) (*connect.Response[T], error) {
	return transportbridge.InvokeJSON(ctx, headers, handler, method, "/topics/"+url.PathEscape(id)+suffix, body, map[string]string{"id": id}, out)
}
