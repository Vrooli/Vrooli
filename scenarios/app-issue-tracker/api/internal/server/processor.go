package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
)

const maxProcessorPayloadBytes int64 = 256 << 10 // 256 KiB

func (s *Server) currentProcessorState() ProcessorState {
	return s.processor.CurrentState()
}

func (s *Server) updateProcessorState(active *bool, slots *int, interval *int, maxIssues *int, maxIssuesDisabled *bool) {
	s.processor.UpdateState(active, slots, interval, maxIssues, maxIssuesDisabled)
}

func (s *Server) resetIssueCounterHandler(w http.ResponseWriter, r *http.Request) {
	s.processor.ResetCounter()

	logging.LogInfo("Processor issue counter reset")

	response := ApiResponse{
		Success: true,
		Message: "Issue counter reset to 0",
		Data:    ProcessorCounterResetData{IssuesProcessed: 0},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write processor response", err)
	}
}

func (s *Server) incrementProcessedCount() int {
	return s.processor.IncrementProcessedCount()
}

func (s *Server) getProcessorHandler(w http.ResponseWriter, r *http.Request) {
	state := s.currentProcessorState()
	issuesProcessed := s.processor.ProcessedCount()

	var remainingPtr *int
	remainingMode := "unlimited"
	if state.MaxIssuesDisabled {
		remainingMode = "disabled"
	} else if state.MaxIssues > 0 {
		remaining := state.MaxIssues - issuesProcessed
		if remaining < 0 {
			remaining = 0
		}
		remainingMode = "count"
		remainingPtr = new(int)
		*remainingPtr = remaining
	}

	response := ApiResponse{
		Success: true,
		Data: ProcessorSummary{
			Processor:           state,
			IssuesProcessed:     issuesProcessed,
			IssuesRemainingMode: remainingMode,
			IssuesRemaining:     remainingPtr,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write processor state response", err)
	}
}

func (s *Server) updateProcessorHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active            *bool `json:"active"`
		ConcurrentSlots   *int  `json:"concurrent_slots"`
		RefreshInterval   *int  `json:"refresh_interval"`
		MaxIssues         *int  `json:"max_issues"`
		MaxIssuesDisabled *bool `json:"max_issues_disabled"`
	}

	if err := handlers.DecodeJSON(r, &req, maxProcessorPayloadBytes); err != nil {
		handlers.WriteDecodeError(w, err)
		return
	}

	// Validate concurrent_slots
	if req.ConcurrentSlots != nil {
		if *req.ConcurrentSlots < MinConcurrentSlots || *req.ConcurrentSlots > MaxConcurrentSlots {
			handlers.WriteError(w, http.StatusBadRequest,
				fmt.Sprintf("concurrent_slots must be between %d and %d", MinConcurrentSlots, MaxConcurrentSlots))
			return
		}
	}

	// Validate refresh_interval
	if req.RefreshInterval != nil {
		if *req.RefreshInterval < MinRefreshInterval || *req.RefreshInterval > MaxRefreshInterval {
			handlers.WriteError(w, http.StatusBadRequest,
				fmt.Sprintf("refresh_interval must be between %d and %d seconds", MinRefreshInterval, MaxRefreshInterval))
			return
		}
	}

	// Validate max_issues (allow 0 for unlimited, but no negative)
	if req.MaxIssues != nil {
		if *req.MaxIssues < 0 {
			handlers.WriteError(w, http.StatusBadRequest, "max_issues must be 0 or greater")
			return
		}
	}

	s.updateProcessorState(req.Active, req.ConcurrentSlots, req.RefreshInterval, req.MaxIssues, req.MaxIssuesDisabled)
	state := s.currentProcessorState()

	logging.LogInfo(
		"Processor state updated",
		"active", state.Active,
		"slots", state.ConcurrentSlots,
		"interval_seconds", state.RefreshInterval,
		"max_issues", state.MaxIssues,
		"max_issues_disabled", state.MaxIssuesDisabled,
	)

	response := ApiResponse{
		Success: true,
		Message: "Processor settings updated",
		Data:    ProcessorUpdatedData{Processor: state},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write processor update response", err)
	}
}

func (s *Server) getRunningProcessesHandler(w http.ResponseWriter, r *http.Request) {
	processes := s.processor.RunningProcesses()

	response := ApiResponse{
		Success: true,
		Data:    RunningProcessesData{Processes: processes},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write running processes response", err)
	}
}

func (s *Server) stopRunningProcessHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	if !s.processor.CancelRunningProcess(issueID, "user_stop") {
		handlers.WriteError(w, http.StatusNotFound, "Agent is not running for the specified issue")
		return
	}

	logging.LogInfo("Stop requested for running agent", "issue_id", issueID)

	response := ApiResponse{
		Success: true,
		Message: "Agent stop requested",
		Data:    StopProcessData{IssueID: issueID},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write stop process response", err)
	}
}

func (s *Server) registerRunningProcess(issueID, agentID, startTime string, targets []Target, cancel context.CancelFunc) {
	if _, currentFolder, findErr := s.findIssueDirectory(issueID); findErr == nil && currentFolder != StatusActive {
		if moveErr := s.moveIssue(issueID, StatusActive); moveErr != nil {
			logging.LogErrorErr("Failed to mark running issue as active", moveErr, "issue_id", issueID)
		} else {
			logging.LogInfo("Normalized issue status for running process", "issue_id", issueID, "previous_status", currentFolder)
		}
	}

	s.processor.RegisterRunningProcess(issueID, agentID, startTime, targets, cancel)

	startTimeParsed, _ := time.Parse(time.RFC3339, startTime)
	s.hub.Publish(NewEvent(EventAgentStarted, AgentStartedData{
		IssueID:   issueID,
		AgentID:   agentID,
		StartTime: startTimeParsed,
	}))
}

func (s *Server) unregisterRunningProcess(issueID string) {
	s.processor.UnregisterRunningProcess(issueID)
}

func (s *Server) isIssueRunning(issueID string) bool {
	return s.processor.IsRunning(issueID)
}

func (s *Server) getRateLimitStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := s.investigations.RateLimitStatus()
	response := ApiResponse{
		Success: true,
		Data:    status,
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write rate limit response", err)
	}
}

// TriggerInvestigation delegates to the shared investigation pipeline.
func (s *Server) TriggerInvestigation(issueID, agentID string, autoResolve bool) error {
	return s.investigations.TriggerInvestigation(issueID, agentID, autoResolve)
}
