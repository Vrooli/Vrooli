package topics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"prompt-manager/store"
	"testing"

	"github.com/gorilla/mux"
)

// MockTopicStore implements store.TopicStore for testing
type MockTopicStore struct {
	topics    map[string]*store.Topic
	children  map[string][]store.Topic
	ancestors map[string][]store.Topic
	skills    map[string][]string
}

func NewMockTopicStore() *MockTopicStore {
	return &MockTopicStore{
		topics:    make(map[string]*store.Topic),
		children:  make(map[string][]store.Topic),
		ancestors: make(map[string][]store.Topic),
		skills:    make(map[string][]string),
	}
}

func (m *MockTopicStore) List(_ context.Context) ([]store.Topic, error) {
	result := make([]store.Topic, 0, len(m.topics))
	for _, t := range m.topics {
		result = append(result, *t)
	}
	return result, nil
}

func (m *MockTopicStore) Get(_ context.Context, id string) (*store.Topic, error) {
	if t, ok := m.topics[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("topic not found: %s", id)
}

func (m *MockTopicStore) GetWithContent(_ context.Context, id string) (*store.Topic, string, error) {
	if t, ok := m.topics[id]; ok {
		return t, "# " + t.Name, nil
	}
	return nil, "", fmt.Errorf("topic not found: %s", id)
}

func (m *MockTopicStore) Create(_ context.Context, topic *store.Topic, _ string) error {
	if _, ok := m.topics[topic.ID]; ok {
		return fmt.Errorf("topic already exists: %s", topic.ID)
	}
	m.topics[topic.ID] = topic
	return nil
}

func (m *MockTopicStore) Update(_ context.Context, id string, updates *store.Topic, _ *string) error {
	existing, ok := m.topics[id]
	if !ok {
		return fmt.Errorf("topic not found: %s", id)
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.ParentTopicID != nil {
		if *updates.ParentTopicID == "" {
			existing.ParentTopicID = nil
		} else {
			existing.ParentTopicID = updates.ParentTopicID
		}
	}
	if updates.Skills != nil {
		existing.Skills = updates.Skills
	}
	if updates.Icon != "" {
		existing.Icon = updates.Icon
	}
	if updates.Status != "" {
		existing.Status = updates.Status
	}
	return nil
}

func (m *MockTopicStore) Delete(_ context.Context, id string) error {
	if _, ok := m.topics[id]; !ok {
		return fmt.Errorf("topic not found: %s", id)
	}
	delete(m.topics, id)
	return nil
}

func (m *MockTopicStore) GetAncestors(_ context.Context, id string) ([]store.Topic, error) {
	if anc, ok := m.ancestors[id]; ok {
		return anc, nil
	}
	return []store.Topic{}, nil
}

func (m *MockTopicStore) GetChildren(_ context.Context, id string) ([]store.Topic, error) {
	if ch, ok := m.children[id]; ok {
		return ch, nil
	}
	return []store.Topic{}, nil
}

func (m *MockTopicStore) AccumulateSkills(_ context.Context, id string) ([]string, error) {
	if sk, ok := m.skills[id]; ok {
		return sk, nil
	}
	return []string{}, nil
}

// MockIndexStore implements store.IndexStore for testing
type MockIndexStore struct{}

func (m *MockIndexStore) RegenerateAll(_ context.Context) error    { return nil }
func (m *MockIndexStore) RegenerateSkills(_ context.Context) error { return nil }
func (m *MockIndexStore) RegenerateAgents(_ context.Context) error { return nil }
func (m *MockIndexStore) RegenerateTeams(_ context.Context) error  { return nil }
func (m *MockIndexStore) RegenerateTopics(_ context.Context) error { return nil }

func (m *MockIndexStore) GetSkillsIndex(_ context.Context) (*store.SkillsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) GetAgentsIndex(_ context.Context) (*store.AgentsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) GetTeamsIndex(_ context.Context) (*store.TeamsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) GetTopicsIndex(_ context.Context) (*store.TopicsIndex, error) {
	return nil, nil
}

func TestMatch_WithoutMatchFn_ReturnsEmpty(t *testing.T) {
	h := &Handlers{}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{"test"},
		Limit:   5,
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp MatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Method != "none" {
		t.Errorf("expected method 'none', got %q", resp.Method)
	}
	if len(resp.Topics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(resp.Topics))
	}
	if len(resp.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(resp.Skills))
	}
}

