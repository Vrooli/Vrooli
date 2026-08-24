package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSidecarUsesFrontmatterAsSourceOfTruth(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "example")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	body := "---\nname: \"example\"\ndescription: \"Current description\"\n---\nbody\n"
	if err := os.WriteFile(skillPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	old := `{"id":"example","description":"stale","custom":"preserve"}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateSidecar(skillPath); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	data, err := os.ReadFile(filepath.Join(skillDir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["description"] != "Current description" || got["custom"] != "preserve" {
		t.Fatalf("generated sidecar did not use frontmatter and preserve metadata: %#v", got)
	}
}

func TestGenerateSidecarReconstructsMetadataWhenSidecarIsMissing(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "example")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	body := "---\nname: example\ndescription: Current description\nlicense: CC-BY-4.0\nmetadata:\n  kind: skill\n  tags: [practice, topology]\n  requires:\n    commands: [prompt-manager skill]\n---\nbody\n"
	if err := os.WriteFile(skillPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateSidecar(skillPath); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	data, err := os.ReadFile(filepath.Join(skillDir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["license"] != "CC-BY-4.0" || got["kind"] != "skill" {
		t.Fatalf("generated sidecar lost top-level metadata: %#v", got)
	}
	tags, ok := got["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "practice" {
		t.Fatalf("generated sidecar lost metadata tags: %#v", got["tags"])
	}
	requires, ok := got["requires"].(map[string]any)
	if !ok || requires["commands"].([]any)[0] != "prompt-manager skill" {
		t.Fatalf("generated sidecar lost nested requirements: %#v", got["requires"])
	}
}

func TestParseFrontmatterRejectsLegacySkill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("legacy body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFrontmatter(path); err == nil || !strings.Contains(err.Error(), "missing YAML frontmatter") {
		t.Fatalf("expected missing-frontmatter error, got %v", err)
	}
}

func TestValidateCorpusUsesDiscoveredSet(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"one", "two"} {
		path := filepath.Join(root, id, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: \"" + id + "\"\ndescription: \"A skill\"\n---\nbody\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := ValidateCorpus(root); err != nil {
		t.Fatal(err)
	}
}

// A skill directory that lost its body is the exact damage the corpus guard
// exists to catch. Deriving both sides of the count from one walk over the
// files that still exist cannot see it, so the guard asserts over directories.
func TestValidateCorpusDetectsSkillDirectoryWithoutBody(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(good), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("---\nname: \"good\"\ndescription: \"A skill\"\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(root, "orphan")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatal(err)
	}
	sidecar, err := json.Marshal(map[string]string{"id": "orphan", "entry": "SKILL.md"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "skill.json"), sidecar, 0o644); err != nil {
		t.Fatal(err)
	}

	err = ValidateCorpus(root)
	if err == nil {
		t.Fatal("expected a sidecar without a body to fail validation")
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("error must name the offending directory, got %v", err)
	}
	if strings.Contains(err.Error(), filepath.Join(root, "good")) {
		t.Fatalf("error must not implicate the healthy directory, got %v", err)
	}
}

func TestPromptManagerSkillCorpusHasSpecificationFrontmatter(t *testing.T) {
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(working, "..", "..", "..", "store", "skills", "packs")
	if err := ValidateCorpus(root); err != nil {
		t.Fatal(err)
	}
}
