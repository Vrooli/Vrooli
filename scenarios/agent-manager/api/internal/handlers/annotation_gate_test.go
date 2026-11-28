package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnnotationGateServesTheMeasureRowsOverRESTForTheUI(t *testing.T) {
	router, _, service := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/annotations/run/run-1", strings.NewReader(`{}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("annotate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/gate?window=all", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("gate -> %d: %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Fields []map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Fields) == 0 || body.Fields[0]["kind"] != "coverage" {
		t.Fatalf("rows = %+v, want the coverage row first", body.Fields)
	}
	declared := len(service.Pack().Pack.Labels)
	labels := 0
	for _, row := range body.Fields {
		if row["kind"] != "label" {
			continue
		}
		labels++
		if row["gate_state"] == "" || row["scope"] == "" {
			t.Fatalf("label row names no gate state or scope: %+v", row)
		}
		if _, hasRate := row["rate"]; hasRate && row["proven_observed"] == "0" {
			t.Fatalf("a rate without a proven observation: %+v", row)
		}
	}
	if labels != declared {
		t.Fatalf("label rows = %d, pack declares %d", labels, declared)
	}
}

func TestAnnotationGateRejectsABadWindow(t *testing.T) {
	router, _, _ := annotationHandlerForTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/annotations/gate?window=junk", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("gate junk window -> %d", recorder.Code)
	}
}
