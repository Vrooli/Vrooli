package reconcile

import (
	"context"
	"os"
	"path/filepath"
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
	return Resolver{ScenariosRoot: root, Scanner: scanner}, scanner
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

func TestExtraFilesAreReported(t *testing.T) {
	r, _ := fixture(t)
	results, err := r.Resolve(context.Background(), "demo", "page")
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Extra && result.FilePath == "ui/src/components/Extra.tsx" {
			return
		}
	}
	t.Fatal("extra file not reported")
}
