package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"


	"github.com/gorilla/mux"
)

// MockStore implements SkillStore for testing.
type MockStore struct {
	skills   map[string][]Metadata // folder -> skills
	contents map[string]string     // folder/filename -> content
}

type storeContextKey struct{}

// contextualMockStore models the Routed FileSkillStore contract: selecting a
// request-scoped view must keep mutations out of the primary store.
type contextualMockStore struct {
	*MockStore
	test *MockStore
}

func (m *contextualMockStore) WithContext(ctx context.Context) SkillStore {
	if testMode, _ := ctx.Value(storeContextKey{}).(bool); testMode {
		return m.test
	}
	return m.MockStore
}

type fixedIdentityVerifier struct {
	identity *VerifiedWorkflowIdentity
	err      error
}

func (v fixedIdentityVerifier) VerifyWorkflowIdentity(_ context.Context, _ string) (*VerifiedWorkflowIdentity, error) {
	return v.identity, v.err
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

func (m *MockStore) LoadVersions(folder string) (map[string]*VersionFile, error) {
	return make(map[string]*VersionFile), nil
}

func (m *MockStore) SaveVersions(folder string, versions map[string]*VersionFile) error {
	return nil
}

func (m *MockStore) Rename(oldID, newID string) (*Metadata, error) {
	// Find and rename the skill
	for folder, skills := range m.skills {
		for i, p := range skills {
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
	handlers := NewHandlers(store, metrics, "/test/store")

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
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
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
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
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
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.ID != "new-skill-2" {
		t.Errorf("third skill: expected ID 'new-skill-2', got '%s'", resp3.ID)
	}
}

func TestUpdate_RejectsRemovingExistingTemplateVariable(t *testing.T) {
	store := NewMockStore()
	handlers := NewHandlers(store, &MockMetricsService{}, "/test/store")
	create := CreateRequest{Name: "Variable Guard", Content: "# Skill\n{{goal}} {{actor}}", Folder: "local"}
	createBody, _ := json.Marshal(create)
	createRequest := httptest.NewRequest(http.MethodPost, "/skills", bytes.NewReader(createBody))
	createResponse := httptest.NewRecorder()
	handlers.Create(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create: got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	replacement := "# Skill\n{{goal}}"
	updateBody, _ := json.Marshal(UpdateRequest{Content: &replacement})
	updateRequest := mux.SetURLVars(httptest.NewRequest(http.MethodPut, "/skills/variable-guard", bytes.NewReader(updateBody)), map[string]string{"id": "variable-guard"})
	updateResponse := httptest.NewRecorder()
	handlers.Update(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusBadRequest {
		t.Fatalf("update status = %d, want 400: %s", updateResponse.Code, updateResponse.Body.String())
	}
	if !strings.Contains(updateResponse.Body.String(), "actor") {
		t.Fatalf("missing variable not named: %s", updateResponse.Body.String())
	}
	content, err := store.GetContent("local", "variable-guard.md")
	if err != nil {
		t.Fatalf("read stored content: %v", err)
	}
	if content != create.Content {
		t.Fatalf("rejected update changed content: %q", content)
	}
}

func TestUpdate_ScenarioOwnedSkillReturnsActionableReadOnlyError(t *testing.T) {
	store := NewMockStore()
	store.skills["scenario"] = []Metadata{{ID: "program-runtime", File: "SKILL.md"}}
	handlers := NewHandlers(store, &MockMetricsService{}, "/test/store")
	content := "updated content"
	body, err := json.Marshal(UpdateRequest{Content: &content})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/skills/program-runtime", bytes.NewReader(body))
	request = mux.SetURLVars(request, map[string]string{"id": "program-runtime"})
	response := httptest.NewRecorder()

	handlers.Update(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "scenario-owned skills are read-only in Prompt Manager") || !strings.Contains(response.Body.String(), "source SKILL.md") {
		t.Fatalf("response does not identify the owner source: %s", response.Body.String())
	}
}

func TestCreate_UsesRequestScopedStore(t *testing.T) {
	primary := NewMockStore()
	testStore := NewMockStore()
	handlers := NewHandlers(&contextualMockStore{MockStore: primary, test: testStore}, &MockMetricsService{}, "/test/store")

	body, err := json.Marshal(CreateRequest{Name: "Isolated Skill", Content: "test content", Folder: "local"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/skills", bytes.NewReader(body)).WithContext(context.WithValue(context.Background(), storeContextKey{}, true))
	response := httptest.NewRecorder()
	handlers.Create(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if len(primary.skills["local"]) != 0 || len(primary.contents) != 0 {
		t.Fatalf("primary store was mutated by a request-scoped write: skills=%d contents=%d", len(primary.skills["local"]), len(primary.contents))
	}
	if len(testStore.skills["local"]) != 1 || testStore.contents["local/isolated-skill.md"] != "test content" {
		t.Fatalf("test store did not receive scoped write: skills=%#v contents=%#v", testStore.skills, testStore.contents)
	}
}

func TestCreate_ExplicitIDConflict(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

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
	handlers := NewHandlers(store, metrics, "/test/store")

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
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
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
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.ID != "skill-1" {
		t.Errorf("expected ID 'skill-1', got '%s'", resp2.ID)
	}
}

func addMockSkill(store *MockStore, folder, id, file, name, content string) {
	createdAt := "2024-01-01T00:00:00Z"
	skill := Metadata{
		ID:          id,
		File:        file,
		Name:        name,
		Description: "",
		Modes:       []string{},
		Tags:        []string{},
		Icon:        "",
		Draft:       false,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
		Revision:    3,
	}
	store.skills[folder] = append(store.skills[folder], skill)
	store.contents[folder+"/"+file] = content
}

func TestReadReturnsImmutableSourceMetadata(t *testing.T) {
	store := NewMockStore()
	handlers := NewHandlers(store, &MockMetricsService{}, "/test/store")
	const source = "Run {{TARGET}} with {{CONFIG}}"
	addMockSkill(store, "core", "source-skill", "source-skill.md", "Source Skill", source)

	req := ReadRequest{Identifiers: []string{"source-skill"}, Output: "both"}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	handlers.Read(w, httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("read: status %d: %s", w.Code, w.Body.String())
	}
	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Skills) != 1 || resp.Skills[0].Revision != 3 {
		t.Fatalf("skills = %+v", resp.Skills)
	}
	if resp.Skills[0].Content != source || resp.Skills[0].ContentHash != contentSHA256(source) {
		t.Fatalf("source metadata = %+v", resp.Skills[0])
	}
	if resp.Combined == "" || resp.CombinedHash != contentSHA256(resp.Combined) {
		t.Fatalf("combined metadata = %q / %q", resp.Combined, resp.CombinedHash)
	}
	if got := variableNames(resp.Skills[0].Variables); strings.Join(got, ",") != "CONFIG,TARGET" {
		t.Fatalf("variables = %v", got)
	}
}

func TestRead_ResolveAutoByID(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "ui-health", "ui-health.md", "React Coherence", "content-a")

	req := ReadRequest{Identifiers: []string{"ui-health"}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}
	if resp.Skills[0].ID != "ui-health" {
		t.Errorf("expected ID 'ui-health', got '%s'", resp.Skills[0].ID)
	}
	if resp.Skills[0].Content != "content-a" {
		t.Errorf("expected content 'content-a', got '%s'", resp.Skills[0].Content)
	}
}

func TestRead_RejectsRemovedExperimentSelectors(t *testing.T) {
	store := NewMockStore()
	handlers := NewHandlers(store, &MockMetricsService{}, "/test/store")
	addMockSkill(store, "core", "ui-health", "ui-health.md", "React Coherence", "content-a")

	for name, req := range map[string]ReadRequest{
		"experiment": {Identifiers: []string{"ui-health"}, ExperimentID: "exp-1"},
		"variant":    {Identifiers: []string{"ui-health"}, VariantID: "treatment"},
	} {
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		handlers.Read(w, httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body)))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s selector: expected status 400, got %d: %s", name, w.Code, w.Body.String())
		}
	}
}

func TestRead_ResolveFileWithoutExtension(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "ui-health", "ui-health.md", "React Coherence", "content-a")

	req := ReadRequest{Identifiers: []string{"ui-health"}, Resolve: "file"}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}
	if resp.Skills[0].ID != "ui-health" {
		t.Errorf("expected ID 'ui-health', got '%s'", resp.Skills[0].ID)
	}
}

func TestRead_ResolveFilePath(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "ui-health", "ui-health.md", "React Coherence", "content-a")

	req := ReadRequest{Identifiers: []string{"core/ui-health.md"}, Resolve: "file"}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}
	if resp.Skills[0].ID != "ui-health" {
		t.Errorf("expected ID 'ui-health', got '%s'", resp.Skills[0].ID)
	}
}

