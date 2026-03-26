package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTopicStore(t *testing.T) (*FileTopicStore, string) {
	t.Helper()
	dir := t.TempDir()
	topicsDir := filepath.Join(dir, "topics")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return NewFileTopicStore(dir), dir
}

func createTestTopic(t *testing.T, store *FileTopicStore, id, name string, parentID *string, skills []string) {
	t.Helper()
	topic := &Topic{
		ID:            id,
		Name:          name,
		ParentTopicID: parentID,
		Skills:        skills,
	}
	if err := store.Create(context.Background(), topic, ""); err != nil {
		t.Fatalf("creating topic %s: %v", id, err)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestTopicStore_CRUD(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	// Create
	topic := &Topic{
		ID:          "test-topic",
		Name:        "Test Topic",
		Description: "A test topic",
		Skills:      []string{"ux", "documentation-health"},
		Icon:        "book",
	}
	if err := store.Create(ctx, topic, "# Test Topic\n\nSome content."); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify kind and timestamps were set
	got, err := store.Get(ctx, "test-topic")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Kind != KindTopic {
		t.Errorf("Kind = %q, want %q", got.Kind, KindTopic)
	}
	if got.Status != StatusActive {
		t.Errorf("Status = %q, want %q", got.Status, StatusActive)
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}

	// GetWithContent
	_, content, err := store.GetWithContent(ctx, "test-topic")
	if err != nil {
		t.Fatalf("GetWithContent: %v", err)
	}
	if content != "# Test Topic\n\nSome content." {
		t.Errorf("content = %q, want %q", content, "# Test Topic\n\nSome content.")
	}

	// List
	topics, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(topics) != 1 {
		t.Errorf("List returned %d topics, want 1", len(topics))
	}

	// Update
	newContent := "# Updated"
	updates := &Topic{Name: "Updated Topic", Skills: []string{"api-steer"}}
	if err := store.Update(ctx, "test-topic", updates, &newContent); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = store.Get(ctx, "test-topic")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "Updated Topic" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Topic")
	}
	if len(got.Skills) != 1 || got.Skills[0] != "api-steer" {
		t.Errorf("Skills = %v, want [api-steer]", got.Skills)
	}

	// Delete
	if err := store.Delete(ctx, "test-topic"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, "test-topic"); err == nil {
		t.Error("Get after delete should fail")
	}
}

func TestTopicStore_DuplicateCreate(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "dup", "Dup", nil, nil)

	err := store.Create(ctx, &Topic{ID: "dup", Name: "Dup2"}, "")
	if err == nil {
		t.Error("duplicate create should fail")
	}
}

func TestTopicStore_SelfParent(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "self", "Self", nil, nil)

	updates := &Topic{ParentTopicID: strPtr("self")}
	if err := store.Update(ctx, "self", updates, nil); err == nil {
		t.Error("self-referencing parent should fail")
	}
}

func TestTopicStore_GetAncestors(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "root", "Root", nil, []string{"doc-health"})
	createTestTopic(t, store, "child", "Child", strPtr("root"), []string{"ux"})
	createTestTopic(t, store, "grandchild", "Grandchild", strPtr("child"), []string{"perf"})

	ancestors, err := store.GetAncestors(ctx, "grandchild")
	if err != nil {
		t.Fatalf("GetAncestors: %v", err)
	}
	if len(ancestors) != 2 {
		t.Fatalf("GetAncestors returned %d, want 2", len(ancestors))
	}
	if ancestors[0].ID != "child" {
		t.Errorf("first ancestor = %q, want %q", ancestors[0].ID, "child")
	}
	if ancestors[1].ID != "root" {
		t.Errorf("second ancestor = %q, want %q", ancestors[1].ID, "root")
	}
}

