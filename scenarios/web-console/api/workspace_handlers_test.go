package main

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"
	workspaceH "web-console/handlers/workspace"
)

// Workspace Connect-handler tests. The adapter in workspace_handlers.go
// owns the merge-with-existing behavior, event emission, and the
// "group not found" → ErrGroupNotFound translation; these tests exercise
// that surface end-to-end via the Connect handler.

type workspaceConnectTestHarness struct {
	h interface {
		GetLayout(context.Context, *connect.Request[workspacev1.GetLayoutRequest]) (*connect.Response[workspacev1.GetLayoutResponse], error)
		SaveLayout(context.Context, *connect.Request[workspacev1.SaveLayoutRequest]) (*connect.Response[workspacev1.SaveLayoutResponse], error)
		UpdatePane(context.Context, *connect.Request[workspacev1.UpdatePaneRequest]) (*connect.Response[workspacev1.UpdatePaneResponse], error)
		DeletePane(context.Context, *connect.Request[workspacev1.DeletePaneRequest]) (*connect.Response[workspacev1.DeletePaneResponse], error)
		CreateGroup(context.Context, *connect.Request[workspacev1.CreateGroupRequest]) (*connect.Response[workspacev1.CreateGroupResponse], error)
		UpdateGroup(context.Context, *connect.Request[workspacev1.UpdateGroupRequest]) (*connect.Response[workspacev1.UpdateGroupResponse], error)
		DeleteGroup(context.Context, *connect.Request[workspacev1.DeleteGroupRequest]) (*connect.Response[workspacev1.DeleteGroupResponse], error)
	}
}

func newWorkspaceConnectHandler(t *testing.T) (*workspaceConnectTestHarness, *Server) {
	t.Helper()
	srv := newFakeTestServer()
	h := workspaceH.NewConnectHandler(workspaceH.Deps{Service: newWorkspaceAdapter(srv)})
	return &workspaceConnectTestHarness{h: h}, srv
}

// --- Layout ---

func TestConnect_GetLayout_Empty(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	resp, err := harness.h.GetLayout(context.Background(), connect.NewRequest(&workspacev1.GetLayoutRequest{}))
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(resp.Msg.GetPanes()) != 0 {
		t.Errorf("expected 0 panes, got %d", len(resp.Msg.GetPanes()))
	}
	if len(resp.Msg.GetGroups()) != 0 {
		t.Errorf("expected 0 groups, got %d", len(resp.Msg.GetGroups()))
	}
}

func TestConnect_GetLayout_WithPanes(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "bash", SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "node", SortOrder: 0, IsActive: true})

	resp, err := harness.h.GetLayout(context.Background(), connect.NewRequest(&workspacev1.GetLayoutRequest{}))
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(resp.Msg.GetPanes()) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(resp.Msg.GetPanes()))
	}
	if resp.Msg.GetPanes()[0].GetSessionId() != "s2" {
		t.Errorf("expected first pane s2, got %s", resp.Msg.GetPanes()[0].GetSessionId())
	}
	if resp.Msg.GetActivePane() != "s2" {
		t.Errorf("expected active_pane s2, got %s", resp.Msg.GetActivePane())
	}
}

func TestConnect_SaveLayout_Reorder(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{SessionID: "a", Name: "a", SortOrder: 0})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "b", Name: "b", SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "c", Name: "c", SortOrder: 2})

	_, err := harness.h.SaveLayout(context.Background(), connect.NewRequest(&workspacev1.SaveLayoutRequest{
		ActivePane: "b",
		PaneOrder:  []string{"c", "a", "b"},
	}))
	if err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}

	layout, _ := store.GetLayout()
	if layout.Panes[0].SessionID != "c" {
		t.Errorf("expected first pane c, got %s", layout.Panes[0].SessionID)
	}
	if layout.Panes[1].SessionID != "a" {
		t.Errorf("expected second pane a, got %s", layout.Panes[1].SessionID)
	}
	if layout.ActivePane != "b" {
		t.Errorf("expected active pane b, got %s", layout.ActivePane)
	}
}

// --- Pane ---

func TestConnect_UpdatePane_Create(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	resp, err := harness.h.UpdatePane(context.Background(), connect.NewRequest(&workspacev1.UpdatePaneRequest{
		SessionId:      "s1",
		Name:           "custom",
		HasName:        true,
		HeaderColor:    "#ff0000",
		HasHeaderColor: true,
		ThemeId:        "dracula",
		HasThemeId:     true,
		FontSize:       16,
		HasFontSize:    true,
	}))
	if err != nil {
		t.Fatalf("UpdatePane: %v", err)
	}
	p := resp.Msg.GetPane()
	if p.GetName() != "custom" {
		t.Errorf("expected name custom, got %s", p.GetName())
	}
	if p.GetHeaderColor() != "#ff0000" {
		t.Errorf("expected color #ff0000, got %s", p.GetHeaderColor())
	}
	if p.GetThemeId() != "dracula" {
		t.Errorf("expected theme dracula, got %s", p.GetThemeId())
	}
	if p.GetFontSize() != 16 {
		t.Errorf("expected font_size 16, got %d", p.GetFontSize())
	}
}

func TestConnect_UpdatePane_PartialUpdate(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{
		SessionID: "s1", Name: "original", HeaderColor: "#00ff00",
		ThemeID: "monokai", FontSize: 14,
	})

	resp, err := harness.h.UpdatePane(context.Background(), connect.NewRequest(&workspacev1.UpdatePaneRequest{
		SessionId:      "s1",
		HeaderColor:    "#ff0000",
		HasHeaderColor: true,
	}))
	if err != nil {
		t.Fatalf("UpdatePane: %v", err)
	}
	p := resp.Msg.GetPane()
	if p.GetName() != "original" {
		t.Errorf("name should be preserved, got %s", p.GetName())
	}
	if p.GetHeaderColor() != "#ff0000" {
		t.Errorf("color should be updated, got %s", p.GetHeaderColor())
	}
	if p.GetThemeId() != "monokai" {
		t.Errorf("theme should be preserved, got %s", p.GetThemeId())
	}
}