func TestRead_ResolveNameAmbiguous(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "alpha-core", "alpha.md", "Alpha", "content-a")
	addMockSkill(store, "local", "alpha-local", "alpha.md", "Alpha", "content-b")

	req := ReadRequest{Identifiers: []string{"Alpha"}, Resolve: "name"}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Ambiguous) != 1 {
		t.Fatalf("expected 1 ambiguous entry, got %d", len(resp.Ambiguous))
	}
	if len(resp.Ambiguous[0].Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(resp.Ambiguous[0].Candidates))
	}
}

func TestRead_StrictMissingReturnsNotFound(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	allowMissing := false
	req := ReadRequest{
		Identifiers:  []string{"missing"},
		AllowMissing: &allowMissing,
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("read: expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Missing) != 1 {
		t.Fatalf("expected 1 missing entry, got %d", len(resp.Missing))
	}
}

func TestRead_CombinedResolveFileWithoutExtension(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "ui-health", "ui-health.md", "React Coherence", "content-a")

	req := ReadRequest{
		Identifiers: []string{"ui-health"},
		Resolve:     "file",
		Output:      "combined",
		Format:      "xml",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if resp.SkillCount != 1 {
		t.Fatalf("expected 1 skill, got %d", resp.SkillCount)
	}
	if !strings.Contains(resp.Combined, "<skills count=\"1\">") {
		t.Fatalf("expected XML root with count, got: %s", resp.Combined)
	}
	if !strings.Contains(resp.Combined, "<![CDATA[\ncontent-a\n]]>") {
		t.Fatalf("expected CDATA content, got: %s", resp.Combined)
	}
}

func TestRead_CombinedStrictMissingReturnsNotFound(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	allowMissing := false
	req := ReadRequest{
		Identifiers:  []string{"missing"},
		AllowMissing: &allowMissing,
		Output:       "combined",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("read: expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Missing) != 1 {
		t.Fatalf("expected 1 missing entry, got %d", len(resp.Missing))
	}
}

func TestRead_ExtractsVariablesFromContent(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "test-skill", "test-skill.md", "Test Skill",
		"Run: cd scenarios/{{TARGET}}/ui && pnpm lint\nAlso check {{CONFIG}}")

	req := ReadRequest{Identifiers: []string{"test-skill"}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}

	vars := resp.Skills[0].Variables
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d: %+v", len(vars), vars)
	}

	// Variables should be sorted alphabetically
	if vars[0].Name != "CONFIG" {
		t.Errorf("expected first variable 'CONFIG', got '%s'", vars[0].Name)
	}
	if vars[1].Name != "TARGET" {
		t.Errorf("expected second variable 'TARGET', got '%s'", vars[1].Name)
	}
}

