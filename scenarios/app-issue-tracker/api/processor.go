package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// currentProcessorState returns the current state of the processor
func (s *Server) currentProcessorState() ProcessorState {
	s.processorMutex.RLock()
	defer s.processorMutex.RUnlock()
	return s.processorState
}

// updateProcessorState updates the processor configuration
func (s *Server) updateProcessorState(active *bool, slots *int, interval *int) {
	s.processorMutex.Lock()
	defer s.processorMutex.Unlock()

	if active != nil {
		s.processorState.Active = *active
	}
	if slots != nil && *slots > 0 {
		s.processorState.ConcurrentSlots = *slots
	}
	if interval != nil && *interval > 0 {
		s.processorState.RefreshInterval = *interval
	}
}

// initializeProcessorState initializes the processor with default settings
func (s *Server) initializeProcessorState() {
	s.processorMutex.Lock()
	defer s.processorMutex.Unlock()

	s.processorState = ProcessorState{
		Active:           false,
		ConcurrentSlots:  2,
		RefreshInterval:  45,
		CurrentlyRunning: 0,
	}
}

// getProcessorHandler returns the current processor state
func (s *Server) getProcessorHandler(w http.ResponseWriter, r *http.Request) {
	state := s.currentProcessorState()

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"processor": state,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateProcessorHandler updates the processor configuration
func (s *Server) updateProcessorHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active          *bool `json:"active"`
		ConcurrentSlots *int  `json:"concurrent_slots"`
		RefreshInterval *int  `json:"refresh_interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.updateProcessorState(req.Active, req.ConcurrentSlots, req.RefreshInterval)
	state := s.currentProcessorState()

	log.Printf("Processor state updated: active=%v, slots=%d, interval=%d",
		state.Active, state.ConcurrentSlots, state.RefreshInterval)

	response := ApiResponse{
		Success: true,
		Message: "Processor settings updated",
		Data: map[string]interface{}{
			"processor": state,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRateLimitStatusHandler returns information about rate limited issues
func (s *Server) getRateLimitStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Check waiting issues for rate limit info
	waitingIssues, err := s.loadIssuesFromFolder("waiting")
	if err != nil {
		log.Printf("Error loading waiting issues: %v", err)
		waitingIssues = []Issue{}
	}

	var earliestReset time.Time
	rateLimitedCount := 0
	var rateLimitType string
	var rateLimitAgent string

	for _, issue := range waitingIssues {
		if issue.Metadata.Extra != nil {
			resetTimeStr := issue.Metadata.Extra["rate_limit_until"]
			if resetTimeStr != "" {
				resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
				if err == nil && time.Now().Before(resetTime) {
					rateLimitedCount++
					if earliestReset.IsZero() || resetTime.Before(earliestReset) {
						earliestReset = resetTime
						rateLimitAgent = issue.Metadata.Extra["rate_limit_agent"]
						rateLimitType = "api_rate_limit"
					}
				}
			}
		}
	}

	rateLimited := rateLimitedCount > 0
	var resetTime string
	var secondsUntilReset int64

	if rateLimited {
		resetTime = earliestReset.Format(time.RFC3339)
		secondsUntilReset = int64(time.Until(earliestReset).Seconds())
		if secondsUntilReset < 0 {
			secondsUntilReset = 0
		}
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"rate_limited":         rateLimited,
			"rate_limited_count":   rateLimitedCount,
			"rate_limit_type":      rateLimitType,
			"rate_limit_agent":     rateLimitAgent,
			"reset_time":           resetTime,
			"seconds_until_reset":  secondsUntilReset,
			"waiting_issues_count": len(waitingIssues),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// runProcessorLoop is the background goroutine that automatically processes open issues
func (s *Server) runProcessorLoop() {
	log.Println("Processor loop started")

	for {
		state := s.currentProcessorState()

		// Sleep if processor is inactive
		if !state.Active {
			time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
			continue
		}

		// Check waiting issues and move back to open if rate limit expired
		waitingIssues, _ := s.loadIssuesFromFolder("waiting")
		for _, issue := range waitingIssues {
			if issue.Metadata.Extra != nil {
				resetTimeStr := issue.Metadata.Extra["rate_limit_until"]
				if resetTimeStr != "" {
					resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
					if err == nil && time.Now().After(resetTime) {
						log.Printf("Processor: Rate limit expired for issue %s, moving back to open", issue.ID)
						s.moveIssue(issue.ID, "open")

						// Clear rate limit metadata
						issueDir, _, _ := s.findIssueDirectory(issue.ID)
						if issueDir != "" {
							loadedIssue, _ := s.loadIssueFromDir(issueDir)
							if loadedIssue != nil && loadedIssue.Metadata.Extra != nil {
								delete(loadedIssue.Metadata.Extra, "rate_limit_until")
								delete(loadedIssue.Metadata.Extra, "rate_limit_agent")
								s.writeIssueMetadata(issueDir, loadedIssue)
							}
						}
					}
				}
			}
		}

		// Load open issues
		openIssues, err := s.loadIssuesFromFolder("open")
		if err != nil {
			log.Printf("Error loading open issues in processor loop: %v", err)
			time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
			continue
		}

		if len(openIssues) == 0 {
			// No open issues to process
			time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
			continue
		}

		// Calculate how many issues to process
		processCount := state.ConcurrentSlots
		if len(openIssues) < processCount {
			processCount = len(openIssues)
		}

		log.Printf("Processor: Found %d open issues, processing %d concurrently", len(openIssues), processCount)

		// Process issues in parallel up to concurrent slot limit
		for i := 0; i < processCount; i++ {
			issue := openIssues[i]
			agentID := "unified-resolver" // Default agent

			if err := s.triggerInvestigation(issue.ID, agentID, true); err != nil {
				log.Printf("Processor: Failed to trigger investigation for issue %s: %v", issue.ID, err)
			}
		}

		// Sleep before next iteration
		time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
	}
}
