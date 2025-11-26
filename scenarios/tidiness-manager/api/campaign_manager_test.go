package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:TM-SS-003] Test campaign creation
func TestCampaignManager_CreateCampaign(t *testing.T) {
	// Mock visited-tracker server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cm := &CampaignManager{
		visitedTrackerURL: server.URL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}

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
	// Mock visited-tracker server with existing campaigns
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cm := &CampaignManager{
		visitedTrackerURL: server.URL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}

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

	// Mock visited-tracker server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cm := &CampaignManager{
		visitedTrackerURL: server.URL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}

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
