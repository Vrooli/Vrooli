package externalurl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestExternalURLRouteResolvesAllowlistedScenario(t *testing.T) {
	var called string
	router := mux.NewRouter()
	moduleWithResolver(func(_ context.Context, scenario string) (string, error) {
		called = scenario
		return "https://web-console.example.test", nil
	}).Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/external-url?scenario=web-console", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.JSONEq(t, `{"scenario":"web-console","url":"https://web-console.example.test"}`, recorder.Body.String())
	require.Equal(t, "web-console", called)
}

func TestExternalURLRouteDoesNotResolveOtherScenarios(t *testing.T) {
	called := false
	router := mux.NewRouter()
	moduleWithResolver(func(context.Context, string) (string, error) {
		called = true
		return "https://unexpected.example.test", nil
	}).Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/external-url?scenario=other-scenario", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"scenario":"other-scenario","url":""}`, recorder.Body.String())
	require.False(t, called)
}

func TestExternalURLRouteHidesResolverFailure(t *testing.T) {
	router := mux.NewRouter()
	moduleWithResolver(func(context.Context, string) (string, error) {
		return "", errors.New("internal tunnel-manager detail")
	}).Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/external-url?scenario=web-console", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"scenario":"web-console","url":""}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "internal tunnel-manager detail")
}
