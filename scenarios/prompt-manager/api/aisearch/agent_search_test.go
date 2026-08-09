package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prompt-manager/search"
	"prompt-manager/store"
)

// --- Mock agent stores for AI search tests ---

type mockAgentStoreReader struct {
	agents []store.Agent
	err    error
}

func (m *mockAgentStoreReader) List(ctx context.Context) ([]store.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.agents, nil
}

func (m *mockAgentStoreReader) Get(ctx context.Context, id string) (*store.Agent, error) {
	for _, a := range m.agents {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, errForTesting
}

func (m *mockAgentStoreReader) Create(ctx context.Context, agent *store.Agent) error { return nil }
func (m *mockAgentStoreReader) Update(ctx context.Context, id string, agent *store.Agent) error {
	return nil
}
func (m *mockAgentStoreReader) Delete(ctx context.Context, id string) error { return nil }

// --- Compose embedding text tests ---

func TestComposeAgentEmbeddingText(t *testing.T) {
	agent := &store.Agent{
		DisplayName: "Researcher",
		Description: "A research assistant",
		Tags:        []string{"research", "analysis"},
		Status:      "active",
	}

	result := composeAgentEmbeddingText(agent, "You are a researcher.")

	if !strings.Contains(result, "Researcher") {
		t.Error("expected result to contain display name")
	}
	if !strings.Contains(result, "A research assistant") {
		t.Error("expected result to contain description")
	}
	if !strings.Contains(result, "Tags: research, analysis") {
		t.Error("expected result to contain tags")
	}
	if !strings.Contains(result, "Status: active") {
		t.Error("expected result to contain status")
	}
	if !strings.Contains(result, "You are a researcher.") {
		t.Error("expected result to contain soul content")
	}
}

func TestComposeAgentEmbeddingText_TruncatesLongSoul(t *testing.T) {
	agent := &store.Agent{DisplayName: "Test"}
	longSoul := strings.Repeat("x", 3000)

	result := composeAgentEmbeddingText(agent, longSoul)

	if len(result) > 2100 {
		t.Errorf("expected truncated result, got length %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected truncated content to end with ...")
	}
}

func TestComposeAgentEmbeddingText_EmptyFields(t *testing.T) {
	agent := &store.Agent{DisplayName: "Just a Name"}

	result := composeAgentEmbeddingText(agent, "")

	if result != "Just a Name" {
		t.Errorf("expected only name, got: %s", result)
	}
}

// --- Agent point ID tests ---

func TestAgentPointID_DifferentFromSkillPointID(t *testing.T) {
	agentID := agentPointID("my-agent")
	skillID := qdrantPointID("my-agent")

	if agentID == skillID {
		t.Error("agent and skill point IDs should differ for the same input")
	}
}

func TestAgentPointID_EmptyInput(t *testing.T) {
	id := agentPointID("")
	if id == "" {
		t.Error("expected non-empty UUID for empty input")
	}
}

// --- SearchAgents tests ---

func TestSearchAgents_AISuccess(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer ollamaServer.Close()

	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/search") {
			resp := searchResponse{
				Result: []struct {
					ID      interface{}            `json:"id"`
					Score   float64                `json:"score"`
					Payload map[string]interface{} `json:"payload"`
				}{
					{ID: "uuid-1", Score: 0.85, Payload: map[string]interface{}{
						"agent_id":     "researcher",
						"display_name": "Researcher",
						"description":  "A research agent",
						"status":       "active",
						"tags":         []interface{}{"research"},
					}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 3)
	agentVS := NewVectorStore(qdrantServer.URL, "", "agents", 3)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	agentStore := &mockAgentStoreReader{agents: []store.Agent{
		{ID: "researcher", DisplayName: "Researcher", Description: "A research agent", Status: "active"},
	}}
	service.SetAgentSearch(agentVS, agentStore, nil)

	result, err := service.SearchAgents(context.Background(), "research", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "ai" {
		t.Errorf("expected method 'ai', got '%s'", result.Method)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].ID != "researcher" {
		t.Errorf("expected ID 'researcher', got '%s'", result.Results[0].ID)
	}
	if result.Results[0].DisplayName != "Researcher" {
		t.Errorf("expected DisplayName 'Researcher', got '%s'", result.Results[0].DisplayName)
	}
	if result.Results[0].ScorePercent != 85 {
		t.Errorf("expected score percent 85, got %d", result.Results[0].ScorePercent)
	}
}

func TestSearchAgents_FallbackToText(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollamaServer.Close()

	embedder := fakeEmbedderOK()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	agentStore := &mockAgentStoreReader{agents: []store.Agent{
		{ID: "researcher", DisplayName: "Researcher", Status: "active"},
	}}
	agentSearchSvc := search.NewAgentSearchService(agentStore, nil)
	service.SetAgentSearch(nil, agentStore, agentSearchSvc)

	result, err := service.SearchAgents(context.Background(), "Researcher", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "text" {
		t.Errorf("expected method 'text' for fallback, got '%s'", result.Method)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least one fallback result")
	}
}

func TestSearchAgents_NoVectorStore_FallsBack(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	// No agent vector store set — should fall back immediately
	agentStore := &mockAgentStoreReader{agents: []store.Agent{
		{ID: "a1", DisplayName: "Alpha", Status: "active"},
	}}
	agentSearchSvc := search.NewAgentSearchService(agentStore, nil)
	service.SetAgentSearch(nil, agentStore, agentSearchSvc)

	result, err := service.SearchAgents(context.Background(), "Alpha", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "text" {
		t.Errorf("expected text fallback, got '%s'", result.Method)
	}
}

func TestSearchAgents_NoSearchSvc_EmptyResults(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	// No vector store and no search service
	service.SetAgentSearch(nil, nil, nil)

	result, err := service.SearchAgents(context.Background(), "anything", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 results, got %d", result.Total)
	}
	if result.Method != "text" {
		t.Errorf("expected method 'text', got '%s'", result.Method)
	}
}

// --- IndexAgent tests ---

func TestIndexAgent_Success(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer ollamaServer.Close()

	upsertCalled := false
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points") && r.Method == "PUT" {
			upsertCalled = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 3)
	agentVS := NewVectorStore(qdrantServer.URL, "", "agents", 3)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	agentStore := &mockAgentStoreReader{agents: []store.Agent{
		{ID: "a1", DisplayName: "Test Agent", Description: "Desc", Status: "active"},
	}}
	service.SetAgentSearch(agentVS, agentStore, nil)

	err := service.IndexAgent(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected upsert to be called")
	}
}

func TestIndexAgent_NotFound(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	agentVS := NewVectorStore("http://localhost:99999", "", "agents", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	agentStore := &mockAgentStoreReader{agents: nil}
	service.SetAgentSearch(agentVS, agentStore, nil)

	err := service.IndexAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestIndexAgent_NilVectorStore(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	// No agent search configured — should be a no-op
	err := service.IndexAgent(context.Background(), "a1")
	if err != nil {
		t.Fatalf("expected no error for nil vector store, got: %v", err)
	}
}

// --- DeleteAgentFromIndex tests ---

func TestDeleteAgentFromIndex_Success(t *testing.T) {
	deleteCalled := false
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/delete") {
			deleteCalled = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	agentVS := NewVectorStore(qdrantServer.URL, "", "agents", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)
	service.SetAgentSearch(agentVS, nil, nil)

	err := service.DeleteAgentFromIndex(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected delete to be called")
	}
}

func TestDeleteAgentFromIndex_NilVectorStore(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	err := service.DeleteAgentFromIndex(context.Background(), "a1")
	if err != nil {
		t.Fatalf("expected no error for nil vector store, got: %v", err)
	}
}

// --- toAIAgentSearchResult tests ---

func TestToAIAgentSearchResult(t *testing.T) {
	r := SearchResult{
		ID:    "uuid-1",
		Score: 0.92,
		Payload: map[string]interface{}{
			"agent_id":     "researcher",
			"display_name": "Researcher",
			"description":  "Research agent",
			"status":       "active",
			"tags":         []interface{}{"research", "analysis"},
		},
	}

	result := toAIAgentSearchResult(r)

	if result.ID != "researcher" {
		t.Errorf("expected ID 'researcher', got '%s'", result.ID)
	}
	if result.DisplayName != "Researcher" {
		t.Errorf("expected DisplayName 'Researcher', got '%s'", result.DisplayName)
	}
	if result.ScorePercent != 92 {
		t.Errorf("expected ScorePercent 92, got %d", result.ScorePercent)
	}
	if len(result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Tags))
	}
}

func TestToAIAgentSearchResult_MissingPayloadFields(t *testing.T) {
	r := SearchResult{
		ID:      "uuid-1",
		Score:   0.5,
		Payload: map[string]interface{}{},
	}

	result := toAIAgentSearchResult(r)

	if result.ID != "uuid-1" {
		t.Errorf("expected ID from SearchResult, got '%s'", result.ID)
	}
	if result.DisplayName != "" {
		t.Errorf("expected empty DisplayName, got '%s'", result.DisplayName)
	}
}
