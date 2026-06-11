package aisearch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedsearch "github.com/vrooli/ai-go/search"

	"prompt-manager/search"
	"prompt-manager/skills"
)

// --- Mock implementations ---

// MockSkillStore implements skills.SkillStore for testing.
type MockSkillStore struct {
	skills   map[string][]skills.Metadata // folder -> skills
	contents map[string]string            // folder/filename -> content
	findErr  error
	allErr   error
}

func NewMockSkillStore() *MockSkillStore {
	return &MockSkillStore{
		skills:   make(map[string][]skills.Metadata),
		contents: make(map[string]string),
	}
}

func (m *MockSkillStore) GetAll() ([]skills.Metadata, error) {
	if m.allErr != nil {
		return nil, m.allErr
	}
	var all []skills.Metadata
	for _, skillList := range m.skills {
		all = append(all, skillList...)
	}
	return all, nil
}

func (m *MockSkillStore) FindByID(id string) (*skills.Metadata, string, error) {
	if m.findErr != nil {
		return nil, "", m.findErr
	}
	for folder, skillList := range m.skills {
		for _, p := range skillList {
			if p.ID == id {
				return &p, folder, nil
			}
		}
	}
	return nil, "", errors.New("not found")
}

func (m *MockSkillStore) LoadMetadata(folder string) ([]skills.Metadata, error) {
	return m.skills[folder], nil
}

func (m *MockSkillStore) SaveMetadata(folder string, skillList []skills.Metadata) error {
	m.skills[folder] = skillList
	return nil
}

func (m *MockSkillStore) GetContent(folder, filename string) (string, error) {
	key := folder + "/" + filename
	if content, ok := m.contents[key]; ok {
		return content, nil
	}
	return "", errors.New("content not found")
}

func (m *MockSkillStore) SaveContent(folder, filename, content string) error {
	key := folder + "/" + filename
	m.contents[key] = content
	return nil
}

func (m *MockSkillStore) DeleteContent(folder, filename string) error {
	key := folder + "/" + filename
	delete(m.contents, key)
	return nil
}

func (m *MockSkillStore) GetVersions(skillID string) ([]skills.SkillVersion, error) {
	return []skills.SkillVersion{}, nil
}

func (m *MockSkillStore) SaveVersion(skillID, folder string, skill *skills.Metadata, content string) error {
	return nil
}

func (m *MockSkillStore) GetVersionContent(skillID string, version int) (*skills.SkillVersion, error) {
	return nil, errors.New("version not found")
}

func (m *MockSkillStore) LoadVersions(folder string) (map[string]*skills.VersionFile, error) {
	return make(map[string]*skills.VersionFile), nil
}

func (m *MockSkillStore) SaveVersions(folder string, versions map[string]*skills.VersionFile) error {
	return nil
}

func (m *MockSkillStore) Rename(oldID, newID string) (*skills.Metadata, error) {
	// Find and rename the skill
	for folder, skillList := range m.skills {
		for i, p := range skillList {
			if p.ID == oldID {
				// Update the ID
				m.skills[folder][i].ID = newID
				m.skills[folder][i].File = newID + ".md"
				updated := m.skills[folder][i]

				// Move content
				oldKey := folder + "/" + oldID + ".md"
				newKey := folder + "/" + newID + ".md"
				if content, ok := m.contents[oldKey]; ok {
					m.contents[newKey] = content
					delete(m.contents, oldKey)
				}

				return &updated, nil
			}
		}
	}
	return nil, errors.New("skill not found")
}

func (m *MockSkillStore) AddSkill(folder string, skill skills.Metadata, content string) {
	m.skills[folder] = append(m.skills[folder], skill)
	// Extract filename from the skill.File path for content storage
	// skill.File is like "local/test.md", we need to store at "local/test.md"
	_, filename := extractFolderAndFile(skill.File)
	m.contents[folder+"/"+filename] = content
}

// Embedder tests live in embedder_test.go (CLI runner-based, not HTTP). The
// fakes used by Service-level tests below come from that file too.

// --- VectorStore tests ---

