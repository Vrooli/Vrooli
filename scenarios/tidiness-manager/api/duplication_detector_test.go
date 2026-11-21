package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/api-core/factgraph"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"tidiness-manager/internal/testutil"
)

func TestDuplicationProjectsSharedFactsForEachLanguage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		language      Language
		graphLanguage string
		files         []string
	}{
		{"go", LanguageGo, "go", []string{"api/a.go", "api/b.go"}},
		{"typescript", LanguageTypeScript, "typescript", []string{"ui/src/a.ts", "ui/src/b.ts"}},
		{"javascript", LanguageJavaScript, "typescript", []string{"ui/src/a.js", "ui/src/b.js"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			symbolLanguage := string(tc.language)
			nodes := []*common.CodeGraphNode{
				testutil.GraphNode("a", "first", sourcePath(tc.files[0]), bodyAttributes(symbolLanguage, 11, 18, "same")),
				testutil.GraphNode("b", "second", sourcePath(tc.files[1]), bodyAttributes(symbolLanguage, 21, 28, "same")),
			}
			unitRoot := "/scenario"
			if tc.graphLanguage == "typescript" {
				unitRoot = "/scenario/ui"
			}
			report := testutil.GraphReport("/scenario", testutil.GraphUnit{ID: "unit", Language: tc.graphLanguage, RootPath: unitRoot, Nodes: nodes})
			detector := NewDuplicationDetector("/scenario")
			got := detector.DetectDuplicationForLanguagesFromIndex(map[string][]string{string(tc.language): tc.files}, factgraph.NewIndex(report))[string(tc.language)]
			if got.Skipped || got.Tool != "code-facts" || got.TotalDuplicates != 1 || got.TotalLines != 8 {
				t.Fatalf("result = %+v", got)
			}
			if len(got.DuplicateBlocks) != 1 || len(got.DuplicateBlocks[0].Files) != 2 {
				t.Fatalf("duplicate block = %+v", got.DuplicateBlocks)
			}
		})
	}
}

func TestDuplicationProjectsOnlyRequestedFiles(t *testing.T) {
	nodes := []*common.CodeGraphNode{
		testutil.GraphNode("a", "first", "src/a.js", bodyAttributes("javascript", 1, 8, "same")),
		testutil.GraphNode("b", "middle", "src/b.js", bodyAttributes("javascript", 11, 18, "same")),
		testutil.GraphNode("c", "third", "src/c.js", bodyAttributes("javascript", 21, 28, "same")),
	}
	report := testutil.GraphReport("/scenario", testutil.GraphUnit{ID: "unit", Language: "typescript", RootPath: "/scenario/ui", Nodes: nodes})
	files := []string{"ui/src/a.js", "ui/src/c.js"}
	got := NewDuplicationDetector("/scenario").DetectDuplicationForLanguagesFromIndex(map[string][]string{"javascript": files}, factgraph.NewIndex(report))["javascript"]
	if got.Skipped || got.TotalDuplicates != 1 || len(got.DuplicateBlocks[0].Files) != 2 {
		t.Fatalf("result = %+v", got)
	}
	if got.DuplicateBlocks[0].Files[0].Path != files[0] || got.DuplicateBlocks[0].Files[1].Path != files[1] {
		t.Fatalf("requested file scope = %+v", got.DuplicateBlocks[0].Files)
	}
}

func TestDuplicationSkipsMissingEvidenceAndEscapingPaths(t *testing.T) {
	detector := NewDuplicationDetector("/scenario")
	missing := detector.DetectDuplicationForLanguagesFromIndex(map[string][]string{"javascript": {"ui/src/app.js"}}, factgraph.NewIndex(&facts.CodeFactsReport{}))["javascript"]
	if !missing.Skipped || missing.Tool != "code-facts" || missing.SkipReason == "" || missing.TotalDuplicates != 0 {
		t.Fatalf("missing evidence result = %+v", missing)
	}
	escaping := detector.DetectDuplicationForLanguagesFromIndex(map[string][]string{"javascript": {"../outside.js"}}, factgraph.NewIndex(&facts.CodeFactsReport{}))["javascript"]
	if !escaping.Skipped || escaping.Tool != "" || escaping.SkipReason == "" {
		t.Fatalf("path validation result = %+v", escaping)
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
