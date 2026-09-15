package workspace

import (
	"context"
	"testing"
)

func TestMemStoreLifecycle(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	if err := s.UpsertPane(ctx, Pane{SessionID: "s1", IsActive: true, SortOrder: 1}); err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPane(ctx, Pane{SessionID: "s1", GroupID: g.ID, Name: "updated", SortOrder: 0}); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePaneOrder(ctx, "s1", []string{"s1"}); err != nil {
		t.Fatal(err)
	}
	name := "renamed"
	collapsed := true
	if _, err := s.UpdateGroup(ctx, g.ID, &name, nil, &collapsed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateGroup(ctx, "missing", nil, nil, nil); err != ErrGroupNotFound {
		t.Fatal(err)
	}
	layout, err := s.GetLayout(ctx)
	if err != nil || len(layout.Panes) != 1 || len(layout.Groups) != 1 {
		t.Fatalf("layout=%#v err=%v", layout, err)
	}
	if ok, err := s.DeleteGroup(ctx, g.ID); err != nil || !ok {
		t.Fatalf("delete group=%v err=%v", ok, err)
	}
	if err := s.DeletePane(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
}
