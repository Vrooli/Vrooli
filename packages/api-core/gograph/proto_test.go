package gograph

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestToProtoGraphPreservesGoKindsAndImportEdges(t *testing.T) {
	graph := Graph{
		Nodes: []Node{
			{ID: "file:a.go", Kind: NodeKindFile, Name: "a.go", Path: "a.go"},
			{ID: "go_func:package:demo:Run", Kind: NodeKindFunc, Name: "Run", Path: "a.go", Attributes: map[string]string{"exported": "true"}},
		},
		Edges: []Edge{{ID: "import:package:demo->package:fmt", Kind: EdgeKindImport, From: "package:demo", To: "package:fmt"}},
	}

	wire := ToProtoGraph(graph)
	if len(wire.GetNodes()) != 2 || len(wire.GetEdges()) != 1 {
		t.Fatalf("wire graph lost rows: %+v", wire)
	}
	var file, function *commonv1.CodeGraphNode
	for _, node := range wire.GetNodes() {
		switch node.GetId() {
		case "file:a.go":
			file = node
		case "go_func:package:demo:Run":
			function = node
		}
	}
	if file == nil || file.GetKind() != commonv1.NodeKind_NODE_KIND_FILE {
		t.Fatalf("file node mapping = %+v", file)
	}
	if function == nil || function.GetKind() != commonv1.NodeKind_NODE_KIND_PACKAGE || function.GetAttributes()["kind"] != "go_func" || function.GetAttributes()["exported"] != "true" {
		t.Fatalf("typed symbol mapping = %+v", function)
	}
	if edge := wire.GetEdges()[0]; edge.GetKind() != commonv1.EdgeKind_EDGE_KIND_IMPORT || edge.GetFromNodeId() != "package:demo" || edge.GetToNodeId() != "package:fmt" {
		t.Fatalf("import edge mapping = %+v", edge)
	}
}

func TestWarningsToProtoPreservesWarningKindAndLocation(t *testing.T) {
	warnings := WarningsToProto([]Warning{{Kind: WarningKindTypeCheckFailure, File: "a.go", Message: "unresolved type"}})
	if len(warnings) != 1 || warnings[0].GetKind() != commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE || warnings[0].GetFile() != "a.go" || warnings[0].GetMessage() != "unresolved type" {
		t.Fatalf("warning mapping = %+v", warnings)
	}
	if got := WarningsToProto(nil); got != nil {
		t.Fatalf("nil warning input = %#v, want nil", got)
	}
}
