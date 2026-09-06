package reconcile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"react-component-library/internal/gates"
	"testing"
)

type fakeScanner struct {
	calls int
	files []ObservedFile
}

func (f *fakeScanner) ScanScenario(context.Context, string) ([]ObservedFile, error) {
	f.calls++
	return f.files, nil
}

func fixture(t *testing.T) (Resolver, *fakeScanner) {
	t.Helper()
	root := t.TempDir()
	pageDir := filepath.Join(root, "demo", "experience", "pages")
	uiDir := filepath.Join(root, "demo", "ui", "src", "components")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	page := `{"regions":[
		{"id":"first-region","required":true,"component":{"local":"wrong"}},
		{"id":"second-region","required":true,"component":{"local":"second"}},
		{"id":"third-region","required":false,"component":{"local":"third-card"}},
		{"id":"lost-region","required":false,"component":{"local":"lost"}}
	],"bindings":{"regions":{"first-region":{"testid":"first-hit"}}},"sketch":{"placements":[
		{"region":"first-region","fills":{"asset":"components.second"}},
		{"region":"second-region","fills":{"asset":"components.second"}}
	]}}`
	if err := os.WriteFile(filepath.Join(pageDir, "page.json"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"First.tsx": `<div data-testid="first-hit"/>`, "Second.tsx": `<div/>`, "ThirdCard.tsx": `<div/>`, "Extra.tsx": `<div/>`} {
		if err := os.WriteFile(filepath.Join(uiDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scanner := &fakeScanner{files: []ObservedFile{
		{Path: "ui/src/components/First.tsx", DisplayName: "First", Provenance: ProvenanceCustom},
		{Path: "ui/src/components/Second.tsx", DisplayName: "Second", ComponentName: "second", Provenance: ProvenanceUnknown},
		{Path: "ui/src/components/ThirdCard.tsx", DisplayName: "ThirdCard", Provenance: ProvenanceCustom},
		{Path: "ui/src/components/Extra.tsx", DisplayName: "Extra", Provenance: ProvenanceCustom},
	}}
	return Resolver{ScenariosRoot: root, Scanner: scanner, Facts: func(context.Context, string, ...string) (map[string]gates.SourceFacts, error) {
		var fact gates.SourceFacts
		if err := json.Unmarshal([]byte(`{"elements":[{"tag":"div","attributes":{"data-testid":["\"first-hit\""]}}]}`), &fact); err != nil {
			t.Fatal(err)
		}
		return map[string]gates.SourceFacts{filepath.Join(uiDir, "First.tsx"): fact}, nil
	}}, scanner
}

func TestJoinRuleOrderStopsAtFirstHit(t *testing.T) {
	r, scanner := fixture(t)
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	if scanner.calls != 1 {
		t.Fatalf("ScanScenario calls = %d, want 1", scanner.calls)
	}
	if results[0].Region != "first-region" || results[0].JoinRule != "binding-testid" || results[0].FilePath != "ui/src/components/First.tsx" {
		t.Fatalf("first result = %#v", results[0])
	}
}

func TestRuleThreeIsMarkedHeuristic(t *testing.T) {
	r, _ := fixture(t)
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Region == "third-region" {
			if result.JoinRule != "component-slug" || result.Proven || !result.Heuristic {
				t.Fatalf("result = %#v", result)
			}
			return
		}
	}
	t.Fatal("third-region result missing")
}

func TestUnknownAndCustomAreNotMerged(t *testing.T) {
	r, _ := fixture(t)
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	states := map[Provenance]bool{}
	for _, result := range results {
		states[result.Provenance] = true
	}
	if !states[ProvenanceUnknown] || !states[ProvenanceCustom] {
		t.Fatalf("states = %#v", states)
	}
}

func TestUnrelatedScenarioFilesAreNotPageExtras(t *testing.T) {
	r, _ := fixture(t)
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Extra {
			t.Fatalf("scenario inventory cannot establish page extras: %+v", result)
		}
	}
}

func TestAmbiguousBindingDoesNotChooseFirstFile(t *testing.T) {
	r, scanner := fixture(t)
	previous := r.Facts
	r.Facts = func(ctx context.Context, root string, paths ...string) (map[string]gates.SourceFacts, error) {
		facts, err := previous(ctx, root, paths...)
		facts[filepath.Join(r.ScenariosRoot, "demo", scanner.files[1].Path)] = facts[filepath.Join(r.ScenariosRoot, "demo", scanner.files[0].Path)]
		return facts, err
	}
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	got := results[0]
	if got.Proven || got.FilePath != "" || got.ReasonCode != "ambiguous_binding" || len(got.Candidates) != 2 {
		t.Fatalf("ambiguous binding chose a file: %+v", got)
	}
}

func TestTestIDRequiresLiteralIntrinsicJSXAttribute(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   bool
	}{
		{`{"elements":[{"tag":"div","attributes":{"data-testid":["\"target\""]}}]}`, true},
		{`{"elements":[{"tag":"div","attributes":{"data-testid":["{'target'}"]}}]}`, true},
		{`{"elements":[{"tag":"Card","attributes":{"data-testid":["\"target\""]}}]}`, false},
		{`{"elements":[{"tag":"div","attributes":{"data-testid":["{target}"]}}]}`, false},
		{`{"elements":[{"tag":"div","attributes":{"title":["\"target\""]}}]}`, false},
		{`{"elements":[]}`, false},
	} {
		var fact gates.SourceFacts
		if err := json.Unmarshal([]byte(tc.source), &fact); err != nil {
			t.Fatal(err)
		}
		if got := hasLiteralTestID(fact, "target"); got != tc.want {
			t.Fatalf("%s: got %v", tc.source, got)
		}
	}
}

func TestRealASTIgnoresCommentAndStringBindings(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(t.TempDir(), "Fixture.tsx")
	source := `// data-testid="comment-only"
const misleading = 'data-testid="string-only"';
export const Fixture = () => <div data-testid="actual"><Card data-testid="unforwarded" /></div>;`
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := gates.ReadSourceFacts(context.Background(), repoRoot, sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"actual", "comment-only", "string-only", "unforwarded"} {
		if got := hasLiteralTestID(facts[sourcePath], id); got != (id == "actual") {
			t.Fatalf("binding %q proved=%v", id, got)
		}
	}
}
