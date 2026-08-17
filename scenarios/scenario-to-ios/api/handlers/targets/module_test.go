package targets

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"scenario-to-ios/internal/targets"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestModule_UsesVersionedOperatorRoute(t *testing.T) {
	m := Module(targets.Prober{GOOS: "linux"})
	router := mux.NewRouter()
	m.Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ios/targets", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "ios:simulator:linux")

	legacyRequest := httptest.NewRequest(http.MethodGet, "/ios/targets", nil)
	legacyResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyResponse, legacyRequest)
	require.Equal(t, http.StatusNotFound, legacyResponse.Code)
}
