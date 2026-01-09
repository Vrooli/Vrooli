package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test helpers

// setupTestCampaignManager creates a mock server with the provided handler and returns
// a CampaignManager configured to use it, along with a cleanup function
func setupTestCampaignManager(handler http.HandlerFunc) (*CampaignManager, func()) {
	server := httptest.NewServer(handler)
	cm := &CampaignManager{
		visitedTrackerURL: server.URL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}
	return cm, server.Close
}

// [REQ:TM-SS-003] Test campaign creation
func TestCampaignManager_CreateCampaign(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/campaigns" && r.Method == http.MethodPost {
			var req CreateCampaignRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			// Validate request fields
			if req.FromAgent != "tidiness-manager" {
				t.Errorf("Expected from_agent='tidiness-manager', got '%s'", req.FromAgent)
			}

			if len(req.Patterns) == 0 {
				t.Error("Expected patterns to be set")
			}

			// Return mock campaign
			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:        "test-campaign-id",
					Name:      req.Name,
					FromAgent: req.FromAgent,
					Patterns:  req.Patterns,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Status:    "active",
					Metadata:  req.Metadata,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	})
	defer cleanup()

	campaign, err := cm.CreateCampaign("test-scenario")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	if campaign.ID != "test-campaign-id" {
		t.Errorf("Expected campaign ID 'test-campaign-id', got '%s'", campaign.ID)
	}

	if campaign.Metadata["scenario"] != "test-scenario" {
		t.Error("Expected scenario in metadata")
	}
}

// [REQ:TM-SS-004] Test finding existing campaign
func TestCampaignManager_FindCampaignByScenario(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/campaigns" && r.Method == http.MethodGet {
			resp := ListCampaignsResponse{
				Campaigns: []Campaign{
					{
						ID:        "campaign-1",
						Name:      "tidiness-test-scenario-123456",
						FromAgent: "tidiness-manager",
						Status:    "active",
					},
					{
						ID:        "campaign-2",
						Name:      "other-campaign",
						FromAgent: "other-agent",
						Status:    "active",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	})
	defer cleanup()

	campaign, err := cm.FindCampaignByScenario("test-scenario")
	if err != nil {
		t.Fatalf("FindCampaignByScenario failed: %v", err)
	}

	if campaign == nil {
		t.Fatal("Expected to find campaign, got nil")
	}

	if campaign.ID != "campaign-1" {
		t.Errorf("Expected campaign-1, got %s", campaign.ID)
	}
}

// [REQ:TM-SS-003] [REQ:TM-SS-004] Test get-or-create flow
func TestCampaignManager_GetOrCreateCampaign(t *testing.T) {
	attemptedCreate := false

	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/campaigns" && r.Method == http.MethodGet {
			// First call: return empty list (no existing campaign)
			resp := ListCampaignsResponse{
				Campaigns: []Campaign{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/api/v1/campaigns" && r.Method == http.MethodPost {
			attemptedCreate = true

			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)

			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:        "new-campaign-id",
					Name:      req.Name,
					FromAgent: "tidiness-manager",
					Status:    "active",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	})
	defer cleanup()

	campaign, err := cm.GetOrCreateCampaign("new-scenario")
	if err != nil {
		t.Fatalf("GetOrCreateCampaign failed: %v", err)
	}

	if !attemptedCreate {
		t.Error("Expected campaign creation to be attempted")
	}

	if campaign.ID != "new-campaign-id" {
		t.Errorf("Expected new-campaign-id, got %s", campaign.ID)
	}
}

// [REQ:TM-SS-003] Test graceful degradation when visited-tracker unavailable
func TestCampaignManager_GracefulDegradation(t *testing.T) {
	cm := &CampaignManager{
		visitedTrackerURL: "http://localhost:99999", // Invalid port
		httpClient:        &http.Client{Timeout: 1 * time.Second},
	}

	available := cm.IsVisitedTrackerAvailable()
	if available {
		t.Error("Expected visited-tracker to be unavailable")
	}

	// Should return error but not panic
	_, err := cm.GetOrCreateCampaign("test-scenario")
	if err == nil {
		t.Error("Expected error when visited-tracker unavailable")
	}
}

