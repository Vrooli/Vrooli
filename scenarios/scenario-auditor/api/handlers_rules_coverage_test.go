package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	re "scenario-auditor/internal/ruleengine"
)

func TestGetTestCoverageHandler_NoNaN(t *testing.T) {
	// Ensure handler runs within repository context
	t.Setenv("VROOLI_ROOT", projectRootDir(t))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/test-coverage", nil)
	recorder := httptest.NewRecorder()

	getTestCoverageHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "NaN") {
		t.Fatalf("response contains NaN values: %s", recorder.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

// projectRootDir attempts to find the repo root for tests.
func projectRootDir(t *testing.T) string {
	t.Helper()

	root, err := re.DiscoverRepoRoot()
	if err != nil {
		t.Fatalf("resolveVrooliRoot failed: %v", err)
	}
	return root
}
