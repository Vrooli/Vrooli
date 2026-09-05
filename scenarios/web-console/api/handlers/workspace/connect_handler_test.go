package workspace

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"

	wsdomain "web-console/internal/workspace"
)

type fakeWorkspaceService struct {
	err    error
	pane   Pane
	group  Group
	layout Layout
	role   Role
	// roleErr is separate from err so a role test can exercise the role
	// error mapping without also breaking the pane and group assertions.
	roleErr error
}

func (f *fakeWorkspaceService) GetLayout(context.Context) (Layout, error)          { return f.layout, f.err }
func (f *fakeWorkspaceService) SaveLayout(context.Context, string, []string) error { return f.err }
func (f *fakeWorkspaceService) UpdatePane(context.Context, UpdatePaneRequest) (Pane, error) {
	return f.pane, f.err
}
func (f *fakeWorkspaceService) DeletePane(context.Context, string) {}
func (f *fakeWorkspaceService) CreateGroup(context.Context, string, string) (Group, error) {
	return f.group, f.err
}

func (f *fakeWorkspaceService) UpdateGroup(context.Context, UpdateGroupRequest) (Group, error) {
	return f.group, f.err
}
func (f *fakeWorkspaceService) DeleteGroup(context.Context, string) {}

func (f *fakeWorkspaceService) ListRoles(context.Context, string) ([]Role, error) {
	return []Role{f.role}, f.roleErr
}

func (f *fakeWorkspaceService) CreateRole(context.Context, CreateRoleRequest) (Role, error) {
	return f.role, f.roleErr
}

func (f *fakeWorkspaceService) UpdateRole(context.Context, UpdateRoleRequest) (Role, error) {
	return f.role, f.roleErr
}

func (f *fakeWorkspaceService) DeleteRole(context.Context, string) error { return f.roleErr }

func TestConnectHandlerWorkspaceOperations(t *testing.T) {
	svc := &fakeWorkspaceService{
		layout: Layout{ActivePane: "s1", Panes: []Pane{{SessionID: "s1", Name: "main", FontSize: 14, SupportsMessagesView: true}}, Groups: []Group{{ID: "g1", Name: "Group", IsCollapsed: true}}},
		pane:   Pane{SessionID: "s1", Name: "edited", FontSize: 16}, group: Group{ID: "g1", Name: "Group"},
	}
	h := NewConnectHandler(Deps{Service: svc})
	ctx := context.Background()
	if resp, err := h.GetLayout(ctx, connect.NewRequest(&workspacev1.GetLayoutRequest{})); err != nil || resp.Msg.ActivePane != "s1" || len(resp.Msg.Panes) != 1 || len(resp.Msg.Groups) != 1 {
		t.Fatalf("layout: %#v %v", resp, err)
	}
	if _, err := h.SaveLayout(ctx, connect.NewRequest(&workspacev1.SaveLayoutRequest{ActivePane: "s1", PaneOrder: []string{"s1"}})); err != nil {
		t.Fatal(err)
	}
	if resp, err := h.UpdatePane(ctx, connect.NewRequest(&workspacev1.UpdatePaneRequest{SessionId: "s1", HasName: true, Name: "edited", HasFontSize: true, FontSize: 16})); err != nil || resp.Msg.Pane.Name != "edited" {
		t.Fatalf("pane: %#v %v", resp, err)
	}
	if _, err := h.DeletePane(ctx, connect.NewRequest(&workspacev1.DeletePaneRequest{SessionId: "s1"})); err != nil {
		t.Fatal(err)
	}
	if resp, err := h.CreateGroup(ctx, connect.NewRequest(&workspacev1.CreateGroupRequest{Name: "Group", Color: "blue"})); err != nil || resp.Msg.Group.Id != "g1" {
		t.Fatalf("create group: %#v %v", resp, err)
	}
	if resp, err := h.UpdateGroup(ctx, connect.NewRequest(&workspacev1.UpdateGroupRequest{Id: "g1", HasName: true, Name: "Group"})); err != nil || resp.Msg.Group.Id != "g1" {
		t.Fatalf("update group: %#v %v", resp, err)
	}
	if _, err := h.DeleteGroup(ctx, connect.NewRequest(&workspacev1.DeleteGroupRequest{Id: "g1"})); err != nil {
		t.Fatal(err)
	}
}