func TestMatch_WithMatchFn_ReturnsResults(t *testing.T) {
	parentID := "parent"
	h := &Handlers{
		topicMatchFn: func(_ context.Context, queries []string, limit int) ([]MatchedTopic, []string, string, error) {
			return []MatchedTopic{
				{
					ID:            "topic-1",
					Name:          "Testing",
					Description:   "Testing practices",
					ParentTopicID: &parentID,
					Score:         0.85,
					ScorePercent:  85,
				},
			}, []string{"skill-a", "skill-b"}, "ai", nil
		},
	}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{"testing"},
		Limit:   5,
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp MatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Method != "ai" {
		t.Errorf("expected method 'ai', got %q", resp.Method)
	}
	if len(resp.Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(resp.Topics))
	}
	if resp.Topics[0].ID != "topic-1" {
		t.Errorf("expected topic ID 'topic-1', got %q", resp.Topics[0].ID)
	}
	if len(resp.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(resp.Skills))
	}
}

func TestMatch_MissingQueries_Returns400(t *testing.T) {
	h := &Handlers{}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{},
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// --- List tests ---

func TestList_ReturnsTopics(t *testing.T) {
	ts := NewMockTopicStore()
	ts.topics["go-basics"] = &store.Topic{
		ID:     "go-basics",
		Name:   "Go Basics",
		Status: "active",
	}
	ts.topics["testing"] = &store.Topic{
		ID:     "testing",
		Name:   "Testing",
		Status: "active",
	}

	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp []Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 topics, got %d", len(resp))
	}
}

func TestList_Empty(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp []Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 topics, got %d", len(resp))
	}
}

// --- Get tests ---

func TestGet_Found(t *testing.T) {
	ts := NewMockTopicStore()
	ts.topics["go-basics"] = &store.Topic{
		ID:          "go-basics",
		Name:        "Go Basics",
		Description: "Fundamentals of Go",
		Status:      "active",
	}

	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics/go-basics", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "go-basics"})
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "go-basics" {
		t.Errorf("expected ID 'go-basics', got %q", resp.ID)
	}
	if resp.Name != "Go Basics" {
		t.Errorf("expected name 'Go Basics', got %q", resp.Name)
	}
	if resp.Description != "Fundamentals of Go" {
		t.Errorf("expected description 'Fundamentals of Go', got %q", resp.Description)
	}
}

