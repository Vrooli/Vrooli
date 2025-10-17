package main

import (
	"testing"
)

// Benchmark tests for performance validation

func BenchmarkPerformLineDiff(b *testing.B) {
	text1 := GenerateTestText("medium")
	text2 := GenerateTestText("medium") + "\nModified"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		performLineDiff(text1, text2, DiffOptions{})
	}
}

func BenchmarkPerformWordDiff(b *testing.B) {
	text1 := GenerateTestText("medium")
	text2 := GenerateTestText("medium") + " modified"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		performWordDiff(text1, text2, DiffOptions{})
	}
}

func BenchmarkPerformSearch(b *testing.B) {
	text := GenerateTestText("large")
	pattern := "test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		performSearch(text, pattern, SearchOptions{})
	}
}

func BenchmarkPerformSearchRegex(b *testing.B) {
	text := GenerateTestText("large")
	pattern := "test[0-9]+"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		performSearch(text, pattern, SearchOptions{Regex: true})
	}
}

func BenchmarkApplyTransformation(b *testing.B) {
	text := GenerateTestText("medium")
	transform := Transformation{Type: "upper"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyTransformation(text, transform)
	}
}

func BenchmarkCalculateTextStatistics(b *testing.B) {
	text := GenerateTestText("large")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateTextStatistics(text)
	}
}

func BenchmarkExtractEntities(b *testing.B) {
	text := "Contact me at test@example.com or visit https://example.com for more info at admin@test.org"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractEntities(text)
	}
}

func BenchmarkExtractKeywords(b *testing.B) {
	text := GenerateTestText("large")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractKeywords(text)
	}
}

func BenchmarkAnalyzeSentiment(b *testing.B) {
	text := "This is a wonderful and amazing product that works great!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzeSentiment(text)
	}
}

func BenchmarkTrackTransformationSteps(b *testing.B) {
	text := GenerateTestText("medium")
	transforms := []Transformation{
		{Type: "upper"},
		{Type: "lower"},
		{Type: "title"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trackTransformationSteps(text, transforms)
	}
}

// Performance test functions using test framework

func TestPerformanceDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	text1 := GenerateTestText("large")
	text2 := GenerateTestText("large") + "\nModified"

	// Should complete in reasonable time
	changes, _ := performLineDiff(text1, text2, DiffOptions{})

	t.Logf("Diff found %d changes in large text", len(changes))
}

func TestPerformanceSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	text := GenerateTestText("large")
	pattern := "Line"

	matches := performSearch(text, pattern, SearchOptions{})

	t.Logf("Search found %d matches in large text", len(matches))
}

func TestPerformanceTransform(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	text := GenerateTestText("large")
	transforms := []Transformation{
		{Type: "upper"},
		{Type: "lower"},
	}

	result := trackTransformationSteps(text, transforms)

	if len(result) != 3 {
		t.Errorf("Expected 3 transformation steps, got %d", len(result))
	}
}

func TestPerformanceAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	text := GenerateTestText("large")

	stats := calculateTextStatistics(text)
	keywords := extractKeywords(text)

	t.Logf("Analysis completed: %d words, %d keywords", stats.WordCount, len(keywords))
}