func TestVectorStore_NewVectorStore_Defaults_CreateUsesDefaults(t *testing.T) {
	// We assert defaults via the EnsureCollection request the store sends.
	var gotPath string
	var gotSize int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method == "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req createCollectionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotSize = req.Vectors.Size
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "", 3)
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	if !strings.HasSuffix(gotPath, "/collections/prompt-manager-skills") {
		t.Errorf("expected default collection 'prompt-manager-skills' in path, got %q", gotPath)
	}
	if gotSize != 3 {
		t.Errorf("expected resolved vector size 3, got %d", gotSize)
	}
}

func TestVectorStoreForRole_CreateUsesResolvedPolicyDimensions(t *testing.T) {
	old := resolveEmbeddingPolicy
	t.Cleanup(func() { resolveEmbeddingPolicy = old })
	resolveEmbeddingPolicy = func(_ context.Context, role string) (sharedsearch.EmbeddingPolicy, error) {
		if role != "embedding.default" {
			t.Fatalf("role = %q, want embedding.default", role)
		}
		return sharedsearch.EmbeddingPolicy{
			Role:       role,
			Model:      "fixture-embed-model:latest",
			Dimensions: 1234,
		}, nil
	}

	var gotSize int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req createCollectionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotSize = req.Vectors.Size
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStoreForRole(server.URL, "", "", "")
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	if gotSize != 1234 {
		t.Errorf("expected resolved vector size 1234, got %d", gotSize)
	}
}

