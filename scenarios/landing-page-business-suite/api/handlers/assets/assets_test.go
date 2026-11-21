package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadRejectsMissingMultipartFileBeforeServiceCall(t *testing.T) {
	called := false
	handler := Upload(Dependencies{
		Upload: func(_ context.Context, _ UploadInput) (UploadResult, error) {
			called = true
			return UploadResult{}, nil
		},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) { w.WriteHeader(status) },
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("upload service called for malformed multipart request")
	}
}
