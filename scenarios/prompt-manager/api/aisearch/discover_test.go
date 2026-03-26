package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prompt-manager/search"
	"prompt-manager/skills"
	"prompt-manager/store"
)

// MockTopicStoreReader implements TopicStoreReader for testing.
type MockTopicStoreReader struct {
	topics    map[string]*store.Topic
	ancestors map[string][]store.Topic // topicID -> ancestor chain
	skills    map[string][]string      // topicID -> accumulated skill IDs
}

func NewMockTopicStoreReader() *MockTopicStoreReader {
	return &MockTopicStoreReader{
		topics:    make(map[string]*store.Topic),
		ancestors: make(map[string][]store.Topic),
		skills:    make(map[string][]string),
	}
}

func (m *MockTopicStoreReader) List(_ context.Context) ([]store.Topic, error) {
	var result []store.Topic
	for _, t := range m.topics {
		result = append(result, *t)
	}
	return result, nil
}

func (m *MockTopicStoreReader) Get(_ context.Context, id string) (*store.Topic, error) {
	if t, ok := m.topics[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}

func (m *MockTopicStoreReader) GetWithContent(_ context.Context, id string) (*store.Topic, string, error) {
	if t, ok := m.topics[id]; ok {
		return t, "topic content", nil
	}
	return nil, "", errors.New("not found")
}

func (m *MockTopicStoreReader) GetAncestors(_ context.Context, id string) ([]store.Topic, error) {
	if ancestors, ok := m.ancestors[id]; ok {
		return ancestors, nil
	}
	return nil, nil
}

func (m *MockTopicStoreReader) AccumulateSkills(_ context.Context, id string) ([]string, error) {
	if skills, ok := m.skills[id]; ok {
		return skills, nil
	}
	return nil, nil
}

func TestValidComplexity(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"minor", true},
		{"moderate", true},
		{"major", true},
		{"architectural", true},
		{"", false},
		{"unknown", false},
		{"MINOR", false},
	}

	for _, tc := range tests {
		got := ValidComplexity(tc.input)
		if got != tc.want {
			t.Errorf("ValidComplexity(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestComplexityBudgets_AllPositive(t *testing.T) {
	expected := []string{"minor", "moderate", "major", "architectural"}
	for _, c := range expected {
		v, ok := ComplexityBudgets[c]
		if !ok {
			t.Errorf("ComplexityBudgets missing key %q", c)
			continue
		}
		if v <= 0 {
			t.Errorf("ComplexityBudgets[%q] = %d, want positive", c, v)
		}
	}
	// Budgets should increase with complexity
	if ComplexityBudgets["minor"] >= ComplexityBudgets["moderate"] {
		t.Error("minor budget should be less than moderate")
	}
	if ComplexityBudgets["moderate"] >= ComplexityBudgets["major"] {
		t.Error("moderate budget should be less than major")
	}
	if ComplexityBudgets["major"] >= ComplexityBudgets["architectural"] {
		t.Error("major budget should be less than architectural")
	}
}

func TestDiscover_EmptyQueries(t *testing.T) {
	svc := &Service{}
	_, err := svc.Discover(context.Background(), nil, "moderate", 10)
	if err == nil {
		t.Fatal("expected error for empty queries")
	}
}

func TestDiscover_NoTopicStore_ReturnsResponseWithBudget(t *testing.T) {
	// Create a service with no topic store and no embedder.
	// Discover will log errors for failed searches but still return a valid response.
	mockSkills := NewMockSkillStore()

	svc := &Service{
		skillStore:    mockSkills,
		searchService: search.NewService(mockSkills),
		reindex:       &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"api"}, "moderate", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Complexity != "moderate" {
		t.Errorf("expected complexity 'moderate', got %q", resp.Complexity)
	}
	if resp.BudgetChars != 8000 {
		t.Errorf("expected budget 8000, got %d", resp.BudgetChars)
	}
	if resp.BudgetStatus != "under" {
		t.Errorf("expected budgetStatus 'under', got %q", resp.BudgetStatus)
	}
}

func TestDiscover_BudgetCalculation_Under(t *testing.T) {
	mockSkills := NewMockSkillStore()
	svc := &Service{
		skillStore:    mockSkills,
		searchService: search.NewService(mockSkills),
		reindex:       &reindexState{},
	}

	// No AI services = no results, 0 chars → under budget
	resp, err := svc.Discover(context.Background(), []string{"test"}, "minor", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.BudgetStatus != "under" {
		t.Errorf("expected budgetStatus 'under', got %q", resp.BudgetStatus)
	}
	if resp.TotalContentChars != 0 {
		t.Errorf("expected 0 total chars, got %d", resp.TotalContentChars)
	}
}

func TestDiscover_ReadCommand_Empty(t *testing.T) {
	mockSkills := NewMockSkillStore()
	svc := &Service{
		skillStore:    mockSkills,
		searchService: search.NewService(mockSkills),
		reindex:       &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"nonexistent"}, "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ReadCommand != "" {
		t.Errorf("expected empty readCommand for no results, got %q", resp.ReadCommand)
	}
}

func TestSortDiscoverTopicResults(t *testing.T) {
	depth0 := 0
	depth1 := 1
	depth2 := 2

	results := []DiscoverResult{
		{ID: "c", TopicDepth: &depth2, Score: 0.9},
		{ID: "a", TopicDepth: &depth0, Score: 0.7},
		{ID: "b", TopicDepth: &depth1, Score: 0.8},
		{ID: "a2", TopicDepth: &depth0, Score: 0.9},
	}

	sortDiscoverTopicResults(results)

	// depth 0 first (higher score first within same depth)
	if results[0].ID != "a2" {
		t.Errorf("expected first result to be 'a2' (depth 0, score 0.9), got %q", results[0].ID)
	}
	if results[1].ID != "a" {
		t.Errorf("expected second result to be 'a' (depth 0, score 0.7), got %q", results[1].ID)
	}
	if results[2].ID != "b" {
		t.Errorf("expected third result to be 'b' (depth 1), got %q", results[2].ID)
	}
	if results[3].ID != "c" {
		t.Errorf("expected fourth result to be 'c' (depth 2), got %q", results[3].ID)
	}
}

func TestSortDiscoverSearchResults(t *testing.T) {
	results := []DiscoverResult{
		{ID: "low", Score: 0.3},
		{ID: "high", Score: 0.9},
		{ID: "mid", Score: 0.6},
	}

	sortDiscoverSearchResults(results)

	if results[0].ID != "high" {
		t.Errorf("expected first result 'high', got %q", results[0].ID)
	}
	if results[1].ID != "mid" {
		t.Errorf("expected second result 'mid', got %q", results[1].ID)
	}
	if results[2].ID != "low" {
		t.Errorf("expected third result 'low', got %q", results[2].ID)
	}
}

func TestSearchTopics_NoVectorStore(t *testing.T) {
	svc := &Service{
		reindex: &reindexState{},
	}

	results, method, err := svc.SearchTopics(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if method != "none" {
		t.Errorf("expected method 'none', got %q", method)
	}
}

// --- Full pipeline tests with mock Ollama + Qdrant ---

// newTestEmbedder creates an Embedder backed by an httptest server that returns
// a fixed embedding vector for any prompt.
func newTestEmbedder(t *testing.T) (*Embedder, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(embeddingResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		})
	}))
	return NewEmbedder(srv.URL, "test-model"), srv.Close
}

