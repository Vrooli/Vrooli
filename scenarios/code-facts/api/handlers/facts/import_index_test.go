package facts

import (
	"fmt"
	"strings"
	"testing"

	"code-facts/internal/catalog"
	"code-facts/internal/retrieval"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestGoImportDocumentsPreserveStableSourceProvenance(t *testing.T) {
	source := catalog.SourceFile{
		ID: "source-1", Path: "scenarios/demo/api/server.go", Language: "go", Role: catalog.RoleImplementation,
		Scope: "scenario:demo", Authority: "authoritative", Owner: "scenario:demo", Hash: "sha256:source",
	}
	payload := []byte("package demo\nimport (\n\t\"context\"\n\tweb \"net/http\"\n)\n// import \\\"false/edge\\\"\nvar note = `import \\\"also/false\\\"`\n")
	first, incomplete, err := goImportDocuments(source, payload)
	if err != nil || incomplete || len(first) != 2 {
		t.Fatalf("import documents=%+v incomplete=%t err=%v", first, incomplete, err)
	}
	second, _, err := goImportDocuments(source, payload)
	if err != nil || len(second) != len(first) {
		t.Fatalf("repeat import parse: documents=%+v err=%v", second, err)
	}
	wantPaths := []string{"context", "net/http"}
	wantLines := []int{3, 4}
	for index, document := range first {
		if document.ID != second[index].ID || document.Kind != "imports" || document.ExactText != wantPaths[index] || document.StartLine != wantLines[index] {
			t.Fatalf("import document %d = %+v", index, document)
		}
		if document.Path != source.Path || document.SourceHash != source.Hash || document.Scope != source.Scope {
			t.Fatalf("import provenance %d = %+v", index, document)
		}
	}
	if !strings.Contains(first[1].Body, "as web") {
		t.Fatalf("named import alias missing from evidence: %q", first[1].Body)
	}
}

func TestGoImportDocumentsMarksMalformedAndOversizedInputsPartial(t *testing.T) {
	source := catalog.SourceFile{ID: "source-1", Path: "scenarios/demo/api/server.go", Language: "go", Scope: "scenario:demo", Hash: "sha256:source"}
	malformed, incomplete, err := goImportDocuments(source, []byte("package demo\nimport (\n\"context\"\n"))
	if err != nil || !incomplete || len(malformed) != 1 || malformed[0].Kind != "imports_partial" {
		t.Fatalf("malformed import evidence=%+v incomplete=%t err=%v", malformed, incomplete, err)
	}

	var builder strings.Builder
	builder.WriteString("package demo\nimport (\n")
	for index := 0; index <= maxImportEdgesPerFile; index++ {
		fmt.Fprintf(&builder, "\"example.test/pkg%d\"\n", index)
	}
	builder.WriteString(")\n")
	bounded, incomplete, err := goImportDocuments(source, []byte(builder.String()))
	if err != nil || !incomplete || len(bounded) != maxImportEdgesPerFile+1 {
		t.Fatalf("bounded import evidence count=%d incomplete=%t err=%v", len(bounded), incomplete, err)
	}
	if got := bounded[len(bounded)-1].Kind; got != "imports_partial" {
		t.Fatalf("cap marker kind = %q", got)
	}
}

func TestGoImportDocumentsDoesNotInventUnsupportedLanguageEdges(t *testing.T) {
	source := catalog.SourceFile{ID: "source-1", Path: "scenarios/demo/ui/main.ts", Language: "typescript", Scope: "scenario:demo", Hash: "sha256:source"}
	documents, incomplete, err := goImportDocuments(source, []byte("import React from 'react';\n"))
	if err != nil || incomplete || len(documents) != 0 {
		t.Fatalf("unsupported import evidence=%+v incomplete=%t err=%v", documents, incomplete, err)
	}
}

func TestImportSearchScopeRejectsBroadMismatchedAndUnsupportedRequests(t *testing.T) {
	valid := &factsv1.SearchRequest{
		Query: "which packages import context", Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}
	if err := validateImportSearchScope(valid); err != nil {
		t.Fatalf("scenario-scoped import request rejected: %v", err)
	}
	valid.Scope = "scenario:demo"
	valid.Languages = []string{"go"}
	if err := validateImportSearchScope(valid); err != nil {
		t.Fatalf("matching scoped Go import request rejected: %v", err)
	}
	valid.Languages = []string{"typescript", "javascript"}
	if err := validateImportSearchScope(valid); err != nil {
		t.Fatalf("supported polyglot import request rejected: %v", err)
	}

	cases := []struct {
		name string
		edit func(*factsv1.SearchRequest)
	}{
		{name: "target required", edit: func(request *factsv1.SearchRequest) { request.Target = nil }},
		{name: "project target rejected", edit: func(request *factsv1.SearchRequest) {
			request.Target = &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT}
		}},
		{name: "invalid slug rejected", edit: func(request *factsv1.SearchRequest) { request.Target.Scenario = "../outside" }},
		{name: "mismatched scope rejected", edit: func(request *factsv1.SearchRequest) { request.Scope = "scenario:other" }},
		{name: "unsupported language rejected", edit: func(request *factsv1.SearchRequest) { request.Languages = []string{"typescript"} }},
		{name: "all family cannot be mixed", edit: func(request *factsv1.SearchRequest) {
			request.Families = append(request.Families, factsv1.FactFamily_FACT_FAMILY_ALL)
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := &factsv1.SearchRequest{
				Query: valid.GetQuery(), Families: append([]factsv1.FactFamily(nil), valid.GetFamilies()...),
				Target: &factsv1.CodeTarget{Kind: valid.GetTarget().GetKind(), Scenario: valid.GetTarget().GetScenario()},
			}
			testCase.edit(request)
			if err := validateImportSearchScope(request); err == nil {
				t.Fatal("invalid import scope was accepted")
			}
		})
	}
}