func TestRead_SubstitutesVariablesWhenProvided(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "test-skill", "test-skill.md", "Test Skill",
		"Run: cd scenarios/{{TARGET}}/ui && pnpm lint")

	req := ReadRequest{
		Identifiers: []string{"test-skill"},
		Variables:   map[string]string{"TARGET": "my-scenario"},
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("read: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}

	// Content should have substituted value
	expectedContent := "Run: cd scenarios/my-scenario/ui && pnpm lint"
	if resp.Skills[0].Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, resp.Skills[0].Content)
	}

	// Variables should still be reported (from original content)
	if len(resp.Skills[0].Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(resp.Skills[0].Variables))
	}
	if resp.Skills[0].Variables[0].Name != "TARGET" {
		t.Errorf("expected variable 'TARGET', got '%s'", resp.Skills[0].Variables[0].Name)
	}
}

func TestRead_PartialSubstitutionLeavesUnknownVariables(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "test-skill", "test-skill.md", "Test Skill",
		"{{KNOWN}} and {{UNKNOWN}}")

	req := ReadRequest{
		Identifiers: []string{"test-skill"},
		Variables:   map[string]string{"KNOWN": "replaced"},
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// KNOWN substituted, UNKNOWN left as-is
	expectedContent := "replaced and {{UNKNOWN}}"
	if resp.Skills[0].Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, resp.Skills[0].Content)
	}
}