func TestConnectHandlerWorkspaceErrors(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeWorkspaceService{err: errors.New("store")}})
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, e := h.GetLayout(ctx, connect.NewRequest(&workspacev1.GetLayoutRequest{})); return e },
		func() error {
			_, e := h.SaveLayout(ctx, connect.NewRequest(&workspacev1.SaveLayoutRequest{}))
			return e
		},
		func() error {
			_, e := h.UpdatePane(ctx, connect.NewRequest(&workspacev1.UpdatePaneRequest{}))
			return e
		},
		func() error {
			_, e := h.CreateGroup(ctx, connect.NewRequest(&workspacev1.CreateGroupRequest{}))
			return e
		},
		func() error {
			_, e := h.UpdateGroup(ctx, connect.NewRequest(&workspacev1.UpdateGroupRequest{}))
			return e
		},
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeInternal {
			t.Errorf("got %v", err)
		}
	}
	missing := NewConnectHandler(Deps{Service: &fakeWorkspaceService{err: ErrGroupNotFound}})
	if _, err := missing.UpdateGroup(ctx, connect.NewRequest(&workspacev1.UpdateGroupRequest{})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing group: %v", err)
	}
}

// TestConnectHandlerRoleOperations covers the four role RPCs and, more
// importantly, the error mapping: a missing id and a malformed write are
// caller mistakes and must not surface as CodeInternal.
func TestConnectHandlerRoleOperations(t *testing.T) {
	ctx := context.Background()
	svc := &fakeWorkspaceService{role: Role{ID: "r1", GroupID: "g1", Label: "Implementer", Command: "codex --yolo"}}
	h := NewConnectHandler(Deps{Service: svc})

	if resp, err := h.ListRoles(ctx, connect.NewRequest(&workspacev1.ListRolesRequest{GroupId: "g1"})); err != nil || len(resp.Msg.GetRoles()) != 1 || resp.Msg.GetRoles()[0].GetId() != "r1" {
		t.Fatalf("list roles: %#v %v", resp, err)
	}
	if resp, err := h.CreateRole(ctx, connect.NewRequest(&workspacev1.CreateRoleRequest{GroupId: "g1", Label: "Implementer"})); err != nil || resp.Msg.GetRole().GetLabel() != "Implementer" {
		t.Fatalf("create role: %#v %v", resp, err)
	}
	if resp, err := h.UpdateRole(ctx, connect.NewRequest(&workspacev1.UpdateRoleRequest{Id: "r1", HasLabel: true, Label: "Implementer"})); err != nil || resp.Msg.GetRole().GetId() != "r1" {
		t.Fatalf("update role: %#v %v", resp, err)
	}
	if _, err := h.DeleteRole(ctx, connect.NewRequest(&workspacev1.DeleteRoleRequest{Id: "r1"})); err != nil {
		t.Fatalf("delete role: %v", err)
	}

	// A waiting role must survive the proto round trip with an EMPTY
	// session id: an encoder that substituted a placeholder would make every
	// waiting role look running to the client.
	waiting := &fakeWorkspaceService{role: Role{ID: "r2", GroupID: "g1", Label: "Critic"}}
	waitingHandler := NewConnectHandler(Deps{Service: waiting})
	resp, err := waitingHandler.ListRoles(ctx, connect.NewRequest(&workspacev1.ListRolesRequest{}))
	if err != nil {
		t.Fatalf("list waiting roles: %v", err)
	}
	if got := resp.Msg.GetRoles()[0].GetSessionId(); got != "" {
		t.Fatalf("waiting role session id = %q, want empty", got)
	}

	notFound := &fakeWorkspaceService{roleErr: wsdomain.ErrRoleNotFound}
	if _, err := NewConnectHandler(Deps{Service: notFound}).UpdateRole(ctx, connect.NewRequest(&workspacev1.UpdateRoleRequest{Id: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing role code = %v, want not_found", connect.CodeOf(err))
	}

	invalid := &fakeWorkspaceService{roleErr: wsdomain.ErrInvalidRole}
	if _, err := NewConnectHandler(Deps{Service: invalid}).CreateRole(ctx, connect.NewRequest(&workspacev1.CreateRoleRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid role code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

// TestGetLayoutReturnsRolesInOneResponse pins the round-trip contract: adding
// roles must not cost a second call on load.
func TestGetLayoutReturnsRolesInOneResponse(t *testing.T) {
	svc := &fakeWorkspaceService{layout: Layout{
		ActivePane: "s1",
		Panes:      []Pane{{SessionID: "s1", Name: "planner"}},
		Groups:     []Group{{ID: "g1", Name: "Ship it"}},
		Roles:      []Role{{ID: "r1", GroupID: "g1", Label: "Implementer"}},
	}}
	resp, err := NewConnectHandler(Deps{Service: svc}).GetLayout(context.Background(), connect.NewRequest(&workspacev1.GetLayoutRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetPanes()) != 1 || len(resp.Msg.GetGroups()) != 1 || len(resp.Msg.GetRoles()) != 1 {
		t.Fatalf("GetLayout returned panes=%d groups=%d roles=%d, want 1/1/1",
			len(resp.Msg.GetPanes()), len(resp.Msg.GetGroups()), len(resp.Msg.GetRoles()))
	}
}
