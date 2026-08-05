package workspace

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"
)

// Deps wires the seams the Connect workspace handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// WorkspaceServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ErrGroupNotFound is the sentinel the Service implementation returns
// when an UpdateGroup call references a missing id. Mapped to
// CodeNotFound.
var ErrGroupNotFound = errors.New("group not found")

func (h *connectHandler) GetLayout(ctx context.Context, _ *connect.Request[workspacev1.GetLayoutRequest]) (*connect.Response[workspacev1.GetLayoutResponse], error) {
	layout, err := h.deps.Service.GetLayout(ctx)
	if err != nil {
		h.deps.Logger.Printf("workspace.GetLayout: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workspacev1.GetLayoutResponse{
		ActivePane: layout.ActivePane,
		Panes:      panesToProto(layout.Panes),
		Groups:     groupsToProto(layout.Groups),
	}), nil
}

func (h *connectHandler) SaveLayout(ctx context.Context, req *connect.Request[workspacev1.SaveLayoutRequest]) (*connect.Response[workspacev1.SaveLayoutResponse], error) {
	if err := h.deps.Service.SaveLayout(ctx, req.Msg.GetActivePane(), req.Msg.GetPaneOrder()); err != nil {
		h.deps.Logger.Printf("workspace.SaveLayout: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workspacev1.SaveLayoutResponse{}), nil
}

func (h *connectHandler) UpdatePane(ctx context.Context, req *connect.Request[workspacev1.UpdatePaneRequest]) (*connect.Response[workspacev1.UpdatePaneResponse], error) {
	in := UpdatePaneRequest{
		SessionID:               req.Msg.GetSessionId(),
		Name:                    req.Msg.GetName(),
		HasName:                 req.Msg.GetHasName(),
		HeaderColor:             req.Msg.GetHeaderColor(),
		HasHeaderColor:          req.Msg.GetHasHeaderColor(),
		ThemeID:                 req.Msg.GetThemeId(),
		HasThemeID:              req.Msg.GetHasThemeId(),
		FontSize:                int(req.Msg.GetFontSize()),
		HasFontSize:             req.Msg.GetHasFontSize(),
		SortOrder:               int(req.Msg.GetSortOrder()),
		HasSortOrder:            req.Msg.GetHasSortOrder(),
		GroupID:                 req.Msg.GetGroupId(),
		HasGroupID:              req.Msg.GetHasGroupId(),
		SupportsMessagesView:    req.Msg.GetSupportsMessagesView(),
		HasSupportsMessagesView: req.Msg.GetHasSupportsMessagesView(),
		ManuallyUnread:          req.Msg.GetManuallyUnread(),
		HasManuallyUnread:       req.Msg.GetHasManuallyUnread(),
	}
	p, err := h.deps.Service.UpdatePane(ctx, in)
	if err != nil {
		h.deps.Logger.Printf("workspace.UpdatePane: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workspacev1.UpdatePaneResponse{Pane: paneToProto(p)}), nil
}

func (h *connectHandler) DeletePane(ctx context.Context, req *connect.Request[workspacev1.DeletePaneRequest]) (*connect.Response[workspacev1.DeletePaneResponse], error) {
	h.deps.Service.DeletePane(ctx, req.Msg.GetSessionId())
	return connect.NewResponse(&workspacev1.DeletePaneResponse{}), nil
}

func (h *connectHandler) CreateGroup(ctx context.Context, req *connect.Request[workspacev1.CreateGroupRequest]) (*connect.Response[workspacev1.CreateGroupResponse], error) {
	g, err := h.deps.Service.CreateGroup(ctx, req.Msg.GetName(), req.Msg.GetColor())
	if err != nil {
		h.deps.Logger.Printf("workspace.CreateGroup: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workspacev1.CreateGroupResponse{Group: groupToProto(g)}), nil
}

func (h *connectHandler) UpdateGroup(ctx context.Context, req *connect.Request[workspacev1.UpdateGroupRequest]) (*connect.Response[workspacev1.UpdateGroupResponse], error) {
	in := UpdateGroupRequest{
		ID:             req.Msg.GetId(),
		Name:           req.Msg.GetName(),
		HasName:        req.Msg.GetHasName(),
		Color:          req.Msg.GetColor(),
		HasColor:       req.Msg.GetHasColor(),
		IsCollapsed:    req.Msg.GetIsCollapsed(),
		HasIsCollapsed: req.Msg.GetHasIsCollapsed(),
	}
	g, err := h.deps.Service.UpdateGroup(ctx, in)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("workspace.UpdateGroup: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workspacev1.UpdateGroupResponse{Group: groupToProto(g)}), nil
}

func (h *connectHandler) DeleteGroup(ctx context.Context, req *connect.Request[workspacev1.DeleteGroupRequest]) (*connect.Response[workspacev1.DeleteGroupResponse], error) {
	h.deps.Service.DeleteGroup(ctx, req.Msg.GetId())
	return connect.NewResponse(&workspacev1.DeleteGroupResponse{}), nil
}

func paneToProto(p Pane) *workspacev1.Pane {
	return &workspacev1.Pane{
		SessionId:            p.SessionID,
		Name:                 p.Name,
		HeaderColor:          p.HeaderColor,
		ThemeId:              p.ThemeID,
		FontSize:             int32(p.FontSize),
		SortOrder:            int32(p.SortOrder),
		GroupId:              p.GroupID,
		SupportsMessagesView: p.SupportsMessagesView,
		ManuallyUnread:       p.ManuallyUnread,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func panesToProto(in []Pane) []*workspacev1.Pane {
	out := make([]*workspacev1.Pane, 0, len(in))
	for _, p := range in {
		out = append(out, paneToProto(p))
	}
	return out
}

func groupToProto(g Group) *workspacev1.Group {
	return &workspacev1.Group{
		Id:          g.ID,
		Name:        g.Name,
		Color:       g.Color,
		SortOrder:   int32(g.SortOrder),
		IsCollapsed: g.IsCollapsed,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func groupsToProto(in []Group) []*workspacev1.Group {
	out := make([]*workspacev1.Group, 0, len(in))
	for _, g := range in {
		out = append(out, groupToProto(g))
	}
	return out
}
