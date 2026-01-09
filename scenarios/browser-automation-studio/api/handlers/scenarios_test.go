package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// GetScenarioPort Tests
// ============================================================================

func TestGetScenarioPort_MissingName(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios//port", nil)
	req = withURLParam(req, "name", "")
	rr := httptest.NewRecorder()

	handler.GetScenarioPort(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response["code"] != "MISSING_REQUIRED_FIELD" {
		t.Errorf("expected MISSING_REQUIRED_FIELD error code, got %v", response["code"])
	}
}

func TestGetScenarioPort_ValidScenario(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Test with a known scenario name
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/browser-automation-studio/port", nil)
	req = withURLParam(req, "name", "browser-automation-studio")
	rr := httptest.NewRecorder()

	handler.GetScenarioPort(rr, req)

	// The scenario port lookup may succeed or fail depending on system state
	// We're testing that the handler doesn't panic and returns a valid response
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 200 or 500, got %d", rr.Code)
	}

	if rr.Code == http.StatusOK {
		var response ScenarioPortInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// Port should be a valid number (0 or positive)
		if response.Port < 0 {
			t.Errorf("expected non-negative port, got %d", response.Port)
		}
	}
}

func TestGetScenarioPort_UnknownScenario(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/unknown-scenario-12345/port", nil)
	req = withURLParam(req, "name", "unknown-scenario-12345")
	rr := httptest.NewRecorder()

	handler.GetScenarioPort(rr, req)

	// Unknown scenario should return 500 (internal error from port lookup failure)
	if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusOK {
		t.Fatalf("expected status 500 or 200, got %d", rr.Code)
	}
}

// ============================================================================
// ListScenarios Tests
// ============================================================================

func TestListScenarios_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	rr := httptest.NewRecorder()

	handler.ListScenarios(rr, req)

	// Listing may succeed or fail depending on system state
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 200 or 500, got %d: %s", rr.Code, rr.Body.String())
	}

	if rr.Code == http.StatusOK {
		var response map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// Should have scenarios key
		if _, ok := response["scenarios"]; !ok {
			t.Error("expected 'scenarios' key in response")
		}
	}
}

// ============================================================================
// ScenarioPortInfo Struct Tests
// ============================================================================

func TestScenarioPortInfo_Struct(t *testing.T) {
	info := ScenarioPortInfo{
		Port:   8080,
		Status: "running",
		URL:    "http://localhost:8080",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ScenarioPortInfo
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Port != info.Port {
		t.Errorf("expected Port %d, got %d", info.Port, decoded.Port)
	}
	if decoded.Status != info.Status {
		t.Errorf("expected Status %q, got %q", info.Status, decoded.Status)
	}
	if decoded.URL != info.URL {
		t.Errorf("expected URL %q, got %q", info.URL, decoded.URL)
	}
}

func TestScenarioPortInfo_EmptyStruct(t *testing.T) {
	info := ScenarioPortInfo{}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ScenarioPortInfo
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Port != 0 {
		t.Errorf("expected Port 0, got %d", decoded.Port)
	}
	if decoded.Status != "" {
		t.Errorf("expected empty Status, got %q", decoded.Status)
	}
	if decoded.URL != "" {
		t.Errorf("expected empty URL, got %q", decoded.URL)
	}
}
