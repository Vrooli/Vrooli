package initiatives

import (
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
)

// TestDelete_CascadesToMemberItemsAndDeps verifies initiative deletion
// clears the initiative field on every member item and scrubs the
// deleted name from every other initiative's depends_on array.
func TestDelete_CascadesToMemberItemsAndDeps(t *testing.T) {
	items := map[string]backlog.BacklogItem{
		"idea/a": {Kind: "idea", Name: "a", Title: "A", Status: "backlog", Priority: 5, Initiative: "target"},
		"idea/b": {Kind: "idea", Name: "b", Title: "B", Status: "backlog", Priority: 5, Initiative: "target"},
		"idea/c": {Kind: "idea", Name: "c", Title: "C", Status: "backlog", Priority: 5, Initiative: "other"},
	}
	svc := newTestService(t, items)

	if _, err := svc.Create(CreateRequest{Name: "target", Title: "Target", Items: []string{"idea/a", "idea/b"}}); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "other", Title: "Other", Items: []string{"idea/c"}, DependsOn: []string{"target"}}); err != nil {
		t.Fatalf("create other: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "unrelated", Title: "Unrelated"}); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}

	if err := svc.Delete("target"); err != nil {
		t.Fatalf("delete target: %v", err)
	}

	loader := svc.backlogLoader.(*mockBacklogLoader)
	if got := loader.items["idea/a"].Initiative; got != "" {
		t.Errorf("idea/a.initiative should be cleared, got %q", got)
	}
	if got := loader.items["idea/b"].Initiative; got != "" {
		t.Errorf("idea/b.initiative should be cleared, got %q", got)
	}
	if got := loader.items["idea/c"].Initiative; got != "other" {
		t.Errorf("idea/c.initiative should be untouched (other), got %q", got)
	}

	other, err := svc.store.Load("other")
	if err != nil {
		t.Fatalf("load other: %v", err)
	}
	if stringSliceContains(other.DependsOn, "target") {
		t.Errorf("other.depends_on should have 'target' scrubbed, got %v", other.DependsOn)
	}
	if svc.store.Exists("target") {
		t.Errorf("target should be deleted")
	}
}

// TestAddItems_RejectsItemsInDifferentInitiative verifies AddItems refuses
// to silently steal items from another initiative. Callers must use PATCH
// on the item to move it.
func TestAddItems_RejectsItemsInDifferentInitiative(t *testing.T) {
	items := map[string]backlog.BacklogItem{
		"idea/taken": {Kind: "idea", Name: "taken", Title: "Taken", Status: "backlog", Priority: 5, Initiative: "owner"},
	}
	svc := newTestService(t, items)

	if _, err := svc.Create(CreateRequest{Name: "owner", Title: "Owner", Items: []string{"idea/taken"}}); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "thief", Title: "Thief"}); err != nil {
		t.Fatalf("create thief: %v", err)
	}

	err := svc.AddItems("thief", []string{"idea/taken"})
	if err == nil {
		t.Fatalf("expected AddItems to reject cross-initiative attach, got nil")
	}
	if !strings.Contains(err.Error(), "already belongs to initiative") {
		t.Errorf("expected 'already belongs to initiative' error, got %q", err.Error())
	}

	thief, err := svc.store.Load("thief")
	if err != nil {
		t.Fatalf("load thief: %v", err)
	}
	if len(thief.Items) != 0 {
		t.Errorf("thief.items should be untouched on reject, got %v", thief.Items)
	}
}

// TestAddItems_SetsItemInitiativeField verifies AddItems sets the item's
// initiative field so the two sides stay symmetric.
func TestAddItems_SetsItemInitiativeField(t *testing.T) {
	items := map[string]backlog.BacklogItem{
		"idea/orphan": {Kind: "idea", Name: "orphan", Title: "Orphan", Status: "backlog", Priority: 5, Initiative: ""},
	}
	svc := newTestService(t, items)

	if _, err := svc.Create(CreateRequest{Name: "home", Title: "Home"}); err != nil {
		t.Fatalf("create home: %v", err)
	}
	if err := svc.AddItems("home", []string{"idea/orphan"}); err != nil {
		t.Fatalf("AddItems: %v", err)
	}

	loader := svc.backlogLoader.(*mockBacklogLoader)
	if got := loader.items["idea/orphan"].Initiative; got != "home" {
		t.Errorf("idea/orphan.initiative should be 'home' after AddItems, got %q", got)
	}
}

// TestRemoveItems_ClearsItemInitiativeField verifies RemoveItems clears
// the item's initiative field when it matches this initiative.
func TestRemoveItems_ClearsItemInitiativeField(t *testing.T) {
	items := map[string]backlog.BacklogItem{
		"idea/member": {Kind: "idea", Name: "member", Title: "Member", Status: "backlog", Priority: 5, Initiative: "home"},
	}
	svc := newTestService(t, items)

	if _, err := svc.Create(CreateRequest{Name: "home", Title: "Home", Items: []string{"idea/member"}}); err != nil {
		t.Fatalf("create home: %v", err)
	}

	if err := svc.RemoveItems("home", []string{"idea/member"}); err != nil {
		t.Fatalf("RemoveItems: %v", err)
	}

	loader := svc.backlogLoader.(*mockBacklogLoader)
	if got := loader.items["idea/member"].Initiative; got != "" {
		t.Errorf("idea/member.initiative should be cleared after RemoveItems, got %q", got)
	}
	home, err := svc.store.Load("home")
	if err != nil {
		t.Fatalf("load home: %v", err)
	}
	if len(home.Items) != 0 {
		t.Errorf("home.items should be empty after RemoveItems, got %v", home.Items)
	}
}

// TestRememberItem_Idempotent verifies RememberItem is a no-op when the
// ref is already present (keeps symmetric batch create from double-adding).
func TestRememberItem_Idempotent(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "init", Title: "Init", Items: []string{"idea/x"}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.RememberItem("init", "idea/x"); err != nil {
		t.Fatalf("RememberItem (duplicate): %v", err)
	}
	got, _ := svc.store.Load("init")
	if len(got.Items) != 1 {
		t.Errorf("items should remain [idea/x], got %v", got.Items)
	}
}

// TestForgetItem_RemovesRefOnly verifies ForgetItem removes the ref from
// initiative.items[] without touching the item side.
func TestForgetItem_RemovesRefOnly(t *testing.T) {
	items := map[string]backlog.BacklogItem{
		"idea/x": {Kind: "idea", Name: "x", Title: "X", Status: "backlog", Priority: 5, Initiative: "init"},
	}
	svc := newTestService(t, items)
	if _, err := svc.Create(CreateRequest{Name: "init", Title: "Init", Items: []string{"idea/x"}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ForgetItem("init", "idea/x"); err != nil {
		t.Fatalf("ForgetItem: %v", err)
	}
	got, _ := svc.store.Load("init")
	if len(got.Items) != 0 {
		t.Errorf("items should be empty, got %v", got.Items)
	}
	loader := svc.backlogLoader.(*mockBacklogLoader)
	if initField := loader.items["idea/x"].Initiative; initField != "init" {
		t.Errorf("item.initiative should be untouched by ForgetItem, got %q", initField)
	}
}
