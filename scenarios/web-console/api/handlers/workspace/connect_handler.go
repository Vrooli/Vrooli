package workspace

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"

	wsdomain "web-console/internal/workspace"
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

// roleError maps the role sentinels the store raises onto Connect codes. A
// missing id and a malformed write are caller mistakes, not server faults,
// so neither should read as CodeInternal in the client's error handling.
func (h *connectHandler) roleError(op string, err error) error {
	switch {
	case errors.Is(err, wsdomain.ErrRoleNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, wsdomain.ErrInvalidRole):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.deps.Logger.Printf("workspace.%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

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
		Roles:      rolesToProto(layout.Roles),
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

func (h *connectHandler) ListRoles(ctx context.Context, req *connect.Request[workspacev1.ListRolesRequest]) (*connect.Response[workspacev1.ListRolesResponse], error) {
	roles, err := h.deps.Service.ListRoles(ctx, req.Msg.GetGroupId())
	if err != nil {
		return nil, h.roleError("ListRoles", err)
	}
	return connect.NewResponse(&workspacev1.ListRolesResponse{Roles: rolesToProto(roles)}), nil
}

func (h *connectHandler) CreateRole(ctx context.Context, req *connect.Request[workspacev1.CreateRoleRequest]) (*connect.Response[workspacev1.CreateRoleResponse], error) {
	in := CreateRoleRequest{
		GroupID:        req.Msg.GetGroupId(),
		Label:          req.Msg.GetLabel(),
		Command:        req.Msg.GetCommand(),
		WorkingDir:     req.Msg.GetWorkingDir(),
		IncomingPrompt: req.Msg.GetIncomingPrompt(),
		Backend:        req.Msg.GetBackend(),
		TargetID:       req.Msg.GetTargetId(),
		SessionID:      req.Msg.GetSessionId(),
		SortOrder:      int(req.Msg.GetSortOrder()),
		// The proto has no has_sort_order: a create either names a position
		// or appends. Treating any non-zero value as explicit keeps zero
		// meaning "append", which is what an unset int32 field looks like.
		HasSortOrder: req.Msg.GetSortOrder() != 0,
	}
	role, err := h.deps.Service.CreateRole(ctx, in)
	if err != nil {
		return nil, h.roleError("CreateRole", err)
	}
	return connect.NewResponse(&workspacev1.CreateRoleResponse{Role: roleToProto(role)}), nil
}

func (h *connectHandler) UpdateRole(ctx context.Context, req *connect.Request[workspacev1.UpdateRoleRequest]) (*connect.Response[workspacev1.UpdateRoleResponse], error) {
	in := UpdateRoleRequest{
		ID:                req.Msg.GetId(),
		Label:             req.Msg.GetLabel(),
		HasLabel:          req.Msg.GetHasLabel(),
		Command:           req.Msg.GetCommand(),
		HasCommand:        req.Msg.GetHasCommand(),
		WorkingDir:        req.Msg.GetWorkingDir(),
		HasWorkingDir:     req.Msg.GetHasWorkingDir(),
		IncomingPrompt:    req.Msg.GetIncomingPrompt(),
		HasIncomingPrompt: req.Msg.GetHasIncomingPrompt(),
		SessionID:         req.Msg.GetSessionId(),
		HasSessionID:      req.Msg.GetHasSessionId(),
		SortOrder:         int(req.Msg.GetSortOrder()),
		HasSortOrder:      req.Msg.GetHasSortOrder(),
		Backend:           req.Msg.GetBackend(),
		HasBackend:        req.Msg.GetHasBackend(),
		TargetID:          req.Msg.GetTargetId(),
		HasTargetID:       req.Msg.GetHasTargetId(),
		GroupID:           req.Msg.GetGroupId(),
		HasGroupID:        req.Msg.GetHasGroupId(),
	}
	role, err := h.deps.Service.UpdateRole(ctx, in)
	if err != nil {
		return nil, h.roleError("UpdateRole", err)
	}
	return connect.NewResponse(&workspacev1.UpdateRoleResponse{Role: roleToProto(role)}), nil
}

func (h *connectHandler) DeleteRole(ctx context.Context, req *connect.Request[workspacev1.DeleteRoleRequest]) (*connect.Response[workspacev1.DeleteRoleResponse], error) {
	if err := h.deps.Service.DeleteRole(ctx, req.Msg.GetId()); err != nil {
		return nil, h.roleError("DeleteRole", err)
	}
	return connect.NewResponse(&workspacev1.DeleteRoleResponse{}), nil
}

func roleToProto(r Role) *workspacev1.Role {
	return &workspacev1.Role{
		Id:             r.ID,
		GroupId:        r.GroupID,
		Label:          r.Label,
		Command:        r.Command,
		WorkingDir:     r.WorkingDir,
		IncomingPrompt: r.IncomingPrompt,
		SessionId:      r.SessionID,
		SortOrder:      int32(r.SortOrder),
		Backend:        r.Backend,
		TargetId:       r.TargetID,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func rolesToProto(in []Role) []*workspacev1.Role {
	out := make([]*workspacev1.Role, 0, len(in))
	for _, r := range in {
		out = append(out, roleToProto(r))
	}
	return out
}
