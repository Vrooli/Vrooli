package chat

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	internalchat "portal/internal/chat"

	chatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat"
)

type Handler struct {
	service *internalchat.Service
}

func NewHandler(service *internalchat.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListChats(ctx context.Context, req *connect.Request[chatv1.ListChatsRequest]) (*connect.Response[chatv1.ListChatsResponse], error) {
	chats, groups, err := h.service.ListChats(ctx, internalchat.SearchInput{
		GroupID: req.Msg.GetGroupId(),
		Query:   req.Msg.GetQuery(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.ListChatsResponse{
		Chats:  internalchat.ToProtoChats(chats),
		Groups: internalchat.ToProtoGroups(groups),
	}), nil
}

func (h *Handler) CreateChat(ctx context.Context, req *connect.Request[chatv1.CreateChatRequest]) (*connect.Response[chatv1.CreateChatResponse], error) {
	created, err := h.service.CreateChat(ctx, internalchat.CreateChatInput{
		Title:            req.Msg.GetTitle(),
		GroupID:          req.Msg.GetGroupId(),
		Model:            req.Msg.GetModel(),
		WebSearchEnabled: req.Msg.GetWebSearchEnabled(),
		Mode:             internalchat.ChatModeFromProto(req.Msg.GetMode()),
		AgentHarness:     internalchat.AgentHarnessFromProto(req.Msg.GetAgentHarness()),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.CreateChatResponse{Chat: internalchat.ToProtoChat(created)}), nil
}

func (h *Handler) GetChat(ctx context.Context, req *connect.Request[chatv1.GetChatRequest]) (*connect.Response[chatv1.GetChatResponse], error) {
	got, err := h.service.GetChat(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.GetChatResponse{Chat: internalchat.ToProtoChat(got)}), nil
}

func (h *Handler) UpdateChat(ctx context.Context, req *connect.Request[chatv1.UpdateChatRequest]) (*connect.Response[chatv1.UpdateChatResponse], error) {
	input := internalchat.UpdateChatInput{ID: req.Msg.GetId()}
	if req.Msg.GetHasTitle() {
		v := req.Msg.GetTitle()
		input.Title = &v
	}
	if req.Msg.GetHasGroupId() {
		v := req.Msg.GetGroupId()
		input.GroupID = &v
	}
	if req.Msg.GetHasModel() {
		v := req.Msg.GetModel()
		input.Model = &v
	}
	if req.Msg.GetHasWebSearchEnabled() {
		v := req.Msg.GetWebSearchEnabled()
		input.WebSearchEnabled = &v
	}
	if req.Msg.GetHasActiveLeafMessageId() {
		v := req.Msg.GetActiveLeafMessageId()
		input.ActiveLeafMessageID = &v
		input.ClearActiveLeafMessageID = v == ""
	}
	updated, err := h.service.UpdateChat(ctx, input)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.UpdateChatResponse{Chat: internalchat.ToProtoChat(updated)}), nil
}

func (h *Handler) DeleteChat(ctx context.Context, req *connect.Request[chatv1.DeleteChatRequest]) (*connect.Response[chatv1.DeleteChatResponse], error) {
	if _, err := h.service.DeleteChat(ctx, req.Msg.GetId()); err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.DeleteChatResponse{}), nil
}

func (h *Handler) ListGroups(ctx context.Context, _ *connect.Request[chatv1.ListGroupsRequest]) (*connect.Response[chatv1.ListGroupsResponse], error) {
	groups, err := h.service.ListGroups(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.ListGroupsResponse{Groups: internalchat.ToProtoGroups(groups)}), nil
}

func (h *Handler) CreateGroup(ctx context.Context, req *connect.Request[chatv1.CreateGroupRequest]) (*connect.Response[chatv1.CreateGroupResponse], error) {
	created, err := h.service.CreateGroup(ctx, internalchat.CreateGroupInput{
		Name:  req.Msg.GetName(),
		Color: req.Msg.GetColor(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.CreateGroupResponse{Group: internalchat.ToProtoGroup(created)}), nil
}

func (h *Handler) UpdateGroup(ctx context.Context, req *connect.Request[chatv1.UpdateGroupRequest]) (*connect.Response[chatv1.UpdateGroupResponse], error) {
	input := internalchat.UpdateGroupInput{ID: req.Msg.GetId()}
	if req.Msg.GetHasName() {
		v := req.Msg.GetName()
		input.Name = &v
	}
	if req.Msg.GetHasColor() {
		v := req.Msg.GetColor()
		input.Color = &v
	}
	if req.Msg.GetHasCollapsed() {
		v := req.Msg.GetCollapsed()
		input.Collapsed = &v
	}
	if req.Msg.GetHasSortOrder() {
		v := req.Msg.GetSortOrder()
		input.SortOrder = &v
	}
	updated, err := h.service.UpdateGroup(ctx, input)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.UpdateGroupResponse{Group: internalchat.ToProtoGroup(updated)}), nil
}

func (h *Handler) DeleteGroup(ctx context.Context, req *connect.Request[chatv1.DeleteGroupRequest]) (*connect.Response[chatv1.DeleteGroupResponse], error) {
	if _, err := h.service.DeleteGroup(ctx, req.Msg.GetId()); err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&chatv1.DeleteGroupResponse{}), nil
}

func connectError(err error) error {
	var notFound internalchat.ErrNotFound
	switch {
	case errors.As(err, &notFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, internalchat.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