// [REQ:TM-SS-003] Test campaign creation with various metadata
func TestCampaignManager_CreateCampaignMetadata(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/campaigns" && r.Method == http.MethodPost {
			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)

			// Verify metadata fields
			if req.Metadata == nil {
				t.Error("Expected metadata to be set")
			}

			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:        "campaign-123",
					Name:      req.Name,
					Metadata:  req.Metadata,
					CreatedAt: time.Now(),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	campaign, err := cm.CreateCampaign("test-scenario-with-metadata")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	if campaign.Metadata["scenario"] != "test-scenario-with-metadata" {
		t.Error("Expected scenario in metadata")
	}
}

// [REQ:TM-SS-004] Test finding campaign when multiple campaigns exist
func TestCampaignManager_FindCampaignMultiple(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		resp := ListCampaignsResponse{
			Campaigns: []Campaign{
				{ID: "c1", Name: "tidiness-scenario-a-123", FromAgent: "tidiness-manager"},
				{ID: "c2", Name: "tidiness-scenario-b-456", FromAgent: "tidiness-manager"},
				{ID: "c3", Name: "tidiness-scenario-c-789", FromAgent: "tidiness-manager"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer cleanup()

	tests := []struct {
		scenario   string
		expectedID string
	}{
		{"scenario-a", "c1"},
		{"scenario-b", "c2"},
		{"scenario-c", "c3"},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			campaign, err := cm.FindCampaignByScenario(tt.scenario)
			if err != nil {
				t.Fatalf("FindCampaignByScenario failed: %v", err)
			}

			if campaign == nil {
				t.Fatal("Expected to find campaign")
			}

			if campaign.ID != tt.expectedID {
				t.Errorf("Expected %s, got %s", tt.expectedID, campaign.ID)
			}
		})
	}
}

// [REQ:TM-SS-004] Test not finding campaign
func TestCampaignManager_CampaignNotFound(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		resp := ListCampaignsResponse{
			Campaigns: []Campaign{
				{ID: "c1", Name: "tidiness-other-scenario-123", FromAgent: "tidiness-manager"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer cleanup()

	campaign, err := cm.FindCampaignByScenario("nonexistent-scenario")
	if err != nil {
		t.Fatalf("FindCampaignByScenario failed: %v", err)
	}

	if campaign != nil {
		t.Errorf("Expected nil campaign, got %+v", campaign)
	}
}

// [REQ:TM-SS-003] Test campaign creation error handling
func TestCampaignManager_CreateCampaignError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
	}{
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal server error"}`,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"error": "invalid request"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			})
			defer cleanup()

			_, err := cm.CreateCampaign("test-scenario")
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// [REQ:TM-SS-004] Test list campaigns error handling
func TestCampaignManager_ListCampaignsError(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database error"}`))
	})
	defer cleanup()

	_, err := cm.FindCampaignByScenario("test-scenario")
	if err == nil {
		t.Error("Expected error when list campaigns fails")
	}
}

// [REQ:TM-SS-003] Test campaign name generation
func TestCampaignManager_CampaignNameFormat(t *testing.T) {
	var capturedName string

	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)
			capturedName = req.Name

			resp := CreateCampaignResponse{
				Campaign: Campaign{ID: "test-id", Name: req.Name},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	_, err := cm.CreateCampaign("my-scenario")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Verify name format
	expectedPrefix := "tidiness-my-scenario-"
	if len(capturedName) < len(expectedPrefix) {
		t.Errorf("Campaign name too short: %s", capturedName)
	}

	if capturedName[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Campaign name should start with %s, got %s", expectedPrefix, capturedName)
	}
}

// [REQ:TM-SS-003] Test timeout handling
func TestCampaignManager_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cm := &CampaignManager{
		visitedTrackerURL: server.URL,
		httpClient:        &http.Client{Timeout: 100 * time.Millisecond},
	}

	_, err := cm.CreateCampaign("test-scenario")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// [REQ:TM-SS-004] Test campaign pattern matching
func TestCampaignManager_PatternConfiguration(t *testing.T) {
	var capturedPatterns []string

	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)
			capturedPatterns = req.Patterns

			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:       "test-id",
					Patterns: req.Patterns,
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	campaign, err := cm.CreateCampaign("test-scenario")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Verify patterns are set correctly
	if len(capturedPatterns) == 0 {
		t.Error("Expected patterns to be set")
	}

	// Verify campaign has patterns
	if len(campaign.Patterns) == 0 {
		t.Error("Campaign should have patterns")
	}
}

// [REQ:TM-SS-003] Test concurrent campaign operations
func TestCampaignManager_ConcurrentOperations(t *testing.T) {
	requestCount := 0
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Method == http.MethodGet {
			resp := ListCampaignsResponse{Campaigns: []Campaign{}}
			json.NewEncoder(w).Encode(resp)
		} else if r.Method == http.MethodPost {
			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)
			resp := CreateCampaignResponse{
				Campaign: Campaign{ID: "campaign-" + req.Name},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	const numConcurrent = 5
	results := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			_, err := cm.GetOrCreateCampaign("scenario-" + string(rune('0'+id)))
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numConcurrent; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent operation %d failed: %v", i, err)
		}
	}
}

