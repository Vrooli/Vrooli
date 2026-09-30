package slice

import (
	"reflect"
	"testing"
)

func TestBuild_DerivesRungsAndSurfaces(t *testing.T) {
	snap := Snapshot{
		ID:       "snap:architecture-cartographer:h1",
		Scenario: "architecture-cartographer",
		Packages: []PackageNode{
			{ID: "pkg:handler", ImportPath: "architecture-cartographer/handlers/graph", RepoPath: "api/handlers/graph"},
			{ID: "pkg:internal", ImportPath: "architecture-cartographer/internal/graph", RepoPath: "api/internal/graph"},
			{ID: "pkg:cli", ImportPath: "architecture-cartographer/cli/domains/graph", RepoPath: "cli/domains/graph"},
			{ID: "pkg:ui", ImportPath: "architecture-cartographer/ui/src/features/graph", RepoPath: "ui/src/features/graph"},
			{ID: "pkg:proto", ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"},
		},
		Imports: []ImportEdge{{ToPackageID: "pkg:proto"}},
	}
	domainMap := DomainMap{
		Scenario: "architecture-cartographer",
	}
	classifier := fakeClassifier{
		"api/handlers/graph":      {Domain: "graph", Zone: ZoneTransport},
		"api/internal/graph":      {Domain: "graph", Zone: ZoneDomain},
		"cli/domains/graph":       {Domain: "graph", Zone: ZoneCLI},
		"ui/src/features/graph":   {Domain: "graph", Zone: ZoneUI},
		"packages/proto/ignored":  {Domain: "graph", Zone: ZoneDomain},
		"api/internal/unrelated":  {Domain: "other", Zone: ZoneDomain},
		"cli/domains/unrelated":   {Domain: "other", Zone: ZoneCLI},
		"ui/src/features/ignored": {Domain: "other", Zone: ZoneUI},
	}

	got := Build(snap, domainMap, classifier, "graph")

	if got.SnapshotID != snap.ID {
		t.Fatalf("snapshot id = %q", got.SnapshotID)
	}
	for _, rung := range got.Rungs {
		if !rung.Present {
			t.Fatalf("rung %s not present: %#v", rung.Name, got.Rungs)
		}
	}
	if want := []string{"api", "cli", "ui"}; !reflect.DeepEqual(got.Surfaces, want) {
		t.Fatalf("surfaces = %v, want %v", got.Surfaces, want)
	}
}

func TestBuild_FilesSymbolsAndLayerEdges(t *testing.T) {
	snap := Snapshot{
		ID:       "snap:x:h1",
		Scenario: "x",
		Packages: []PackageNode{
			{ID: "pkg:handler", RepoPath: "api/handlers/graph"},
			{ID: "pkg:internal", RepoPath: "api/internal/graph"},
			{ID: "pkg:proto", ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/x/v1/graph"},
		},
		Files: []FileNode{
			{Path: "api/handlers/graph/handler.go", PackageID: "pkg:handler", Lines: 120},
			{Path: "api/internal/graph/graph.go", PackageID: "pkg:internal", Lines: 300},
			{Path: "api/internal/graph/graph_test.go", PackageID: "pkg:internal", Lines: 80, IsTest: true},
		},
		Symbols: []SymbolNode{
			{Name: "Handler", Kind: "type", PackageID: "pkg:handler", FilePath: "api/handlers/graph/handler.go", Exported: true},
			{Name: "unexported", Kind: "func", PackageID: "pkg:internal", Exported: false},
			{Name: "Snapshot", Kind: "type", PackageID: "pkg:internal", FilePath: "api/internal/graph/graph.go", Exported: true},
		},
		Imports: []ImportEdge{
			{FromPackageID: "pkg:handler", ToPackageID: "pkg:internal"},
			{FromPackageID: "pkg:internal", ToPackageID: "pkg:proto"},
		},
	}
	classifier := fakeClassifier{
		"api/handlers/graph": {Domain: "graph", Zone: ZoneTransport},
		"api/internal/graph": {Domain: "graph", Zone: ZoneDomain},
	}
	got := Build(snap, DomainMap{Scenario: "x"}, classifier, "graph")

	byRung := map[string]Rung{}
	for _, r := range got.Rungs {
		byRung[r.Name] = r
	}
	if n := len(byRung[RungInternal].Files); n != 2 {
		t.Fatalf("internal rung should have 2 files, got %d", n)
	}
	if n := len(byRung[RungInternal].Symbols); n != 1 || byRung[RungInternal].Symbols[0].Name != "Snapshot" {
		t.Fatalf("internal rung should have 1 exported symbol Snapshot, got %#v", byRung[RungInternal].Symbols)
	}
	if n := len(byRung[RungHandler].Symbols); n != 1 || byRung[RungHandler].Symbols[0].Name != "Handler" {
		t.Fatalf("handler rung symbols = %#v", byRung[RungHandler].Symbols)
	}
	// Layer edges: handler->internal and internal->proto.
	wantEdges := map[string]bool{"handler->internal": false, "internal->proto": false}
	for _, e := range got.LayerEdges {
		wantEdges[e.FromRung+"->"+e.ToRung] = true
	}
	for edge, seen := range wantEdges {
		if !seen {
			t.Fatalf("expected layer edge %s, got %#v", edge, got.LayerEdges)
		}
	}
}

func TestBuild_DerivesRungsFromDeclaredDomainPaths(t *testing.T) {
	snap := Snapshot{ID: "snap:architecture-cartographer:h1", Scenario: "architecture-cartographer"}
	domainMap := DomainMap{
		Scenario: "architecture-cartographer",
		Domains: []Domain{{
			Name: "graph",
			Paths: []string{
				"api/internal/graph/",
				"api/handlers/graph/",
				"cli/domains/graph/",
				"ui/src/features/graph/",
				"packages/proto/schemas/architecture-cartographer/v1/graph/",
			},
		}},
	}

	got := Build(snap, domainMap, fakeClassifier{}, "graph")

	present := map[string]bool{}
	for _, rung := range got.Rungs {
		present[rung.Name] = rung.Present
	}
	for _, rung := range []string{RungProto, RungHandler, RungInternal, RungCLI, RungUI} {
		if !present[rung] {
			t.Fatalf("rung %s not present from declared paths: %#v", rung, got.Rungs)
		}
	}
	if want := []string{"api", "cli", "ui"}; !reflect.DeepEqual(got.Surfaces, want) {
		t.Fatalf("surfaces = %v, want %v", got.Surfaces, want)
	}
}

func TestBuild_MarksMissingRungsAndIgnoresTestOnlyProtoImports(t *testing.T) {
	snap := Snapshot{
		ID:       "snap:demo:h1",
		Scenario: "demo",
		Packages: []PackageNode{
			{ID: "pkg:internal", ImportPath: "demo/internal/billing", RepoPath: "api/internal/billing"},
			{ID: "pkg:proto", ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/demo/v1/billing"},
		},
		Imports: []ImportEdge{{ToPackageID: "pkg:proto", TestOnly: true}},
	}
	domainMap := DomainMap{
		Scenario: "demo",
	}
	classifier := fakeClassifier{"api/internal/billing": {Domain: "billing", Zone: ZoneDomain}}

	got := Build(snap, domainMap, classifier, "billing")

	present := map[string]bool{}
	for _, rung := range got.Rungs {
		present[rung.Name] = rung.Present
	}
	if present[RungProto] {
		t.Fatalf("proto rung should ignore test-only imports: %#v", got.Rungs)
	}
	if !present[RungInternal] {
		t.Fatalf("internal rung should be present: %#v", got.Rungs)
	}
	if len(got.Surfaces) != 0 {
		t.Fatalf("surfaces = %v, want none without handler/cli/ui rungs", got.Surfaces)
	}
}

type fakeClassifier map[string]ZoneInfo

func (f fakeClassifier) Classify(repoPath string) ZoneInfo {
	return f[repoPath]
}
