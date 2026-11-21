package main

import (
	"strconv"
	"testing"

	"github.com/vrooli/api-core/factgraph"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"tidiness-manager/internal/testutil"
)

func TestComplexityProjectsSharedFactsForRequestedFiles(t *testing.T) {
	node := func(id, path string, complexity, line int) *common.CodeGraphNode {
		return testutil.GraphNode(id, id, path, map[string]string{
			"signature": "func()", "cyclomatic_complexity": strconv.Itoa(complexity),
			"start_line": strconv.Itoa(line), "end_line": strconv.Itoa(line + 20),
		})
	}
	report := testutil.GraphReport("/repo/demo", testutil.GraphUnit{
		ID: "api", Language: "go", RootPath: "/repo/demo/api",
		Nodes: []*common.CodeGraphNode{
			node("simple", "a.go", 1, 1), node("complex", "a.go", 15, 25),
			node("test", "a_test.go", 3, 1), node("excluded", "outside.go", 99, 1),
		},
	})

	result := complexityFromIndex([]string{"api/a.go", "api/a_test.go"}, factgraph.NewIndex(report))
	if result.Skipped || result.TotalFunctions != 3 || result.MaxComplexity != 15 || result.AverageComplexity != 19.0/3 || result.HighComplexityCount != 1 || result.Tool != "code-facts" {
		t.Fatalf("shared facts aggregation: %+v", result)
	}
	if result.FileComplexity["api/a.go"].Avg != 8 || result.HighComplexityFiles[0].Line != 25 {
		t.Fatal(result)
	}
}

func TestComplexityMissingGraphIsSkipped(t *testing.T) {
	result := complexityFromIndex([]string{"a.go"}, factgraph.NewIndex(&facts.CodeFactsReport{}))
	if !result.Skipped || result.SkipReason != "Code Facts returned no Go graph evidence" {
		t.Fatalf("missing graph evidence was reported as metrics: %+v", result)
	}
}
