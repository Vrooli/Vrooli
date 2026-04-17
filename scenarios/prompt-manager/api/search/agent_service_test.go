package search

import (
	"context"
	"errors"
	"prompt-manager/store"
	"testing"
)

type mockAgentStore struct {
	agents []store.Agent
}

func (m *mockAgentStore) List(ctx context.Context) ([]store.Agent, error) {
	return m.agents, nil
}

func (m *mockAgentStore) Get(ctx context.Context, id string) (*store.Agent, error) {
	for _, a := range m.agents {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAgentStore) Create(ctx context.Context, agent *store.Agent) error {
	return nil
}

func (m *mockAgentStore) Update(ctx context.Context, id string, updates *store.Agent) error {
	return nil
}

func (m *mockAgentStore) Delete(ctx context.Context, id string) error {
	return nil
}

type mockAgentFileReader struct {
	files    map[string][]store.AgentFileEntry
	contents map[string]string // key: "agentID/path"
}

func (m *mockAgentFileReader) ListFiles(ctx context.Context, id string) ([]store.AgentFileEntry, error) {
	files, ok := m.files[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return files, nil
}

func (m *mockAgentFileReader) ReadFile(ctx context.Context, id, path string) (string, error) {
	key := id + "/" + path
	content, ok := m.contents[key]
	if !ok {
		return "", errors.New("not found")
	}
	return content, nil
}

func newTestAgentService(agents []store.Agent, files map[string][]store.AgentFileEntry, contents map[string]string) *AgentSearchService {
	return NewAgentSearchService(
		&mockAgentStore{agents: agents},
		&mockAgentFileReader{files: files, contents: contents},
	)
}

func TestAgentSearch_ExactNameMatch(t *testing.T) {
	svc := newTestAgentService([]store.Agent{
		{ID: "researcher", DisplayName: "Researcher", Status: "active"},
		{ID: "writer", DisplayName: "Writer", Status: "active"},
	}, nil, nil)

	resp, err := svc.Search(context.Background(), AgentSearchQuery{Query: "Researcher"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one result")
	}
	if resp.Results[0].ID != "researcher" {
		t.Errorf("expected first result to be researcher, got %s", resp.Results[0].ID)
	}
	if resp.Results[0].Score < 10.0 {
		t.Errorf("expected exact name match score >= 10, got %f", resp.Results[0].Score)
	}
}

func TestAgentSearch_FilterByTag(t *testing.T) {
	svc := newTestAgentService([]store.Agent{
		{ID: "a1", DisplayName: "Agent A", Tags: []string{"research"}, Status: "active"},
		{ID: "a2", DisplayName: "Agent B", Tags: []string{"coding"}, Status: "active"},
	}, nil, nil)

	resp, err := svc.Search(context.Background(), AgentSearchQuery{Tag: "research"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 result, got %d", resp.Total)
	}
	if resp.Results[0].ID != "a1" {
		t.Errorf("expected a1, got %s", resp.Results[0].ID)
	}
}

func TestAgentSearch_FilterByStatus(t *testing.T) {
	svc := newTestAgentService([]store.Agent{
		{ID: "a1", DisplayName: "Active Agent", Status: "active"},
		{ID: "a2", DisplayName: "Suspended Agent", Status: "suspended"},
	}, nil, nil)

	resp, err := svc.Search(context.Background(), AgentSearchQuery{Status: "suspended"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 result, got %d", resp.Total)
	}
	if resp.Results[0].ID != "a2" {
		t.Errorf("expected a2, got %s", resp.Results[0].ID)
	}
}

func TestAgentSearch_NoQuery_ReturnsAll(t *testing.T) {
	svc := newTestAgentService([]store.Agent{
		{ID: "a1", DisplayName: "Agent A", Status: "active"},
		{ID: "a2", DisplayName: "Agent B", Status: "active"},
	}, nil, nil)

	resp, err := svc.Search(context.Background(), AgentSearchQuery{})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 2 {
		t.Fatalf("expected 2 results, got %d", resp.Total)
	}
}

func TestAgentSearch_DescriptionMatch(t *testing.T) {
	svc := newTestAgentService([]store.Agent{
		{ID: "a1", DisplayName: "Alpha", Description: "Handles data processing tasks", Status: "active"},
		{ID: "a2", DisplayName: "Beta", Description: "Manages user interfaces", Status: "active"},
	}, nil, nil)

	resp, err := svc.Search(context.Background(), AgentSearchQuery{Query: "data processing"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one result")
	}
	if resp.Results[0].ID != "a1" {
		t.Errorf("expected a1, got %s", resp.Results[0].ID)
	}
}

func TestAgentContentSearch_Basic(t *testing.T) {
	svc := newTestAgentService(
		[]store.Agent{
			{ID: "a1", DisplayName: "Agent One", Status: "active"},
		},
		map[string][]store.AgentFileEntry{
			"a1": {
				{Path: "SOUL.md", IsDir: false, Size: 100},
			},
		},
		map[string]string{
			"a1/SOUL.md": "Line one\nThis has the keyword\nLine three",
		},
	)

	resp, err := svc.SearchContent(context.Background(), AgentContentSearchQuery{
		Query: "keyword",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 match, got %d", resp.Total)
	}
	if resp.Matches[0].LineNumber != 2 {
		t.Errorf("expected line 2, got %d", resp.Matches[0].LineNumber)
	}
	if resp.Matches[0].AgentID != "a1" {
		t.Errorf("expected agent a1, got %s", resp.Matches[0].AgentID)
	}
}

func TestAgentContentSearch_EmptyQuery(t *testing.T) {
	svc := newTestAgentService(nil, nil, nil)

	resp, err := svc.SearchContent(context.Background(), AgentContentSearchQuery{
		Query: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 0 {
		t.Errorf("expected 0 matches for empty query, got %d", resp.Total)
	}
}

func TestAgentContentSearch_CaseSensitive(t *testing.T) {
	svc := newTestAgentService(
		[]store.Agent{
			{ID: "a1", DisplayName: "Agent", Status: "active"},
		},
		map[string][]store.AgentFileEntry{
			"a1": {{Path: "notes.md", IsDir: false}},
		},
		map[string]string{
			"a1/notes.md": "Hello World\nhello world\nHELLO WORLD",
		},
	)

	resp, err := svc.SearchContent(context.Background(), AgentContentSearchQuery{
		Query:         "Hello",
		CaseSensitive: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 case-sensitive match, got %d", resp.Total)
	}
	if resp.Matches[0].LineNumber != 1 {
		t.Errorf("expected line 1, got %d", resp.Matches[0].LineNumber)
	}
}

func TestAgentContentSearch_Limit(t *testing.T) {
	svc := newTestAgentService(
		[]store.Agent{
			{ID: "a1", DisplayName: "Agent", Status: "active"},
		},
		map[string][]store.AgentFileEntry{
			"a1": {{Path: "data.md", IsDir: false}},
		},
		map[string]string{
			"a1/data.md": "match\nmatch\nmatch\nmatch\nmatch",
		},
	)

	resp, err := svc.SearchContent(context.Background(), AgentContentSearchQuery{
		Query: "match",
		Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 2 {
		t.Errorf("expected 2 matches (limited), got %d", resp.Total)
	}
}
