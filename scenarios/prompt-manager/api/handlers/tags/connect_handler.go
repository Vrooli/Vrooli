package tags

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	tagsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags"
	tagsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags/tags_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/tags"
)

type connectHandler struct {
	tagsconnect.UnimplementedTagsServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return tagsconnect.NewTagsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListTags(ctx context.Context, req *connect.Request[tagsv1.ListTagsRequest]) (*connect.Response[tagsv1.ListTagsResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.List, http.MethodGet, "/tags", nil, nil)
	if err != nil {
		return nil, err
	}
	out := &tagsv1.ListTagsResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "tags", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateTag(ctx context.Context, req *connect.Request[tagsv1.CreateTagRequest]) (*connect.Response[tagsv1.CreateTagResponse], error) {
	payload := map[string]any{"name": req.Msg.GetName()}
	if req.Msg.Color != nil {
		payload["color"] = req.Msg.GetColor()
	}
	if req.Msg.Description != nil {
		payload["description"] = req.Msg.GetDescription()
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/tags", payload, nil)
	if err != nil {
		return nil, err
	}
	out := &tagsv1.CreateTagResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "tag", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}