func TestTopicStore_GetAncestors_CycleDetection(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	// Create a cycle: a -> b -> a (requires manual file manipulation)
	createTestTopic(t, store, "a", "A", nil, nil)
	createTestTopic(t, store, "b", "B", strPtr("a"), nil)

	// Now manually update a's parent to b to create a cycle
	topicA, _ := store.Get(ctx, "a")
	topicA.ParentTopicID = strPtr("b")
	topicPath := filepath.Join(store.topicsDir(), "a", "topic.json")
	if err := SaveJSON(topicPath, topicA); err != nil {
		t.Fatal(err)
	}

	// GetAncestors should terminate without infinite loop
	ancestors, err := store.GetAncestors(ctx, "a")
	if err != nil {
		t.Fatalf("GetAncestors with cycle: %v", err)
	}
	// Should find b but then stop (since a is already visited)
	if len(ancestors) != 1 {
		t.Errorf("GetAncestors with cycle returned %d, want 1", len(ancestors))
	}
}

func TestTopicStore_GetChildren(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "parent", "Parent", nil, nil)
	createTestTopic(t, store, "child1", "Child 1", strPtr("parent"), nil)
	createTestTopic(t, store, "child2", "Child 2", strPtr("parent"), nil)
	createTestTopic(t, store, "other", "Other", nil, nil)

	children, err := store.GetChildren(ctx, "parent")
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("GetChildren returned %d, want 2", len(children))
	}
}

func TestTopicStore_AccumulateSkills(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "root", "Root", nil, []string{"doc-health", "seam-discovery"})
	createTestTopic(t, store, "mid", "Mid", strPtr("root"), []string{"ux", "doc-health"}) // doc-health is duplicate
	createTestTopic(t, store, "leaf", "Leaf", strPtr("mid"), []string{"performance"})

	skills, err := store.AccumulateSkills(ctx, "leaf")
	if err != nil {
		t.Fatalf("AccumulateSkills: %v", err)
	}

	// Should be: performance + ux + doc-health + seam-discovery (deduplicated, doc-health appears once)
	if len(skills) != 4 {
		t.Errorf("AccumulateSkills returned %d skills, want 4: %v", len(skills), skills)
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, s := range skills {
		if seen[s] {
			t.Errorf("duplicate skill: %s", s)
		}
		seen[s] = true
	}
}

func TestTopicStore_UpdateCycleDetection(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "a", "A", nil, nil)
	createTestTopic(t, store, "b", "B", strPtr("a"), nil)
	createTestTopic(t, store, "c", "C", strPtr("b"), nil)

	// Try to set a's parent to c (would create a -> c -> b -> a cycle)
	updates := &Topic{ParentTopicID: strPtr("c")}
	if err := store.Update(ctx, "a", updates, nil); err == nil {
		t.Error("setting parent that creates a cycle should fail")
	}
}

func TestTopicStore_DeleteWithChildren(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	createTestTopic(t, store, "parent", "Parent", nil, nil)
	createTestTopic(t, store, "child", "Child", strPtr("parent"), nil)

	// Delete parent should succeed (child becomes orphaned)
	if err := store.Delete(ctx, "parent"); err != nil {
		t.Fatalf("Delete parent: %v", err)
	}

	// Child should still exist
	child, err := store.Get(ctx, "child")
	if err != nil {
		t.Fatalf("Get child after parent delete: %v", err)
	}
	if child.ParentTopicID == nil || *child.ParentTopicID != "parent" {
		t.Error("child should still reference deleted parent")
	}

	// GetAncestors for child should handle missing parent gracefully
	ancestors, err := store.GetAncestors(ctx, "child")
	if err != nil {
		t.Fatalf("GetAncestors with missing parent: %v", err)
	}
	if len(ancestors) != 0 {
		t.Errorf("GetAncestors with missing parent returned %d, want 0", len(ancestors))
	}
}

func TestTopicStore_ParentValidation(t *testing.T) {
	store, _ := setupTopicStore(t)
	ctx := context.Background()

	// Create with nonexistent parent should fail
	err := store.Create(ctx, &Topic{
		ID:            "bad-parent",
		Name:          "Bad Parent",
		ParentTopicID: strPtr("nonexistent"),
	}, "")
	if err == nil {
		t.Error("create with nonexistent parent should fail")
	}
}
