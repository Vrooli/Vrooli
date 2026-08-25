package workspace

import (
	"context"
	"log"
	"testing"

	wsdomain "web-console/internal/workspace"
)

type workspaceEvents struct{ count int }

func (e *workspaceEvents) Emit(string, string, map[string]string) { e.count++ }

func TestAdapterCoversWorkspacePersistenceAndEvents(t *testing.T) {
	ctx := context.Background()
	store := wsdomain.NewMemStore()
	events := &workspaceEvents{}
	a := &Adapter{Store: store, Events: events, Logger: log.Default()}

	if err := a.SaveLayout(ctx, "s1", []string{"s1"}); err != nil {
		t.Fatal(err)
	}
	pane, err := a.UpdatePane(ctx, UpdatePaneRequest{SessionID: "s1", HasName: true, Name: "Shell", HasHeaderColor: true, HeaderColor: "#fff", HasThemeID: true, ThemeID: "dark", HasFontSize: true, FontSize: 16, HasSortOrder: true, SortOrder: 1, HasGroupID: true, GroupID: "", HasSupportsMessagesView: true, SupportsMessagesView: true, HasManuallyUnread: true, ManuallyUnread: true})
	if err != nil || pane.Name != "Shell" || pane.FontSize != 16 {
		t.Fatalf("UpdatePane() = %#v, %v", pane, err)
	}
	if got, err := a.GetLayout(ctx); err != nil || len(got.Panes) != 1 {
		t.Fatalf("GetLayout() = %#v, %v", got, err)
	}
	group, err := a.CreateGroup(ctx, "Agents", "#123")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := a.UpdateGroup(ctx, UpdateGroupRequest{ID: group.ID, HasName: true, Name: "Updated", HasColor: true, Color: "#456", HasIsCollapsed: true, IsCollapsed: true})
	if err != nil || updated.Name != "Updated" {
		t.Fatalf("UpdateGroup() = %#v, %v", updated, err)
	}
	if _, err := a.UpdateGroup(ctx, UpdateGroupRequest{ID: "missing"}); err != ErrGroupNotFound {
		t.Fatalf("missing group error = %v", err)
	}
	a.DeletePane(ctx, "s1")
	a.DeleteGroup(ctx, group.ID)
	if events.count < 4 {
		t.Fatalf("event count = %d", events.count)
	}
}
