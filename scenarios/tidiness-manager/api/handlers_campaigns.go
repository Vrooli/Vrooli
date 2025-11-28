package main

import (
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
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	// Validate inputs
	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	// Security: Limit max sessions to prevent resource exhaustion
	const maxAllowedSessions = 100
	if req.MaxSessions <= 0 {
		req.MaxSessions = 10 // Default
	} else if req.MaxSessions > maxAllowedSessions {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("max_sessions cannot exceed %d", maxAllowedSessions))
		return
	}

	// Security: Limit max files per session
	const maxAllowedFilesPerSession = 50
	if req.MaxFilesPerSession <= 0 {
		req.MaxFilesPerSession = 5 // Default
	} else if req.MaxFilesPerSession > maxAllowedFilesPerSession {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("max_files_per_session cannot exceed %d", maxAllowedFilesPerSession))
		return
	}

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		s.log("failed to create orchestrator", map[string]interface{}{"error": err.Error()})
		// Security: Don't leak internal error details
		respondError(w, http.StatusInternalServerError, "failed to create campaign orchestrator")
		return
	}

	// Create campaign
	campaign, err := orchestrator.CreateAutoCampaign(req.Scenario, req.MaxSessions, req.MaxFilesPerSession)
	if err != nil {
		s.log("failed to create campaign", map[string]interface{}{"error": err.Error(), "scenario": req.Scenario})
		respondError(w, http.StatusInternalServerError, "failed to create campaign")
		return
	}

	// Auto-start the campaign
	if err := orchestrator.StartCampaign(campaign.ID); err != nil {
		s.log("failed to start campaign", map[string]interface{}{"error": err.Error(), "campaign_id": campaign.ID})
		respondError(w, http.StatusInternalServerError, "failed to start campaign")
		return
	}

	// Refresh campaign data to get updated status
	campaign, err = orchestrator.GetCampaign(campaign.ID)
	if err != nil {
		s.log("failed to get campaign", map[string]interface{}{"error": err.Error(), "campaign_id": campaign.ID})
		respondError(w, http.StatusInternalServerError, "failed to retrieve campaign status")
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
		s.log("failed to create orchestrator", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to create campaign orchestrator")
		return
	}

	campaigns, err := orchestrator.ListCampaigns(statusFilter)
	if err != nil {
		s.log("failed to list campaigns", map[string]interface{}{"error": err.Error(), "status_filter": statusFilter})
		respondError(w, http.StatusInternalServerError, "failed to list campaigns")
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
		s.log("failed to create orchestrator", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to create campaign orchestrator")
		return
	}

	campaign, err := orchestrator.GetCampaign(campaignID)
	if err != nil {
		s.log("failed to get campaign", map[string]interface{}{"error": err.Error(), "campaign_id": campaignID})
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
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	// Create orchestrator
	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(s.db, campaignMgr)
	if err != nil {
		s.log("failed to create orchestrator", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to create campaign orchestrator")
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
		s.log("failed to execute campaign action", map[string]interface{}{"error": err.Error(), "action": req.Action, "campaign_id": campaignID})
		respondError(w, http.StatusInternalServerError, "failed to execute campaign action")
		return
	}

	// Return updated campaign
	campaign, err := orchestrator.GetCampaign(campaignID)
	if err != nil {
		s.log("failed to get campaign after action", map[string]interface{}{"error": err.Error(), "campaign_id": campaignID})
		respondError(w, http.StatusInternalServerError, "failed to retrieve campaign status")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"campaign": campaign,
	})
}
