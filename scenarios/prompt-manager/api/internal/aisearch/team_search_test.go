package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prompt-manager/internal/search"
	"prompt-manager/internal/store"
)

// --- Mock team stores for AI search tests ---

type mockTeamStoreReader struct {
	teams []store.Team
	err   error
}

func (m *mockTeamStoreReader) List(ctx context.Context) ([]store.Team, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.teams, nil
}

func (m *mockTeamStoreReader) Get(ctx context.Context, id string) (*store.Team, error) {
	for _, t := range m.teams {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, errForTesting
}

type mockTeamRelReader struct {
	members map[string][]store.TeamMemberRelation
}

func (m *mockTeamRelReader) ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return m.members[teamID], nil
}

// --- Compose embedding text tests ---

func TestComposeTeamEmbeddingText(t *testing.T) {
	team := &store.Team{
		DisplayName: "Engineering",
		Mission:     "Build great software",
	}

	result := composeTeamEmbeddingText(team, []string{"alice", "bob"})

	if !strings.Contains(result, "Engineering") {
		t.Error("expected result to contain display name")
	}
	if !strings.Contains(result, "Build great software") {
		t.Error("expected result to contain mission")
	}
	if !strings.Contains(result, "Members: alice, bob") {
		t.Error("expected result to contain member names")
	}
}

func TestComposeTeamEmbeddingText_EmptyFields(t *testing.T) {
	team := &store.Team{DisplayName: "Just a Name"}

	result := composeTeamEmbeddingText(team, nil)

	if result != "Just a Name" {
		t.Errorf("expected only name, got: %s", result)
	}
}

// --- Team point ID tests ---

func TestTeamPointID_DifferentFromOtherTypes(t *testing.T) {
	teamID := teamPointID("my-team")
	skillID := qdrantPointID("my-team")
	agentID := agentPointID("my-team")

	if teamID == skillID {
		t.Error("team and skill point IDs should differ")
	}
	if teamID == agentID {
		t.Error("team and agent point IDs should differ")
	}
}

// --- SearchTeams tests ---

