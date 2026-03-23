package main

import (
	"math"
	"testing"
)

func TestBuildFileDuplicationMap_UsesActualLineCounts(t *testing.T) {
	langMetrics := map[Language]*LanguageMetrics{
		LanguageGo: {
			Duplicates: &DuplicateResult{
				DuplicateBlocks: []DuplicateBlock{
					{
						Files: []DuplicateLocation{
							{Path: "api/handler.go", StartLine: 10, EndLine: 20},
							{Path: "api/util.go", StartLine: 5, EndLine: 15},
						},
						Lines: 11,
					},
				},
			},
		},
	}

	languages := map[Language]*LanguageInfo{
		LanguageGo: {Files: []string{"api/handler.go", "api/util.go"}},
	}

	// Actual line counts: handler.go has 200 lines, util.go has 100 lines
	fileLineCounts := map[string]int{
		"api/handler.go": 200,
		"api/util.go":    100,
	}

	result := buildFileDuplicationMap(langMetrics, languages, fileLineCounts)

	// handler.go: 11 dup lines / 200 total = 5.5%
	handlerPct, ok := result["api/handler.go"]
	if !ok {
		t.Fatal("expected duplication entry for api/handler.go")
	}
	if math.Abs(handlerPct-5.5) > 0.1 {
		t.Errorf("handler.go: expected ~5.5%%, got %.1f%%", handlerPct)
	}

	// util.go: 11 dup lines / 100 total = 11.0%
	utilPct, ok := result["api/util.go"]
	if !ok {
		t.Fatal("expected duplication entry for api/util.go")
	}
	if math.Abs(utilPct-11.0) > 0.1 {
		t.Errorf("util.go: expected ~11.0%%, got %.1f%%", utilPct)
	}
}

func TestBuildFileDuplicationMap_FallsBackToEndLine(t *testing.T) {
	langMetrics := map[Language]*LanguageMetrics{
		LanguageGo: {
			Duplicates: &DuplicateResult{
				DuplicateBlocks: []DuplicateBlock{
					{
						Files: []DuplicateLocation{
							{Path: "api/unknown.go", StartLine: 10, EndLine: 30},
						},
						Lines: 21,
					},
				},
			},
		},
	}

	languages := map[Language]*LanguageInfo{
		LanguageGo: {Files: []string{"api/unknown.go"}},
	}

	// File NOT in fileLineCounts — should fall back to EndLine (30)
	fileLineCounts := map[string]int{}

	result := buildFileDuplicationMap(langMetrics, languages, fileLineCounts)

	pct, ok := result["api/unknown.go"]
	if !ok {
		t.Fatal("expected duplication entry for api/unknown.go")
	}
	// 21 dup lines / 30 (EndLine fallback) = 70.0%
	if math.Abs(pct-70.0) > 0.1 {
		t.Errorf("expected ~70.0%% (EndLine fallback), got %.1f%%", pct)
	}
}

func TestBuildFileDuplicationMap_ZeroLineCounts(t *testing.T) {
	langMetrics := map[Language]*LanguageMetrics{
		LanguageGo: {
			Duplicates: &DuplicateResult{
				DuplicateBlocks: []DuplicateBlock{
					{
						Files: []DuplicateLocation{
							{Path: "api/empty.go", StartLine: 0, EndLine: 0},
						},
						Lines: 0,
					},
				},
			},
		},
	}

	languages := map[Language]*LanguageInfo{
		LanguageGo: {Files: []string{"api/empty.go"}},
	}

	fileLineCounts := map[string]int{
		"api/empty.go": 0,
	}

	// Should not panic or produce an entry
	result := buildFileDuplicationMap(langMetrics, languages, fileLineCounts)

	if _, ok := result["api/empty.go"]; ok {
		t.Error("should not produce duplication entry for file with 0 lines")
	}
}

func TestBuildFileDuplicationMap_NilLangMetrics(t *testing.T) {
	result := buildFileDuplicationMap(nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map for nil input, got %d entries", len(result))
	}
}
