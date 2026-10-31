package sessioncontext

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentsessions"
)

func writeJSONFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestResolveBulkVerdicts(t *testing.T) {
	root := t.TempDir()
	// Real goal.
	writeJSONFile(t, filepath.Join(root, "goals", "ship-cockpit", "goal.json"),
		`{"name":"ship-cockpit","title":"Ship the cockpit","status":"active"}`)
	// Real backlog item.
	writeJSONFile(t, filepath.Join(root, "execute", "wire-snapshot", "spec.json"),
		`{"name":"wire-snapshot","title":"Wire the snapshot","status":"queued"}`)

	r := NewResolver(root, filepath.Dir(root), nil)
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 2000}

	candidates := []ReferenceCandidate{
		{Type: "goal", Name: "ship-cockpit"},             // real
		{Type: "goal", Name: "does-not-exist"},           // fake
		{Type: "backlog", Name: "execute/wire-snapshot"}, // real
		{Type: "backlog", Name: "execute/ghost"},         // fake
		{Type: "operating-mode", Name: "holistic-loop"},  // retired reference
		{Type: "operating-mode", Name: "not-a-mode"},     // retired reference
		{Type: "bogus-marker", Name: "whatever"},         // unknown marker
		{Type: "goal", Name: ""},                         // empty ref
	}

	got := r.ResolveBulk(context.Background(), candidates, limits)
	if len(got) != len(candidates) {
		t.Fatalf("expected %d results, got %d", len(candidates), len(got))
	}

	type want struct {
		exists     bool
		detailPath string
	}
	wants := []want{
		{true, "/goals/ship-cockpit"},
		{false, ""},
		{true, "/backlog/execute/wire-snapshot"},
		{false, ""},
		{false, ""},
		{false, ""},
		{false, ""},
		{false, ""},
	}
	for i, w := range wants {
		if got[i].Exists != w.exists {
			t.Errorf("candidate %d (%s:%s) Exists = %v, want %v", i, candidates[i].Type, candidates[i].Name, got[i].Exists, w.exists)
		}
		if got[i].DetailPath != w.detailPath {
			t.Errorf("candidate %d (%s:%s) DetailPath = %q, want %q", i, candidates[i].Type, candidates[i].Name, got[i].DetailPath, w.detailPath)
		}
	}

	// The real goal carries its loaded title.
	if got[0].Title != "Ship the cockpit" {
		t.Errorf("ship-cockpit Title = %q, want %q", got[0].Title, "Ship the cockpit")
	}
}

func TestExtractReferenceCandidates(t *testing.T) {
	content := "Start `goal:ship-cockpit` then check `backlog:execute/wire-snapshot`.\n" +
		"Run `goals list` to see more (a command, has a space — not a ref).\n" +
		"This `unknownmarker:foo` should be ignored, and `goal:ship-cockpit` again is a dup.\n" +
		"A bare http link `http://example.com` is not a reference either.\n" +
		"Mode `operating-mode:holistic-loop` is valid syntax."

	got := extractReferenceCandidates(content)
	want := []ReferenceCandidate{
		{Type: "goal", Name: "ship-cockpit"},
		{Type: "backlog", Name: "execute/wire-snapshot"},
		{Type: "operating-mode", Name: "holistic-loop"},
	}
	if len(got) != len(want) {
		t.Fatalf("extracted %d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("candidate %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEnrichMessageReferencesAttachesOnlyRealRefs(t *testing.T) {
	root := t.TempDir()
	writeJSONFile(t, filepath.Join(root, "goals", "ship-cockpit", "goal.json"),
		`{"name":"ship-cockpit","title":"Ship the cockpit","status":"active"}`)

	r := NewResolver(root, filepath.Dir(root), nil)

	content := "I recommend `goal:ship-cockpit` next. Avoid `goal:ghost` (does not exist). " +
		"Use `goals list` to browse."
	items := r.EnrichMessageReferences(context.Background(), content)

	if len(items) != 1 {
		t.Fatalf("expected exactly 1 resolved context item, got %d: %+v", len(items), items)
	}
	if items[0].Ref != "ship-cockpit" {
		t.Errorf("resolved ref = %q, want ship-cockpit", items[0].Ref)
	}
	if items[0].Type != agentsessions.ContextGoal {
		t.Errorf("resolved type = %q, want %q", items[0].Type, agentsessions.ContextGoal)
	}
	if items[0].NodeID != "goal/ship-cockpit" {
		t.Errorf("resolved NodeID = %q, want goal/ship-cockpit", items[0].NodeID)
	}
}

func TestEnrichMessageReferencesEmptyWhenNoTypedSpans(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp", nil)
	if got := r.EnrichMessageReferences(context.Background(), "Just prose mentioning ship-cockpit with no code span."); got != nil {
		t.Errorf("expected nil context items for untyped prose, got %+v", got)
	}
}

func TestDetailPathFromNodeID(t *testing.T) {
	cases := map[string]string{
		"goal/ship-cockpit":                  "/goals/ship-cockpit",
		"backlog-item/execute/wire-snapshot": "/backlog/execute/wire-snapshot",
		"scenario/swarm-manager":             "/scenarios/swarm-manager",
		"execution-record/exec-123":          "/executions/exec-123",
		"capture/cap-7":                      "/captures/cap-7",
		"operatingMode/holistic-loop":        "/operating-modes/holistic-loop",
		"/sessions/sess-1":                   "/sessions/sess-1", // already a path
		"/operations":                        "/operations",
		"agent-activity/act-9":               "", // not navigable
		"backlog-item/onlyonepart":           "", // malformed backlog id
		"":                                   "",
	}
	for nodeID, wantPath := range cases {
		if got := detailPathFromNodeID(nodeID); got != wantPath {
			t.Errorf("detailPathFromNodeID(%q) = %q, want %q", nodeID, got, wantPath)
		}
	}
}