// [REQ:TM-SS-004] Test IsVisitedTrackerAvailable with healthy server
func TestCampaignManager_IsAvailableHealthy(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy"}`))
		}
	})
	defer cleanup()

	if !cm.IsVisitedTrackerAvailable() {
		t.Error("Expected visited-tracker to be available")
	}
}

// [REQ:TM-SS-003] Test empty scenario name
func TestCampaignManager_EmptyScenario(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer cleanup()

	// Should handle empty scenario gracefully
	campaign, err := cm.CreateCampaign("")
	if err == nil {
		t.Log("Created campaign with empty scenario name")
	}

	if campaign != nil && campaign.ID == "" {
		t.Error("Campaign ID should not be empty")
	}
}

// [REQ:TM-SS-003] Test campaign creation with special characters in scenario name
func TestCampaignManager_SpecialCharactersInScenario(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
	}{
		{"dashes", "my-scenario-name"},
		{"underscores", "my_scenario_name"},
		{"mixed", "my-scenario_name-123"},
		{"numbers", "scenario-123-test"},
		{"unicode", "测试场景"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedScenario string
			cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					var req CreateCampaignRequest
					json.NewDecoder(r.Body).Decode(&req)
					capturedScenario = req.Metadata["scenario"].(string)

					resp := CreateCampaignResponse{
						Campaign: Campaign{
							ID:       "test-id",
							Metadata: req.Metadata,
						},
					}
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(resp)
				}
			})
			defer cleanup()

			campaign, err := cm.CreateCampaign(tt.scenario)
			if err != nil {
				t.Fatalf("CreateCampaign failed: %v", err)
			}

			if capturedScenario != tt.scenario {
				t.Errorf("Expected scenario %q, got %q", tt.scenario, capturedScenario)
			}

			if campaign.Metadata["scenario"] != tt.scenario {
				t.Errorf("Campaign metadata scenario = %v, want %s", campaign.Metadata["scenario"], tt.scenario)
			}
		})
	}
}

