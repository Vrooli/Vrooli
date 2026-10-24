package agentsessions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterRoutesDoesNotMountRetiredProposalDecisionAlias(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(nil).RegisterRoutes(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/agent-sessions/session-1/proposals/proposal-1/decide", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("proposal decision alias status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
