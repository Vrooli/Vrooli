package memberflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriterSkillRegistry_AllTaggedHaveWritesTo enforces the Pillar 2 canon
// rule: every skill tagged `writer-skill` in the live store must declare a
// non-empty `writes_to[]` array. Without this, the prose-scan join falls
// back to the strict "no topic strings allowed" rule for that skill, which
// silently re-categorizes a writer skill as a generic skill — exactly the
// drift Pillar 2 was built to prevent.
//
// Mirrors classifier_purity_test.go's belt-and-suspenders approach: keep
// the runtime validator as the catch-all, but lock the registry shape with
// a test that fails fast in CI.
func TestWriterSkillRegistry_AllTaggedHaveWritesTo(t *testing.T) {
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}

	skills, err := loadAllSkillJSON(storeDir)
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}

	checked := 0
	for id, s := range skills {
		if !hasTag(s.Tags, "writer-skill") {
			continue
		}
		checked++
		if len(s.WritesTo) == 0 {
			t.Errorf("skill %q is tagged 'writer-skill' but has empty writes_to[] (file: %s); declare every topic prefix the skill writes (e.g. \"bug-inbox/*\")", id, s.path)
			continue
		}
		for i, prefix := range s.WritesTo {
			trimmed := strings.TrimSpace(prefix)
			if trimmed == "" {
				t.Errorf("skill %q writes_to[%d] is empty/whitespace (file: %s)", id, i, s.path)
				continue
			}
			if trimmed != prefix {
				t.Errorf("skill %q writes_to[%d] = %q has surrounding whitespace (file: %s)", id, i, prefix, s.path)
			}
			if strings.Contains(prefix, " ") {
				t.Errorf("skill %q writes_to[%d] = %q contains a space (file: %s); topic prefixes are kebab-case path segments", id, i, prefix, s.path)
			}
		}
	}

	if checked == 0 {
		t.Errorf("no writer-skill tagged skills found in registry; expected at least report-bug, report-friction, morning-vision-walk")
	}
}

// TestWriterSkillRegistry_WritesToOnlyOnWriterTagged is the inverse rule:
// a skill that is NOT tagged `writer-skill` must not declare `writes_to[]`,
// because the prose scanner will not consult it. Catching this in the
// registry test prevents a silent author mistake (forgetting the tag while
// adding writes_to[]) that would look "fine" in skill.json review but
// cause every topic ref in the skill's prose to fire.
func TestWriterSkillRegistry_WritesToOnlyOnWriterTagged(t *testing.T) {
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}

	skills, err := loadAllSkillJSON(storeDir)
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}

	for id, s := range skills {
		if hasTag(s.Tags, "writer-skill") {
			continue
		}
		if len(s.WritesTo) > 0 {
			t.Errorf("skill %q declares writes_to[] but is not tagged 'writer-skill' (file: %s); the prose scanner only consults writes_to[] for writer-tagged skills, so this is dead config — add the tag or remove the field", id, s.path)
		}
	}
}

// TestBuildProseSkillIndex_ReadsTagsAndWritesTo is the focused unit test for
// the bridge between skill.json and the prose scanner: the index must
// surface both the writer-skill tag (via IsWriter) and the writes_to[]
// payload (via WritesTo) as a single coherent record.
func TestBuildProseSkillIndex_ReadsTagsAndWritesTo(t *testing.T) {
	root := t.TempDir()

	mk := func(rel, body string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mk("scenarios/prompt-manager/store/skills/packs/core/writer-with-decl/skill.json",
		`{"id":"writer-with-decl","tags":["skill","writer-skill"],"writes_to":["foo/*","bar/*"]}`)
	mk("scenarios/prompt-manager/store/skills/packs/core/writer-without-decl/skill.json",
		`{"id":"writer-without-decl","tags":["skill","writer-skill"]}`)
	mk("scenarios/prompt-manager/store/skills/packs/core/generic-skill/skill.json",
		`{"id":"generic-skill","tags":["skill","practice"]}`)
	mk("scenarios/prompt-manager/store/skills/packs/core/malformed-skill/skill.json",
		`{not valid json`)

	idx := buildProseSkillIndex([]string{root})

	cases := []struct {
		id           string
		wantInIndex  bool
		wantIsWriter bool
		wantWritesTo []string
	}{
		{"writer-with-decl", true, true, []string{"foo/*", "bar/*"}},
		{"writer-without-decl", true, true, nil},
		{"generic-skill", true, false, nil},
		{"malformed-skill", false, false, nil},
	}
	for _, c := range cases {
		entry, ok := idx[c.id]
		if ok != c.wantInIndex {
			t.Errorf("skill %q: in index = %v, want %v", c.id, ok, c.wantInIndex)
			continue
		}
		if !ok {
			continue
		}
		if entry.IsWriter != c.wantIsWriter {
			t.Errorf("skill %q: IsWriter = %v, want %v", c.id, entry.IsWriter, c.wantIsWriter)
		}
		if !equalStringSlice(entry.WritesTo, c.wantWritesTo) {
			t.Errorf("skill %q: WritesTo = %v, want %v", c.id, entry.WritesTo, c.wantWritesTo)
		}
	}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

type rawSkillJSON struct {
	ID       string   `json:"id"`
	Tags     []string `json:"tags"`
	WritesTo []string `json:"writes_to"`
	path     string   // populated by loader
}

// loadAllSkillJSON walks <storeDir>/skills/packs/<pack>/<id>/skill.json and
// returns id -> raw record. Mirrors LoadSkillIDs / LoadSkillPaths but
// extracts the fields the writer-skill registry tests need.
func loadAllSkillJSON(storeDir string) (map[string]rawSkillJSON, error) {
	out := map[string]rawSkillJSON{}
	packsDir := filepath.Join(storeDir, "skills", "packs")
	packs, err := os.ReadDir(packsDir)
	if err != nil {
		return nil, err
	}
	for _, p := range packs {
		if !p.IsDir() {
			continue
		}
		skillRoots, err := os.ReadDir(filepath.Join(packsDir, p.Name()))
		if err != nil {
			continue
		}
		for _, s := range skillRoots {
			if !s.IsDir() {
				continue
			}
			path := filepath.Join(packsDir, p.Name(), s.Name(), "skill.json")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var raw rawSkillJSON
			if err := json.Unmarshal(data, &raw); err != nil {
				continue
			}
			raw.path = path
			out[s.Name()] = raw
		}
	}
	return out, nil
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
