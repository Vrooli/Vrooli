package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/runannotations"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	_ "modernc.org/sqlite"
)

type annotationDecisionFake struct{ calls int }

func (f *annotationDecisionFake) DecideBatch(_ context.Context, request *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
	f.calls++
	response := &inferencev1.DecideBatchResponse{}
	for range request.Items {
		result := &inferencev1.DecideResponse{Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL, Provider: "ollaya", Model: "winnow"}
		for _, question := range request.Questions {
			noul := 0.75
			result.Answers = append(result.Answers, &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType(), Noul: &noul})
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

const handlerPack = `{"pack":{"name":"default","version":"1","labels":[
  {
    "id":"run.type","group":"run","description":"kind","scope":"run","method":"deterministic",
    "rule":"workload_kind","evidence":[{"source":"workload","field":"kind"}],
    "gate":{"precision":0.9,"recall":0.9,"min_gold":10},
    "search":{"facet":"label:run.type","min_probability":0.6},
    "friction":{"enabled":false,"cost_unit":"run","importance":"low"},
    "compatibility":{"change":"breaking"}
  },
  {
    "id":"operator.frustration","group":"operator","description":"frustration","scope":"run","method":"decision",
    "question":{"type":"noul","instructions":"frustrated?","true_description":"yes","false_description":"no"},
    "evidence":[{"source":"operator_messages","field":"text"}],
    "gate":{"precision":0.7,"recall":0.5,"min_gold":10},
    "search":{"facet":"label:operator.frustration","min_probability":0.6},
    "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"},
    "compatibility":{"change":"breaking"}
  }
]}}`

func annotationHandlerForTest(t *testing.T) (*mux.Router, *annotationDecisionFake, *runannotations.Service) {
	t.Helper()
	dir := t.TempDir()
	packPath := filepath.Join(dir, "pack.json")
	if err := os.WriteFile(packPath, []byte(handlerPack), 0o600); err != nil {
		t.Fatal(err)
	}
	pack, err := runannotations.LoadCompiled(packPath)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(dir, "annotations.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := runannotations.NewStore(db)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixture := runannotations.NewFixtureSource()
	fixture.Set(runannotations.RunFacts{
		RunID: "run-1", Workload: runannotations.WorkloadFacts{Kind: "adhoc"},
		OperatorMessages: []runannotations.MessageFact{{EventID: "m1", Text: "please fix this"}},
		Timings:          runannotations.TimingFacts{DurationMinutes: 20},
		Terminal:         runannotations.TerminalFacts{Status: "completed"},
	})
	fake := &annotationDecisionFake{}
	service, err := runannotations.NewService(pack, runannotations.Options{
		Store: store, Features: fixture, Decisions: fake,
		Clock: func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	NewAnnotationsHandler(service).RegisterRoutes(router)
	return router, fake, service
}

func TestAnnotationRoutesExposePacksLabelsAndCoverage(t *testing.T) {
	router, _, _ := annotationHandlerForTest(t)
	for _, path := range []string{"/api/v1/annotations/packs", "/api/v1/annotations/labels?pack=default", "/api/v1/annotations/coverage?window=7d"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s -> %d: %s", path, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "default") {
			t.Fatalf("%s body = %s", path, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/coverage?window=7d", nil))
	var coverage runannotations.CoverageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &coverage); err != nil {
		t.Fatal(err)
	}
	if len(coverage.Notes) == 0 || !strings.Contains(coverage.Notes[0], "unknown, not zero") {
		t.Fatalf("empty coverage notes = %v", coverage.Notes)
	}
}

func TestAnnotationResponsesCarryTheContractKeys(t *testing.T) {
	router, _, _ := annotationHandlerForTest(t)

	get := func(path string) map[string]any {
		t.Helper()
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s -> %d: %s", path, recorder.Code, recorder.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}

	coverage := get("/api/v1/annotations/coverage?window=7d")
	for _, key := range []string{"window", "pack", "eligible_runs", "observed_runs", "coverage", "weighting", "labels"} {
		if _, ok := coverage[key]; !ok {
			t.Fatalf("coverage missing %q: %v", key, coverage)
		}
	}

	labels := get("/api/v1/annotations/labels?pack=default")
	rows, _ := labels["labels"].([]any)
	if len(rows) != 2 {
		t.Fatalf("labels = %v", labels["labels"])
	}
	first, _ := rows[0].(map[string]any)
	for _, key := range []string{"id", "definition_hash", "compatibility", "method", "scope", "gate", "search", "friction"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("label missing %q: %v", key, first)
		}
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/run/run-1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("annotate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	var run map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	runCoverage, _ := run["coverage"].(map[string]any)
	if runCoverage == nil {
		t.Fatalf("run coverage = %v", run["coverage"])
	}
	if _, ok := runCoverage["observed"]; !ok {
		t.Fatalf("run coverage missing observed: %v", runCoverage)
	}
	if _, ok := runCoverage["labels"]; !ok {
		t.Fatalf("run coverage missing labels: %v", runCoverage)
	}
	runLabels, _ := run["labels"].([]any)
	if len(runLabels) != 2 {
		t.Fatalf("run labels = %v", run["labels"])
	}
	runFirst, _ := runLabels[0].(map[string]any)
	if _, ok := runFirst["id"]; !ok {
		t.Fatalf("run label missing id: %v", runFirst)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/evaluate",
		strings.NewReader(`{"run_ids":["run-1"],"limit":1,"dry_run":true}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("evaluate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	var evaluate map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &evaluate); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"window", "submitted", "observed", "unavailable", "coverage", "labels"} {
		if _, ok := evaluate[key]; !ok {
			t.Fatalf("evaluate missing %q: %v", key, evaluate)
		}
	}
	if evaluate["submitted"].(float64) != 1 {
		t.Fatalf("submitted = %v", evaluate["submitted"])
	}
}

func TestAnnotationRunEndpointsAnnotateAndRead(t *testing.T) {
	router, fake, _ := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/run/run-1", strings.NewReader(`{"include_optional":false}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("annotate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	if fake.calls != 1 {
		t.Fatalf("decision calls = %d, want 1", fake.calls)
	}
	var view runannotations.RunView
	if err := json.Unmarshal(recorder.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.RunID != "run-1" || len(view.Labels) != 2 {
		t.Fatalf("view = %+v", view)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/run/run-1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("read -> %d: %s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Labels) != 2 || view.Counts.Observed != 2 {
		t.Fatalf("read view = %+v", view)
	}
}

func TestAnnotationRunUnknownIs404NotEmpty(t *testing.T) {
	router, _, _ := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/run/missing-run", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAnnotationBatchEvaluateAndBadInputs(t *testing.T) {
	router, fake, _ := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/evaluate", strings.NewReader(`{"run_ids":["run-1"],"limit":1}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("evaluate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	var result runannotations.EvaluateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 1 || fake.calls != 1 {
		t.Fatalf("result = %+v calls = %d", result, fake.calls)
	}
	if result.Submitted != 1 || result.Observed != 1 || result.Coverage != 1 || len(result.Labels) != 2 {
		t.Fatalf("contract result = %+v", result)
	}
	if result.Window.Until == "" {
		t.Fatalf("window = %+v", result.Window)
	}
	for _, label := range result.Labels {
		if label.ID == "" || label.State == "" || label.Gate.State == "" {
			t.Fatalf("incomplete coverage row = %+v", label)
		}
	}

	for name, body := range map[string]string{
		"invalid json":  `{`,
		"unknown field": `{"nope":true}`,
		"bad since":     `{"since":"yesterday"}`,
	} {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/evaluate", strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s -> %d: %s", name, recorder.Code, recorder.Body.String())
		}
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/coverage?window=eternity", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad window -> %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAnnotationGoldDraftIsBlindAndComplete(t *testing.T) {
	router, _, _ := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/run/run-1",
		strings.NewReader(`{"include_optional":true}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("annotate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/gold?label=operator.frustration&window=7d", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("gold -> %d: %s", recorder.Code, recorder.Body.String())
	}
	var draft map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	if draft["label"] != "operator.frustration" || draft["definition_hash"] == "" {
		t.Fatalf("draft identity = %v", draft)
	}
	runs, _ := draft["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("draft runs = %v", draft["runs"])
	}
	run := runs[0].(map[string]any)
	if run["run_id"] != "run-1" || strings.TrimSpace(run["digest"].(string)) == "" {
		t.Fatalf("draft run = %v", run)
	}
	for _, leaked := range []string{"probability", "confidence", "state"} {
		if _, ok := run[leaked]; ok {
			t.Fatalf("draft leaked %q: %v", leaked, run)
		}
	}
}
