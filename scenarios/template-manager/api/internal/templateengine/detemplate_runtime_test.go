package templateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// writeFile creates parent dirs and writes content (test helper).
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// detemplateFixture builds a temp repo root with a generated scenario "myapp"
// carrying notes example residue (scenario-local files, a relocated proto
// module + its gen artifacts) plus a real health domain. Returns root, the
// scenario model, and the synthetic template info.
func detemplateFixture(t *testing.T) (string, scenariomodel.Scenario, templatecontracts.TemplateInfo) {
	t.Helper()
	root := t.TempDir()
	sc := filepath.Join(root, "scenarios", "myapp")

	// Pure example files/dirs (deleted wholesale).
	writeFile(t, filepath.Join(sc, "api/internal/notes/service.go"), "package notes\n")
	writeFile(t, filepath.Join(sc, "api/handlers/notes/module.go"), "package notes\n")
	writeFile(t, filepath.Join(sc, "cli/domains/notes/register.go"), "package notes\n")
	writeFile(t, filepath.Join(sc, "ui/src/features/notes/NotesCard.tsx"), "export const NotesCard = () => null;\n")
	writeFile(t, filepath.Join(sc, "ui/src/api/notes.ts"), "export const notes = 1;\n")
	writeFile(t, filepath.Join(sc, "ui/src/api/notes.test.ts"), "test('notes');\n")
	writeFile(t, filepath.Join(sc, "ui/src/pages/NotesPage.tsx"), "export const NotesPage = () => null;\n")

	// Relocated proto schema + a generated go artifact.
	writeFile(t, filepath.Join(root, "packages/proto/schemas/myapp/v1/notes/notes.proto"), "syntax = \"proto3\";\n")
	writeFile(t, filepath.Join(root, "packages/proto/gen/go/myapp/v1/notes/notes.pb.go"), "package notes\n")

	// Real health domain (must survive untouched).
	writeFile(t, filepath.Join(sc, "api/internal/health/health.go"), "package health\n")

	// Registration file with a marked notes import beside a kept health import.
	writeFile(t, filepath.Join(sc, "api/main.go"), strings.Join([]string{
		"package main",
		"import (",
		`	healthH "myapp/handlers/health"`,
		`	notesH "myapp/handlers/notes" // EXAMPLE-DOMAIN:notes`,
		")",
		"",
	}, "\n"))

	// Doc with a fenced example block + binding zone.
	writeFile(t, filepath.Join(sc, "docs/concepts/DOMAINS.md"), strings.Join([]string{
		"# Domains",
		"",
		"The real `health` domain lives here.",
		"",
		"<!-- EXAMPLE-DOMAIN:notes START -->",
		"## notes (example)",
		"<!-- EXAMPLE-DOMAIN:notes END -->",
		"",
		"Binding guidance continues.",
		"",
	}, "\n"))

	// Clean doc (must not be edited).
	writeFile(t, filepath.Join(sc, "docs/clean.md"), "# Clean\nNo markers here.\n")

	item := scenariomodel.Scenario{Slug: "myapp", Path: sc}
	info := templatecontracts.TemplateInfo{
		Name: "react-vite",
		Manifest: templatecontracts.TemplateManifest{
			Relocations: []templatecontracts.TemplateRelocation{{From: "proto/", To: "packages/proto/schemas/{{SCENARIO_ID}}/"}},
			ExampleDomain: &templatecontracts.TemplateExampleDomain{
				Marker: "notes",
				Paths: []string{
					"api/handlers/notes", "api/internal/notes", "cli/domains/notes",
					"ui/src/api/notes.ts", "ui/src/api/notes.test.ts",
					"ui/src/features/notes", "ui/src/pages/NotesPage.tsx",
					"proto/v1/notes",
				},
			},
		},
	}
	return root, item, info
}

func TestResolveDetemplateDeletionsMapsProto(t *testing.T) {
	root, item, info := detemplateFixture(t)
	dels := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)

	got := map[string]bool{}
	for _, d := range dels {
		got[d.Display] = d.IsProto
	}
	// Scenario-local deletions present and not flagged proto.
	for _, p := range []string{"api/handlers/notes", "api/internal/notes", "cli/domains/notes", "ui/src/api/notes.ts", "ui/src/features/notes", "ui/src/pages/NotesPage.tsx"} {
		if _, ok := got[p]; !ok {
			t.Errorf("missing scenario-local deletion %q", p)
		}
		if got[p] {
			t.Errorf("%q wrongly flagged proto", p)
		}
	}
	// Proto schema mapped through relocation.
	if proto, ok := got["packages/proto/schemas/myapp/v1/notes"]; !ok || !proto {
		t.Errorf("proto schema deletion missing/not flagged: %v", got)
	}
	// Generated proto artifact present.
	if proto, ok := got["packages/proto/gen/go/myapp/v1/notes"]; !ok || !proto {
		t.Errorf("generated proto dir deletion missing: %v", got)
	}
}

