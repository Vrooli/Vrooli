package main

import "testing"

func TestLanguagesFromFileMetricsGroupsCodeFactsInventory(t *testing.T) {
	languages := languagesFromFileMetrics([]FileMetric{
		{Path: "api/main.go", Extension: ".go", Language: "go", Lines: 4},
		{Path: "api/handler.go", Extension: ".go", Language: "go", Lines: 7},
		{Path: "ui/src/App.tsx", Extension: ".tsx", Language: "typescript", Lines: 8},
		{Path: "docs/guide.md", Extension: ".md", Language: "markdown", Lines: 20},
	})
	if len(languages) != 2 {
		t.Fatalf("languages = %+v, want only Tidiness-supported analyzer languages", languages)
	}
	goInfo := languages[LanguageGo]
	if goInfo == nil || goInfo.FileCount != 2 || goInfo.TotalLines != 11 || goInfo.PrimaryDir != "api" {
		t.Fatalf("Go inventory = %+v", goInfo)
	}
	tsInfo := languages[LanguageTypeScript]
	if tsInfo == nil || tsInfo.FileCount != 1 || tsInfo.TotalLines != 8 || tsInfo.PrimaryDir != "ui" {
		t.Fatalf("TypeScript inventory = %+v", tsInfo)
	}
}

func TestLanguageForCodeFactsFencesUnsupportedMetrics(t *testing.T) {
	for _, lang := range []string{"go", "typescript", "javascript", "python", "rust"} {
		if _, ok := languageForCodeFacts(lang); !ok {
			t.Errorf("languageForCodeFacts(%q) should be supported", lang)
		}
	}
	for _, lang := range []string{"ruby", "markdown", "java", ""} {
		if _, ok := languageForCodeFacts(lang); ok {
			t.Errorf("languageForCodeFacts(%q) should remain outside language analyzers", lang)
		}
	}
}
