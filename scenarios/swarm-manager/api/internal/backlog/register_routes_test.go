package backlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterRoutes_List(t *testing.T) {
	rootDir := t.TempDir()
	handler := NewHandler(rootDir, rootDir)
	r := mux.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestRegisterRoutes_DoesNotExposeLegacyReviewDecide(t *testing.T) {
	rootDir := t.TempDir()
	handler := NewHandler(rootDir, rootDir)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/item/review-decide", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusNotFound {
		t.Fatalf("legacy review-decide route status = %d, want 404", response.Code)
	}
}