func TestSearchTeams_AISuccess(t *testing.T) {
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
					{ID: "uuid-1", Score: 0.9, Payload: map[string]interface{}{
						"team_id":      "eng",
						"display_name": "Engineering",
						"mission":      "Build stuff",
						"enabled":      true,
						"member_count": float64(5),
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
	teamVS := NewVectorStore(qdrantServer.URL, "", "teams", 3)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	teamStore := &mockTeamStoreReader{teams: []store.Team{
		{ID: "eng", DisplayName: "Engineering", Mission: "Build stuff", Enabled: true},
	}}
	service.SetTeamSearch(teamVS, teamStore, nil, nil)

	result, err := service.SearchTeams(context.Background(), "engineering", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "ai" {
		t.Errorf("expected method 'ai', got '%s'", result.Method)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].ID != "eng" {
		t.Errorf("expected ID 'eng', got '%s'", result.Results[0].ID)
	}
	if result.Results[0].MemberCount != 5 {
		t.Errorf("expected MemberCount 5, got %d", result.Results[0].MemberCount)
	}
	if result.Results[0].ScorePercent != 90 {
		t.Errorf("expected ScorePercent 90, got %d", result.Results[0].ScorePercent)
	}
}

func TestSearchTeams_FallbackToText(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollamaServer.Close()

	embedder := fakeEmbedderOK()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	teamStore := &mockTeamStoreReader{teams: []store.Team{
		{ID: "eng", DisplayName: "Engineering", Enabled: true},
	}}
	// Need a mock that satisfies store.TeamStore for TeamSearchService
	teamSearchSvc := search.NewTeamSearchService(
		&mockFullTeamStore{teams: teamStore.teams},
		nil, nil,
	)
	service.SetTeamSearch(nil, teamStore, nil, teamSearchSvc)

	result, err := service.SearchTeams(context.Background(), "Engineering", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "text" {
		t.Errorf("expected method 'text' for fallback, got '%s'", result.Method)
	}
}

func TestSearchTeams_NoSearchSvc_EmptyResults(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	service.SetTeamSearch(nil, nil, nil, nil)

	result, err := service.SearchTeams(context.Background(), "anything", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 results, got %d", result.Total)
	}
}

// --- IndexTeam tests ---

func TestIndexTeam_Success(t *testing.T) {
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
	teamVS := NewVectorStore(qdrantServer.URL, "", "teams", 3)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	teamStore := &mockTeamStoreReader{teams: []store.Team{
		{ID: "eng", DisplayName: "Engineering", Mission: "Build", Enabled: true},
	}}
	relStore := &mockTeamRelReader{members: map[string][]store.TeamMemberRelation{
		"eng": {{TeamID: "eng", AgentID: "alice"}, {TeamID: "eng", AgentID: "bob"}},
	}}
	service.SetTeamSearch(teamVS, teamStore, relStore, nil)

	err := service.IndexTeam(context.Background(), "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected upsert to be called")
	}
}

func TestIndexTeam_NotFound(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	teamVS := NewVectorStore("http://localhost:99999", "", "teams", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	teamStore := &mockTeamStoreReader{teams: nil}
	service.SetTeamSearch(teamVS, teamStore, nil, nil)

	err := service.IndexTeam(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent team")
	}
}

func TestIndexTeam_NilVectorStore(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	err := service.IndexTeam(context.Background(), "eng")
	if err != nil {
		t.Fatalf("expected no error for nil vector store, got: %v", err)
	}
}

// --- DeleteTeamFromIndex tests ---

func TestDeleteTeamFromIndex_Success(t *testing.T) {
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
	teamVS := NewVectorStore(qdrantServer.URL, "", "teams", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)
	service.SetTeamSearch(teamVS, nil, nil, nil)

	err := service.DeleteTeamFromIndex(context.Background(), "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected delete to be called")
	}
}

func TestDeleteTeamFromIndex_NilVectorStore(t *testing.T) {
	embedder := fakeEmbedderErr()
	skillVS := NewVectorStore("http://localhost:99999", "", "skills", 768)
	skillStore := NewMockSkillStore()
	searchSvc := search.NewService(skillStore)
	service := NewService(embedder, skillVS, skillStore, searchSvc, 0.5)

	err := service.DeleteTeamFromIndex(context.Background(), "eng")
	if err != nil {
		t.Fatalf("expected no error for nil vector store, got: %v", err)
	}
}

// --- toAITeamSearchResult tests ---

func TestToAITeamSearchResult(t *testing.T) {
	r := SearchResult{
		ID:    "uuid-1",
		Score: 0.88,
		Payload: map[string]interface{}{
			"team_id":      "eng",
			"display_name": "Engineering",
			"mission":      "Build great software",
			"enabled":      true,
			"member_count": float64(3),
		},
	}

	result := toAITeamSearchResult(r)

	if result.ID != "eng" {
		t.Errorf("expected ID 'eng', got '%s'", result.ID)
	}
	if result.DisplayName != "Engineering" {
		t.Errorf("expected DisplayName 'Engineering', got '%s'", result.DisplayName)
	}
	if result.Mission != "Build great software" {
		t.Errorf("expected Mission, got '%s'", result.Mission)
	}
	if !result.Enabled {
		t.Error("expected Enabled to be true")
	}
	if result.MemberCount != 3 {
		t.Errorf("expected MemberCount 3, got %d", result.MemberCount)
	}
	if result.ScorePercent != 88 {
		t.Errorf("expected ScorePercent 88, got %d", result.ScorePercent)
	}
}

// --- mockFullTeamStore satisfies store.TeamStore for search.NewTeamSearchService ---

type mockFullTeamStore struct {
	teams []store.Team
}

func (m *mockFullTeamStore) List(ctx context.Context) ([]store.Team, error) {
	return m.teams, nil
}

func (m *mockFullTeamStore) Get(ctx context.Context, id string) (*store.Team, error) {
	for _, t := range m.teams {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, errForTesting
}

func (m *mockFullTeamStore) Create(ctx context.Context, team *store.Team) error         { return nil }
func (m *mockFullTeamStore) Update(ctx context.Context, id string, u *store.Team) error { return nil }
func (m *mockFullTeamStore) Delete(ctx context.Context, id string) error                { return nil }
func (m *mockFullTeamStore) GetRoles(ctx context.Context, id string) (*store.TeamRoles, error) {
	return &store.TeamRoles{}, nil
}

func (m *mockFullTeamStore) SetRoles(ctx context.Context, id string, r *store.TeamRoles) error {
	return nil
}

func (m *mockFullTeamStore) GetOrgChart(ctx context.Context, teamID string) (*store.OrgChart, error) {
	return &store.OrgChart{}, nil
}

func (m *mockFullTeamStore) SetOrgChart(ctx context.Context, teamID string, o *store.OrgChart) error {
	return nil
}

func (m *mockFullTeamStore) GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

func (m *mockFullTeamStore) GetInbox(ctx context.Context, teamID, agentID string) (*store.TeamInbox, error) {
	return &store.TeamInbox{}, nil
}

func (m *mockFullTeamStore) SetInbox(ctx context.Context, teamID, agentID string, inbox *store.TeamInbox) error {
	return nil
}
