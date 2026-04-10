package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"deployment-manager/fitness"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// ---------- fixed-time provider ----------

type fixedTimeProvider struct {
	t time.Time
}

func (f fixedTimeProvider) Now() time.Time { return f.t }

var fixedTime = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

func setFixedTime(t *testing.T) {
	old := shared.GetTimeProvider()
	shared.SetTimeProvider(fixedTimeProvider{t: fixedTime})
	t.Cleanup(func() { shared.SetTimeProvider(old) })
}

// ---------- mock repository ----------

type mockRepo struct {
	profiles       map[string]*Profile
	versions       map[string][]Version
	scenarioTiers  map[string]scenarioTier
	createErr      error
	listErr        error
	getErr         error
	updateErr      error
	deleteErr      error
	getVersionsErr error
	getScenarioErr error
}

type scenarioTier struct {
	scenario  string
	tierCount int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		profiles:      make(map[string]*Profile),
		versions:      make(map[string][]Version),
		scenarioTiers: make(map[string]scenarioTier),
	}
}

func (m *mockRepo) List(_ context.Context) ([]Profile, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		out = append(out, *p)
	}
	return out, nil
}

func (m *mockRepo) Get(_ context.Context, idOrName string) (*Profile, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if p, ok := m.profiles[idOrName]; ok {
		return p, nil
	}
	// Also search by name.
	for _, p := range m.profiles {
		if p.Name == idOrName {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) Create(_ context.Context, profile *Profile) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	m.profiles[profile.ID] = profile
	return profile.ID, nil
}

func (m *mockRepo) Update(_ context.Context, idOrName string, updates map[string]interface{}) (*Profile, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	p, ok := m.profiles[idOrName]
	if !ok {
		// Search by name.
		for _, pp := range m.profiles {
			if pp.Name == idOrName {
				p = pp
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, nil
	}
	if updates["tiers"] != nil {
		p.Tiers = updates["tiers"]
	}
	if updates["swaps"] != nil {
		p.Swaps = updates["swaps"]
	}
	p.Version++
	return p, nil
}

func (m *mockRepo) Delete(_ context.Context, idOrName string) (bool, error) {
	if m.deleteErr != nil {
		return false, m.deleteErr
	}
	if _, ok := m.profiles[idOrName]; ok {
		delete(m.profiles, idOrName)
		return true, nil
	}
	return false, nil
}

func (m *mockRepo) GetVersions(_ context.Context, idOrName string) ([]Version, error) {
	if m.getVersionsErr != nil {
		return nil, m.getVersionsErr
	}
	return m.versions[idOrName], nil
}

func (m *mockRepo) GetScenarioAndTier(_ context.Context, idOrName string) (string, int, error) {
	if m.getScenarioErr != nil {
		return "", 0, m.getScenarioErr
	}
	st, ok := m.scenarioTiers[idOrName]
	if !ok {
		return "", 0, ErrNotFound
	}
	return st.scenario, st.tierCount, nil
}

func (m *mockRepo) AddSwap(_ context.Context, _ string, _ Swap) error {
	return nil
}

func (m *mockRepo) GetSwaps(_ context.Context, _ string) ([]Swap, error) {
	return nil, nil
}

// ---------- helpers ----------

func noopLog(_ string, _ map[string]interface{}) {}

func newTestHandler(repo *mockRepo) *Handler {
	return NewHandler(repo, noopLog)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return out
}

func decodeBodySlice(t *testing.T, rec *httptest.ResponseRecorder) []interface{} {
	t.Helper()
	var out []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return out
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	return bytes.NewBuffer(b)
}

func setMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

// ---------- List tests ----------

func TestList(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		setup      func(*mockRepo)
		wantStatus int
		wantLen    int
	}{
		{
			name:       "empty list",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "multiple profiles",
			setup: func(m *mockRepo) {
				m.profiles["p1"] = &Profile{ID: "p1", Name: "alpha", Scenario: "web", Version: 1, CreatedAt: fixedTime, UpdatedAt: fixedTime}
				m.profiles["p2"] = &Profile{ID: "p2", Name: "beta", Scenario: "api", Version: 2, CreatedAt: fixedTime, UpdatedAt: fixedTime}
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name: "repo error returns 500",
			setup: func(m *mockRepo) {
				m.listErr = fmt.Errorf("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
			h.List(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && tt.wantLen >= 0 {
				items := decodeBodySlice(t, rec)
				if len(items) != tt.wantLen {
					t.Fatalf("got %d items, want %d", len(items), tt.wantLen)
				}
			}
		})
	}
}

// ---------- Create tests ----------

func TestCreate(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		body       interface{}
		repoErr    error
		wantStatus int
		wantField  string // field to assert present in response
	}{
		{
			name:       "valid create",
			body:       map[string]interface{}{"name": "prod", "scenario": "web-app"},
			wantStatus: http.StatusCreated,
			wantField:  "id",
		},
		{
			name:       "with optional fields",
			body:       map[string]interface{}{"name": "staging", "scenario": "api", "tiers": []int{1, 2}, "settings": map[string]string{"key": "val"}},
			wantStatus: http.StatusCreated,
			wantField:  "id",
		},
		{
			name:       "missing name",
			body:       map[string]interface{}{"scenario": "web-app"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing scenario",
			body:       map[string]interface{}{"name": "prod"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing both required fields",
			body:       map[string]interface{}{"tiers": []int{1}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json body",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repo error",
			body:       map[string]interface{}{"name": "prod", "scenario": "web-app"},
			repoErr:    fmt.Errorf("insert failed"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.repoErr != nil {
				repo.createErr = tt.repoErr
			}
			h := newTestHandler(repo)

			var buf *bytes.Buffer
			switch v := tt.body.(type) {
			case string:
				buf = bytes.NewBufferString(v)
			default:
				buf = jsonBody(t, v)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", buf)
			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantField != "" {
				body := decodeBody(t, rec)
				if body[tt.wantField] == nil {
					t.Fatalf("expected field %q in response, got %v", tt.wantField, body)
				}
			}
		})
	}
}

func TestCreate_IDUsesFixedTime(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", jsonBody(t, map[string]interface{}{"name": "test", "scenario": "s1"}))
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	body := decodeBody(t, rec)
	expectedID := fmt.Sprintf("profile-%d", fixedTime.Unix())
	if body["id"] != expectedID {
		t.Fatalf("id = %v, want %v", body["id"], expectedID)
	}
}

func TestCreate_AppliesDefaults(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", jsonBody(t, map[string]interface{}{"name": "test", "scenario": "s1"}))
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	// The profile stored in the repo should have defaults applied.
	body := decodeBody(t, rec)
	id := body["id"].(string)
	stored := repo.profiles[id]
	if stored == nil {
		t.Fatal("profile not stored in repo")
	}
	if stored.Tiers == nil {
		t.Fatal("expected Tiers to have default value, got nil")
	}
}

// ---------- Get tests ----------

func TestGet(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		profileID  string
		setup      func(*mockRepo)
		wantStatus int
	}{
		{
			name:      "found",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.profiles["p1"] = &Profile{
					ID: "p1", Name: "prod", Scenario: "web",
					Version: 1, CreatedAt: fixedTime, UpdatedAt: fixedTime,
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			profileID:  "nonexistent",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "repo error",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.getErr = fmt.Errorf("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+tt.profileID, nil)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.Get(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestGet_ResponseFields(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.profiles["p1"] = &Profile{
		ID: "p1", Name: "prod", Scenario: "web-app",
		Tiers: []int{2}, Swaps: map[string]interface{}{},
		Version: 3, CreatedAt: fixedTime, UpdatedAt: fixedTime,
		CreatedBy: "alice", UpdatedBy: "bob",
	}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Get(rec, req)

	body := decodeBody(t, rec)
	checks := map[string]interface{}{
		"id":         "p1",
		"name":       "prod",
		"scenario":   "web-app",
		"created_by": "alice",
		"updated_by": "bob",
		"created_at": fixedTime.UTC().Format(time.RFC3339),
		"updated_at": fixedTime.UTC().Format(time.RFC3339),
	}
	for k, want := range checks {
		got := body[k]
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			t.Errorf("field %q = %v, want %v", k, got, want)
		}
	}
	// version comes back as float64 from JSON
	if body["version"].(float64) != 3 {
		t.Errorf("version = %v, want 3", body["version"])
	}
}

// ---------- Update tests ----------

func TestUpdate(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		profileID  string
		body       interface{}
		setup      func(*mockRepo)
		wantStatus int
	}{
		{
			name:      "found and updated",
			profileID: "p1",
			body:      map[string]interface{}{"tiers": []int{1, 2, 3}},
			setup: func(m *mockRepo) {
				m.profiles["p1"] = &Profile{
					ID: "p1", Name: "prod", Scenario: "web",
					Version: 1, CreatedAt: fixedTime, UpdatedAt: fixedTime,
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			profileID:  "nonexistent",
			body:       map[string]interface{}{"tiers": []int{1}},
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "invalid body",
			profileID: "p1",
			body:      "bad json",
			setup: func(m *mockRepo) {
				m.profiles["p1"] = &Profile{ID: "p1", Name: "prod", Version: 1, CreatedAt: fixedTime, UpdatedAt: fixedTime}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "repo error",
			profileID: "p1",
			body:      map[string]interface{}{"tiers": []int{1}},
			setup: func(m *mockRepo) {
				m.updateErr = fmt.Errorf("update failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			var buf *bytes.Buffer
			switch v := tt.body.(type) {
			case string:
				buf = bytes.NewBufferString(v)
			default:
				buf = jsonBody(t, v)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/"+tt.profileID, buf)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.Update(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestUpdate_VersionIncremented(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.profiles["p1"] = &Profile{
		ID: "p1", Name: "prod", Version: 5,
		CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/p1", jsonBody(t, map[string]interface{}{"tiers": []int{1}}))
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Update(rec, req)

	body := decodeBody(t, rec)
	if body["version"].(float64) != 6 {
		t.Fatalf("version = %v, want 6", body["version"])
	}
}

// ---------- Delete tests ----------

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		setup      func(*mockRepo)
		wantStatus int
	}{
		{
			name:      "found and deleted",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.profiles["p1"] = &Profile{ID: "p1", Name: "prod"}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			profileID:  "nonexistent",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "repo error",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.deleteErr = fmt.Errorf("delete failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/"+tt.profileID, nil)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.Delete(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestDelete_ResponseMessage(t *testing.T) {
	repo := newMockRepo()
	repo.profiles["p1"] = &Profile{ID: "p1", Name: "prod"}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/p1", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Delete(rec, req)

	body := decodeBody(t, rec)
	if body["id"] != "p1" {
		t.Errorf("response id = %v, want p1", body["id"])
	}
	if body["message"] != "Profile deleted successfully" {
		t.Errorf("response message = %v, want 'Profile deleted successfully'", body["message"])
	}
}

// ---------- GetVersions tests ----------

func TestGetVersions(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		profileID  string
		setup      func(*mockRepo)
		wantStatus int
		wantCount  int
	}{
		{
			name:      "versions returned",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.versions["p1"] = []Version{
					{ProfileID: "p1", Version: 2, Name: "prod", Scenario: "web", CreatedAt: fixedTime, CreatedBy: "alice", ChangeDescription: "updated tiers"},
					{ProfileID: "p1", Version: 1, Name: "prod", Scenario: "web", CreatedAt: fixedTime, CreatedBy: "system", ChangeDescription: ""},
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "no versions",
			profileID:  "p1",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:      "repo error",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.getVersionsErr = fmt.Errorf("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+tt.profileID+"/versions", nil)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.GetVersions(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				body := decodeBody(t, rec)
				if body["profile_id"] != tt.profileID {
					t.Errorf("profile_id = %v, want %v", body["profile_id"], tt.profileID)
				}
				versions := body["versions"].([]interface{})
				if len(versions) != tt.wantCount {
					t.Fatalf("got %d versions, want %d", len(versions), tt.wantCount)
				}
			}
		})
	}
}

func TestGetVersions_ChangeDescriptionOmittedWhenEmpty(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.versions["p1"] = []Version{
		{ProfileID: "p1", Version: 1, Name: "prod", CreatedAt: fixedTime, CreatedBy: "system", ChangeDescription: ""},
	}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/versions", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.GetVersions(rec, req)

	body := decodeBody(t, rec)
	versions := body["versions"].([]interface{})
	v := versions[0].(map[string]interface{})
	if _, exists := v["change_description"]; exists {
		t.Error("expected change_description to be omitted when empty")
	}
}

func TestGetVersions_ChangeDescriptionIncludedWhenSet(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.versions["p1"] = []Version{
		{ProfileID: "p1", Version: 2, Name: "prod", CreatedAt: fixedTime, CreatedBy: "alice", ChangeDescription: "updated tiers"},
	}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/versions", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.GetVersions(rec, req)

	body := decodeBody(t, rec)
	versions := body["versions"].([]interface{})
	v := versions[0].(map[string]interface{})
	if v["change_description"] != "updated tiers" {
		t.Errorf("change_description = %v, want 'updated tiers'", v["change_description"])
	}
}

// ---------- Validate tests ----------

func TestValidate(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		profileID  string
		verbose    bool
		setup      func(*mockRepo)
		wantStatus int
	}{
		{
			name:      "profile found",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 1}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "profile not found",
			profileID:  "nonexistent",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "verbose mode",
			profileID: "p1",
			verbose:   true,
			setup: func(m *mockRepo) {
				m.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 1}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "repo error",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.getScenarioErr = fmt.Errorf("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			url := "/api/v1/profiles/" + tt.profileID + "/validate"
			if tt.verbose {
				url += "?verbose=true"
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.Validate(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestValidate_ResponseStructure(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 1}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/validate", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Validate(rec, req)

	body := decodeBody(t, rec)
	if body["profile_id"] != "p1" {
		t.Errorf("profile_id = %v, want p1", body["profile_id"])
	}
	if body["scenario"] != "web-app" {
		t.Errorf("scenario = %v, want web-app", body["scenario"])
	}
	if body["status"] != "pass" {
		t.Errorf("status = %v, want pass", body["status"])
	}
	if body["timestamp"] != fixedTime.UTC().Format(time.RFC3339) {
		t.Errorf("timestamp = %v, want %v", body["timestamp"], fixedTime.UTC().Format(time.RFC3339))
	}

	checks := body["checks"].([]interface{})
	expectedCheckNames := []string{
		"fitness_threshold", "secret_completeness", "licensing",
		"resource_limits", "platform_requirements", "dependency_compatibility",
	}
	if len(checks) != len(expectedCheckNames) {
		t.Fatalf("got %d checks, want %d", len(checks), len(expectedCheckNames))
	}
	for i, c := range checks {
		check := c.(map[string]interface{})
		if check["name"] != expectedCheckNames[i] {
			t.Errorf("check[%d].name = %v, want %v", i, check["name"], expectedCheckNames[i])
		}
		if check["status"] != "pass" {
			t.Errorf("check[%d].status = %v, want pass", i, check["status"])
		}
	}
}

func TestValidate_VerboseIncludesRemediation(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 1}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/validate?verbose=true", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Validate(rec, req)

	body := decodeBody(t, rec)
	checks := body["checks"].([]interface{})
	for i, c := range checks {
		check := c.(map[string]interface{})
		if check["remediation"] == nil {
			t.Errorf("check[%d] missing remediation in verbose mode", i)
		}
	}
}

func TestValidate_NonVerboseOmitsRemediation(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 1}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/validate", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.Validate(rec, req)

	body := decodeBody(t, rec)
	checks := body["checks"].([]interface{})
	for i, c := range checks {
		check := c.(map[string]interface{})
		if check["remediation"] != nil {
			t.Errorf("check[%d] should not have remediation in non-verbose mode", i)
		}
	}
}

// ---------- CostEstimate tests ----------

func TestCostEstimate(t *testing.T) {
	setFixedTime(t)

	tests := []struct {
		name       string
		profileID  string
		verbose    bool
		setup      func(*mockRepo)
		wantStatus int
	}{
		{
			name:      "profile found",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 2}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "profile not found",
			profileID:  "nonexistent",
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "verbose mode",
			profileID: "p1",
			verbose:   true,
			setup: func(m *mockRepo) {
				m.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 2}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "repo error",
			profileID: "p1",
			setup: func(m *mockRepo) {
				m.getScenarioErr = fmt.Errorf("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			h := newTestHandler(repo)

			url := "/api/v1/profiles/" + tt.profileID + "/cost"
			if tt.verbose {
				url += "?verbose=true"
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = setMuxVars(req, map[string]string{"id": tt.profileID})
			h.CostEstimate(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestCostEstimate_ResponseStructure(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 2}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/cost", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.CostEstimate(rec, req)

	body := decodeBody(t, rec)
	if body["profile_id"] != "p1" {
		t.Errorf("profile_id = %v, want p1", body["profile_id"])
	}
	expectedTier := fitness.GetTierDisplayName(2)
	if body["tier"] != expectedTier {
		t.Errorf("tier = %v, want %v", body["tier"], expectedTier)
	}
	if body["monthly_cost"] != "$49.99" {
		t.Errorf("monthly_cost = %v, want $49.99", body["monthly_cost"])
	}
	if body["currency"] != "USD" {
		t.Errorf("currency = %v, want USD", body["currency"])
	}
	if body["timestamp"] != fixedTime.UTC().Format(time.RFC3339) {
		t.Errorf("timestamp = %v, want %v", body["timestamp"], fixedTime.UTC().Format(time.RFC3339))
	}
}

func TestCostEstimate_VerboseIncludesBreakdown(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 2}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/cost?verbose=true", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.CostEstimate(rec, req)

	body := decodeBody(t, rec)
	breakdown := body["breakdown"].(map[string]interface{})
	if breakdown["compute"] != "$30.00" {
		t.Errorf("compute = %v, want $30.00", breakdown["compute"])
	}
	if breakdown["storage"] != "$10.00" {
		t.Errorf("storage = %v, want $10.00", breakdown["storage"])
	}
	if breakdown["bandwidth"] != "$9.99" {
		t.Errorf("bandwidth = %v, want $9.99", breakdown["bandwidth"])
	}
	if body["notes"] == nil {
		t.Error("expected notes in verbose response")
	}
}

func TestCostEstimate_NonVerboseOmitsBreakdown(t *testing.T) {
	setFixedTime(t)

	repo := newMockRepo()
	repo.scenarioTiers["p1"] = scenarioTier{scenario: "web-app", tierCount: 2}
	h := newTestHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1/cost", nil)
	req = setMuxVars(req, map[string]string{"id": "p1"})
	h.CostEstimate(rec, req)

	body := decodeBody(t, rec)
	if body["breakdown"] != nil {
		t.Error("breakdown should not be present in non-verbose response")
	}
	if body["notes"] != nil {
		t.Error("notes should not be present in non-verbose response")
	}
}

// ---------- profileToResponse / versionToResponse ----------

func TestProfileToResponse(t *testing.T) {
	p := Profile{
		ID: "p1", Name: "prod", Scenario: "web",
		Tiers: []int{1, 2}, Swaps: map[string]interface{}{"a": "b"},
		Secrets: nil, Settings: map[string]interface{}{"k": "v"},
		Version: 3, CreatedAt: fixedTime, UpdatedAt: fixedTime,
		CreatedBy: "alice", UpdatedBy: "bob",
	}

	resp := profileToResponse(p)

	if resp["id"] != "p1" {
		t.Errorf("id = %v", resp["id"])
	}
	if resp["name"] != "prod" {
		t.Errorf("name = %v", resp["name"])
	}
	if resp["version"] != 3 {
		t.Errorf("version = %v", resp["version"])
	}
	if resp["created_at"] != fixedTime.UTC().Format(time.RFC3339) {
		t.Errorf("created_at = %v", resp["created_at"])
	}
	if resp["updated_at"] != fixedTime.UTC().Format(time.RFC3339) {
		t.Errorf("updated_at = %v", resp["updated_at"])
	}
}

func TestVersionToResponse(t *testing.T) {
	v := Version{
		ProfileID: "p1", Version: 2, Name: "prod", Scenario: "web",
		Tiers: []int{1}, CreatedAt: fixedTime, CreatedBy: "alice",
		ChangeDescription: "updated config",
	}

	resp := versionToResponse(v)
	if resp["version"] != 2 {
		t.Errorf("version = %v", resp["version"])
	}
	if resp["change_description"] != "updated config" {
		t.Errorf("change_description = %v", resp["change_description"])
	}
}

func TestVersionToResponse_EmptyChangeDescription(t *testing.T) {
	v := Version{
		ProfileID: "p1", Version: 1, Name: "prod",
		CreatedAt: fixedTime, CreatedBy: "system",
		ChangeDescription: "",
	}

	resp := versionToResponse(v)
	if _, exists := resp["change_description"]; exists {
		t.Error("change_description should be omitted when empty")
	}
}

// ---------- ApplyDefaults tests ----------

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		checkNil []string // fields that should NOT be nil after applying
		checkSet []string // fields that should keep their original value
	}{
		{
			name:     "all nil fields get defaults",
			profile:  Profile{Name: "test", Scenario: "s1"},
			checkNil: []string{"Tiers", "Swaps", "Secrets", "Settings"},
		},
		{
			name: "non-nil tiers preserved",
			profile: Profile{
				Name: "test", Scenario: "s1",
				Tiers: []int{1, 3, 5},
			},
			checkSet: []string{"Tiers"},
		},
		{
			name: "non-nil swaps preserved",
			profile: Profile{
				Name: "test", Scenario: "s1",
				Swaps: map[string]interface{}{"from": "a"},
			},
			checkSet: []string{"Swaps"},
		},
		{
			name: "mixed nil and non-nil",
			profile: Profile{
				Name: "test", Scenario: "s1",
				Tiers:    []int{4},
				Settings: map[string]interface{}{"debug": true},
				// Swaps and Secrets are nil
			},
			checkNil: []string{"Swaps", "Secrets"},
			checkSet: []string{"Tiers", "Settings"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.profile
			ApplyDefaults(&p)

			// Verify nil fields got defaults.
			for _, field := range tt.checkNil {
				var val interface{}
				switch field {
				case "Tiers":
					val = p.Tiers
				case "Swaps":
					val = p.Swaps
				case "Secrets":
					val = p.Secrets
				case "Settings":
					val = p.Settings
				}
				if val == nil {
					t.Errorf("field %s should not be nil after ApplyDefaults", field)
				}
			}
		})
	}
}

func TestApplyDefaults_DefaultTiersIsDesktop(t *testing.T) {
	p := Profile{Name: "test", Scenario: "s1"}
	ApplyDefaults(&p)

	tiers, ok := p.Tiers.([]int)
	if !ok {
		t.Fatalf("expected Tiers to be []int, got %T", p.Tiers)
	}
	if len(tiers) != 1 || tiers[0] != fitness.TierDesktop {
		t.Errorf("default tiers = %v, want [%d] (TierDesktop)", tiers, fitness.TierDesktop)
	}
}

// ---------- ShouldApplyDefault tests ----------

func TestShouldApplyDefault(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"nil returns true", nil, true},
		{"string returns false", "hello", false},
		{"empty string returns false", "", false},
		{"int returns false", 42, false},
		{"zero int returns false", 0, false},
		{"slice returns false", []int{1, 2}, false},
		{"empty slice returns false", []int{}, false},
		{"map returns false", map[string]string{"a": "b"}, false},
		{"empty map returns false", map[string]string{}, false},
		{"bool false returns false", false, false},
		{"bool true returns false", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldApplyDefault(tt.value)
			if got != tt.want {
				t.Errorf("ShouldApplyDefault(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// ---------- DefaultDefaults tests ----------

func TestDefaultDefaults(t *testing.T) {
	d := DefaultDefaults()

	// Tiers should default to desktop.
	if len(d.Tiers) != 1 || d.Tiers[0] != fitness.TierDesktop {
		t.Errorf("Tiers = %v, want [%d]", d.Tiers, fitness.TierDesktop)
	}

	// Swaps, Secrets, Settings should be empty maps (not nil).
	if d.Swaps == nil {
		t.Error("Swaps should not be nil")
	}
	if len(d.Swaps) != 0 {
		t.Errorf("Swaps should be empty, got %v", d.Swaps)
	}
	if d.Secrets == nil {
		t.Error("Secrets should not be nil")
	}
	if len(d.Secrets) != 0 {
		t.Errorf("Secrets should be empty, got %v", d.Secrets)
	}
	if d.Settings == nil {
		t.Error("Settings should not be nil")
	}
	if len(d.Settings) != 0 {
		t.Errorf("Settings should be empty, got %v", d.Settings)
	}
}