// [REQ:TM-SS-003] Test malformed JSON response handling
func TestCampaignManager_MalformedJSONResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		expectError bool
	}{
		{"incomplete json", `{"id": "test-id", "name": "`, true},
		{"invalid json", `{this is not json}`, true},
		{"empty response", ``, true},
		{"null response", `null`, false}, // null is valid JSON, creates empty campaign
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(tt.response))
			})
			defer cleanup()

			_, err := cm.CreateCampaign("test-scenario")
			if tt.expectError && err == nil {
				t.Error("Expected error for malformed JSON")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// [REQ:TM-SS-004] Test list campaigns with malformed JSON
func TestCampaignManager_ListCampaignsMalformedJSON(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"campaigns": [{"id": "broken"`))
	})
	defer cleanup()

	_, err := cm.FindCampaignByScenario("test-scenario")
	if err == nil {
		t.Error("Expected error for malformed JSON response")
	}
}

// [REQ:TM-SS-003] Test campaign metadata field completeness
func TestCampaignManager_MetadataFields(t *testing.T) {
	var capturedMetadata map[string]interface{}

	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req CreateCampaignRequest
			json.NewDecoder(r.Body).Decode(&req)
			capturedMetadata = req.Metadata

			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:       "test-id",
					Metadata: req.Metadata,
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	_, err := cm.CreateCampaign("test-scenario")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Verify expected metadata fields are present
	expectedFields := []string{"scenario"}
	for _, field := range expectedFields {
		if _, ok := capturedMetadata[field]; !ok {
			t.Errorf("Expected metadata field %q to be present", field)
		}
	}

	// Verify scenario field has correct value
	if capturedMetadata["scenario"] != "test-scenario" {
		t.Errorf("Metadata scenario = %v, want test-scenario", capturedMetadata["scenario"])
	}
}

// [REQ:TM-SS-004] Test finding campaign returns first match regardless of from_agent
func TestCampaignManager_FindCampaignAnyAgent(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		resp := ListCampaignsResponse{
			Campaigns: []Campaign{
				{ID: "c1", Name: "tidiness-test-scenario-123", FromAgent: "other-agent"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer cleanup()

	campaign, err := cm.FindCampaignByScenario("test-scenario")
	if err != nil {
		t.Fatalf("FindCampaignByScenario failed: %v", err)
	}

	// Implementation currently returns campaign regardless of from_agent if name matches
	if campaign == nil {
		t.Error("Expected campaign to be found even with different agent")
	}

	if campaign != nil && campaign.ID != "c1" {
		t.Errorf("Expected campaign c1, got %s", campaign.ID)
	}
}

// [REQ:TM-SS-003] Test campaign response field validation
func TestCampaignManager_ResponseFieldValidation(t *testing.T) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			resp := CreateCampaignResponse{
				Campaign: Campaign{
					ID:        "campaign-123",
					Name:      "tidiness-test-123456",
					FromAgent: "tidiness-manager",
					Patterns:  []string{"**/*"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Status:    "active",
					Metadata:  map[string]interface{}{"scenario": "test"},
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	campaign, err := cm.CreateCampaign("test")
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Verify all fields are populated
	if campaign.ID == "" {
		t.Error("Campaign ID should not be empty")
	}
	if campaign.Name == "" {
		t.Error("Campaign Name should not be empty")
	}
	if campaign.FromAgent != "tidiness-manager" {
		t.Errorf("FromAgent = %s, want tidiness-manager", campaign.FromAgent)
	}
	if len(campaign.Patterns) == 0 {
		t.Error("Patterns should not be empty")
	}
	if campaign.Status != "active" {
		t.Errorf("Status = %s, want active", campaign.Status)
	}
	if campaign.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if campaign.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if campaign.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

// [REQ:TM-SS-003] [REQ:TM-SS-004] Benchmark CreateCampaign
func BenchmarkCreateCampaign(b *testing.B) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		resp := CreateCampaignResponse{
			Campaign: Campaign{ID: "test-id"},
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.CreateCampaign("test-scenario")
	}
}

// [REQ:TM-SS-004] Benchmark FindCampaignByScenario
func BenchmarkFindCampaignByScenario(b *testing.B) {
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		resp := ListCampaignsResponse{
			Campaigns: []Campaign{
				{ID: "c1", Name: "tidiness-test-scenario-123", FromAgent: "tidiness-manager"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.FindCampaignByScenario("test-scenario")
	}
}

// [REQ:TM-SS-003] [REQ:TM-SS-004] Benchmark GetOrCreateCampaign
func BenchmarkGetOrCreateCampaign(b *testing.B) {
	callCount := 0
	cm, cleanup := setupTestCampaignManager(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return empty on first call, existing on subsequent
			if callCount == 0 {
				json.NewEncoder(w).Encode(ListCampaignsResponse{Campaigns: []Campaign{}})
			} else {
				json.NewEncoder(w).Encode(ListCampaignsResponse{
					Campaigns: []Campaign{
						{ID: "c1", Name: "tidiness-test-scenario-123"},
					},
				})
			}
			callCount++
		} else if r.Method == http.MethodPost {
			resp := CreateCampaignResponse{Campaign: Campaign{ID: "new-id"}}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.GetOrCreateCampaign("test-scenario")
	}
}
