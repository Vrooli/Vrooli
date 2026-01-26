package skills

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockStore implements SkillStore for testing.
type MockStore struct {
	skills   map[string][]Metadata // folder -> skills
	contents map[string]string     // folder/filename -> content
}

func NewMockStore() *MockStore {
	return &MockStore{
		skills:   make(map[string][]Metadata),
		contents: make(map[string]string),
	}
}

func (m *MockStore) GetAll() ([]Metadata, error) {
	var all []Metadata
	for _, skills := range m.skills {
		all = append(all, skills...)
	}
	return all, nil
}

func (m *MockStore) FindByID(id string) (*Metadata, string, error) {
	for folder, skills := range m.skills {
		for _, p := range skills {
			if p.ID == id {
				return &p, folder, nil
			}
		}
	}
	return nil, "", errors.New("not found")
}

func (m *MockStore) LoadMetadata(folder string) ([]Metadata, error) {
	return m.skills[folder], nil
}

func (m *MockStore) SaveMetadata(folder string, skills []Metadata) error {
	m.skills[folder] = skills
	return nil
}

func (m *MockStore) GetContent(folder, filename string) (string, error) {
	key := folder + "/" + filename
	if content, ok := m.contents[key]; ok {
		return content, nil
	}
	return "", errors.New("content not found")
}

func (m *MockStore) SaveContent(folder, filename, content string) error {
	key := folder + "/" + filename
	m.contents[key] = content
	return nil
}

func (m *MockStore) DeleteContent(folder, filename string) error {
	key := folder + "/" + filename
	delete(m.contents, key)
	return nil
}

func (m *MockStore) GetVersions(skillID string) ([]SkillVersion, error) {
	return []SkillVersion{}, nil
}

func (m *MockStore) SaveVersion(skillID, folder string, skill *Metadata, content string) error {
	return nil
}

func (m *MockStore) GetVersionContent(skillID string, version int) (*SkillVersion, error) {
	return nil, errors.New("version not found")
}

// MockMetricsService implements MetricsService for testing.
type MockMetricsService struct{}

func (m *MockMetricsService) Get(skillID string) (*SkillMetrics, error) {
	return nil, nil
}

func (m *MockMetricsService) RecordUsage(skillID string) (int, time.Time, error) {
	return 1, time.Now(), nil
}

func (m *MockMetricsService) SetRating(skillID string, rating int, notes *string) error {
	return nil
}

func (m *MockMetricsService) Delete(skillID string) error {
	return nil
}

func TestCreate_AutoIncrementID(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics)

	// Create first skill with name "New Skill"
	req1 := CreateRequest{
		Name:    "New Skill",
		Content: "test content 1",
		Folder:  "local",
	}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handlers.Create(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: expected status 201, got %d: %s", w1.Code, w1.Body.String())
	}

	var resp1 Response
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.ID != "new-skill" {
		t.Errorf("first skill: expected ID 'new-skill', got '%s'", resp1.ID)
	}

	// Create second skill with same name - should auto-increment
	req2 := CreateRequest{
		Name:    "New Skill",
		Content: "test content 2",
		Folder:  "local",
	}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlers.Create(w2, r2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second create: expected status 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp2 Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.ID != "new-skill-1" {
		t.Errorf("second skill: expected ID 'new-skill-1', got '%s'", resp2.ID)
	}

	// Create third skill with same name - should continue incrementing
	req3 := CreateRequest{
		Name:    "New Skill",
		Content: "test content 3",
		Folder:  "local",
	}
	body3, _ := json.Marshal(req3)
	r3 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body3))
	w3 := httptest.NewRecorder()
	handlers.Create(w3, r3)

	if w3.Code != http.StatusCreated {
		t.Fatalf("third create: expected status 201, got %d: %s", w3.Code, w3.Body.String())
	}

	var resp3 Response
	json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.ID != "new-skill-2" {
		t.Errorf("third skill: expected ID 'new-skill-2', got '%s'", resp3.ID)
	}
}

func TestCreate_ExplicitIDConflict(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics)

	// Create first skill with explicit ID
	req1 := CreateRequest{
		ID:      "my-custom-id",
		Name:    "First Skill",
		Content: "test content 1",
		Folder:  "local",
	}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handlers.Create(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: expected status 201, got %d: %s", w1.Code, w1.Body.String())
	}

	// Try to create second skill with same explicit ID - should fail
	req2 := CreateRequest{
		ID:      "my-custom-id",
		Name:    "Second Skill",
		Content: "test content 2",
		Folder:  "local",
	}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlers.Create(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Errorf("second create: expected status 409 Conflict, got %d", w2.Code)
	}
}

func TestCreate_EmptyNameFallback(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics)

	// Create skill with special characters that produce empty slug
	req := CreateRequest{
		Name:    "!!!",
		Content: "test content",
		Folder:  "local",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != DefaultFallbackPrefix {
		t.Errorf("expected ID '%s', got '%s'", DefaultFallbackPrefix, resp.ID)
	}

	// Create another with same special name - should increment
	body2, _ := json.Marshal(req)
	r2 := httptest.NewRequest("POST", "/skills", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlers.Create(w2, r2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second create: expected status 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp2 Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.ID != "skill-1" {
		t.Errorf("expected ID 'skill-1', got '%s'", resp2.ID)
	}
}