func TestConnect_DeletePane_Idempotent(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	_, err := harness.h.DeletePane(context.Background(), connect.NewRequest(&workspacev1.DeletePaneRequest{SessionId: "nonexistent"}))
	if err != nil {
		t.Fatalf("DeletePane: %v", err)
	}
}

func TestConnect_UpdatePane_SetGroup(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Dev", "#3b82f6")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "bash"})

	resp, err := harness.h.UpdatePane(context.Background(), connect.NewRequest(&workspacev1.UpdatePaneRequest{
		SessionId:  "s1",
		GroupId:    group.ID,
		HasGroupId: true,
	}))
	if err != nil {
		t.Fatalf("UpdatePane: %v", err)
	}
	if resp.Msg.GetPane().GetGroupId() != group.ID {
		t.Errorf("expected group_id %s, got %s", group.ID, resp.Msg.GetPane().GetGroupId())
	}
}

// --- Group ---

func TestConnect_CreateGroup(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	resp, err := harness.h.CreateGroup(context.Background(), connect.NewRequest(&workspacev1.CreateGroupRequest{
		Name: "Dev servers", Color: "#e11d48",
	}))
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	g := resp.Msg.GetGroup()
	if g.GetName() != "Dev servers" {
		t.Errorf("expected name 'Dev servers', got %s", g.GetName())
	}
	if g.GetColor() != "#e11d48" {
		t.Errorf("expected color #e11d48, got %s", g.GetColor())
	}
	if g.GetId() == "" {
		t.Error("expected generated ID")
	}
}

func TestConnect_CreateGroup_Defaults(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	resp, err := harness.h.CreateGroup(context.Background(), connect.NewRequest(&workspacev1.CreateGroupRequest{}))
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	g := resp.Msg.GetGroup()
	if g.GetName() != "Group" {
		t.Errorf("expected default name 'Group', got %s", g.GetName())
	}
	if g.GetColor() != "#3b82f6" {
		t.Errorf("expected default color #3b82f6, got %s", g.GetColor())
	}
}

func TestConnect_UpdateGroup(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Old name", "#000000")

	resp, err := harness.h.UpdateGroup(context.Background(), connect.NewRequest(&workspacev1.UpdateGroupRequest{
		Id:             group.ID,
		Name:           "New name",
		HasName:        true,
		IsCollapsed:    true,
		HasIsCollapsed: true,
	}))
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	g := resp.Msg.GetGroup()
	if g.GetName() != "New name" {
		t.Errorf("expected name 'New name', got %s", g.GetName())
	}
	if !g.GetIsCollapsed() {
		t.Error("expected is_collapsed to be true")
	}
	if g.GetColor() != "#000000" {
		t.Errorf("color should be preserved, got %s", g.GetColor())
	}
}

func TestConnect_UpdateGroup_NotFound(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	_, err := harness.h.UpdateGroup(context.Background(), connect.NewRequest(&workspacev1.UpdateGroupRequest{
		Id:      "nonexistent",
		Name:    "x",
		HasName: true,
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	var connErr *connect.Error
	if !errors.As(err, &connErr) || connErr.Code() != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestConnect_DeleteGroup_Idempotent(t *testing.T) {
	harness, _ := newWorkspaceConnectHandler(t)
	_, err := harness.h.DeleteGroup(context.Background(), connect.NewRequest(&workspacev1.DeleteGroupRequest{Id: "nonexistent"}))
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
}

func TestConnect_DeleteGroup_ClearsGroupOnPanes(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)

	group, _ := store.CreateGroup("Test", "#ff0000")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "a", GroupID: group.ID})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "b", GroupID: group.ID})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s3", Name: "c"})

	_, err := harness.h.DeleteGroup(context.Background(), connect.NewRequest(&workspacev1.DeleteGroupRequest{Id: group.ID}))
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}

	layout, _ := store.GetLayout()
	for _, p := range layout.Panes {
		if p.GroupID != "" {
			t.Errorf("pane %s still has group_id %s", p.SessionID, p.GroupID)
		}
	}
}

func TestConnect_GetLayout_WithGroups(t *testing.T) {
	harness, srv := newWorkspaceConnectHandler(t)
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Servers", "#e11d48")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "web", GroupID: group.ID, SortOrder: 0})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "db", GroupID: group.ID, SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s3", Name: "logs", SortOrder: 2})

	resp, err := harness.h.GetLayout(context.Background(), connect.NewRequest(&workspacev1.GetLayoutRequest{}))
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(resp.Msg.GetPanes()) != 3 {
		t.Fatalf("expected 3 panes, got %d", len(resp.Msg.GetPanes()))
	}
	if len(resp.Msg.GetGroups()) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Msg.GetGroups()))
	}
	if resp.Msg.GetGroups()[0].GetName() != "Servers" {
		t.Errorf("expected group name 'Servers', got %s", resp.Msg.GetGroups()[0].GetName())
	}

	groupedCount := 0
	for _, p := range resp.Msg.GetPanes() {
		if p.GetGroupId() == group.ID {
			groupedCount++
		}
	}
	if groupedCount != 2 {
		t.Errorf("expected 2 panes in group, got %d", groupedCount)
	}
}