func TestPlanDetemplateEditsStripsAndKeeps(t *testing.T) {
	root, item, info := detemplateFixture(t)
	dels := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)
	edits, dangling, err := planDetemplateEdits(item.Path, "notes", dels)
	if err != nil {
		t.Fatal(err)
	}
	if len(dangling) != 0 {
		t.Fatalf("unexpected dangling refs: %+v", dangling)
	}
	editsByPath := map[string]detemplateEdit{}
	for _, e := range edits {
		editsByPath[e.Summary.Path] = e
	}
	main, ok := editsByPath["api/main.go"]
	if !ok {
		t.Fatal("api/main.go not edited")
	}
	if strings.Contains(string(main.Content), "notes") {
		t.Errorf("notes residue in main.go:\n%s", main.Content)
	}
	if !strings.Contains(string(main.Content), "handlers/health") {
		t.Errorf("kept health import lost:\n%s", main.Content)
	}
	dom, ok := editsByPath["docs/concepts/DOMAINS.md"]
	if !ok {
		t.Fatal("DOMAINS.md not edited")
	}
	if strings.Contains(string(dom.Content), "notes") {
		t.Errorf("notes residue in DOMAINS.md:\n%s", dom.Content)
	}
	if !strings.Contains(string(dom.Content), "Binding guidance continues.") {
		t.Errorf("binding zone lost:\n%s", dom.Content)
	}
	if _, edited := editsByPath["docs/clean.md"]; edited {
		t.Error("clean.md should not be edited")
	}
}

func TestPlanDetemplateEditsIdempotent(t *testing.T) {
	root, item, info := detemplateFixture(t)
	dels := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)
	edits, _, err := planDetemplateEdits(item.Path, "notes", dels)
	if err != nil {
		t.Fatal(err)
	}
	// Apply edits + deletions.
	for _, e := range edits {
		if err := os.WriteFile(e.Abs, e.Content, e.Mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range dels {
		if err := os.RemoveAll(d.Abs); err != nil {
			t.Fatal(err)
		}
	}
	// Re-plan: nothing left to do.
	dels2 := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)
	if len(dels2) != 0 {
		t.Errorf("expected no deletions on re-run, got %+v", dels2)
	}
	edits2, _, err := planDetemplateEdits(item.Path, "notes", dels2)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits2) != 0 {
		t.Errorf("expected no edits on re-run, got %d", len(edits2))
	}
}

func TestPlanDetemplateEditsRefusesDanglingRef(t *testing.T) {
	root, item, info := detemplateFixture(t)
	// A kept (unmarked) production file still imports the deleted notes package.
	writeFile(t, filepath.Join(item.Path, "api/internal/server/server.go"), strings.Join([]string{
		"package server",
		`import _ "myapp/internal/notes"`,
		"",
	}, "\n"))
	dels := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)
	_, dangling, err := planDetemplateEdits(item.Path, "notes", dels)
	if err != nil {
		t.Fatal(err)
	}
	if len(dangling) == 0 {
		t.Fatal("expected a dangling reference to the deleted notes package")
	}
	found := false
	for _, d := range dangling {
		if d.File == "api/internal/server/server.go" && strings.Contains(d.Reference, "internal/notes") {
			found = true
		}
	}
	if !found {
		t.Errorf("dangling ref not reported for server.go: %+v", dangling)
	}
}

func TestScanTreeForTextResidueGate(t *testing.T) {
	root, item, info := detemplateFixture(t)
	// Add an EXAMPLE-DOMAIN marker that survives outside the deleted paths.
	writeFile(t, filepath.Join(item.Path, "docs/leftover.md"), "x\n<!-- EXAMPLE-DOMAIN:notes START -->\ny\n<!-- EXAMPLE-DOMAIN:notes END -->\n")

	hits, err := scanTreeForText(item.Path, "EXAMPLE-DOMAIN")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected residue gate to find the surviving marker")
	}
	foundLeftover, foundMain := false, false
	for _, h := range hits {
		switch h {
		case "docs/leftover.md":
			foundLeftover = true
		case "api/main.go":
			foundMain = true
		}
	}
	if !foundLeftover || !foundMain {
		t.Errorf("residue gate missed markers, hits=%v", hits)
	}

	// After applying detemplate (strip edits + delete leftover marker), the
	// gate should report clean.
	dels := resolveDetemplateDeletions(root, item, info, info.Manifest.ExampleDomain.Paths)
	edits, _, err := planDetemplateEdits(item.Path, "notes", dels)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range edits {
		if err := os.WriteFile(e.Abs, e.Content, e.Mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range dels {
		if err := os.RemoveAll(d.Abs); err != nil {
			t.Fatal(err)
		}
	}
	hits2, err := scanTreeForText(item.Path, "EXAMPLE-DOMAIN")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits2) != 0 {
		t.Errorf("residue gate still flags markers after detemplate: %v", hits2)
	}
}

func TestPlanDetemplateFinalizersConditional(t *testing.T) {
	root, item, _ := detemplateFixture(t)
	// Fixture has ui/ (package.json absent), api/ (go.mod absent), cli/ (go.mod absent).
	// Add the surface markers the planner keys on.
	writeFile(t, filepath.Join(item.Path, "ui/package.json"), "{}\n")
	writeFile(t, filepath.Join(item.Path, "api/go.mod"), "module myapp\n")
	writeFile(t, filepath.Join(item.Path, "cli/go.mod"), "module myapp/cli\n")
	writeFile(t, filepath.Join(root, "packages/proto/Makefile"), "generate:\n\techo gen\n")

	plans := planDetemplateFinalizers(root, item, true)
	var cmds []string
	for _, p := range plans {
		cmds = append(cmds, p.commandLine())
	}
	want := []string{
		"make generate",
		"corepack pnpm run strings:gen",
		"go mod tidy",  // api
		"gofumpt -w .", // api
		"go mod tidy",  // cli
		"gofumpt -w .", // cli
	}
	if strings.Join(cmds, "|") != strings.Join(want, "|") {
		t.Errorf("finalizer plan mismatch:\n got=%v\nwant=%v", cmds, want)
	}
}
