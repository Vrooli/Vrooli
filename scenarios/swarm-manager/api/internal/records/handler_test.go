package records

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func newTestRouter(t *testing.T, check ScenarioChecker) *mux.Router {
	t.Helper()
	svc := NewService(NewFileStore(t.TempDir()), nil, nil)
	h := NewHandler(svc, nil)
	h.SetScenarioChecker(check)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func postCreate(t *testing.T, router *mux.Router, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCreateWarnsOnUnknownScenario(t *testing.T) {
	router := newTestRouter(t, func(slug string) (bool, string) {
		if slug == "web-console" {
			return true, ""
		}
		return false, "web-console"
	})

	rec := postCreate(t, router, `{"kind":"fix","scenario":"web-consol","trigger":"t","outcome":"shipped"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Record   Record   `json:"record"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Record.ID == "" {
		t.Error("record not created despite warning-only design")
	}
	if len(resp.Warnings) != 1 || !strings.Contains(resp.Warnings[0], `did you mean "web-console"?`) {
		t.Errorf("warnings = %v, want unknown-slug warning with suggestion", resp.Warnings)
	}

	rec = postCreate(t, router, `{"kind":"fix","scenario":"web-console","trigger":"t","outcome":"shipped"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "warnings") {
		t.Errorf("known slug should not warn: %s", rec.Body.String())
	}
}

func TestCreateAcceptsOutcomeAliasAndEvidence(t *testing.T) {
	router := newTestRouter(t, nil)
	rec := postCreate(t, router,
		`{"kind":"feature","scenario":"web-console","trigger":"t","evidence":"all suites green","outcome":"done"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Record Record `json:"record"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Record.Kind != KindExecute || resp.Record.Outcome != OutcomeShipped {
		t.Errorf("aliases not canonicalized: kind=%s outcome=%s", resp.Record.Kind, resp.Record.Outcome)
	}
	if resp.Record.Evidence != "all suites green" {
		t.Errorf("evidence not persisted: %q", resp.Record.Evidence)
	}
}

func TestCreateRejectsProseOutcomeWithGuidance(t *testing.T) {
	router := newTestRouter(t, nil)
	rec := postCreate(t, router,
		`{"kind":"fix","scenario":"x","trigger":"t","outcome":"All validation green with zero regressions across api and cli suites"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "--evidence") {
		t.Errorf("prose-outcome error should point at --evidence: %s", rec.Body.String())
	}
}

func TestCaptureDraftThenRepairPublishes(t *testing.T) {
	router := newTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/records/capture", strings.NewReader(`{"kind":"feature","scenario":"x","trigger":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	first := httptest.NewRecorder()
	router.ServeHTTP(first, req)
	if first.Code != http.StatusCreated || !strings.Contains(first.Body.String(), `"disposition":"draft"`) {
		t.Fatalf("capture = %d %s", first.Code, first.Body.String())
	}
	var draft CaptureResult
	if err := json.Unmarshal(first.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	if draft.Record.ID == "" || !strings.Contains(strings.Join(draft.NextAction, " "), draft.Record.ID) {
		t.Fatalf("missing durable repair command: %+v", draft)
	}
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/records/"+draft.Record.ID+"/capture", strings.NewReader(`{"outcome":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	second := httptest.NewRecorder()
	router.ServeHTTP(second, req)
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"disposition":"published"`) {
		t.Fatalf("repair = %d %s", second.Code, second.Body.String())
	}
}
