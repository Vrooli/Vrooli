package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScenarioSkill(t *testing.T, root, scenario, id string) {
	t.Helper()
	dir := filepath.Join(root, scenario, "skills", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + id + "\ndescription: scenario skill\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFileSkillStoreIndexesScenarioOwnedRoots(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	writeScenarioSkill(t, scenarios, "hello-plugin", "hello")
	writeScenarioSkill(t, scenarios, "workspace-sandbox", "sandbox")
	store := NewFileSkillStoreWithScenarioRoots(config, scenarios)
	items, err := store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.ID] = true
	}
	if !seen["hello"] || !seen["sandbox"] {
		t.Fatalf("scenario-owned skills missing: %#v", seen)
	}
	_, content, err := store.GetWithContent(t.Context(), "hello")
	if err != nil || !strings.Contains(content, "body") {
		t.Fatalf("read scenario skill: %q %v", content, err)
	}
}

func TestFileSkillStoreRejectsDuplicateScenarioSkillIdentifiers(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	writeScenarioSkill(t, scenarios, "one", "duplicate")
	writeScenarioSkill(t, scenarios, "two", "duplicate")
	_, err := NewFileSkillStoreWithScenarioRoots(config, scenarios).List(t.Context())
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate root error, got %v", err)
	}
}

func TestScenarioSkillCacheRebuildsFromRootsAfterDeletion(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	config := filepath.Join(root, "config")
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	writeScenarioSkill(t, scenarios, "hello-plugin", "hello")
	s := NewFileSkillStoreWithScenarioRootsAndCache(config, cache, scenarios)
	if err := s.RebuildScenarioSkillCache(t.Context()); err != nil {
		t.Fatal(err)
	}
	path := s.ScenarioSkillCachePath()
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildScenarioSkillCache(t.Context()); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("cache rebuild changed content:\nfirst=%s\nsecond=%s", first, second)
	}
}
