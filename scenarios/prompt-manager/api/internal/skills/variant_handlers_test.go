package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

// mockVariantStore implements store.VariantStore for testing.
type mockVariantStore struct {
	variants map[string]map[string]*store.Variant // skillID -> variantID -> variant
	contents map[string]string                    // skillID/variantID -> content
}

func newMockVariantStore() *mockVariantStore {
	return &mockVariantStore{
		variants: make(map[string]map[string]*store.Variant),
		contents: make(map[string]string),
	}
}

func (m *mockVariantStore) List(_ context.Context, skillID string) ([]store.Variant, error) {
	vmap := m.variants[skillID]
	var result []store.Variant
	for _, v := range vmap {
		result = append(result, *v)
	}
	return result, nil
}

func (m *mockVariantStore) Get(_ context.Context, skillID, variantID string) (*store.Variant, error) {
	if vmap, ok := m.variants[skillID]; ok {
		if v, ok := vmap[variantID]; ok {
			return v, nil
		}
	}
	return nil, errors.New("variant not found")
}

func (m *mockVariantStore) GetWithContent(_ context.Context, skillID, variantID string) (*store.Variant, string, error) {
	v, err := m.Get(context.Background(), skillID, variantID)
	if err != nil {
		return nil, "", err
	}
	key := skillID + "/" + variantID
	return v, m.contents[key], nil
}

func (m *mockVariantStore) Create(_ context.Context, skillID string, variant *store.Variant, content string) error {
	if _, ok := m.variants[skillID]; !ok {
		m.variants[skillID] = make(map[string]*store.Variant)
	}
	if _, ok := m.variants[skillID][variant.ID]; ok {
		return errors.New("variant already exists")
	}
	variant.SkillID = skillID
	variant.Entry = "VARIANT.md"
	variant.Timestamps = store.NewTimestamps()
	m.variants[skillID][variant.ID] = variant
	m.contents[skillID+"/"+variant.ID] = content
	return nil
}

func (m *mockVariantStore) Update(_ context.Context, skillID, variantID string, updates *store.Variant, content *string) error {
	v, err := m.Get(context.Background(), skillID, variantID)
	if err != nil {
		return err
	}
	if updates.Name != "" {
		v.Name = updates.Name
	}
	if content != nil {
		m.contents[skillID+"/"+variantID] = *content
	}
	v.UpdateTimestamp()
	return nil
}

func (m *mockVariantStore) Delete(_ context.Context, skillID, variantID string) error {
	if vmap, ok := m.variants[skillID]; ok {
		if _, ok := vmap[variantID]; ok {
			delete(vmap, variantID)
			delete(m.contents, skillID+"/"+variantID)
			return nil
		}
	}
	return errors.New("variant not found")
}

// mockPackSkillStore implements store.SkillStore minimally for tests.
type mockPackSkillStore struct {
	skills  map[string]*store.Skill
	updates int
}

func newMockPackSkillStore() *mockPackSkillStore {
	return &mockPackSkillStore{skills: make(map[string]*store.Skill)}
}

func (m *mockPackSkillStore) List(_ context.Context) ([]store.Skill, error) {
	var result []store.Skill
	for _, s := range m.skills {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockPackSkillStore) Get(_ context.Context, id string) (*store.Skill, error) {
	if s, ok := m.skills[id]; ok {
		return s, nil
	}
	return nil, errors.New("skill not found")
}

func (m *mockPackSkillStore) GetWithContent(_ context.Context, id string) (*store.Skill, string, error) {
	s, err := m.Get(context.Background(), id)
	if err != nil {
		return nil, "", err
	}
	return s, "original content", nil
}

func (m *mockPackSkillStore) Create(_ context.Context, _ string, s *store.Skill, _ string) error {
	m.skills[s.ID] = s
	return nil
}

func (m *mockPackSkillStore) Update(_ context.Context, id string, s *store.Skill, _ *string) error {
	if _, ok := m.skills[id]; !ok {
		return errors.New("skill not found")
	}
	m.skills[id] = s
	m.updates++
	return nil
}

func (m *mockPackSkillStore) Delete(_ context.Context, id string) error {
	delete(m.skills, id)
	return nil
}

func (m *mockPackSkillStore) GetVersionHistory(_ context.Context, _ string) ([]store.HistoryEntry, error) {
	return nil, nil
}

func (m *mockPackSkillStore) ContentPath(_, _ string) string { return "" }

func (m *mockPackSkillStore) Rename(_ context.Context, _, _ string) (*store.Skill, error) {
	return nil, errors.New("not implemented")
}

func TestVariantHandlers_CreateAndGet(t *testing.T) {
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["test-skill"] = &store.Skill{ID: "test-skill", Name: "Test", Pack: "local"}
	h := NewVariantHandlers(vs, ss)

	// Create
	reqBody := CreateVariantRequest{ID: "v1", Name: "Concise", Content: "short content"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/skills/test-skill/variants", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "test-skill"})
	w := httptest.NewRecorder()
	h.CreateVariant(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp VariantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != "v1" || resp.Name != "Concise" {
		t.Errorf("unexpected response: %+v", resp)
	}

	// Get
	req2 := httptest.NewRequest("GET", "/skills/test-skill/variants/v1", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "test-skill", "vid": "v1"})
	w2 := httptest.NewRecorder()
	h.GetVariant(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w2.Code)
	}

	var resp2 VariantResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatal(err)
	}
	if resp2.Content != "short content" {
		t.Errorf("expected content %q, got %q", "short content", resp2.Content)
	}
}

func TestVariantHandlers_List(t *testing.T) {
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	h := NewVariantHandlers(vs, ss)

	// Create two variants
	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v1", Name: "V1"}, "c1"); err != nil {
		t.Fatal(err)
	}
	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v2", Name: "V2"}, "c2"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/skills/s1/variants", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	h.ListVariants(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	var resp []VariantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 variants, got %d", len(resp))
	}
}

func TestVariantHandlers_Delete(t *testing.T) {
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	h := NewVariantHandlers(vs, ss)

	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v1", Name: "V1"}, "c1"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/skills/s1/variants/v1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1", "vid": "v1"})
	w := httptest.NewRecorder()
	h.DeleteVariant(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVariantHandlers_CreateMissingFields(t *testing.T) {
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	h := NewVariantHandlers(vs, ss)

	reqBody := CreateVariantRequest{Name: "Missing ID"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/skills/s1/variants", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	h.CreateVariant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
