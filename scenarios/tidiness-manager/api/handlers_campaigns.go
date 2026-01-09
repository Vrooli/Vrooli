package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// CreateCampaignRequest represents a campaign creation request
type CreateAutoCampaignRequest struct {
	Scenario           string `json:"scenario"`
	MaxSessions        int    `json:"max_sessions"`
	MaxFilesPerSession int    `json:"max_files_per_session"`
	Category           string `json:"category,omitempty"`
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

	// Basic validation before orchestration to avoid silently defaulting invalid values
	if strings.TrimSpace(req.Scenario) == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}
	if req.MaxSessions <= 0 || req.MaxFilesPerSession <= 0 {
		respondError(w, http.StatusBadRequest, "max_sessions and max_files_per_session must be positive integers")
		return
	}

	orchestrator, err := s.getCampaignOrchestrator()
	if err != nil {
		s.log("failed to create orchestrator", map[string]interface{}{"error": err.Error()})
		// Security: Don't leak internal error details
		respondError(w, http.StatusInternalServerError, "failed to create campaign orchestrator")
		return
	}

	// Create campaign
	campaign, err := orchestrator.CreateAutoCampaign(req.Scenario, req.MaxSessions, req.MaxFilesPerSession)
	if err != nil {
		var validationErr *CampaignValidationError
		if errors.As(err, &validationErr) {
			respondError(w, http.StatusBadRequest, validationErr.Error())
			return
		}

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

	orchestrator, err := s.getCampaignOrchestrator()
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

	orchestrator, err := s.getCampaignOrchestrator()
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

	orchestrator, err := s.getCampaignOrchestrator()
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
