package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// HealthResponse structure for test assertions
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// CapabilityInfo for test assertions
type CapabilityInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response HealthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status to be healthy, got %s", response.Status)
	}

	if response.Service != "agent-dashboard-api" {
		t.Errorf("expected service to be agent-dashboard-api, got %s", response.Service)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version to be 1.0.0, got %s", response.Version)
	}
}

func TestAgentsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/agents", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(agentsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response struct {
		Agents         interface{} `json:"agents"`
		LastScan       time.Time   `json:"last_scan"`
		ScanInProgress bool        `json:"scan_in_progress"`
	}

	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Agents == nil {
		t.Error("expected agents array to be initialized")
	}
}

func TestCapabilitiesHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/capabilities", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(capabilitiesHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response struct {
		Capabilities []CapabilityInfo `json:"capabilities"`
		Total        int              `json:"total"`
	}

	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Total < 0 {
		t.Error("total should not be negative")
	}
}

func TestSearchByCapabilityHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/agents/search?capability=text-generation", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(searchByCapabilityHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// The response is wrapped in an APIResponse
	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}

	// Check the data fields
	if _, ok := response.Data["capability"]; !ok {
		t.Error("expected response data to have capability field")
	}
	if _, ok := response.Data["agents"]; !ok {
		t.Error("expected response data to have agents field")
	}
}

func TestSearchByCapabilityHandlerMissingCapability(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/agents/search", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(searchByCapabilityHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestIsValidResourceName(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		expected bool
	}{
		{"valid claude-code", "claude-code", true},
		{"valid ollama", "ollama", true},
		{"invalid resource", "invalid-resource", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidResourceName(tt.resource)
			if result != tt.expected {
				t.Errorf("isValidResourceName(%s) = %v, want %v", tt.resource, result, tt.expected)
			}
		})
	}
}

func TestIsValidAgentID(t *testing.T) {
	tests := []struct {
		name     string
		agentID  string
		expected bool
	}{
		{"valid agent ID", "claude-code:agent-123", true},
		{"valid with underscore", "ollama:agent_123", true},
		{"invalid with special char", "resource:agent@123", false},
		{"empty string", "", false},
		{"missing colon", "agent-123", false},
		{"invalid format", "resource", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAgentID(tt.agentID)
			if result != tt.expected {
				t.Errorf("isValidAgentID(%s) = %v, want %v", tt.agentID, result, tt.expected)
			}
		})
	}
}

func TestIsValidLineCount(t *testing.T) {
	tests := []struct {
		name     string
		lines    string
		expected bool
	}{
		{"valid count", "100", true},
		{"max count", "10000", true},
		{"too high", "10001", false},
		{"negative", "-1", false},
		{"not a number", "abc", false},
		{"empty string", "", false}, // empty string is invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidLineCount(tt.lines)
			if result != tt.expected {
				t.Errorf("isValidLineCount(%s) = %v, want %v", tt.lines, result, tt.expected)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := http.NewRequest("OPTIONS", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if header := rr.Header().Get("Access-Control-Allow-Origin"); header != "*" {
		t.Errorf("expected CORS header to be *, got %s", header)
	}
}
