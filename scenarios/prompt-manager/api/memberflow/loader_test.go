package memberflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeStore builds a minimal scenarios/prompt-manager/store-shaped tree
// rooted at t.TempDir() and returns the absolute store path.
func makeStore(t *testing.T, members map[string]map[string]string) string {
	t.Helper()
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	if err := os.MkdirAll(teams, 0o755); err != nil {
		t.Fatalf("mkdir teams: %v", err)
	}
	for ref, content := range mergeMembers(members) {
		dir := filepath.Join(teams, ref.Team, "members", ref.Member)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if content == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, "topics.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("write topics.json for %s: %v", ref, err)
		}
	}
	return root
}

func mergeMembers(in map[string]map[string]string) map[MemberRef]string {
	out := make(map[MemberRef]string)
	for team, members := range in {
		for member, content := range members {
			out[MemberRef{Team: team, Member: member}] = content
		}
	}
	return out
}

func TestLoadAll_ValidMixedTopics(t *testing.T) {
	store := makeStore(t, map[string]map[string]string{
		"marketing-crew": {
			"researcher": `{
				"intake": [{"prefix": "research-inbox/*", "taxonomy": "marketing-research", "classifier_skill": "signal-classifier"}],
				"output": [{"prefix": "audience-scan/*", "destination_kind": "knowledge", "schema": "audience-scan"}],
				"raises_capability_gaps": true
			}`,
			"publisher":     `{}`,
			"empty-no-file": "", // no file written
		},
		"monetization": {
			"opportunity-router": `{
				"intake": [{"prefix": "monetization-inbox/*", "taxonomy": "monetization-opportunity", "classifier_skill": "signal-classifier"}]
			}`,
		},
	})

	all, err := LoadAll(store)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if got, want := len(all), 4; got != want {
		t.Fatalf("len(all) = %d, want %d", got, want)
	}

	// Ordering is team, then member.
	wantOrder := []string{
		"marketing-crew/empty-no-file",
		"marketing-crew/publisher",
		"marketing-crew/researcher",
		"monetization/opportunity-router",
	}
	for i, m := range all {
		if got := m.Ref.String(); got != wantOrder[i] {
			t.Errorf("all[%d].Ref = %q, want %q", i, got, wantOrder[i])
		}
	}

	// no-file member: Exists=false, IsEmpty=true
	noFile := all[0]
	if noFile.Exists {
		t.Errorf("expected Exists=false for member without topics.json, got %+v", noFile)
	}
	if !noFile.Topics.IsEmpty() {
		t.Errorf("expected empty Topics for member without topics.json")
	}

	// {} member: Exists=true, IsEmpty=true
	pub := all[1]
	if !pub.Exists {
		t.Errorf("expected Exists=true for publisher with {} file")
	}
	if !pub.Topics.IsEmpty() {
		t.Errorf("expected empty Topics for {} file")
	}

	// researcher fully populated
	res := all[2]
	if !res.Exists {
		t.Errorf("expected researcher.Exists=true")
	}
	if len(res.Topics.Intake) != 1 || res.Topics.Intake[0].Taxonomy != "marketing-research" {
		t.Errorf("researcher intake mismatch: %+v", res.Topics.Intake)
	}
	if !res.Topics.RaisesCapabilityGaps {
		t.Errorf("researcher should raise capability gaps")
	}
}

func TestLoadAll_MalformedJSON(t *testing.T) {
	store := makeStore(t, map[string]map[string]string{
		"team-a": {
			"member-1": `{ this is not json`,
		},
	})
	_, err := LoadAll(store)
	if err == nil {
		t.Fatalf("LoadAll() returned nil, expected parse error")
	}
}

func TestLoadAll_SchemaViolation(t *testing.T) {
	store := makeStore(t, map[string]map[string]string{
		"team-a": {
			"member-1": `{"intake": [{"prefix": "*", "taxonomy": "x"}]}`,
		},
	})
	_, err := LoadAll(store)
	if err == nil {
		t.Fatalf("LoadAll() returned nil, expected validation error for bare-* prefix")
	}
}

func TestLoadAll_NoTeamsDir(t *testing.T) {
	root := t.TempDir() // no "teams" subdir
	_, err := LoadAll(root)
	if err == nil {
		t.Fatalf("LoadAll() on store without teams/ should error")
	}
}

func TestLoadAll_TeamWithoutMembersDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "lonely-team"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := LoadAll(root)
	if err != nil {
		t.Fatalf("LoadAll on team without members dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 members, got %d", len(got))
	}
}

