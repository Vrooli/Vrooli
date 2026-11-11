package main

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"strconv"
	"strings"
	"testing"
	"tidiness-manager/internal/testutil"
	"time"
)

func TestComplexityUsesScopedSharedFacts(t *testing.T) {
	analyzer := NewComplexityAnalyzer("/repo/demo", time.Second)
	analyzer.source = reductionSourceFunc(func(ctx context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		if request.Msg.Target.Path != "/repo/demo" || !request.Msg.IncludeGraphs {
			t.Fatal(request.Msg)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("source call is not bounded")
		}
		node := func(id, path string, complexity, line int) *common.CodeGraphNode {
			return testutil.GraphNode(id, id, path, map[string]string{"signature": "func()", "cyclomatic_complexity": strconv.Itoa(complexity), "start_line": strconv.Itoa(line), "end_line": strconv.Itoa(line + 20)})
		}
		return connect.NewResponse(testutil.GraphReport("/repo/demo", testutil.GraphUnit{ID: "api", Language: "go", RootPath: "/repo/demo/api", Nodes: []*common.CodeGraphNode{node("simple", "a.go", 1, 1), node("complex", "a.go", 15, 25), node("test", "a_test.go", 3, 1), node("excluded", "outside.go", 99, 1)}})), nil
	})
	result, err := analyzer.AnalyzeComplexity(context.Background(), LanguageGo, []string{"api/a.go", "api/a_test.go"})
	if err != nil || result.Skipped || result.TotalFunctions != 3 || result.MaxComplexity != 15 || result.AverageComplexity != 19.0/3 || result.HighComplexityCount != 1 || result.Tool != "code-facts" {
		t.Fatalf("shared facts aggregation: %+v %v", result, err)
	}
	if result.FileComplexity["api/a.go"].Avg != 8 || result.HighComplexityFiles[0].Line != 25 {
		t.Fatal(result)
	}
}
func TestComplexityFailuresCannotLookLikeCleanSource(t *testing.T) {
	analyzer := NewComplexityAnalyzer("/repo/demo", time.Second)
	analyzer.source = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return nil, errors.New("source unavailable")
	})
	for _, c := range []struct {
		files  []string
		lang   Language
		reason string
	}{
		{[]string{"a.go"}, LanguageGo, "source unavailable"},
		{[]string{"../outside.go"}, LanguageGo, "escapes scenario root"},
		{[]string{"/outside.go"}, LanguageGo, "escapes scenario root"},
		{nil, LanguageGo, "no files"},
		{[]string{"a.ts"}, LanguageTypeScript, "not yet available"},
	} {
		result, err := analyzer.AnalyzeComplexity(context.Background(), c.lang, c.files)
		if err != nil || !result.Skipped || !strings.Contains(result.SkipReason, c.reason) {
			t.Fatalf("failure reported as metrics: %+v %v", result, err)
		}
	}
	analyzer.source = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return connect.NewResponse(&facts.CodeFactsReport{}), nil
	})
	result, _ := analyzer.AnalyzeComplexity(context.Background(), LanguageGo, []string{"a.go"})
	if !result.Skipped || !strings.Contains(result.SkipReason, "no Go graph") {
		t.Fatal(result)
	}
}
