package memberflow

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestLoadWriterSkillProducers_UnionsTaggedWritesTo asserts the loader walks
// every skill.json under <repoRoot>/scenarios/prompt-manager/store/skills/packs
// and returns the sorted union of writes_to[] entries from skills tagged
// "writer-skill". Untagged skills are ignored even when they declare
// writes_to[]. Empty / whitespace prefixes are filtered out.
func TestLoadWriterSkillProducers_UnionsTaggedWritesTo(t *testing.T) {
	root := t.TempDir()
	packsDir := filepath.Join(root, "scenarios", "prompt-manager", "store", "skills", "packs")

	writeSkill := func(pack, id, body string) {
		t.Helper()
		dir := filepath.Join(packsDir, pack, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skill.json"), []byte(body), 0o644); err != nil {
			t.Fatalf("write skill.json: %v", err)
		}
	}

	// Two writer skills with overlapping prefixes; the union should
	// dedupe.
	writeSkill("core", "report-bug",
		`{"id":"report-bug","tags":["skill","writer-skill"],"writes_to":["bug-inbox/*"]}`)
	writeSkill("core", "report-friction",
		`{"id":"report-friction","tags":["writer-skill","observability"],"writes_to":["friction-inbox/*"]}`)
	// Multi-prefix writer skill exercises the full union.
	writeSkill("core", "morning-vision-walk",
		`{"id":"morning-vision-walk","tags":["practice","writer-skill"],"writes_to":["research-inbox/*","opportunity-inbox/*","validation-inbox/*","vision-walk/*"]}`)
	// Non-writer skill with writes_to should be ignored.
	writeSkill("core", "non-writer-skill",
		`{"id":"non-writer","tags":["skill"],"writes_to":["should-be-ignored/*"]}`)
	// Writer skill with empty/whitespace entries — entries are filtered.
	writeSkill("core", "edge-cases",
		`{"id":"edge-cases","tags":["writer-skill"],"writes_to":["","valid/*"]}`)
	// Malformed skill.json — silently skipped (loader tolerance).
	writeSkill("core", "broken", `{not valid json`)

	got, err := LoadWriterSkillProducers(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"bug-inbox/*",
		"friction-inbox/*",
		"opportunity-inbox/*",
		"research-inbox/*",
		"valid/*",
		"validation-inbox/*",
		"vision-walk/*",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadWriterSkillProducers returned wrong set:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestLoadWriterSkillProducers_MissingTreeIsSilentNoOp confirms the loader
// returns (nil, nil) — not an error — when the skills tree is absent. The
// rule consumer treats nil as "no writer-skill producers known," which is
// the safe default for tests and bare scaffolds.
func TestLoadWriterSkillProducers_MissingTreeIsSilentNoOp(t *testing.T) {
	root := t.TempDir()
	got, err := LoadWriterSkillProducers(root)
	if err != nil {
		t.Errorf("missing skills tree must not error; got %v", err)
	}
	if got != nil {
		t.Errorf("missing skills tree must return nil; got %v", got)
	}
}

// TestLoadWriterSkillProducers_EmptyRepoRootIsNoOp confirms empty repoRoot
// short-circuits to (nil, nil), keeping the lazy-load in Validate a silent
// no-op when no RepoRoot is configured.
func TestLoadWriterSkillProducers_EmptyRepoRootIsNoOp(t *testing.T) {
	got, err := LoadWriterSkillProducers("")
	if err != nil || got != nil {
		t.Errorf("empty repoRoot must return (nil, nil); got (%v, %v)", got, err)
	}
}