func TestVectorStore_EnsureCollection_AlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/collections/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result": {}}`))
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test-collection", 768)
	err := vs.EnsureCollection(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_EnsureCollection_Create(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == "PUT" {
			created = true
			var req createCollectionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Vectors.Distance != "Cosine" {
				t.Errorf("expected Cosine distance, got %s", req.Vectors.Distance)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "new-collection", 768)
	err := vs.EnsureCollection(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected collection to be created")
	}
}

func TestVectorStore_Upsert_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/points") {
			t.Errorf("expected /points path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("wait") != "true" {
			t.Error("expected wait=true query param")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	err := vs.Upsert(context.Background(), "skill-1", []float64{0.1, 0.2}, map[string]interface{}{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_Upsert_EmptyID(t *testing.T) {
	vs := NewVectorStore("http://localhost:6333", "", "test", 768)
	err := vs.Upsert(context.Background(), "", []float64{0.1}, nil)

	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestVectorStore_Delete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/points/delete") {
			t.Errorf("expected /points/delete path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	err := vs.Delete(context.Background(), "skill-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_Search_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/points/search") {
			t.Errorf("expected /points/search path, got %s", r.URL.Path)
		}

		resp := searchResponse{
			Result: []struct {
				ID      interface{}            `json:"id"`
				Score   float64                `json:"score"`
				Payload map[string]interface{} `json:"payload"`
			}{
				{ID: "skill-1", Score: 0.95, Payload: map[string]interface{}{"name": "Test Skill"}},
				{ID: "skill-2", Score: 0.80, Payload: map[string]interface{}{"name": "Another Skill"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	results, err := vs.Search(context.Background(), []float64{0.1, 0.2}, 5, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "skill-1" {
		t.Errorf("expected first result ID 'skill-1', got '%s'", results[0].ID)
	}
	if results[0].Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", results[0].Score)
	}
}

func TestVectorStore_CountPoints_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/points/count") {
			t.Errorf("expected /points/count path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
			Count int `json:"count"`
		}{Count: 42}})
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	count, err := vs.CountPoints(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestVectorStore_Available_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if !vs.Available(context.Background()) {
		t.Error("expected Available() to return true")
	}
}

func TestVectorStore_Available_EmptyURL(t *testing.T) {
	vs := NewVectorStore("", "", "test", 768)
	if vs.Available(context.Background()) {
		t.Error("expected Available() to return false for empty URL")
	}
}

func TestVectorStore_APIKeyHeader(t *testing.T) {
	apiKeyReceived := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKeyReceived = r.Header.Get("api-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "my-secret-key", "test", 768)
	vs.Available(context.Background())

	if apiKeyReceived != "my-secret-key" {
		t.Errorf("expected API key 'my-secret-key', got '%s'", apiKeyReceived)
	}
}

// --- Service tests ---

func TestService_Search_AISuccess(t *testing.T) {
	// Mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer ollamaServer.Close()

	// Mock Qdrant server
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/search") {
			resp := searchResponse{
				Result: []struct {
					ID      interface{}            `json:"id"`
					Score   float64                `json:"score"`
					Payload map[string]interface{} `json:"payload"`
				}{
					{ID: "skill-1", Score: 0.9, Payload: map[string]interface{}{
						"name":        "Test Skill",
						"description": "A test skill",
						"folder":      "local",
						"tags":        []interface{}{"test", "demo"},
					}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	vectorStore := NewVectorStore(qdrantServer.URL, "", "test", 3)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	result, err := service.Search(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "ai" {
		t.Errorf("expected method 'ai', got '%s'", result.Method)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Name != "Test Skill" {
		t.Errorf("expected name 'Test Skill', got '%s'", result.Results[0].Name)
	}
	if result.Results[0].ScorePercent != 90 {
		t.Errorf("expected score percent 90, got %d", result.Results[0].ScorePercent)
	}
}

func TestService_Search_FallbackToText(t *testing.T) {
	// Mock failing Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollamaServer.Close()

	embedder := fakeEmbedderOK()
	vectorStore := NewVectorStore("http://localhost:99999", "", "test", 768)
	skillStore := NewMockSkillStore()
	skillStore.AddSkill("local", skills.Metadata{
		ID:          "skill-1",
		Name:        "Test Skill",
		Description: "A test skill for searching",
		File:        "local/test.md",
		Tags:        []string{"test"},
	}, "Test content")

	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	result, err := service.Search(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "text" {
		t.Errorf("expected method 'text' for fallback, got '%s'", result.Method)
	}
}

func TestService_GetStatus(t *testing.T) {
	// Mock servers
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1}})
	}))
	defer ollamaServer.Close()

	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.Contains(r.URL.Path, "/points/count") {
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: 10}})
			return
		}
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	vectorStore := NewVectorStore(qdrantServer.URL, "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	status := service.GetStatus(context.Background())

	if !status.Available {
		t.Error("expected Available to be true")
	}
	if !status.Ollama {
		t.Error("expected Ollama to be true")
	}
	if !status.Qdrant {
		t.Error("expected Qdrant to be true")
	}
	if status.IndexedCount != 10 {
		t.Errorf("expected IndexedCount 10, got %d", status.IndexedCount)
	}
}

func TestService_GetStatus_Unavailable(t *testing.T) {
	embedder := fakeEmbedderErr()
	vectorStore := NewVectorStore("", "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	status := service.GetStatus(context.Background())

	if status.Available {
		t.Error("expected Available to be false")
	}
	if status.Ollama {
		t.Error("expected Ollama to be false")
	}
	if status.Qdrant {
		t.Error("expected Qdrant to be false")
	}
	if !strings.Contains(status.Message, "Ollama") || !strings.Contains(status.Message, "Qdrant") {
		t.Errorf("expected message mentioning unavailable services, got: %s", status.Message)
	}
}

func TestService_Available(t *testing.T) {
	// Both available
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1}})
	}))
	defer ollamaServer.Close()

	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	vectorStore := NewVectorStore(qdrantServer.URL, "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	if !service.Available(context.Background()) {
		t.Error("expected Available() to return true")
	}
}

func TestService_DefaultThreshold(t *testing.T) {
	embedder := fakeEmbedderErr()
	vectorStore := NewVectorStore("http://localhost:6333", "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)

	// Test with 0 threshold - should default to 0.5
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0)
	if service.threshold != 0.5 {
		t.Errorf("expected default threshold 0.5, got %f", service.threshold)
	}

	// Test with negative threshold - should default to 0.5
	service2 := NewService(embedder, vectorStore, skillStore, searchSvc, -1)
	if service2.threshold != 0.5 {
		t.Errorf("expected default threshold 0.5, got %f", service2.threshold)
	}
}

// --- Index tests ---

func TestComposeEmbeddingText(t *testing.T) {
	skill := &skills.Metadata{
		Name:        "Test Skill",
		Description: "A description of the skill",
		Tags:        []string{"tag1", "tag2"},
		Modes:       []string{"mode1", "mode2"},
	}
	content := "The content of the skill"

	result := composeEmbeddingText(skill, content)

	if !strings.Contains(result, "Test Skill") {
		t.Error("expected result to contain skill name")
	}
	if !strings.Contains(result, "A description of the skill") {
		t.Error("expected result to contain description")
	}
	if !strings.Contains(result, "Tags: tag1, tag2") {
		t.Error("expected result to contain tags")
	}
	if !strings.Contains(result, "Categories: mode1 / mode2") {
		t.Error("expected result to contain modes as categories")
	}
	if !strings.Contains(result, "The content of the skill") {
		t.Error("expected result to contain content")
	}
}

func TestComposeEmbeddingText_TruncatesLongContent(t *testing.T) {
	skill := &skills.Metadata{Name: "Test"}
	longContent := strings.Repeat("x", 3000)

	result := composeEmbeddingText(skill, longContent)

	// Should be truncated to 2000 chars + "..."
	if len(result) > 2100 { // Allow for name and separators
		t.Errorf("expected truncated result, got length %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected truncated content to end with ...")
	}
}

func TestComposeEmbeddingText_EmptyFields(t *testing.T) {
	skill := &skills.Metadata{
		Name: "Just a Name",
	}

	result := composeEmbeddingText(skill, "")

	if result != "Just a Name" {
		t.Errorf("expected only name, got: %s", result)
	}
}

func TestExtractFolderAndFile(t *testing.T) {
	tests := []struct {
		input          string
		expectedFolder string
		expectedFile   string
	}{
		{"local/skill.md", "local", "skill.md"},
		{"shared/subdir/skill.md", "shared", "subdir/skill.md"},
		{"skill.md", "", "skill.md"},
		{"", "", ""},
	}

	for _, tc := range tests {
		folder, file := extractFolderAndFile(tc.input)
		if folder != tc.expectedFolder {
			t.Errorf("input '%s': expected folder '%s', got '%s'", tc.input, tc.expectedFolder, folder)
		}
		if file != tc.expectedFile {
			t.Errorf("input '%s': expected file '%s', got '%s'", tc.input, tc.expectedFile, file)
		}
	}
}

func TestService_IndexSkill_Success(t *testing.T) {
	// Mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer ollamaServer.Close()

	// Mock Qdrant server
	upsertCalled := false
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points") && r.Method == "PUT" {
			upsertCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderOK()
	vectorStore := NewVectorStore(qdrantServer.URL, "", "test", 3)
	skillStore := NewMockSkillStore()
	skillStore.AddSkill("local", skills.Metadata{
		ID:          "skill-1",
		Name:        "Test Skill",
		Description: "A test skill",
		File:        "local/test.md",
	}, "Test content")

	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	err := service.IndexSkill(context.Background(), "skill-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected upsert to be called")
	}
}

func TestService_IndexSkill_SkillNotFound(t *testing.T) {
	embedder := fakeEmbedderErr()
	vectorStore := NewVectorStore("http://localhost:6333", "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	err := service.IndexSkill(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestService_DeleteFromIndex_Success(t *testing.T) {
	deleteCalled := false
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/delete") {
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer qdrantServer.Close()

	embedder := fakeEmbedderErr()
	vectorStore := NewVectorStore(qdrantServer.URL, "", "test", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, vectorStore, skillStore, searchSvc, 0.5)

	err := service.DeleteFromIndex(context.Background(), "skill-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected delete to be called")
	}
}

// --- Helper function tests ---

func TestStringifyID(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"string-id", "string-id"},
		{float64(123), "123"},
		{float64(123.45), "123"},
		{42, "42"},
		{nil, "<nil>"},
	}

	for _, tc := range tests {
		result := stringifyID(tc.input)
		if result != tc.expected {
			t.Errorf("stringifyID(%v): expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}
