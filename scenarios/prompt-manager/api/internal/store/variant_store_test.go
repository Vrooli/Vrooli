package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// setupSkillForVariantTest creates a skill store with a test skill, returning the store dir.
func setupSkillForVariantTest(t *testing.T) (string, *FileSkillStore) {
	t.Helper()
	storeDir := t.TempDir()

	// Create pack order
	packOrderDir := filepath.Join(storeDir, "skills")
	if err := os.MkdirAll(packOrderDir, 0o755); err != nil {
		t.Fatal(err)
	}
	order := &PackOrder{
		ActivePacks:   []string{"local", "core"},
		InactivePacks: []string{"drafts"},
	}
	if err := SaveJSON(filepath.Join(packOrderDir, "_pack-order.json"), order); err != nil {
		t.Fatal(err)
	}

	skillStore := NewFileSkillStore(storeDir)
	ctx := context.Background()

	// Create a test skill
	skill := &Skill{
		ID:   "test-skill",
		Name: "Test Skill",
	}
	if err := skillStore.Create(ctx, "local", skill, "# Test Skill\nOriginal content"); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	return storeDir, skillStore
}

func TestVariantStore_CreateAndGet(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	variant := &Variant{
		ID:          "concise-v1",
		Name:        "Concise Style",
		Description: "A more concise prompt variant",
	}
	content := "# Concise\nShort and sweet."

	if err := vs.Create(ctx, "test-skill", variant, content); err != nil {
		t.Fatalf("create variant: %v", err)
	}

	got, err := vs.Get(ctx, "test-skill", "concise-v1")
	if err != nil {
		t.Fatalf("get variant: %v", err)
	}

	if got.Name != "Concise Style" {
		t.Errorf("expected name %q, got %q", "Concise Style", got.Name)
	}
	if got.Kind != KindVariant {
		t.Errorf("expected kind %q, got %q", KindVariant, got.Kind)
	}
	if got.SkillID != "test-skill" {
		t.Errorf("expected skillId %q, got %q", "test-skill", got.SkillID)
	}
	if got.Entry != "VARIANT.md" {
		t.Errorf("expected entry %q, got %q", "VARIANT.md", got.Entry)
	}
	if got.Revision != 1 {
		t.Errorf("expected revision 1, got %d", got.Revision)
	}
}

func TestVariantStore_GetWithContent(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	variant := &Variant{ID: "detailed-v1", Name: "Detailed Style"}
	content := "# Detailed\nLong and thorough."

	if err := vs.Create(ctx, "test-skill", variant, content); err != nil {
		t.Fatalf("create variant: %v", err)
	}

	got, gotContent, err := vs.GetWithContent(ctx, "test-skill", "detailed-v1")
	if err != nil {
		t.Fatalf("get with content: %v", err)
	}

	if got.Name != "Detailed Style" {
		t.Errorf("expected name %q, got %q", "Detailed Style", got.Name)
	}
	if gotContent != content {
		t.Errorf("expected content %q, got %q", content, gotContent)
	}
}

func TestVariantStore_List(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	// Initially empty
	variants, err := vs.List(ctx, "test-skill")
	if err != nil {
		t.Fatalf("list variants: %v", err)
	}
	if len(variants) != 0 {
		t.Errorf("expected 0 variants, got %d", len(variants))
	}

	// Create two variants
	if err := vs.Create(ctx, "test-skill", &Variant{ID: "v1", Name: "V1"}, "content1"); err != nil {
		t.Fatal(err)
	}
	if err := vs.Create(ctx, "test-skill", &Variant{ID: "v2", Name: "V2"}, "content2"); err != nil {
		t.Fatal(err)
	}

	variants, err = vs.List(ctx, "test-skill")
	if err != nil {
		t.Fatalf("list variants: %v", err)
	}
	if len(variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(variants))
	}
}

func TestVariantStore_Update(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	if err := vs.Create(ctx, "test-skill", &Variant{ID: "v1", Name: "Old Name"}, "old content"); err != nil {
		t.Fatal(err)
	}

	newContent := "new content"
	if err := vs.Update(ctx, "test-skill", "v1", &Variant{Name: "New Name"}, &newContent); err != nil {
		t.Fatalf("update variant: %v", err)
	}

	got, gotContent, err := vs.GetWithContent(ctx, "test-skill", "v1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New Name" {
		t.Errorf("expected name %q, got %q", "New Name", got.Name)
	}
	if gotContent != "new content" {
		t.Errorf("expected content %q, got %q", "new content", gotContent)
	}
	if got.Revision != 2 {
		t.Errorf("expected revision 2, got %d", got.Revision)
	}
}

func TestVariantStore_Delete(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	if err := vs.Create(ctx, "test-skill", &Variant{ID: "v1", Name: "V1"}, "content"); err != nil {
		t.Fatal(err)
	}

	if err := vs.Delete(ctx, "test-skill", "v1"); err != nil {
		t.Fatalf("delete variant: %v", err)
	}

	if _, err := vs.Get(ctx, "test-skill", "v1"); err == nil {
		t.Error("expected error getting deleted variant")
	}
}

func TestVariantStore_RejectControlID(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	err := vs.Create(ctx, "test-skill", &Variant{ID: ControlVariantID, Name: "Control"}, "content")
	if err == nil {
		t.Error("expected error creating variant with reserved control ID")
	}
}

func TestVariantStore_SkillNotFound(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	if _, err := vs.List(ctx, "nonexistent"); err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestVariantStore_DuplicateCreate(t *testing.T) {
	_, skillStore := setupSkillForVariantTest(t)
	vs := NewFileVariantStore(skillStore)
	ctx := context.Background()

	if err := vs.Create(ctx, "test-skill", &Variant{ID: "v1", Name: "V1"}, "content"); err != nil {
		t.Fatal(err)
	}

	err := vs.Create(ctx, "test-skill", &Variant{ID: "v1", Name: "V1 Dup"}, "content2")
	if err == nil {
		t.Error("expected error creating duplicate variant")
	}
}
