package prompts

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockStore implements PromptStore for testing.
type MockStore struct {
	prompts  map[string][]Metadata // folder -> prompts
	contents map[string]string     // folder/filename -> content
}

func NewMockStore() *MockStore {
	return &MockStore{
		prompts:  make(map[string][]Metadata),
		contents: make(map[string]string),
	}
}

func (m *MockStore) GetAll() ([]Metadata, error) {
	var all []Metadata
	for _, prompts := range m.prompts {
		all = append(all, prompts...)
	}
	return all, nil
}

func (m *MockStore) FindByID(id string) (*Metadata, string, error) {
	for folder, prompts := range m.prompts {
		for _, p := range prompts {
			if p.ID == id {
				return &p, folder, nil
			}
		}
	}
	return nil, "", errors.New("not found")
}

func (m *MockStore) LoadMetadata(folder string) ([]Metadata, error) {
	return m.prompts[folder], nil
}

func (m *MockStore) SaveMetadata(folder string, prompts []Metadata) error {
	m.prompts[folder] = prompts
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

// MockMetricsService implements MetricsService for testing.
type MockMetricsService struct{}

func (m *MockMetricsService) Get(promptID string) (*PromptMetrics, error) {
	return nil, nil
}

func (m *MockMetricsService) RecordUsage(promptID string) (int, time.Time, error) {
	return 1, time.Now(), nil
}

func (m *MockMetricsService) SetRating(promptID string, rating int, notes *string) error {
	return nil
}

func (m *MockMetricsService) Delete(promptID string) error {
	return nil
}

func TestCreate_AutoIncrementID(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics)

	// Create first prompt with name "New Prompt"
	req1 := CreateRequest{
		Name:    "New Prompt",
		Content: "test content 1",
		Folder:  "local",
	}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handlers.Create(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: expected status 201, got %d: %s", w1.Code, w1.Body.String())
	}

	var resp1 Response
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.ID != "new-prompt" {
		t.Errorf("first prompt: expected ID 'new-prompt', got '%s'", resp1.ID)
	}

	// Create second prompt with same name - should auto-increment
	req2 := CreateRequest{
		Name:    "New Prompt",
		Content: "test content 2",
		Folder:  "local",
	}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlers.Create(w2, r2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second create: expected status 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp2 Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.ID != "new-prompt-1" {
		t.Errorf("second prompt: expected ID 'new-prompt-1', got '%s'", resp2.ID)
	}

	// Create third prompt with same name - should continue incrementing
	req3 := CreateRequest{
		Name:    "New Prompt",
		Content: "test content 3",
		Folder:  "local",
	}
	body3, _ := json.Marshal(req3)
	r3 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body3))
	w3 := httptest.NewRecorder()
	handlers.Create(w3, r3)

	if w3.Code != http.StatusCreated {
		t.Fatalf("third create: expected status 201, got %d: %s", w3.Code, w3.Body.String())
	}

	var resp3 Response
	json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.ID != "new-prompt-2" {
		t.Errorf("third prompt: expected ID 'new-prompt-2', got '%s'", resp3.ID)
	}
}

func TestCreate_ExplicitIDConflict(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics)

	// Create first prompt with explicit ID
	req1 := CreateRequest{
		ID:      "my-custom-id",
		Name:    "First Prompt",
		Content: "test content 1",
		Folder:  "local",
	}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handlers.Create(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: expected status 201, got %d: %s", w1.Code, w1.Body.String())
	}

	// Try to create second prompt with same explicit ID - should fail
	req2 := CreateRequest{
		ID:      "my-custom-id",
		Name:    "Second Prompt",
		Content: "test content 2",
		Folder:  "local",
	}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body2))
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

	// Create prompt with special characters that produce empty slug
	req := CreateRequest{
		Name:    "!!!",
		Content: "test content",
		Folder:  "local",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body))
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
	r2 := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlers.Create(w2, r2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second create: expected status 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp2 Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.ID != "prompt-1" {
		t.Errorf("expected ID 'prompt-1', got '%s'", resp2.ID)
	}
}
