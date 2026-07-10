package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/modelpolicy"

	"github.com/vrooli/api-core/health"
)

func TestModelPolicyHealthCheckerFailsRequiredInvalidState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-model-policy.json")
	if err := os.WriteFile(path, []byte("{\"schemaVersion\":99}"), 0o644); err != nil {
		t.Fatalf("write invalid catalog: %v", err)
	}
	state, loadErr := modelpolicy.NewState(path, modelpolicy.Requirement{
		Required: true,
		Reason:   "test policy profile",
	})
	if loadErr == nil {
		t.Fatal("expected invalid catalog load")
	}

	result := modelPolicyHealthChecker(state).Check(context.Background())
	if result.Connected || result.Error == nil {
		t.Fatalf("health result = %+v, want failed dependency", result)
	}
	detail, ok := result.Error.(*health.ErrorDetail)
	if !ok {
		t.Fatalf("health error type = %T, want *health.ErrorDetail", result.Error)
	}
	if detail.Code != modelpolicy.DiagnosticCodeCatalogInvalid {
		t.Fatalf("health code = %q", detail.Code)
	}
	if got := detail.Details["path"]; got != path {
		t.Fatalf("health path = %v, want %q", got, path)
	}
	if !strings.Contains(detail.Message, "schemaVersion") {
		t.Fatalf("health message = %q, want validation cause", detail.Message)
	}
}

func TestModelPolicyHealthCheckerAllowsOptionalInvalidState(t *testing.T) {
	state, loadErr := modelpolicy.NewState(filepath.Join(t.TempDir(), "missing.json"), modelpolicy.Requirement{})
	if loadErr == nil {
		t.Fatal("expected missing catalog load")
	}
	result := modelPolicyHealthChecker(state).Check(context.Background())
	if !result.Connected || result.Error != nil {
		t.Fatalf("optional health result = %+v, want connected", result)
	}
}

func TestModelPolicyHealthCheckerMakesStandardHealthEndpointUnready(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	state, loadErr := modelpolicy.NewState(path, modelpolicy.Requirement{Required: true, Reason: "test policy profile"})
	if loadErr == nil {
		t.Fatal("expected missing catalog load")
	}
	handler := health.New("agent-manager-api").
		Check(modelPolicyHealthChecker(state), health.Critical).
		Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("health status = %d, want %d: %s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
	var response health.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if response.Readiness || response.Status != health.StatusUnhealthy {
		t.Fatalf("health response = %+v, want unhealthy and unready", response)
	}
	dependency := response.Dependencies["model_policy_catalog"]
	if dependency.Connected || dependency.Error == nil {
		t.Fatalf("catalog dependency = %+v, want failed diagnostic", dependency)
	}
}
