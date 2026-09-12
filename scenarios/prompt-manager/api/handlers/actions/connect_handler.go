package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions/actions_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/actions"
)

type connectHandler struct {
	actionsconnect.UnimplementedActionsServiceHandler
	legacy *domain.Handlers
}

// NewConnectMount exposes the existing action service through its generated
// Connect contract. The legacy handlers are an in-process compatibility seam;
// their public REST routes are intentionally not mounted.
func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return actionsconnect.NewActionsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListActions(ctx context.Context, req *connect.Request[actionsv1.ListActionsRequest]) (*connect.Response[actionsv1.ListActionsResponse], error) {
	query := url.Values{}
	query.Set("pack", req.Msg.GetPack())
	query.Set("status", req.Msg.GetStatus())
	query.Set("owner", req.Msg.GetOwner())
	query.Set("tag", req.Msg.GetTag())
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.List, http.MethodGet, "/actions?"+query.Encode(), nil, nil)
	if err != nil {
		return nil, err
	}
	out := &actionsv1.ListActionsResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "actions", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetAction(ctx context.Context, req *connect.Request[actionsv1.GetActionRequest]) (*connect.Response[actionsv1.GetActionResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Get, http.MethodGet, "/actions/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &actionsv1.GetActionResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "action", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AuthorAction(ctx context.Context, req *connect.Request[actionsv1.AuthorActionRequest]) (*connect.Response[actionsv1.AuthorActionResponse], error) {
	payload, err := protoObject(req.Msg)
	if err != nil {
		return nil, err
	}
	delete(payload, "apply")
	previewResult, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Preview, http.MethodPost, "/actions/preview", payload, nil)
	if err != nil {
		return nil, err
	}
	out := &actionsv1.AuthorActionResponse{}
	if err := transportbridge.Decode(previewResult.Body, out); err != nil {
		return nil, err
	}
	if !req.Msg.GetApply() {
		return connect.NewResponse(out), nil
	}
	if out.GetRendered() == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("action preview returned no rendered contract"))
	}
	actionPayload, err := protoObject(out.GetRendered())
	if err != nil {
		return nil, err
	}
	actionPayload["pack"] = req.Msg.GetPack()
	created, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/actions", actionPayload, nil)
	if err != nil {
		return nil, err
	}
	var mutation struct {
		Action     json.RawMessage `json:"action"`
		Validation json.RawMessage `json:"validation"`
	}
	if err := json.Unmarshal(created.Body, &mutation); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode action mutation: %w", err))
	}
	wrapper, _ := json.Marshal(map[string]any{"rendered": mutation.Action, "validation": mutation.Validation, "applied": true})
	if err := transportbridge.Decode(wrapper, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateAction(ctx context.Context, req *connect.Request[actionsv1.UpdateActionRequest]) (*connect.Response[actionsv1.UpdateActionResponse], error) {
	if req.Msg.GetAction() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("action is required"))
	}
	payload, err := protoObject(req.Msg.GetAction())
	if err != nil {
		return nil, err
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Update, http.MethodPut, "/actions/"+url.PathEscape(req.Msg.GetId()), payload, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &actionsv1.UpdateActionResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteAction(ctx context.Context, req *connect.Request[actionsv1.DeleteActionRequest]) (*connect.Response[actionsv1.DeleteActionResponse], error) {
	target := "/actions/" + url.PathEscape(req.Msg.GetId())
	if req.Msg.GetHard() {
		target += "?hard=true"
	}
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Delete, http.MethodDelete, target, nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&actionsv1.DeleteActionResponse{Id: req.Msg.GetId(), Deleted: req.Msg.GetHard(), Archived: !req.Msg.GetHard()}), nil
}

func (h *connectHandler) ValidateAction(ctx context.Context, req *connect.Request[actionsv1.ValidateActionRequest]) (*connect.Response[actionsv1.ValidateActionResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Validate, http.MethodPost, "/actions/"+url.PathEscape(req.Msg.GetId())+"/validate", nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &actionsv1.ValidateActionResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "validation", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RunAction(ctx context.Context, req *connect.Request[actionsv1.RunActionRequest]) (*connect.Response[actionsv1.RunActionResponse], error) {
	payload, err := protoObject(req.Msg)
	if err != nil {
		return nil, err
	}
	delete(payload, "id")
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Run, http.MethodPost, "/actions/"+url.PathEscape(req.Msg.GetId())+"/run", payload, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &actionsv1.RunActionResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func protoObject(message proto.Message) (map[string]any, error) {
	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode protobuf request: %w", err))
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode protobuf request: %w", err))
	}
	return payload, nil
}