func TestGet_NotFound(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// --- Create tests ---

func TestCreate_Success(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	body, _ := json.Marshal(CreateRequest{
		Name:        "Go Testing",
		Description: "All about Go testing",
		Skills:      []string{"skill-a", "skill-b"},
	})

	req := httptest.NewRequest("POST", "/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Go Testing" {
		t.Errorf("expected name 'Go Testing', got %q", resp.Name)
	}
	// ID should be slugified from name
	if resp.ID != "go-testing" {
		t.Errorf("expected slugified ID 'go-testing', got %q", resp.ID)
	}
	if len(resp.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(resp.Skills))
	}

	// Verify stored in mock
	if _, ok := ts.topics["go-testing"]; !ok {
		t.Error("expected topic to be stored in mock store")
	}
}

func TestCreate_MissingName(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	body, _ := json.Marshal(CreateRequest{
		Description: "Missing name field",
	})

	req := httptest.NewRequest("POST", "/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// --- Update tests ---

func TestUpdate_Success(t *testing.T) {
	ts := NewMockTopicStore()
	ts.topics["go-basics"] = &store.Topic{
		ID:          "go-basics",
		Name:        "Go Basics",
		Description: "Fundamentals of Go",
		Status:      "active",
	}

	h := NewHandlers(ts, &MockIndexStore{})

	newName := "Go Fundamentals"
	newDesc := "Updated description"
	body, _ := json.Marshal(UpdateRequest{
		Name:        &newName,
		Description: &newDesc,
	})

	req := httptest.NewRequest("PUT", "/topics/go-basics", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "go-basics"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Go Fundamentals" {
		t.Errorf("expected name 'Go Fundamentals', got %q", resp.Name)
	}
	if resp.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", resp.Description)
	}
}

func TestUpdate_ClearParent(t *testing.T) {
	parentID := "programming"
	ts := NewMockTopicStore()
	ts.topics["go-basics"] = &store.Topic{
		ID:            "go-basics",
		Name:          "Go Basics",
		ParentTopicID: &parentID,
		Status:        "active",
	}

	h := NewHandlers(ts, &MockIndexStore{})

	// Send empty string to clear parent (frontend sends "" instead of null)
	emptyParent := ""
	body, _ := json.Marshal(UpdateRequest{
		ParentTopicID: &emptyParent,
	})

	req := httptest.NewRequest("PUT", "/topics/go-basics", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "go-basics"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ParentTopicID != nil {
		t.Errorf("expected parentTopicId to be nil (root topic), got %q", *resp.ParentTopicID)
	}

	// Verify in store
	stored := ts.topics["go-basics"]
	if stored.ParentTopicID != nil {
		t.Errorf("expected stored parentTopicId to be nil, got %q", *stored.ParentTopicID)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	newName := "Nonexistent"
	body, _ := json.Marshal(UpdateRequest{
		Name: &newName,
	})

	req := httptest.NewRequest("PUT", "/topics/nonexistent", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	ts := NewMockTopicStore()
	ts.topics["go-basics"] = &store.Topic{
		ID:     "go-basics",
		Name:   "Go Basics",
		Status: "active",
	}

	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("DELETE", "/topics/go-basics", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "go-basics"})
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	if _, ok := ts.topics["go-basics"]; ok {
		t.Error("expected topic to be deleted from store")
	}
}

func TestDelete_NotFound(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("DELETE", "/topics/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// --- AccumulatedSkills tests ---

func TestAccumulatedSkills_Success(t *testing.T) {
	parentID := "programming"
	ts := NewMockTopicStore()
	ts.topics["go-testing"] = &store.Topic{
		ID:            "go-testing",
		Name:          "Go Testing",
		ParentTopicID: &parentID,
		Skills:        []string{"skill-c"},
		Status:        "active",
	}
	ts.topics["programming"] = &store.Topic{
		ID:     "programming",
		Name:   "Programming",
		Skills: []string{"skill-a"},
		Status: "active",
	}
	ts.ancestors["go-testing"] = []store.Topic{
		{ID: "programming", Name: "Programming"},
	}
	ts.skills["go-testing"] = []string{"skill-a", "skill-c"}

	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics/go-testing/skills", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "go-testing"})
	rr := httptest.NewRecorder()

	h.AccumulatedSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp AccumulatedSkillsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TopicID != "go-testing" {
		t.Errorf("expected topicId 'go-testing', got %q", resp.TopicID)
	}
	if len(resp.Ancestry) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(resp.Ancestry))
	}
	if resp.Ancestry[0] != "programming" {
		t.Errorf("expected ancestor 'programming', got %q", resp.Ancestry[0])
	}
	if len(resp.Skills) != 2 {
		t.Errorf("expected 2 accumulated skills, got %d", len(resp.Skills))
	}
}

func TestAccumulatedSkills_NotFound(t *testing.T) {
	ts := NewMockTopicStore()
	h := NewHandlers(ts, &MockIndexStore{})

	req := httptest.NewRequest("GET", "/topics/nonexistent/skills", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rr := httptest.NewRecorder()

	h.AccumulatedSkills(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}
