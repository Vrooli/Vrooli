package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CreateCampaignRequest represents a campaign creation request
type CreateAutoCampaignRequest struct {
	Scenario           string `json:"scenario"`
	MaxSessions        int    `json:"max_sessions"`
	MaxFilesPerSession int    `json:"max_files_per_session"`
}

// CampaignActionRequest represents pause/resume/terminate requests
type CampaignActionRequest struct {
	Action string `json:"action"` // pause, resume, terminate
}

// handleCreateCampaign creates a new auto-tidiness campaign [REQ:TM-AC-001]
func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var req CreateAutoCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate inputs
	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if req.MaxSessions <= 0 {
		req.MaxSessions = 10 // Default
	}

	if req.MaxFilesPerSession <= 0 {
		req.MaxFilesPerSession = 5 // Default
	}

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create orchestrator: %v", err))
		return
	}

	// Create campaign
	campaign, err := orchestrator.CreateAutoCampaign(req.Scenario, req.MaxSessions, req.MaxFilesPerSession)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create campaign: %v", err))
		return
	}

	// Auto-start the campaign
	if err := orchestrator.StartCampaign(campaign.ID); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start campaign: %v", err))
		return
	}

	// Refresh campaign data to get updated status
	campaign, err = orchestrator.GetCampaign(campaign.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get campaign: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"campaign": campaign,
	})
}

// handleListCampaigns lists all campaigns with optional status filter [REQ:TM-AC-008]
func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create orchestrator: %v", err))
		return
	}

	campaigns, err := orchestrator.ListCampaigns(statusFilter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list campaigns: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"campaigns": campaigns,
		"count":     len(campaigns),
	})
}

// handleGetCampaign retrieves a single campaign by ID [REQ:TM-AC-008]
func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := strconv.Atoi(campaignIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid campaign ID")
		return
	}

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create orchestrator: %v", err))
		return
	}

	campaign, err := orchestrator.GetCampaign(campaignID)
	if err != nil {
		respondError(w, http.StatusNotFound, "campaign not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"campaign": campaign,
	})
}

// handleCampaignAction handles pause/resume/terminate actions [REQ:TM-AC-005, TM-AC-006]
func (s *Server) handleCampaignAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := strconv.Atoi(campaignIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid campaign ID")
		return
	}

	var req CampaignActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create orchestrator: %v", err))
		return
	}

	// Execute action
	switch req.Action {
	case "pause":
		err = orchestrator.PauseCampaign(campaignID)
	case "resume":
		err = orchestrator.ResumeCampaign(campaignID)
	case "terminate":
		err = orchestrator.TerminateCampaign(campaignID)
	default:
		respondError(w, http.StatusBadRequest, "invalid action (must be: pause, resume, terminate)")
		return
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to %s campaign: %v", req.Action, err))
		return
	}

	// Return updated campaign
	campaign, err := orchestrator.GetCampaign(campaignID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get campaign: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"campaign": campaign,
	})
}
