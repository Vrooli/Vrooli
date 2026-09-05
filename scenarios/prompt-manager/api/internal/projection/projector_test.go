package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRejectsQuarantinedImportedSkill(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	dir := filepath.Join(source, "vendor-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: vendor-skill\ndescription: pending\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{"origin": map[string]any{"kind": "imported", "review": map[string]string{"verdict": "pending"}}}
	encoded, _ := json.Marshal(metadata)
	if err := os.WriteFile(filepath.Join(dir, "skill.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Project(source, target, BasePack{Skills: []string{"vendor-skill"}, MaxSkills: 1, MaxTokens: 100}); err == nil {
		t.Fatal("expected quarantined skill rejection")
	}
}

func TestProjectIsScopedIdempotentAndReapsGeneratedSkills(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	for _, id := range []string{"alpha", "beta"} {
		dir := filepath.Join(source, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: \"" + id + "\"\ndescription: \"A skill\"\n---\nbody\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pack := BasePack{Skills: []string{"alpha", "beta"}, MaxSkills: 8, MaxTokens: 1000}
	first, err := Project(source, target, pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Skills) != 2 || first.ResidentTokens == 0 {
		t.Fatalf("unexpected first result: %#v", first)
	}
	before, err := os.ReadFile(filepath.Join(target, "alpha", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Project(source, target, BasePack{Skills: []string{"alpha"}, MaxSkills: 8, MaxTokens: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Removed) != 1 || second.Removed[0] != "beta" {
		t.Fatalf("expected beta reap, got %#v", second.Removed)
	}
	after, err := os.ReadFile(filepath.Join(target, "alpha", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("projection was not idempotent")
	}
}

func TestProjectRejectsResidentOverageBeforeWriting(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	dir := filepath.Join(source, "large")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: large\ndescription: this description is deliberately over the ceiling\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Project(source, target, BasePack{Skills: []string{"large"}, MaxSkills: 1, MaxTokens: 1}); err == nil {
		t.Fatal("expected resident-token ceiling refusal")
	}
	if _, err := os.Stat(filepath.Join(target, "large", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("over-budget projection wrote content: %v", err)
	}
}
