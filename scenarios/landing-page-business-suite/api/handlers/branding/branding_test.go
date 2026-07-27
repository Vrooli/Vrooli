package branding

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClearRejectsBlankFieldBeforeMutation(t *testing.T) {
	called := false
	handler := Clear(Dependencies{
		DecodeJSON: func(_ http.ResponseWriter, _ *http.Request, target any) bool {
			target.(*struct {
				Field string `json:"field"`
			}).Field = " "
			return true
		},
		Clear:      func(string) error { called = true; return nil },
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", strings.NewReader(`{}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("clear mutation was called for blank field")
	}
}