func TestImportSearchRejectsEmbeddedScopeAndFamilyOverrides(t *testing.T) {
	request := &factsv1.SearchRequest{
		Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}
	valid := retrieval.Query{Text: "which packages import context", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial"}}
	if err := validateImportSearchQuery(request, &valid); err != nil {
		t.Fatalf("valid scoped query rejected: %v", err)
	}
	for _, query := range []retrieval.Query{
		{Text: "which packages scope:scenario:other", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial"}},
		{Text: "which packages target:scenarios/other", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial"}},
		{Text: "which packages family:source", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial"}},
	} {
		if err := validateImportSearchQuery(request, &query); err == nil {
			t.Fatalf("unsafe embedded query filter accepted: %+v", query)
		}
	}
	languageRequest := &factsv1.SearchRequest{Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, Languages: []string{"go"}, Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"}}
	languageOverride := retrieval.Query{Text: "which packages language:typescript", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial"}, Languages: []string{"typescript"}}
	if err := validateImportSearchQuery(languageRequest, &languageOverride); err == nil {
		t.Fatal("query language filter overrode explicit request language")
	}
	mixedRequest := &factsv1.SearchRequest{
		Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS, factsv1.FactFamily_FACT_FAMILY_SYMBOLS},
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}
	mixedQuery := retrieval.Query{Text: "which packages import context", Target: "scenarios/demo", Scope: "scenario:demo", Families: []string{"imports", "imports_partial", "source"}}
	if err := validateImportSearchQuery(mixedRequest, &mixedQuery); err != nil {
		t.Fatalf("typed multi-family search rejected its requested source family: %v", err)
	}
}

func TestIndexedFamiliesIncludesImportsAndRejectsUnindexedFamilies(t *testing.T) {
	got, err := indexedFamilies([]factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS})
	if err != nil || strings.Join(got, ",") != "imports,imports_partial" {
		t.Fatalf("indexed import families=%v err=%v", got, err)
	}
	if _, err := indexedFamilies([]factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES}); err == nil {
		t.Fatal("unindexed surfaces family was accepted")
	}
}

func TestImportGenerationCompatibilityPreservesGoOnlyQueries(t *testing.T) {
	legacy := "code-facts-corpus-v2-go-import-edges-v1"
	if !generationSupportsImportLanguages(legacy, []string{"go"}) {
		t.Fatal("legacy Go-only generation should serve an explicitly Go-scoped query")
	}
	for _, languages := range [][]string{nil, {"typescript"}, {"javascript"}, {"go", "typescript"}} {
		if generationSupportsImportLanguages(legacy, languages) {
			t.Fatalf("legacy generation incorrectly supports languages %v", languages)
		}
	}
	if !generationSupportsImportLanguages("code-facts-corpus-v3-polyglot-import-edges-v1", nil) {
		t.Fatal("polyglot generation should serve unfiltered imports")
	}
}

func TestImportSearchRequiresAnIndexGenerationWithImportEdges(t *testing.T) {
	if generationHasGoImportEdges("code-facts-corpus-v1") {
		t.Fatal("pre-import generation was treated as complete for Go imports")
	}
	if !generationHasGoImportEdges("code-facts-corpus-v2-go-import-edges-v1|embedding_role=default") {
		t.Fatal("legacy Go import generation was not recognized")
	}
}