func TestLoadTeam_FiltersByTeam(t *testing.T) {
	store := makeStore(t, map[string]map[string]string{
		"alpha": {"a": `{}`, "b": `{}`},
		"beta":  {"c": `{}`},
	})
	got, err := LoadTeam(store, "alpha")
	if err != nil {
		t.Fatalf("LoadTeam: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d members for alpha, want 2", len(got))
	}
	for _, m := range got {
		if m.Ref.Team != "alpha" {
			t.Errorf("LoadTeam(alpha) returned %s", m.Ref)
		}
	}
}

func TestLoadMember(t *testing.T) {
	store := makeStore(t, map[string]map[string]string{
		"team-a": {
			"member-1": `{"raises_capability_gaps": true}`,
		},
	})
	got, err := LoadMember(store, "team-a", "member-1")
	if err != nil {
		t.Fatalf("LoadMember: %v", err)
	}
	if !got.Exists || !got.Topics.RaisesCapabilityGaps {
		t.Errorf("LoadMember produced %+v", got)
	}

	// non-existent member returns empty, not error
	missing, err := LoadMember(store, "team-a", "no-such")
	if err != nil {
		t.Fatalf("LoadMember(no-such): %v", err)
	}
	if missing.Exists {
		t.Errorf("expected Exists=false for missing member")
	}
}

func TestWriteMember_RoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := Topics{
		Intake: []IntakeEntry{
			{Prefix: "research-inbox/*", Taxonomy: "marketing-research", ClassifierSkill: "signal-classifier"},
		},
		Output: []OutputEntry{
			{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge, Schema: "audience-scan"},
		},
		DecisionsOwned:       []string{"audience-update"},
		RaisesCapabilityGaps: true,
	}
	if err := WriteMember(root, "marketing-crew", "researcher", original); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	got, err := LoadMember(root, "marketing-crew", "researcher")
	if err != nil {
		t.Fatalf("LoadMember after write: %v", err)
	}
	if !got.Exists {
		t.Fatal("expected Exists=true after WriteMember")
	}
	if len(got.Topics.Intake) != 1 || got.Topics.Intake[0].Taxonomy != "marketing-research" {
		t.Errorf("round-trip lost intake: %+v", got.Topics)
	}
}

func TestWriteMember_RefusesInvalid(t *testing.T) {
	root := t.TempDir()
	bad := Topics{Intake: []IntakeEntry{{Prefix: "*", Taxonomy: "x"}}}
	if err := WriteMember(root, "team-a", "member-1", bad); err == nil {
		t.Errorf("WriteMember should reject invalid declarations")
	}
}

// TestWriteMember_PrettyJSON ensures the file written is human-friendly so
// operators can review it in PRs.
func TestWriteMember_PrettyJSON(t *testing.T) {
	root := t.TempDir()
	t.Logf("root: %s", root)
	src := Topics{
		Output: []OutputEntry{{Prefix: "x/*", DestinationKind: DestinationKnowledge}},
	}
	if err := WriteMember(root, "t", "m", src); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "teams", "t", "members", "m", "topics.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Must be parseable, indented (contains newlines + spaces), and end with newline.
	var roundTrip Topics
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if data[len(data)-1] != '\n' {
		t.Errorf("expected trailing newline")
	}
	if !containsAny(data, '\n') {
		t.Errorf("expected pretty-printed (multiline) output")
	}
}

func containsAny(b []byte, needle byte) bool {
	for _, c := range b {
		if c == needle {
			return true
		}
	}
	return false
}

// TestWriteMember_PreservesPlaceholderChars pins that prefixes containing
// `<`, `>`, or `&` (e.g. `decision-application/<decision-id>`) survive a
// round-trip through WriteMember without being escaped to Unicode literals.
// Operators read topics.json in PR review; `<…>` is unreadable.
func TestWriteMember_PreservesPlaceholderChars(t *testing.T) {
	root := t.TempDir()
	src := Topics{
		RequiredRead: []RequiredReadEntry{
			{Prefix: "decision-application/<decision-id>"},
			{Prefix: "team-visited/<team-id>"},
		},
	}
	if err := WriteMember(root, "t", "m", src); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "teams", "t", "members", "m", "topics.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	asString := string(data)
	if !strings.Contains(asString, "<decision-id>") {
		t.Errorf("expected raw `<decision-id>` in output, got:\n%s", asString)
	}
	if strings.Contains(asString, `\u003c`) || strings.Contains(asString, `\u003e`) {
		t.Errorf("found Unicode escape in output (HTML-safe escaping not disabled):\n%s", asString)
	}
}
