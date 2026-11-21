package control

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internal "device-control/internal/control"
	strategyregistry "device-control/strategy/registry"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestForgetLiveFlowRunRequiresCacheOnlyAcknowledgment(t *testing.T) {
	router := mux.NewRouter()
	Module(internal.New(strategyregistry.New())).Mount(router)
	for _, body := range []string{"{}", `{"acknowledge_volatile_cache_only":false}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v1/flows/runs/run-1/cache", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), "cache_only_acknowledgment_required")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v1/flows/runs/run-1/cache", strings.NewReader(`{"acknowledge_volatile_cache_only":true}`)))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), "live_run_not_found")

	var descriptorFound bool
	for _, descriptor := range Endpoints {
		if descriptor.ID == "flow_run_cache_forget" {
			descriptorFound = descriptor.Method == http.MethodDelete && descriptor.Path == "/api/v1/flows/runs/{id}/cache"
		}
	}
	require.True(t, descriptorFound)
}
