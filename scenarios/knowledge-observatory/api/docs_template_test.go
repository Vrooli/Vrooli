package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleDocsTemplateList(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("GET", "/api/v1/docs/templates", nil)
	rec := httptest.NewRecorder()

	srv.handleDocsTemplateList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var items []TemplateListItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 11 {
		t.Fatalf("expected 11 templates, got %d", len(items))
	}
	for _, item := range items {
		if item.DocType == "" {
			t.Fatal("empty doc_type in list")
		}
		if item.ExpectedPath == "" {
			t.Fatalf("empty expected_path for %s", item.DocType)
		}
		if item.Purpose == "" {
			t.Fatalf("empty purpose for %s", item.DocType)
		}
	}
}

func TestHandleDocsTemplateGet(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("GET", "/api/v1/docs/templates/seams", nil)
	req = mux.SetURLVars(req, map[string]string{"doc_type": "seams"})
	rec := httptest.NewRecorder()

	srv.handleDocsTemplateGet(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var detail TemplateDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.DocType != "seams" {
		t.Fatalf("unexpected doc_type: %s", detail.DocType)
	}
	if detail.Purpose == "" {
		t.Fatal("empty purpose")
	}
	if detail.Content == "" {
		t.Fatal("empty content")
	}
}

func TestHandleDocsTemplateGet_Unknown(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("GET", "/api/v1/docs/templates/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"doc_type": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDocsTemplateGet(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleDocsTemplateGet_NoTemplate(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("GET", "/api/v1/docs/templates/readme", nil)
	req = mux.SetURLVars(req, map[string]string{"doc_type": "readme"})
	rec := httptest.NewRecorder()

	srv.handleDocsTemplateGet(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
