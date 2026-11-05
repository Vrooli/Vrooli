package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeModel is a test NER model that returns canned spans, so HTTP-level merge
// behavior can be exercised without ONNX.
type fakeModel struct {
	spans []Span
	id    string
	err   error
}

func (m fakeModel) Detect(context.Context, string) ([]Span, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.spans, nil
}
func (m fakeModel) ModelID() string { return m.id }
func (fakeModel) Close()            {}

func doDetect(t *testing.T, h http.Handler, text string) detectResponse {
	t.Helper()
	body := strings.NewReader(`{"text":` + jsonString(text) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/detect", body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detect status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp detectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode detect response: %v\n%s", err, rec.Body.String())
	}
	return resp
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestDetectRegexOnlyWhenModelUnavailable(t *testing.T) {
	h := newHandler(UnavailableModel{reason: "test"})
	resp := doDetect(t, h, "email "+validEmail+" and card "+validVisa)
	if resp.ModelAvailable {
		t.Errorf("expected model_available=false with stub model")
	}
	if !hasCategory(resp.Spans, CategoryEmail, validEmail) {
		t.Errorf("expected email span, got %+v", resp.Spans)
	}
	if !hasCategory(resp.Spans, CategoryCreditCard, "") {
		t.Errorf("expected credit-card span, got %+v", resp.Spans)
	}
}

func TestDetectMergesModelSpans(t *testing.T) {
	model := fakeModel{id: "test/model", spans: []Span{
		{Category: "PERSON", Start: 0, End: 5, Text: "Alice", Confidence: 0.8, Source: "model"},
	}}
	h := newHandler(model)
	resp := doDetect(t, h, "Alice email a@b.co")
	if !resp.ModelAvailable || resp.ModelID != "test/model" {
		t.Errorf("expected model available + id, got available=%v id=%q", resp.ModelAvailable, resp.ModelID)
	}
	if !hasCategory(resp.Spans, "PERSON", "Alice") {
		t.Errorf("expected PERSON span from model, got %+v", resp.Spans)
	}
	if !hasCategory(resp.Spans, CategoryEmail, "a@b.co") {
		t.Errorf("expected email span from regex, got %+v", resp.Spans)
	}
}

func TestDetectAlwaysReturnsArray(t *testing.T) {
	h := newHandler(UnavailableModel{})
	resp := doDetect(t, h, "nothing sensitive here")
	if resp.Spans == nil {
		t.Errorf("spans must be a non-nil array even when empty")
	}
	if len(resp.Spans) != 0 {
		t.Errorf("expected zero spans, got %+v", resp.Spans)
	}
}

func TestDetectRejectsNonPost(t *testing.T) {
	h := newHandler(UnavailableModel{})
	req := httptest.NewRequest(http.MethodGet, "/detect", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /detect status = %d, want 405", rec.Code)
	}
}

func TestDetectBadJSON(t *testing.T) {
	h := newHandler(UnavailableModel{})
	req := httptest.NewRequest(http.MethodPost, "/detect", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad JSON status = %d, want 400", rec.Code)
	}
}

func TestHealthOKWithStub(t *testing.T) {
	h := newHandler(UnavailableModel{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d", rec.Code)
	}
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if payload["status"] != "ok" {
		t.Errorf("expected status ok, got %v", payload["status"])
	}
	if payload["model_loaded"] != false {
		t.Errorf("expected model_loaded false with stub, got %v", payload["model_loaded"])
	}
}

func TestInfoListsCategories(t *testing.T) {
	h := newHandler(UnavailableModel{})
	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), CategoryEmail) {
		t.Errorf("info missing categories: %s", rec.Body.String())
	}
}
