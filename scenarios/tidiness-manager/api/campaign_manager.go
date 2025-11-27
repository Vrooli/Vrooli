package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// CampaignManager handles visited-tracker campaign integration (TM-SS-003, TM-SS-004)
type CampaignManager struct {
	visitedTrackerURL string
	httpClient        *http.Client
}

// NewCampaignManager creates a CampaignManager instance
func NewCampaignManager() *CampaignManager {
	vtURL := os.Getenv("VISITED_TRACKER_URL")
	if vtURL == "" {
		// Try to get from vrooli CLI
		vtURL = "http://localhost:17693" // Default port from scenario status
	}

	return &CampaignManager{
		visitedTrackerURL: vtURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Campaign represents a visited-tracker campaign
type Campaign struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	FromAgent    string                 `json:"from_agent"`
	Patterns     []string               `json:"patterns"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	TrackedFiles []TrackedFile          `json:"tracked_files,omitempty"`
}

// TrackedFile represents a file tracked by visited-tracker
type TrackedFile struct {
	ID             string                 `json:"id"`
	FilePath       string                 `json:"file_path"`
	AbsolutePath   string                 `json:"absolute_path"`
	VisitCount     int                    `json:"visit_count"`
	FirstSeen      time.Time              `json:"first_seen"`
	LastVisited    time.Time              `json:"last_visited"`
	LastModified   time.Time              `json:"last_modified"`
	SizeBytes      int64                  `json:"size_bytes"`
	StalenessScore float64                `json:"staleness_score"`
	Deleted        bool                   `json:"deleted"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CreateCampaignRequest represents a campaign creation request
type CreateCampaignRequest struct {
	Name      string                 `json:"name"`
	FromAgent string                 `json:"from_agent"`
	Patterns  []string               `json:"patterns"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CreateCampaignResponse represents a campaign creation response
type CreateCampaignResponse struct {
	Campaign Campaign `json:"campaign"`
}

// ListCampaignsResponse represents the response from listing campaigns
type ListCampaignsResponse struct {
	Campaigns []Campaign `json:"campaigns"`
}

// GetOrCreateCampaign attempts to find an existing campaign for the scenario,
// or creates a new one if none exists (TM-SS-003, TM-SS-004)
func (cm *CampaignManager) GetOrCreateCampaign(scenario string) (*Campaign, error) {
	// First, try to find existing campaign
	campaign, err := cm.FindCampaignByScenario(scenario)
	if err == nil && campaign != nil {
		return campaign, nil
	}

	// No campaign found or error - try to create one
	return cm.CreateCampaign(scenario)
}

// FindCampaignByScenario searches for an existing campaign for the scenario (TM-SS-004)
func (cm *CampaignManager) FindCampaignByScenario(scenario string) (*Campaign, error) {
	// List all campaigns
	resp, err := cm.httpClient.Get(cm.visitedTrackerURL + "/api/v1/campaigns")
	if err != nil {
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("visited-tracker returned %d: %s", resp.StatusCode, string(body))
	}

	var listResp ListCampaignsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode campaigns: %w", err)
	}

	// Find campaign matching this scenario
	campaignPrefix := fmt.Sprintf("tidiness-%s", scenario)
	for i := range listResp.Campaigns {
		name := listResp.Campaigns[i].Name
		if len(name) >= len(campaignPrefix) && name[:len(campaignPrefix)] == campaignPrefix {
			return &listResp.Campaigns[i], nil
		}
	}

	return nil, nil // No campaign found (not an error)
}

// CreateCampaign creates a new visited-tracker campaign for the scenario (TM-SS-003)
func (cm *CampaignManager) CreateCampaign(scenario string) (*Campaign, error) {
	req := CreateCampaignRequest{
		Name:      fmt.Sprintf("tidiness-%s-%d", scenario, time.Now().Unix()),
		FromAgent: "tidiness-manager",
		Patterns:  []string{"**/*"}, // Track all files
		Metadata: map[string]interface{}{
			"scenario":      scenario,
			"campaign_type": "tidiness-scan",
			"created_by":    "tidiness-manager",
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal campaign request: %w", err)
	}

	resp, err := cm.httpClient.Post(
		cm.visitedTrackerURL+"/api/v1/campaigns",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("visited-tracker returned %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp CreateCampaignResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode campaign response: %w", err)
	}

	return &createResp.Campaign, nil
}

// GetCampaignFiles retrieves all tracked files for a campaign
func (cm *CampaignManager) GetCampaignFiles(campaignID string) ([]TrackedFile, error) {
	url := fmt.Sprintf("%s/api/v1/campaigns/%s", cm.visitedTrackerURL, campaignID)
	resp, err := cm.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("visited-tracker returned %d: %s", resp.StatusCode, string(body))
	}

	var campaign Campaign
	if err := json.NewDecoder(resp.Body).Decode(&campaign); err != nil {
		return nil, fmt.Errorf("failed to decode campaign: %w", err)
	}

	return campaign.TrackedFiles, nil
}

// IsVisitedTrackerAvailable checks if visited-tracker is accessible
func (cm *CampaignManager) IsVisitedTrackerAvailable() bool {
	resp, err := cm.httpClient.Get(cm.visitedTrackerURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
