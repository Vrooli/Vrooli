package settings

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type errorWriter struct {
	header   http.Header
	statuses []int
}

func (e *errorWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (e *errorWriter) WriteHeader(statusCode int) {
	e.statuses = append(e.statuses, statusCode)
}

func TestHandler_GetEncodeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	writer := &errorWriter{}
	handler.Get(writer, req)

	found := false
	for _, status := range writer.statuses {
		if status == http.StatusInternalServerError {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 500 status to be written, got %v", writer.statuses)
	}
}
