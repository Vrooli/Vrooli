package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferRequirementsResolvesSkillIdentifiersAndReportsUnknowns(t *testing.T) {
	reqs, unresolved := InferRequirements("Use `prompt-manager skill read known-skill` and run `vrooli scenario test demo`.", map[string]bool{"known-skill": true})
	if len(unresolved) != 0 || len(reqs.Scenarios) != 2 || len(reqs.Commands) != 3 {
		t.Fatalf("unexpected requirements: %#v unresolved=%v", reqs, unresolved)
	}
	_, unresolved = InferRequirements("prompt-manager skill read missing-skill", map[string]bool{})
	if len(unresolved) != 1 || unresolved[0] != "skill:missing-skill" {
		t.Fatalf("expected unresolved identifier, got %v", unresolved)
	}
}

func TestReplaceFrontmatterRequirementsPreservesBody(t *testing.T) {
	content := "---\nname: demo\ndescription: demo\nmetadata:\n  requires:\n    scenarios: []\n    commands: []\n  origin:\n    kind: authored\n---\n\nBody\n"
	updated, err := replaceFrontmatterRequirements(content, Requirements{Scenarios: []string{"prompt-manager"}, Commands: []string{"prompt-manager skill read"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated, `commands: ["prompt-manager skill read"]`) || !strings.HasSuffix(updated, "Body\n") {
		t.Fatalf("requirements/body not preserved: %s", updated)
	}
}

func TestSyncCorpusRequirementsSkipsThirdPartyVendorPack(t *testing.T) {
	root := t.TempDir()
	vendorSkill := filepath.Join(root, "packs", "vendor", "brag", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(vendorSkill), 0o755); err != nil {
		t.Fatal(err)
	}
	pinned := "---\nname: brag\ndescription: third-party\n---\n\nRun npx hyperframes check.\n"
	if err := os.WriteFile(vendorSkill, []byte(pinned), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncCorpusRequirements(root); err != nil {
		t.Fatalf("vendor skill without a requires block must not fail the sync: %v", err)
	}
	if got, _ := os.ReadFile(vendorSkill); string(got) != pinned {
		t.Fatalf("pinned third-party bytes changed:\n%s", got)
	}
}
