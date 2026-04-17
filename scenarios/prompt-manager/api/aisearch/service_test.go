package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"prompt-manager/search"
	"prompt-manager/skills"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- NeedsReindex tests ---

func TestNeedsReindex(t *testing.T) {
	tests := []struct {
		name         string
		indexedCount int
		diskSkills   int
		wantNeedsIdx bool
	}{
		{"counts match", 60, 60, false},
		{"skills added externally", 55, 60, true},
		{"skills deleted externally", 60, 55, true},
		{"empty index", 0, 60, true},
		{"both empty", 0, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Mock Qdrant returning tc.indexedCount
			qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/points/count") {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
						Count int `json:"count"`
					}{Count: tc.indexedCount}})
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer qdrant.Close()

			skillStore := NewMockSkillStore()
			for i := 0; i < tc.diskSkills; i++ {
				skillStore.AddSkill("local", skills.Metadata{
					ID:   "skill-" + string(rune('a'+i)),
					Name: "Skill",
					File: "local/skill.md",
				}, "content")
			}

			vectorStore := NewVectorStore(qdrant.URL, "", "test", 768)
			embedder := NewEmbedder("http://localhost:11434", "nomic-embed-text")
			searchSvc := search.NewService(skillStore)
			service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

			needs, indexed, disk, err := service.NeedsReindex(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if needs != tc.wantNeedsIdx {
				t.Errorf("NeedsReindex() = %v, want %v", needs, tc.wantNeedsIdx)
			}
			if indexed != tc.indexedCount {
				t.Errorf("indexed = %d, want %d", indexed, tc.indexedCount)
			}
			if disk != tc.diskSkills {
				t.Errorf("disk = %d, want %d", disk, tc.diskSkills)
			}
		})
	}
}

func TestNeedsReindex_QdrantError(t *testing.T) {
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer qdrant.Close()

	skillStore := NewMockSkillStore()
	vectorStore := NewVectorStore(qdrant.URL, "", "test", 768)
	embedder := NewEmbedder("http://localhost:11434", "nomic-embed-text")
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	needs, _, _, err := service.NeedsReindex(context.Background())
	if err == nil {
		t.Fatal("expected error when Qdrant returns 500")
	}
	if needs {
		t.Error("expected needs=false on error")
	}
}

func TestNeedsReindex_SkillStoreError(t *testing.T) {
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/count") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: 5}})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrant.Close()

	skillStore := NewMockSkillStore()
	skillStore.allErr = errForTesting

	vectorStore := NewVectorStore(qdrant.URL, "", "test", 768)
	embedder := NewEmbedder("http://localhost:11434", "nomic-embed-text")
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	needs, _, _, err := service.NeedsReindex(context.Background())
	if err == nil {
		t.Fatal("expected error when skill store fails")
	}
	if needs {
		t.Error("expected needs=false on error")
	}
}

// --- StartPeriodicSync tests ---

func TestStartPeriodicSync_TriggersReindexOnMismatch(t *testing.T) {
	var reindexCalled atomic.Int32

	// Mock Ollama (needed for reindex to succeed)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer ollama.Close()

	// Mock Qdrant — count returns 0 (stale), then accept upserts for reindex
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/count") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: 0}})
			return
		}
		if strings.Contains(r.URL.Path, "/collections/") && r.Method == "GET" {
			// EnsureCollection check
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.Contains(r.URL.Path, "/points") && r.Method == "PUT" {
			reindexCalled.Add(1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrant.Close()

	skillStore := NewMockSkillStore()
	skillStore.AddSkill("local", skills.Metadata{
		ID:   "skill-1",
		Name: "Skill 1",
		File: "local/skill1.md",
	}, "content")

	embedder := NewEmbedder(ollama.URL, "nomic-embed-text")
	vectorStore := NewVectorStore(qdrant.URL, "", "test", 3)
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.StartPeriodicSync(ctx, 50*time.Millisecond)

	// Wait for at least one tick
	time.Sleep(200 * time.Millisecond)

	if reindexCalled.Load() == 0 {
		t.Error("expected reindex to be triggered by periodic sync")
	}
}

func TestStartPeriodicSync_NoReindexWhenMatchingCounts(t *testing.T) {
	var countChecks atomic.Int32

	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/count") {
			countChecks.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: 1}}) // matches disk count
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrant.Close()

	skillStore := NewMockSkillStore()
	skillStore.AddSkill("local", skills.Metadata{
		ID:   "skill-1",
		Name: "Skill 1",
		File: "local/skill1.md",
	}, "content")

	embedder := NewEmbedder("http://localhost:11434", "nomic-embed-text")
	vectorStore := NewVectorStore(qdrant.URL, "", "test", 768)
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	ctx, cancel := context.WithCancel(context.Background())
	service.StartPeriodicSync(ctx, 50*time.Millisecond)

	// Wait for a few ticks
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Count checks happened but no reindex should have been triggered
	if countChecks.Load() == 0 {
		t.Error("expected periodic count checks to occur")
	}

	// Verify no reindex was started (status should show no recent start)
	status := service.ReindexStatus()
	if status.Running {
		t.Error("expected no reindex to be running when counts match")
	}
}

func TestStartPeriodicSync_ContextCanceled(t *testing.T) {
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/count") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: 0}})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrant.Close()

	skillStore := NewMockSkillStore()
	embedder := NewEmbedder("http://localhost:11434", "nomic-embed-text")
	vectorStore := NewVectorStore(qdrant.URL, "", "test", 768)
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	ctx, cancel := context.WithCancel(context.Background())
	service.StartPeriodicSync(ctx, 1*time.Hour) // long interval

	// Cancel immediately
	cancel()

	// Give goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// If we get here without hanging, the goroutine exited cleanly
}

// sentinel error for testing
var errForTesting = &testError{"test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
