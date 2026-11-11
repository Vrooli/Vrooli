package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestLightScannerSharesOneCodeFactsResponseAcrossStructuralMetrics(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nimport \"fmt\"\nfunc first() {\n fmt.Println(1)\n}\nfunc second() {\n fmt.Println(1)\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requestCount := 0
	source := reductionSourceFunc(func(_ context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		requestCount++
		if request.Msg.GetIncludeGraphs() != true || !request.Msg.GetUseCache() {
			t.Errorf("Code Facts request omitted graph/cache options: %+v", request.Msg)
		}
		if got := request.Msg.GetTarget().GetLanguageFilter(); len(got) != 1 || got[0] != "go" {
			t.Errorf("language filter = %v, want [go]", got)
		}
		nodes := []*commonv1.CodeGraphNode{
			{Id: "file:main.go", Kind: commonv1.NodeKind_NODE_KIND_FILE, Path: "main.go", Attributes: map[string]string{"language": "go"}},
			{Id: "import:fmt", Path: "main.go", Attributes: map[string]string{"kind": "go_import_spec", "language": "go", "import_path": "fmt"}},
		}
		for index, name := range []string{"first", "second"} {
			line := fmt.Sprint(index*3 + 3)
			nodes = append(nodes, &commonv1.CodeGraphNode{
				Id: "go_func:" + name, Name: name, Path: "main.go",
				Attributes: map[string]string{
					"kind": "go_func", "language": "go", "signature": "func()", "start_line": line,
					"end_line": fmt.Sprint(index*3 + 5), "body_hash": "same-body", "body_lines": "5",
					"cyclomatic_complexity": "12", "package_id": "main",
				},
			})
		}
		return connect.NewResponse(&facts.CodeFactsReport{
			Target: &facts.TargetContext{RootPath: root},
			Graphs: []*facts.LanguageGraph{{
				ParseUnit: &facts.ParseUnit{Id: "go:" + root, Language: "go", RootPath: root},
				Graph:     &commonv1.CodeGraph{Nodes: nodes},
			}},
		}), nil
	})

	files := []string{"main.go"}
	scanner := NewLightScanner(root, time.Second).withCodeFacts(source)
	got := scanner.collectLanguageMetrics(context.Background(), map[Language]*LanguageInfo{
		LanguageGo: {Language: LanguageGo, Files: files, FileCount: len(files)},
	})[LanguageGo]
	if requestCount != 1 {
		t.Fatalf("Code Facts requests = %d, want exactly one", requestCount)
	}
	if got.CodeMetrics == nil || got.CodeMetrics.MaxImportsInFile != 1 || got.CodeMetrics.MaxFunctionsInFile != 2 {
		t.Fatalf("Code Metrics did not use shared graph counts: %+v", got.CodeMetrics)
	}
	if got.Complexity == nil || got.Complexity.TotalFunctions != 2 || got.Complexity.HighComplexityCount != 2 {
		t.Fatalf("complexity metrics did not use shared graph: %+v", got.Complexity)
	}
	if got.Duplicates == nil || got.Duplicates.TotalDuplicates != 1 || got.Duplicates.Tool != "code-facts" {
		t.Fatalf("duplicate metrics did not use shared graph: %+v", got.Duplicates)
	}
}
