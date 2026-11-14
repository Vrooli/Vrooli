package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/internal/store"
)

type countedSkillRegistry struct {
	*MockSkillStore
	lists       int
	lastContext context.Context
}

func (s *countedSkillRegistry) List(ctx context.Context) ([]store.Skill, error) {
	s.lists++
	s.lastContext = ctx
	return s.MockSkillStore.List(ctx)
}

func TestReadMetadataUsesOneFreshContextualRegistrySnapshot(t *testing.T) {
	registry := &countedSkillRegistry{MockSkillStore: NewMockSkillStore()}
	for i, folder := range append(append([]string(nil), Folders...), "outside-reader-folders") {
		registry.WithSkill(&store.Skill{ID: fmt.Sprintf("skill-%d", i), Name: "Shared name", Pack: folder})
	}
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "lease-identity")
	adapter := NewStoreAdapter(registry, nil).WithContext(ctx)
	indexed, err := loadIndexedSkills(adapter)
	if err != nil {
		t.Fatal(err)
	}
	if registry.lists != 1 {
		t.Fatalf("one read enumerated the registry %d times; want one fresh snapshot", registry.lists)
	}
	if registry.lastContext.Value(contextKey{}) != "lease-identity" {
		t.Fatal("request context was lost")
	}
	if len(indexed) != len(Folders) {
		t.Fatalf("reader folder count=%d", len(indexed))
	}
	for i, folder := range Folders {
		if indexed[i].folder != folder {
			t.Fatalf("folder order=%v", indexed)
		}
	}
	if matches := resolveIdentifier("SHARED NAME", "name", indexed); len(matches) != len(Folders) {
		t.Fatal("name ambiguity was lost")
	}
	if matches := resolveIdentifier("CORE/skill-0.md", "file", indexed); len(matches) != 1 || matches[0].meta.ID != "skill-0" {
		t.Fatal("file resolution changed")
	}
	registry.skills["skill-0"].Name = "Edited immediately"
	next, err := loadIndexedSkills(adapter)
	if err != nil {
		t.Fatal(err)
	}
	if registry.lists != 2 || next[0].meta.Name != "Edited immediately" {
		t.Fatal("a previous request snapshot hid a source edit")
	}
	registry.listErr = errors.New("registry unavailable")
	if _, err := loadIndexedSkills(adapter); err == nil {
		t.Fatal("a previous snapshot hid a registry error")
	}
}

// Hiding the optional batch capability exercises the existing folder interface
// against the very same real registry; no copied legacy implementation.
type folderMetadataStore struct{ SkillStore }

func BenchmarkReadMetadataRegistrySnapshot(b *testing.B) {
	root := b.TempDir()
	for i := range 300 {
		id := fmt.Sprintf("skill-%03d", i)
		dir := filepath.Join(root, "skills", "packs", "core", id)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			b.Fatal(err)
		}
		skill := store.Skill{BaseEntity: store.BaseEntity{Kind: store.KindSkill, SchemaVersion: store.CurrentSchemaVersion}, ID: id, Name: "Synthetic registry metadata", Entry: "SKILL.md", Status: store.StatusActive}
		data, err := json.Marshal(skill)
		if err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skill.json"), data, 0o600); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("Synthetic benchmark skill"), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	adapter := NewStoreAdapter(store.NewFileSkillStore(root), nil)
	for _, variant := range []struct {
		name   string
		source SkillStore
	}{{"folder-interface", folderMetadataStore{adapter}}, {"batch-interface", adapter}} {
		b.Run(variant.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				skills, err := loadIndexedSkills(variant.source)
				if err != nil {
					b.Fatal(err)
				}
				if len(skills) != 300 {
					b.Fatalf("skills=%d", len(skills))
				}
			}
		})
	}
}