func TestRead_NoVariablesInContent(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	addMockSkill(store, "core", "test-skill", "test-skill.md", "Test Skill",
		"Plain text without any variables")

	req := ReadRequest{Identifiers: []string{"test-skill"}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.Read(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("read: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReadResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Skills[0].Variables) != 0 {
		t.Errorf("expected no variables, got %d", len(resp.Skills[0].Variables))
	}
}

func TestSync_IncludesVariables(t *testing.T) {
	store := NewMockStore()
	metrics := &MockMetricsService{}
	handlers := NewHandlers(store, metrics, "/test/store")

	// Add skill with folder prefix in File field (as GetAll returns)
	store.skills["core"] = []Metadata{{
		ID:        "test-skill",
		File:      "core/test-skill.md",
		Name:      "Test Skill",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}}
	store.contents["core/test-skill.md"] = "Use {{TARGET}} and {{CONFIG}}"

	r := httptest.NewRequest("GET", "/skills/sync", nil)
	w := httptest.NewRecorder()
	handlers.Sync(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("sync: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SyncResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("sync: failed to parse response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}

	vars := resp.Skills[0].Variables
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d: %+v", len(vars), vars)
	}

	// Variables should be sorted alphabetically
	if vars[0].Name != "CONFIG" || vars[1].Name != "TARGET" {
		t.Errorf("expected variables CONFIG and TARGET, got %+v", vars)
	}
}

func variableNames(variables []Variable) []string {
	names := make([]string, len(variables))
	for i, variable := range variables {
		names[i] = variable.Name
	}
	return names
}

func TestDeleteSkillRefusesCoreWithoutChangingLineage(t *testing.T) {
	core := Metadata{ID: "core-example", Name: "Core example", File: "SKILL.md"}
	store := NewMockStore()
	store.skills["core"] = []Metadata{core}
	store.contents["core/SKILL.md"] = "preserved source"
	h := NewHandlers(store, &MockMetricsService{}, "/test/store")

	err := h.DeleteSkill(context.Background(), core.ID)
	if err == nil || err.Error() != "Cannot delete core skills" {
		t.Fatalf("DeleteSkill(core) error = %v, want refusal", err)
	}
	if len(store.skills["core"]) != 1 {
		t.Fatalf("core metadata changed after refused delete: %#v", store.skills["core"])
	}
	if got := store.contents["core/SKILL.md"]; got != "preserved source" {
		t.Fatalf("core source changed after refused delete: %q", got)
	}
}
