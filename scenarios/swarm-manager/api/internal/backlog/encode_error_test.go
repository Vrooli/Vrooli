package backlog

import (
	"net/http"
	"net/http/httptest"
	"swarm-manager/internal/testutil"
	"testing"
)

func TestList_EncodeError(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog", nil)
	writer := &testutil.ErrorWriter{}

	h.List(writer, req)

	if !writer.HasStatus(http.StatusInternalServerError) {
		t.Fatalf("expected 500 status to be written, got %v", writer.Statuses)
	}
}
