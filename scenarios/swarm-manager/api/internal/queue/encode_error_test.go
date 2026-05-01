package queue

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/testutil/mocks"
)

func TestList_JSONEncodeError(t *testing.T) {
	handler, _ := setupQueueHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	writer := &mocks.ErrorWriter{}

	handler.List(writer, req)

	if !writer.HasStatus(http.StatusInternalServerError) {
		t.Fatalf("expected 500 status to be written, got %v", writer.Statuses)
	}
}
