package backlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/testutil/mocks"
)

func TestList_EncodeError(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog", nil)
	writer := &mocks.ErrorWriter{}

	h.List(writer, req)

	if !writer.HasStatus(http.StatusInternalServerError) {
		t.Fatalf("expected 500 status to be written, got %v", writer.Statuses)
	}
}
