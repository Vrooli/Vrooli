package swaps

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"deployment-manager/profiles"

	"github.com/gorilla/mux"
)

// mockProfileRepo implements profiles.Repository with just enough for swaps tests.
type mockProfileRepo struct {
	swaps      map[string][]profiles.Swap // keyed by profile ID
	addSwapErr error
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{
		swaps: make(map[string][]profiles.Swap),
	}
}

func (m *mockProfileRepo) List(_ context.Context) ([]profiles.Profile, error) { return nil, nil }
func (m *mockProfileRepo) Get(_ context.Context, _ string) (*profiles.Profile, error) {
	return nil, nil
}

func (m *mockProfileRepo) Create(_ context.Context, _ *profiles.Profile) (string, error) {
	return "", nil
}

func (m *mockProfileRepo) Update(_ context.Context, _ string, _ map[string]interface{}) (*profiles.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) Delete(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockProfileRepo) GetVersions(_ context.Context, _ string) ([]profiles.Version, error) {
	return nil, nil
}

func (m *mockProfileRepo) GetScenarioAndTier(_ context.Context, _ string) (string, int, error) {
	return "", 0, nil
}

func (m *mockProfileRepo) GetSwaps(_ context.Context, idOrName string) ([]profiles.Swap, error) {
	if s, ok := m.swaps[idOrName]; ok {
		return s, nil
	}
	return nil, profiles.ErrNotFound
}

func (m *mockProfileRepo) AddSwap(_ context.Context, idOrName string, swap profiles.Swap) error {
	if m.addSwapErr != nil {
		return m.addSwapErr
	}
	if _, ok := m.swaps[idOrName]; !ok {
		return profiles.ErrNotFound
	}
	m.swaps[idOrName] = append(m.swaps[idOrName], swap)
	return nil
}

// nopLog is a no-op logger for tests.
func nopLog(_ string, _ map[string]interface{}) {}

func newTestHandler(repo profiles.Repository) *Handler {
	return NewHandler(repo, nopLog)
}

// setVars creates a request with mux URL vars injected.
func setVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

// decodeJSON decodes the response body into the given target.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
}

// --- Analyze tests ---

