package factgraph

import (
	"testing"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestNewIndexProjectsFilesAcrossParseUnitRoots(t *testing.T) {
	report := &facts.CodeFactsReport{
		Target: &facts.TargetContext{RootPath: "/repo"},
		Graphs: []*facts.LanguageGraph{{
			ParseUnit:          &facts.ParseUnit{Id: "ui", Language: "typescript", RootPath: "/repo/ui"},
			ExtractionProfile:  "structural",
			ExtractionMs:       11,
			OmittedInformation: []*facts.GraphInformationOmission{{Capability: "resolved_usage_facts", Reason: "structural profile"}},
			Graph: &commonv1.CodeGraph{
				Nodes: []*commonv1.CodeGraphNode{
					{Id: "module", Path: "src/app.ts", Attributes: map[string]string{"kind": "TS_NODE_KIND_MODULE", "language": "typescript"}},
					{Id: "function", Name: "run", Path: "src/app.ts", Attributes: map[string]string{"language": "typescript", "body_hash": "body"}},
					{Id: "package", Kind: commonv1.NodeKind_NODE_KIND_PACKAGE, Attributes: map[string]string{"kind": "package", "import_path": "ui"}},
				},
				Edges: []*commonv1.CodeGraphEdge{
					{FromNodeId: "module", ToNodeId: "other", Kind: commonv1.EdgeKind_EDGE_KIND_IMPORT, Attributes: map[string]string{"test_only": "true"}},
				},
			},
		}},
	}

	index := NewIndex(report)
	file, ok := index.File("ui/src/app.ts", "typescript")
	nodes := file.Nodes()
	if !ok || file.Imports() != 1 || len(nodes) != 2 || nodes[1].Attribute("body_hash") != "body" {
		t.Fatalf("indexed file = %+v, found=%v", file, ok)
	}
	nodes[1].id = "mutated"
	again, _ := index.File("ui/src/app.ts", "typescript")
	if again.Nodes()[1].ID() != "function" {
		t.Fatal("File accessor leaked mutable node state")
	}
	if files := index.Files(); len(files) != 1 || files[0].Path() != "ui/src/app.ts" {
		t.Fatalf("indexed files = %+v", files)
	}
	units := index.Units()
	if len(units) != 1 || units[0].Root() != "ui" || units[0].Language() != "typescript" || units[0].Profile() != "structural" || units[0].ExtractionMS() != 11 {
		t.Fatalf("unit metadata = %+v", units)
	}
	unitNodes, unitEdges, omissions := units[0].Nodes(), units[0].Edges(), units[0].Omissions()
	if len(unitNodes) != 3 || !unitNodes[0].HasTargetPath() || unitNodes[2].HasTargetPath() || unitNodes[2].Kind() != commonv1.NodeKind_NODE_KIND_PACKAGE.String() || unitNodes[2].Path() != "" || len(unitEdges) != 1 || unitEdges[0].Attribute("test_only") != "true" || len(omissions) != 1 || omissions[0].Capability != "resolved_usage_facts" {
		t.Fatalf("unit projection lost graph facts: nodes=%+v edges=%+v omissions=%+v", unitNodes, unitEdges, omissions)
	}
}

func TestNewIndexRejectsEscapingPathsAndParseUnits(t *testing.T) {
	report := &facts.CodeFactsReport{
		Target: &facts.TargetContext{RootPath: "/repo"},
		Graphs: []*facts.LanguageGraph{
			{ParseUnit: &facts.ParseUnit{Language: "go", RootPath: "/outside"}, Graph: &commonv1.CodeGraph{}},
			{ParseUnit: &facts.ParseUnit{Language: "go", RootPath: "/repo"}, Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
				{Id: "escape", Path: "../../outside.go", Attributes: map[string]string{"language": "go"}},
			}}},
		},
	}
	index := NewIndex(report)
	if len(index.Files()) != 0 || len(index.Warnings()) != 2 {
		t.Fatalf("files=%+v warnings=%v", index.Files(), index.Warnings())
	}
	units := index.Units()
	if len(units) != 1 || len(units[0].Nodes()) != 1 || units[0].Nodes()[0].HasTargetPath() {
		t.Fatalf("out-of-target source was not marked: %+v", units)
	}
}

func TestNewIndexCountsGoImportSpecs(t *testing.T) {
	report := &facts.CodeFactsReport{
		Target: &facts.TargetContext{RootPath: "/repo"},
		Graphs: []*facts.LanguageGraph{{
			ParseUnit: &facts.ParseUnit{Language: "go", RootPath: "/repo"},
			Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
				{Id: "fmt", Path: "main.go", Attributes: map[string]string{"kind": "go_import_spec"}},
				{Id: "strings", Path: "main.go", Attributes: map[string]string{"kind": "go_import_spec"}},
			}},
		}},
	}
	file, found := NewIndex(report).File("main.go", "go")
	if !found || file.Imports() != 2 {
		t.Fatalf("Go import projection = %+v, found=%v", file, found)
	}
}

func TestNewIndexProjectsAncestorParseUnitOntoRequestedTarget(t *testing.T) {
	// A caller can request a directory inside a module. The Go unit still owns
	// the module root, so the index must project only the requested subtree and
	// make its paths target-relative rather than reject the unit.
	report := &facts.CodeFactsReport{
		Target: &facts.TargetContext{RootPath: "/repo/internal"},
		Graphs: []*facts.LanguageGraph{{
			ParseUnit: &facts.ParseUnit{Id: "go:/repo", Language: "go", RootPath: "/repo"},
			Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
				{Id: "inside", Path: "internal/app/service.go", Attributes: map[string]string{"language": "go", "kind": "go_func"}},
				{Id: "outside", Path: "packages/shared/code.go", Attributes: map[string]string{"language": "go", "kind": "go_func"}},
			}},
		}},
	}
	index := NewIndex(report)
	if warnings := index.Warnings(); len(warnings) != 0 {
		t.Fatalf("ancestor unit produced warnings: %v", warnings)
	}
	file, ok := index.File("app/service.go", "go")
	if !ok || file.Path() != "app/service.go" || len(file.Nodes()) != 1 {
		t.Fatalf("projected file = %+v, found=%v", file, ok)
	}
	if _, found := index.File("packages/shared/code.go", "go"); found {
		t.Fatal("node outside the requested target must not be indexed")
	}
	units := index.Units()
	if len(units) != 1 || units[0].Root() != "." || len(units[0].Nodes()) != 2 {
		t.Fatalf("unit projection = %+v", units)
	}
	if !units[0].Nodes()[0].HasTargetPath() || units[0].Nodes()[1].HasTargetPath() {
		t.Fatalf("target membership not marked: %+v", units[0].Nodes())
	}
}
