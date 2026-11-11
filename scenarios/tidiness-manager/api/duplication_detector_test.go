package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"tidiness-manager/internal/testutil"
)

func TestNewDuplicationDetectorDefaults(t *testing.T) {
	if got := NewDuplicationDetector("/scenario", 0).timeout; got != 90*time.Second {
		t.Fatalf("default timeout = %s, want 90s", got)
	}
	if got := NewDuplicationDetector("/scenario", time.Second).timeout; got != time.Second {
		t.Fatalf("explicit timeout = %s, want 1s", got)
	}
}

func TestDetectDuplicationUsesOneGraphPathForGoTypeScriptAndJavaScript(t *testing.T) {
	for _, tc := range []struct {
		name          string
		language      Language
		graphLanguage string
		targetFilter  []string
		files         []string
	}{
		{"go", LanguageGo, "go", []string{"go"}, []string{"api/a.go", "api/b.go"}},
		{"typescript", LanguageTypeScript, "typescript", []string{"typescript"}, []string{"ui/src/a.ts", "ui/src/b.ts"}},
		{"javascript in TypeScript project", LanguageJavaScript, "typescript", []string{"javascript", "typescript"}, []string{"ui/src/a.js", "ui/src/b.js"}},
		{"standalone javascript", LanguageJavaScript, "javascript", []string{"javascript", "typescript"}, []string{"ui/src/a.js", "ui/src/b.js"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			symbolLanguage := tc.graphLanguage
			if tc.language == LanguageJavaScript {
				symbolLanguage = "javascript"
			}
			nodes := []*common.CodeGraphNode{
				testutil.GraphNode("a", "first", sourcePath(tc.files[0]), bodyAttributes(symbolLanguage, 11, 18, "same")),
				testutil.GraphNode("b", "second", sourcePath(tc.files[1]), bodyAttributes(symbolLanguage, 21, 28, "same")),
			}
			unitRoot := "/scenario"
			if tc.graphLanguage == "typescript" || tc.graphLanguage == "javascript" {
				unitRoot = "/scenario/ui"
			}
			report := testutil.GraphReport("/scenario", testutil.GraphUnit{ID: "unit", Language: tc.graphLanguage, RootPath: unitRoot, Nodes: nodes})
			dd := NewDuplicationDetector("/scenario", time.Second)
			dd.source = reductionSourceFunc(func(_ context.Context, req *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
				if !reflect.DeepEqual(req.Msg.GetTarget().GetLanguageFilter(), tc.targetFilter) {
					t.Fatalf("language filter = %v, want %v", req.Msg.GetTarget().GetLanguageFilter(), tc.targetFilter)
				}
				return connect.NewResponse(report), nil
			})

			got, err := dd.DetectDuplication(context.Background(), tc.language, tc.files)
			if err != nil {
				t.Fatal(err)
			}
			if got.Skipped || got.Tool != "code-facts" || got.TotalDuplicates != 1 || got.TotalLines != 8 {
				t.Fatalf("result = %+v", got)
			}
			if len(got.DuplicateBlocks) != 1 || len(got.DuplicateBlocks[0].Files) != 2 {
				t.Fatalf("duplicate block = %+v", got.DuplicateBlocks)
			}
		})
	}
}

func TestDetectDuplicationScopesToRequestedFiles(t *testing.T) {
	nodes := []*common.CodeGraphNode{
		testutil.GraphNode("a", "first", "src/a.js", bodyAttributes("javascript", 1, 8, "same")),
		testutil.GraphNode("b", "middle", "src/b.js", bodyAttributes("javascript", 11, 18, "same")),
		testutil.GraphNode("c", "third", "src/c.js", bodyAttributes("javascript", 21, 28, "same")),
	}
	report := testutil.GraphReport("/scenario", testutil.GraphUnit{ID: "unit", Language: "typescript", RootPath: "/scenario/ui", Nodes: nodes})
	dd := NewDuplicationDetector("/scenario", time.Second)
	dd.source = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return connect.NewResponse(report), nil
	})

	got, err := dd.DetectDuplication(context.Background(), LanguageJavaScript, []string{"ui/src/a.js", "ui/src/c.js"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Skipped || got.TotalDuplicates != 1 || len(got.DuplicateBlocks[0].Files) != 2 {
		t.Fatalf("result = %+v", got)
	}
	if got.DuplicateBlocks[0].Files[0].Path != "ui/src/a.js" || got.DuplicateBlocks[0].Files[1].Path != "ui/src/c.js" {
		t.Fatalf("requested file scope = %+v", got.DuplicateBlocks[0].Files)
	}
}

func TestDetectDuplicationDegradesWhenRequestedLanguageEvidenceIsMissing(t *testing.T) {
	dd := NewDuplicationDetector("/scenario", time.Second)
	dd.source = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return connect.NewResponse(&facts.CodeFactsReport{}), nil
	})
	got, err := dd.DetectDuplication(context.Background(), LanguageJavaScript, []string{"ui/src/app.js"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Skipped || got.Tool != "code-facts" || got.SkipReason == "" || got.TotalDuplicates != 0 {
		t.Fatalf("missing evidence result = %+v", got)
	}
}

func TestDetectDuplicationDegradesWhenCodeFactsFails(t *testing.T) {
	dd := NewDuplicationDetector("/scenario", time.Second)
	dd.source = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return nil, errors.New("provider unavailable")
	})
	got, err := dd.DetectDuplication(context.Background(), LanguageGo, []string{"api/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Skipped || got.Tool != "code-facts" || got.SkipReason == "" {
		t.Fatalf("provider failure result = %+v", got)
	}
}

func TestDetectDuplicationRejectsEscapingPaths(t *testing.T) {
	dd := NewDuplicationDetector("/scenario", time.Second)
	got, err := dd.DetectDuplication(context.Background(), LanguageJavaScript, []string{"../outside.js"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Skipped || got.Tool != "" || got.SkipReason == "" {
		t.Fatalf("path validation result = %+v", got)
	}
}

func sourcePath(path string) string {
	return strings.TrimPrefix(path, "ui/")
}

func bodyAttributes(language string, start, end int, hash string) map[string]string {
	return map[string]string{
		"language": language, "body_hash": hash, "body_lines": "8",
		"start_line": fmt.Sprint(start), "end_line": fmt.Sprint(end),
	}
}
