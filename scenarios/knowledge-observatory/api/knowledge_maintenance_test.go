package main

import (
	"testing"
	"time"
)

func TestBuildCollectionDiagnostics(t *testing.T) {
	now := time.Now().UTC().UnixMilli()
	points := []qdrantPoint{
		{
			ID:     "a",
			Vector: []float64{0.1, 0.2, 0.3},
			Payload: map[string]interface{}{
				"namespace":           "alpha",
				"document_id":         "doc-1",
				"chunk_index":         0,
				"content_hash":        "h1",
				"content":             "first chunk",
				"ingested_at_unix_ms": now - 2000,
			},
		},
		{
			ID:     "b",
			Vector: []float64{0.1, 0.2, 0.3},
			Payload: map[string]interface{}{
				"namespace":           "alpha",
				"document_id":         "doc-1",
				"chunk_index":         0,
				"content_hash":        "h1",
				"content":             "first chunk duplicate",
				"ingested_at_unix_ms": now - 1000,
			},
		},
		{
			ID:     "c",
			Vector: []float64{0.9, 0.8, 0.7},
			Payload: map[string]interface{}{
				"namespace":           "beta",
				"document_id":         "doc-2",
				"chunk_index":         1,
				"content_hash":        "h2",
				"content":             "second chunk",
				"ingested_at_unix_ms": now,
			},
		},
	}

	report := buildCollectionDiagnostics(points)
	if report.AnalyzedPoints != 3 {
		t.Fatalf("analyzed_points=%d", report.AnalyzedPoints)
	}
	if report.Redundancy.DuplicatePointCount != 1 {
		t.Fatalf("duplicate points=%d", report.Redundancy.DuplicatePointCount)
	}
	if report.StaleChunks.CandidateDeleteRows != 1 {
		t.Fatalf("stale candidate rows=%d", report.StaleChunks.CandidateDeleteRows)
	}
	if len(report.VectorDimensions) != 1 || report.VectorDimensions[0].Dimension != 3 {
		t.Fatalf("unexpected vector dimensions: %+v", report.VectorDimensions)
	}
}

func TestStaleChunkDeleteCandidates(t *testing.T) {
	points := []qdrantPoint{
		{ID: "old", Payload: map[string]interface{}{"namespace": "n", "document_id": "d", "chunk_index": 0, "ingested_at_unix_ms": float64(1000)}},
		{ID: "new", Payload: map[string]interface{}{"namespace": "n", "document_id": "d", "chunk_index": 0, "ingested_at_unix_ms": float64(2000)}},
		{ID: "other", Payload: map[string]interface{}{"namespace": "n", "document_id": "d", "chunk_index": 1, "ingested_at_unix_ms": float64(1500)}},
	}

	candidates := staleChunkDeleteCandidates(points)
	if len(candidates) != 1 || candidates[0] != "old" {
		t.Fatalf("unexpected stale candidates: %v", candidates)
	}
}
