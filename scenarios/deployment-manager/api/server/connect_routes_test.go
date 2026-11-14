package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deployment-manager/readiness"
	"deployment-manager/releases"

	"github.com/gorilla/mux"
)

func TestSetupRoutesKeepsDesktopBundleExportCompatibilitySeam(t *testing.T) {
	router := mux.NewRouter()
	registerBundleExportCompatibilityRoute(router, func(http.ResponseWriter, *http.Request) {})
	matched := false
	if err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err == nil && path == "/api/v1/bundles/export" {
			matched = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("bundle export compatibility route is not registered")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/bundles/export", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET bundle export status = %d, want 405", response.Code)
	}
}

type stubReadinessState struct{}

func (stubReadinessState) GetLatestReadiness(context.Context, string) (*releases.ReadinessRecord, error) {
	return &releases.ReadinessRecord{ReadinessGoalRef: "goal-1", GoalClosed: true, ApprovedAtCommit: "abc"}, nil
}

func TestSetupRoutesMountsReadinessStateCompatibilitySeam(t *testing.T) {
	router := mux.NewRouter()
	registerReadinessStateCompatibilityRoute(router, readiness.StateHandler(stubReadinessState{}))
	matched := false
	if err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err == nil && path == "/api/v1/readiness/state" {
			matched = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("readiness state compatibility route is not registered")
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/readiness/state?scenario=web-console", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "goal_exists") {
		t.Fatalf("GET readiness state status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/readiness/state?scenario=web-console", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST readiness state status=%d, want 405", response.Code)
	}
}
