package backlog

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestList_EncodeError(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog", nil)
	writer := &errorWriter{}

	h.List(writer, req)

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