func TestAnalyze_ValidSwap(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps/postgres/sqlite/analyze", nil)
	req = setVars(req, map[string]string{"from": "postgres", "to": "sqlite"})
	rec := httptest.NewRecorder()

	h.Analyze(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	if resp["from"] != "postgres" {
		t.Errorf("expected from=postgres, got %v", resp["from"])
	}
	if resp["to"] != "sqlite" {
		t.Errorf("expected to=sqlite, got %v", resp["to"])
	}
	if resp["impact"] != "medium" {
		t.Errorf("expected impact=medium, got %v", resp["impact"])
	}
	if resp["fitness_delta"] == nil {
		t.Error("expected fitness_delta to be present")
	}
	// Verify fitness_delta contains expected tiers
	deltas, ok := resp["fitness_delta"].(map[string]interface{})
	if !ok {
		t.Fatal("fitness_delta is not a map")
	}
	for _, tier := range []string{"local", "desktop", "mobile", "saas", "enterprise"} {
		if _, exists := deltas[tier]; !exists {
			t.Errorf("fitness_delta missing tier %q", tier)
		}
	}
}

func TestAnalyze_MissingFrom(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps//sqlite/analyze", nil)
	req = setVars(req, map[string]string{"from": "", "to": "sqlite"})
	rec := httptest.NewRecorder()

	h.Analyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyze_MissingTo(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps/postgres//analyze", nil)
	req = setVars(req, map[string]string{"from": "postgres", "to": ""})
	rec := httptest.NewRecorder()

	h.Analyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyze_MissingBoth(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps///analyze", nil)
	req = setVars(req, map[string]string{"from": "", "to": ""})
	rec := httptest.NewRecorder()

	h.Analyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// --- Cascade tests ---

func TestCascade_PostgresHasCascadingImpact(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps/postgres/sqlite/cascade", nil)
	req = setVars(req, map[string]string{"from": "postgres", "to": "sqlite"})
	rec := httptest.NewRecorder()

	h.Cascade(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	impacts, ok := resp["cascading_impacts"].([]interface{})
	if !ok {
		t.Fatal("cascading_impacts is not a list")
	}
	if len(impacts) == 0 {
		t.Error("expected at least one cascading impact for postgres swap")
	}

	// Verify the impact structure
	impact := impacts[0].(map[string]interface{})
	if impact["severity"] != "high" {
		t.Errorf("expected severity=high, got %v", impact["severity"])
	}
	if impact["affected_scenario"] == nil {
		t.Error("expected affected_scenario to be present")
	}
}

func TestCascade_NonPostgresNoCascade(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps/redis/in-process/cascade", nil)
	req = setVars(req, map[string]string{"from": "redis", "to": "in-process"})
	rec := httptest.NewRecorder()

	h.Cascade(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	impacts, ok := resp["cascading_impacts"].([]interface{})
	if !ok {
		t.Fatal("cascading_impacts is not a list")
	}
	if len(impacts) != 0 {
		t.Errorf("expected no cascading impacts for redis swap, got %d", len(impacts))
	}
}

func TestCascade_MissingParams(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps///cascade", nil)
	req = setVars(req, map[string]string{"from": "", "to": ""})
	rec := httptest.NewRecorder()

	h.Cascade(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCascade_WarningsAlwaysPresent(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/swaps/ollama/openrouter/cascade", nil)
	req = setVars(req, map[string]string{"from": "ollama", "to": "openrouter"})
	rec := httptest.NewRecorder()

	h.Cascade(rec, req)

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	warnings, ok := resp["warnings"].([]interface{})
	if !ok || len(warnings) == 0 {
		t.Error("expected warnings to be present and non-empty")
	}
}

// --- List tests ---

func TestList_FallsBackToDefaults(t *testing.T) {
	// When GetScenarioDependencies fails (no analyzer), List falls back to defaults
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/my-scenario/swaps", nil)
	req = setVars(req, map[string]string{"scenario": "my-scenario"})
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var suggestions []CLISwapSuggestion
	decodeJSON(t, rec, &suggestions)

	if len(suggestions) == 0 {
		t.Fatal("expected non-empty suggestions from default dependencies")
	}

	// Verify all suggestions have required fields
	for i, s := range suggestions {
		if s.From == "" {
			t.Errorf("suggestion[%d] missing From", i)
		}
		if s.To == "" {
			t.Errorf("suggestion[%d] missing To", i)
		}
		if s.Reason == "" {
			t.Errorf("suggestion[%d] missing Reason", i)
		}
	}
}

func TestList_MissingScenario(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios//swaps", nil)
	req = setVars(req, map[string]string{"scenario": ""})
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestList_ContainsKnownRegistrySwaps(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/test-scenario/swaps", nil)
	req = setVars(req, map[string]string{"scenario": "test-scenario"})
	rec := httptest.NewRecorder()

	h.List(rec, req)

	var suggestions []CLISwapSuggestion
	decodeJSON(t, rec, &suggestions)

	// Build a set of from->to pairs from the response.
	swapSet := make(map[string]CLISwapSuggestion)
	for _, s := range suggestions {
		swapSet[s.From+"->"+s.To] = s
	}

	// The fallback deps include postgres, redis, ollama, n8n, qdrant.
	// We should get at least the postgres->sqlite swap from defaults.
	if s, ok := swapSet["postgres->sqlite"]; ok {
		if s.Score != 15 {
			t.Errorf("expected postgres->sqlite score=15, got %v", s.Score)
		}
		if s.Impact != "medium" {
			t.Errorf("expected postgres->sqlite impact=medium, got %v", s.Impact)
		}
	}

	// Verify every returned suggestion maps back to a valid registry entry.
	for _, s := range suggestions {
		regSwaps, ok := swapRegistry[s.From]
		if !ok {
			t.Errorf("suggestion From=%q not in swapRegistry", s.From)
			continue
		}
		found := false
		for _, rs := range regSwaps {
			if rs.To == s.To {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("suggestion %s->%s not found in swapRegistry", s.From, s.To)
		}
	}
}

// --- Apply tests ---

func TestApply_ValidSwap(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	body := `{"profile_id":"prof-1","from":"postgres","to":"sqlite"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.Apply(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	if resp["status"] != "applied" {
		t.Errorf("expected status=applied, got %v", resp["status"])
	}
	if resp["profile_id"] != "prof-1" {
		t.Errorf("expected profile_id=prof-1, got %v", resp["profile_id"])
	}
	delta, ok := resp["fitness_delta"].(float64)
	if !ok || delta != 15 {
		t.Errorf("expected fitness_delta=15, got %v", resp["fitness_delta"])
	}
}

func TestApply_InvalidBody(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	h.Apply(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApply_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing_profile_id", `{"from":"postgres","to":"sqlite"}`},
		{"missing_from", `{"profile_id":"p1","to":"sqlite"}`},
		{"missing_to", `{"profile_id":"p1","from":"postgres"}`},
		{"all_empty", `{"profile_id":"","from":"","to":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(newMockProfileRepo())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			h.Apply(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestApply_UnknownDependency(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	body := `{"profile_id":"p1","from":"unknown-dep","to":"something"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.Apply(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApply_UnknownTarget(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	body := `{"profile_id":"p1","from":"postgres","to":"unknown-target"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.Apply(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApply_AllRegistrySwaps(t *testing.T) {
	// Verify every swap in the registry can be applied successfully.
	for dep, swaps := range swapRegistry {
		for _, s := range swaps {
			t.Run(dep+"->"+s.To, func(t *testing.T) {
				h := newTestHandler(newMockProfileRepo())
				body, _ := json.Marshal(ApplyRequest{ProfileID: "prof-1", From: s.From, To: s.To})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/swaps/apply", bytes.NewBuffer(body))
				rec := httptest.NewRecorder()

				h.Apply(rec, req)

				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

// --- ApplyToProfile tests ---

func TestApplyToProfile_Valid(t *testing.T) {
	repo := newMockProfileRepo()
	repo.swaps["prof-1"] = []profiles.Swap{} // profile exists
	h := newTestHandler(repo)

	body := `{"from":"postgres","to":"sqlite"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	decodeJSON(t, rec, &resp)

	if resp["status"] != "applied" {
		t.Errorf("expected status=applied, got %v", resp["status"])
	}
	if resp["profile_id"] != "prof-1" {
		t.Errorf("expected profile_id=prof-1, got %v", resp["profile_id"])
	}

	// Verify the swap was persisted to the mock
	if len(repo.swaps["prof-1"]) != 1 {
		t.Fatalf("expected 1 swap persisted, got %d", len(repo.swaps["prof-1"]))
	}
	persisted := repo.swaps["prof-1"][0]
	if persisted.From != "postgres" || persisted.To != "sqlite" {
		t.Errorf("persisted swap mismatch: %+v", persisted)
	}
	if persisted.AppliedAt == "" {
		t.Error("expected AppliedAt to be set")
	}
}

func TestApplyToProfile_ProfileNotFound(t *testing.T) {
	repo := newMockProfileRepo()
	// Do NOT add "nonexistent" to repo.swaps, so AddSwap returns ErrNotFound
	h := newTestHandler(repo)

	body := `{"from":"postgres","to":"sqlite"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/nonexistent/swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestApplyToProfile_MissingProfileID(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	body := `{"from":"postgres","to":"sqlite"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles//swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": ""})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApplyToProfile_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing_from", `{"to":"sqlite"}`},
		{"missing_to", `{"from":"postgres"}`},
		{"both_empty", `{"from":"","to":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockProfileRepo()
			repo.swaps["prof-1"] = []profiles.Swap{}
			h := newTestHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString(tc.body))
			req = setVars(req, map[string]string{"id": "prof-1"})
			rec := httptest.NewRecorder()

			h.ApplyToProfile(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestApplyToProfile_InvalidBody(t *testing.T) {
	h := newTestHandler(newMockProfileRepo())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString("{bad"))
	req = setVars(req, map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApplyToProfile_UnknownSwap(t *testing.T) {
	repo := newMockProfileRepo()
	repo.swaps["prof-1"] = []profiles.Swap{}
	h := newTestHandler(repo)

	body := `{"from":"unknown-dep","to":"something"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApplyToProfile_UnsupportedTarget(t *testing.T) {
	repo := newMockProfileRepo()
	repo.swaps["prof-1"] = []profiles.Swap{}
	h := newTestHandler(repo)

	body := `{"from":"postgres","to":"nonexistent-db"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApplyToProfile_InternalError(t *testing.T) {
	repo := newMockProfileRepo()
	repo.swaps["prof-1"] = []profiles.Swap{}
	repo.addSwapErr = context.DeadlineExceeded // simulate a non-ErrNotFound error
	h := newTestHandler(repo)

	body := `{"from":"postgres","to":"sqlite"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/prof-1/swaps", bytes.NewBufferString(body))
	req = setVars(req, map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ApplyToProfile(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
