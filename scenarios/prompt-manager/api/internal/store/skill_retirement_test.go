package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSkillStoreRetireAndRestorePreservesSkillHistory(t *testing.T) {
	config := t.TempDir()
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	s := NewFileSkillStore(config)
	ctx := context.Background()
	if err := s.Create(ctx, "core", &Skill{ID: "retire-demo", Name: "Retire demo", Status: StatusActive}, "# useful content"); err != nil {
		t.Fatal(err)
	}
	historyPath := filepath.Join(config, "skills", "packs", "core", "retire-demo", "history.jsonl")
	before, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.Retire(ctx, "retire-demo", "operator", "verified no callers", []string{"audit://skill-candidate/retire-demo"})
	if err != nil {
		t.Fatal(err)
	}
	if record.SkillID != "retire-demo" || record.Pack != "core" || record.Revision != 1 {
		t.Fatalf("unexpected retirement record: %#v", record)
	}
	if _, err := s.Get(ctx, "retire-demo"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("retired skill remains discoverable: err=%v", err)
	}
	retiredDir := filepath.Join(config, "skills", "retired", "core", "retire-demo")
	after, err := os.ReadFile(filepath.Join(retiredDir, "history.jsonl"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("retirement changed or lost version history: err=%v\nbefore=%s\nafter=%s", err, before, after)
	}
	lifecyclePath := filepath.Join(retiredDir, "lifecycle.jsonl")
	lifecycle, err := os.ReadFile(lifecyclePath)
	if err != nil || !strings.Contains(string(lifecycle), `"action":"retired"`) || !strings.Contains(string(lifecycle), record.Evidence[0]) {
		t.Fatalf("retirement provenance not persisted: %s err=%v", lifecycle, err)
	}

	if err := s.Restore(ctx, "retire-demo", "operator", "retirement review reversed"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "retire-demo"); err != nil {
		t.Fatalf("restored skill is not discoverable: %v", err)
	}
	after, err = os.ReadFile(historyPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("restore changed or lost version history: err=%v\nbefore=%s\nafter=%s", err, before, after)
	}
	lifecycle, err = os.ReadFile(filepath.Join(config, "skills", "packs", "core", "retire-demo", "lifecycle.jsonl"))
	if err != nil || !strings.Contains(string(lifecycle), `"action":"retired"`) || !strings.Contains(string(lifecycle), `"action":"restored"`) {
		t.Fatalf("restore lifecycle history missing: %s err=%v", lifecycle, err)
	}
}

func TestFileSkillStoreRetireRequiresEvidenceAndRefusesImportedSkills(t *testing.T) {
	config := t.TempDir()
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core", "vendor"}}); err != nil {
		t.Fatal(err)
	}
	s := NewFileSkillStore(config)
	ctx := context.Background()
	for _, id := range []string{"needs-evidence"} {
		if err := s.Create(ctx, "core", &Skill{ID: id, Name: id, Status: StatusActive}, "# content"); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "vendor", "imported-demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	imported := &Skill{ID: "imported-demo", Name: "Imported", Status: StatusActive, Origin: &SkillOrigin{Kind: OriginImported, Review: SkillReview{Verdict: ReviewVerdictPassed}}}
	if err := SaveJSON(filepath.Join(config, "skills", "packs", "vendor", "imported-demo", "skill.json"), imported); err != nil {
		t.Fatal(err)
	}
	if err := WriteContent(filepath.Join(config, "skills", "packs", "vendor", "imported-demo", "SKILL.md"), "# imported"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Retire(ctx, "needs-evidence", "operator", "no callers", nil); err == nil {
		t.Fatal("retirement without evidence must be refused")
	}
	if _, err := s.Get(ctx, "needs-evidence"); err != nil {
		t.Fatalf("refused retirement changed active skill: %v", err)
	}
	if _, err := s.Retire(ctx, "imported-demo", "operator", "no callers", []string{"audit://imported-demo"}); err == nil || !strings.Contains(err.Error(), "imported") {
		t.Fatalf("imported retirement should be refused, got %v", err)
	}
}

func TestFileSkillStoreRetireRefusesScenarioOwnedSkill(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	scenariosRoot := filepath.Join(root, "scenarios")
	skillDir := filepath.Join(scenariosRoot, "example", "skills", "scenario-only")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: scenario-only\ndescription: owner maintained\n---\n\nOwned by its scenario.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewFileSkillStoreWithScenarioRoots(config, scenariosRoot)
	if _, err := s.Retire(context.Background(), "scenario-only", "operator", "unreferenced", []string{"audit://scenario-only"}); err == nil || !strings.Contains(err.Error(), "scenario-owned") {
		t.Fatalf("scenario-owned skill retirement should be refused, got %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md")); err != nil || string(got) != content {
		t.Fatalf("refused retirement changed owner source: content=%q err=%v", got, err)
	}
}