// newTestVectorStore creates a VectorStore backed by an httptest server that
// returns the provided canned results for any search request.
func newTestVectorStore(t *testing.T, collection string, results []searchResultFixture) (*VectorStore, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return canned results for search
		if strings.Contains(r.URL.Path, "/points/search") {
			resp := struct {
				Result []searchResultFixture `json:"result"`
			}{Result: results}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	vs := NewVectorStore(srv.URL, "", collection, 3)
	return vs, srv.Close
}

// searchResultFixture matches the Qdrant search response structure.
type searchResultFixture struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

func TestDiscover_FullPipeline_TopicAndSkillResults(t *testing.T) {
	// Set up mock skill store with 4 skills
	mockSkills := NewMockSkillStore()
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "doc-health", Name: "Documentation Health", Description: "Doc guidance",
		File: "doc-health.md", Tags: []string{"docs"}, Modes: []string{"steer"},
	}, "documentation health content - this is about 50 chars")
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "testing", Name: "Testing", Description: "Test guidance",
		File: "testing.md", Tags: []string{"testing"}, Modes: []string{"steer"},
	}, "testing content - also about 40 characters")
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "api-steer", Name: "API Steer", Description: "API design",
		File: "api-steer.md", Tags: []string{"api"}, Modes: []string{"steer"},
	}, "api steer content that is moderately long for a skill")
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "deploy", Name: "Deploy", Description: "Deployment",
		File: "deploy.md", Tags: []string{"deploy"}, Modes: []string{"steer"},
	}, "deploy content")

	// Mock topic store: "coaching" topic (root, depth 0) accumulates doc-health + testing
	parentID := "coaching"
	mockTopics := NewMockTopicStoreReader()
	mockTopics.topics["coaching"] = &store.Topic{
		ID: "coaching", Name: "Coaching", Description: "General coaching",
		Skills: []string{"doc-health", "testing"},
	}
	mockTopics.topics["api-design"] = &store.Topic{
		ID: "api-design", Name: "API Design", Description: "API topic",
		ParentTopicID: &parentID,
		Skills:        []string{"api-steer"},
	}
	mockTopics.ancestors["coaching"] = nil                                                 // root topic → no ancestors → depth 0
	mockTopics.ancestors["api-design"] = []store.Topic{{ID: "coaching", Name: "Coaching"}} // depth 1
	mockTopics.skills["coaching"] = []string{"doc-health", "testing"}                      // accumulated
	mockTopics.skills["api-design"] = []string{"api-steer", "doc-health", "testing"}       // accumulated (includes ancestors)

	// Set up mock Ollama (embedder) and Qdrant (vector stores)
	embedder, closeEmbed := newTestEmbedder(t)
	defer closeEmbed()

	// Topic vector store returns "coaching" topic hit
	topicVS, closeTopicVS := newTestVectorStore(t, "prompt-manager-topics", []searchResultFixture{
		{
			ID: "uuid-coaching", Score: 0.85,
			Payload: map[string]interface{}{
				"topic_id":    "coaching",
				"name":        "Coaching",
				"description": "General coaching",
				"skills":      []interface{}{"doc-health", "testing"},
			},
		},
	})
	defer closeTopicVS()

	// Skill vector store returns api-steer and deploy (direct skill search results)
	skillVS, closeSkillVS := newTestVectorStore(t, "prompt-manager-skills", []searchResultFixture{
		{
			ID: "uuid-api", Score: 0.82,
			Payload: map[string]interface{}{
				"skill_id":    "api-steer",
				"name":        "API Steer",
				"description": "API design",
				"folder":      "core",
				"tags":        []interface{}{"api"},
				"modes":       []interface{}{"steer"},
			},
		},
		{
			ID: "uuid-deploy", Score: 0.65,
			Payload: map[string]interface{}{
				"skill_id":    "deploy",
				"name":        "Deploy",
				"description": "Deployment",
				"folder":      "core",
				"tags":        []interface{}{"deploy"},
				"modes":       []interface{}{"steer"},
			},
		},
	})
	defer closeSkillVS()

	svc := &Service{
		embedder:         embedder,
		vectorStore:      skillVS,
		skillStore:       mockSkills,
		searchService:    search.NewService(mockSkills),
		threshold:        0.5,
		topicVectorStore: topicVS,
		topicStore:       mockTopics,
		reindex:          &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"testing"}, "moderate", 10)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// --- Verify results exist ---
	if resp.Total == 0 {
		t.Fatal("expected results, got 0")
	}

	// --- Verify topic-sourced skills come first ---
	hasTopicResult := false
	for _, r := range resp.Results {
		if r.Source == "topic" {
			hasTopicResult = true
			break
		}
	}
	if !hasTopicResult {
		t.Error("expected at least one topic-sourced result")
	}

	// All topic results should come before search results in the slice
	seenSearch := false
	for _, r := range resp.Results {
		if r.Source == "search" {
			seenSearch = true
		}
		if r.Source == "topic" && seenSearch {
			t.Errorf("topic result %q appeared after search results (wrong ordering)", r.ID)
		}
	}

	// --- Verify content chars are populated ---
	for _, r := range resp.Results {
		if r.ContentChars == 0 {
			t.Errorf("result %q has 0 ContentChars, expected content size", r.ID)
		}
	}

	// --- Verify totalContentChars ---
	totalChars := 0
	for _, r := range resp.Results {
		totalChars += r.ContentChars
	}
	if resp.TotalContentChars != totalChars {
		t.Errorf("TotalContentChars = %d, expected sum %d", resp.TotalContentChars, totalChars)
	}

	// --- Verify readCommand ---
	if resp.ReadCommand == "" {
		t.Error("expected non-empty readCommand")
	}
	if !strings.HasPrefix(resp.ReadCommand, "prompt-manager skill read ") {
		t.Errorf("readCommand should start with 'prompt-manager skill read ', got %q", resp.ReadCommand)
	}

	// --- Verify budget ---
	if resp.BudgetChars != 8000 {
		t.Errorf("expected budget 8000 for moderate, got %d", resp.BudgetChars)
	}
	if resp.BudgetStatus != "under" {
		t.Errorf("expected budgetStatus 'under' (small test content), got %q", resp.BudgetStatus)
	}

	// --- Verify TopicName is populated on topic-sourced results ---
	for _, r := range resp.Results {
		if r.Source == "topic" && r.TopicName == "" {
			t.Errorf("topic-sourced result %q has empty TopicName", r.ID)
		}
	}
	// The coaching topic should be named "Coaching"
	for _, r := range resp.Results {
		if r.TopicID == "coaching" && r.TopicName != "Coaching" {
			t.Errorf("result %q TopicName = %q, want 'Coaching'", r.ID, r.TopicName)
		}
	}

	// --- Verify dedup: api-steer from topics should not duplicate in search results ---
	apiSteerCount := 0
	for _, r := range resp.Results {
		if r.ID == "api-steer" {
			apiSteerCount++
		}
	}
	if apiSteerCount > 1 {
		t.Errorf("api-steer appeared %d times, expected dedup to 1", apiSteerCount)
	}

	// --- Verify method ---
	if resp.Method != "ai" {
		t.Errorf("expected method 'ai', got %q", resp.Method)
	}
}

