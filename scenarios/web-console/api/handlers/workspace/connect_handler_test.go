package workspace

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"
)

type fakeWorkspaceService struct {
	err    error
	pane   Pane
	group  Group
	layout Layout
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
