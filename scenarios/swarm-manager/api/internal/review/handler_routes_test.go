package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterRoutesDoesNotExposeEvidenceVerificationRESTMutation(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(NewService(ServiceConfig{DataRoot: t.TempDir()})).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/item/review/1/verify/proof", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("verification REST route status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
