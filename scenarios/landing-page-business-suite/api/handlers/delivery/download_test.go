package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeRequiresAppBeforeDependencyCalls(t *testing.T) {
	called := false
	handler := Authorize(Dependencies{
		UserEmail:  func(context.Context) string { called = true; return "user@example.com" },
		WriteError: func(w http.ResponseWriter, status int, message, kind string) { w.WriteHeader(status) },
	})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/downloads?platform=linux", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("identity lookup called before required app validation")
	}
}