func TestDiscover_OverBudget_TrimsReadCommand(t *testing.T) {
	// Create 3 skills: 2000 chars each = 6000 total, budget minor = 4000
	mockSkills := NewMockSkillStore()
	content2k := strings.Repeat("x", 2000)
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "skill-a", Name: "Skill A", Description: "A",
		File: "skill-a.md", Tags: []string{}, Modes: []string{},
	}, content2k)
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "skill-b", Name: "Skill B", Description: "B",
		File: "skill-b.md", Tags: []string{}, Modes: []string{},
	}, content2k)
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "skill-c", Name: "Skill C", Description: "C",
		File: "skill-c.md", Tags: []string{}, Modes: []string{},
	}, content2k)

	embedder, closeEmbed := newTestEmbedder(t)
	defer closeEmbed()

	skillVS, closeSkillVS := newTestVectorStore(t, "prompt-manager-skills", []searchResultFixture{
		{ID: "uuid-a", Score: 0.9, Payload: map[string]interface{}{
			"skill_id": "skill-a", "name": "Skill A", "description": "A",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
		{ID: "uuid-b", Score: 0.8, Payload: map[string]interface{}{
			"skill_id": "skill-b", "name": "Skill B", "description": "B",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
		{ID: "uuid-c", Score: 0.7, Payload: map[string]interface{}{
			"skill_id": "skill-c", "name": "Skill C", "description": "C",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
	})
	defer closeSkillVS()

	svc := &Service{
		embedder:      embedder,
		vectorStore:   skillVS,
		skillStore:    mockSkills,
		searchService: search.NewService(mockSkills),
		threshold:     0.5,
		reindex:       &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"stuff"}, "minor", 10)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// 3 skills x 2000 chars = 6000, budget = 4000
	if resp.BudgetStatus != "over" {
		t.Errorf("expected budgetStatus 'over', got %q", resp.BudgetStatus)
	}
	if resp.TotalContentChars != 6000 {
		t.Errorf("expected totalContentChars 6000, got %d", resp.TotalContentChars)
	}

	// RecommendedReadCommand should only include skills that fit within 4000 chars
	// 2 skills x 2000 = 4000 fits, 3rd doesn't
	if resp.RecommendedReadCommand == "" {
		t.Fatal("expected non-empty RecommendedReadCommand when over budget")
	}
	if strings.Contains(resp.RecommendedReadCommand, "skill-c") {
		t.Error("RecommendedReadCommand should not include skill-c (over budget)")
	}
	if !strings.Contains(resp.RecommendedReadCommand, "skill-a") {
		t.Error("RecommendedReadCommand should include skill-a")
	}
	if !strings.Contains(resp.RecommendedReadCommand, "skill-b") {
		t.Error("RecommendedReadCommand should include skill-b")
	}

	// Full readCommand should include all 3
	if !strings.Contains(resp.ReadCommand, "skill-c") {
		t.Error("readCommand should include all skills including skill-c")
	}
}

func TestDiscover_Deduplication_TopicWinsOverSearch(t *testing.T) {
	// A skill appears from both topic search and skill search.
	// Topic source should win.
	mockSkills := NewMockSkillStore()
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "shared-skill", Name: "Shared", Description: "Shared skill",
		File: "shared-skill.md", Tags: []string{}, Modes: []string{},
	}, "shared skill content")

	mockTopics := NewMockTopicStoreReader()
	mockTopics.topics["t1"] = &store.Topic{
		ID: "t1", Name: "Topic 1",
		Skills: []string{"shared-skill"},
	}
	mockTopics.ancestors["t1"] = nil
	mockTopics.skills["t1"] = []string{"shared-skill"}

	embedder, closeEmbed := newTestEmbedder(t)
	defer closeEmbed()

	topicVS, closeTopicVS := newTestVectorStore(t, "prompt-manager-topics", []searchResultFixture{
		{ID: "uuid-t1", Score: 0.8, Payload: map[string]interface{}{
			"topic_id": "t1", "name": "Topic 1", "description": "",
			"skills": []interface{}{"shared-skill"},
		}},
	})
	defer closeTopicVS()

	skillVS, closeSkillVS := newTestVectorStore(t, "prompt-manager-skills", []searchResultFixture{
		{ID: "uuid-shared", Score: 0.9, Payload: map[string]interface{}{
			"skill_id": "shared-skill", "name": "Shared", "description": "Shared skill",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
	})
	defer closeSkillVS()

	svc := &Service{
		embedder:         embedder,
		vectorStore:      skillVS,
		skillStore:       mockSkills,
		searchService:    search.NewService(mockSkills),
		threshold:        0.5,
		topicVectorStore: topicVS,
		topicStore:       mockTopics,
		reindex:          &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"shared"}, "", 10)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should appear exactly once
	if resp.Total != 1 {
		t.Errorf("expected 1 result (deduped), got %d", resp.Total)
	}
	if len(resp.Results) > 0 && resp.Results[0].Source != "topic" {
		t.Errorf("expected source 'topic' (topic wins over search), got %q", resp.Results[0].Source)
	}
}

func TestDiscover_WithCustomBudgetConfig(t *testing.T) {
	mockSkills := NewMockSkillStore()
	content2k := strings.Repeat("x", 2000)
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "skill-a", Name: "Skill A", Description: "A",
		File: "skill-a.md", Tags: []string{}, Modes: []string{},
	}, content2k)
	mockSkills.AddSkill("core", skills.Metadata{
		ID: "skill-b", Name: "Skill B", Description: "B",
		File: "skill-b.md", Tags: []string{}, Modes: []string{},
	}, content2k)

	embedder, closeEmbed := newTestEmbedder(t)
	defer closeEmbed()

	skillVS, closeSkillVS := newTestVectorStore(t, "prompt-manager-skills", []searchResultFixture{
		{ID: "uuid-a", Score: 0.9, Payload: map[string]interface{}{
			"skill_id": "skill-a", "name": "Skill A", "description": "A",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
		{ID: "uuid-b", Score: 0.8, Payload: map[string]interface{}{
			"skill_id": "skill-b", "name": "Skill B", "description": "B",
			"folder": "core", "tags": []interface{}{}, "modes": []interface{}{},
		}},
	})
	defer closeSkillVS()

	// Custom budget: minor = 3000 (only 1 skill fits at 2000 chars each)
	mockBudget := &MockBudgetConfigProvider{
		cfg: BudgetConfig{Minor: 3000, Moderate: 6000, Major: 10000, Architectural: 20000},
	}

	svc := &Service{
		embedder:      embedder,
		vectorStore:   skillVS,
		skillStore:    mockSkills,
		searchService: search.NewService(mockSkills),
		threshold:     0.5,
		budgetConfig:  mockBudget,
		reindex:       &reindexState{},
	}

	resp, err := svc.Discover(context.Background(), []string{"test"}, "minor", 10)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Budget should use custom value (3000), not default (4000)
	if resp.BudgetChars != 3000 {
		t.Errorf("BudgetChars = %d, want 3000 (custom)", resp.BudgetChars)
	}

	// 2 skills x 2000 = 4000 > 3000 → over budget
	if resp.BudgetStatus != "over" {
		t.Errorf("BudgetStatus = %q, want 'over'", resp.BudgetStatus)
	}

	// RecommendedReadCommand should only include skill-a (2000 <= 3000, but 4000 > 3000)
	if resp.RecommendedReadCommand == "" {
		t.Fatal("expected non-empty RecommendedReadCommand")
	}
	if !strings.Contains(resp.RecommendedReadCommand, "skill-a") {
		t.Error("RecommendedReadCommand should include skill-a")
	}
	if strings.Contains(resp.RecommendedReadCommand, "skill-b") {
		t.Error("RecommendedReadCommand should not include skill-b (over budget)")
	}
}

// --- Handler tests ---

func TestDiscoverHandler_MissingQueries(t *testing.T) {
	h := NewHandlers(&Service{reindex: &reindexState{}})

	body, _ := json.Marshal(DiscoverRequest{})
	req, _ := http.NewRequest("POST", "/api/v1/discover", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Discover(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoverHandler_InvalidComplexity(t *testing.T) {
	h := NewHandlers(&Service{reindex: &reindexState{}})

	body, _ := json.Marshal(DiscoverRequest{
		Queries:    []string{"test"},
		Complexity: "invalid",
	})
	req, _ := http.NewRequest("POST", "/api/v1/discover", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Discover(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}
