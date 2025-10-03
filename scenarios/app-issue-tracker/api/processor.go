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
func (s *Server) updateProcessorState(active *bool, slots *int, interval *int, maxIssues *int, maxIssuesDisabled *bool) {
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
	if maxIssues != nil && *maxIssues >= 0 {
		s.processorState.MaxIssues = *maxIssues
	}
	if maxIssuesDisabled != nil {
		s.processorState.MaxIssuesDisabled = *maxIssuesDisabled
	}
}

// initializeProcessorState initializes the processor with default settings
func (s *Server) initializeProcessorState() {
	s.processorMutex.Lock()
	defer s.processorMutex.Unlock()

	s.processorState = ProcessorState{
		Active:            false,
		ConcurrentSlots:   2,
		RefreshInterval:   45,
		CurrentlyRunning:  0,
		MaxIssues:         0, // 0 = unlimited
		MaxIssuesDisabled: true,
	}
}

// getProcessorHandler returns the current processor state
func (s *Server) getProcessorHandler(w http.ResponseWriter, r *http.Request) {
	state := s.currentProcessorState()

	// Get issue count for max_issues tracking
	s.issuesProcessedMutex.RLock()
	issuesProcessed := s.issuesProcessedCount
	s.issuesProcessedMutex.RUnlock()

	var issuesRemaining interface{}
	if state.MaxIssuesDisabled {
		issuesRemaining = "disabled"
	} else if state.MaxIssues > 0 {
		remaining := state.MaxIssues - issuesProcessed
		if remaining < 0 {
			remaining = 0
		}
		issuesRemaining = remaining
	} else {
		issuesRemaining = "unlimited"
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"processor":        state,
			"issues_processed": issuesProcessed,
			"issues_remaining": issuesRemaining,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateProcessorHandler updates the processor configuration
func (s *Server) updateProcessorHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active            *bool `json:"active"`
		ConcurrentSlots   *int  `json:"concurrent_slots"`
		RefreshInterval   *int  `json:"refresh_interval"`
		MaxIssues         *int  `json:"max_issues"`
		MaxIssuesDisabled *bool `json:"max_issues_disabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.updateProcessorState(req.Active, req.ConcurrentSlots, req.RefreshInterval, req.MaxIssues, req.MaxIssuesDisabled)
	state := s.currentProcessorState()

	log.Printf("Processor state updated: active=%v, slots=%d, interval=%d, max_issues=%d, max_issues_disabled=%v",
		state.Active, state.ConcurrentSlots, state.RefreshInterval, state.MaxIssues, state.MaxIssuesDisabled)

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

// resetIssueCounterHandler resets the issues processed counter to 0
func (s *Server) resetIssueCounterHandler(w http.ResponseWriter, r *http.Request) {
	s.issuesProcessedMutex.Lock()
	s.issuesProcessedCount = 0
	s.issuesProcessedMutex.Unlock()

	log.Printf("Issue counter reset to 0")

	response := ApiResponse{
		Success: true,
		Message: "Issue counter reset to 0",
		Data: map[string]interface{}{
			"issues_processed": 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// incrementProcessedCount increments the processed counter and returns the updated value
func (s *Server) incrementProcessedCount() int {
	s.issuesProcessedMutex.Lock()
	defer s.issuesProcessedMutex.Unlock()

	s.issuesProcessedCount++
	return s.issuesProcessedCount
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
		if issue.Metadata.Extra == nil {
			continue
		}

		resetTimeStr := issue.Metadata.Extra["rate_limit_until"]
		if resetTimeStr == "" {
			continue
		}

		resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
		if err != nil {
			log.Printf("Rate limit metadata for issue %s could not be parsed (%s) — clearing stale entry", issue.ID, resetTimeStr)
			issueDir, _, findErr := s.findIssueDirectory(issue.ID)
			if findErr == nil {
				if loaded, loadErr := s.loadIssueFromDir(issueDir); loadErr == nil {
					if loaded.Metadata.Extra != nil {
						delete(loaded.Metadata.Extra, "rate_limit_until")
						delete(loaded.Metadata.Extra, "rate_limit_agent")
						s.writeIssueMetadata(issueDir, loaded)
					}
				}
			}
			continue
		}

		if time.Now().Before(resetTime) {
			rateLimitedCount++
			if earliestReset.IsZero() || resetTime.Before(earliestReset) {
				earliestReset = resetTime
				rateLimitAgent = issue.Metadata.Extra["rate_limit_agent"]
				rateLimitType = "api_rate_limit"
			}
			continue
		}

		// Reset expired — opportunistically move the issue back to open
		log.Printf("Processor: rate limit window elapsed for issue %s (via status endpoint), moving back to open", issue.ID)
		if err := s.moveIssue(issue.ID, "open"); err != nil {
			log.Printf("Processor: failed to move issue %s back to open after rate limit expiry: %v", issue.ID, err)
		} else {
			issueDir, _, findErr := s.findIssueDirectory(issue.ID)
			if findErr == nil {
				if loaded, loadErr := s.loadIssueFromDir(issueDir); loadErr == nil && loaded.Metadata.Extra != nil {
					delete(loaded.Metadata.Extra, "rate_limit_until")
					delete(loaded.Metadata.Extra, "rate_limit_agent")
					s.writeIssueMetadata(issueDir, loaded)
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

// registerRunningProcess adds a process to the running processes map and ensures status reflects execution
func (s *Server) registerRunningProcess(issueID, agentID, startTime string) {
	// Defensive: ensure the issue is marked active while work is running.
	if _, currentFolder, err := s.findIssueDirectory(issueID); err == nil && currentFolder != "active" {
		if err := s.moveIssue(issueID, "active"); err != nil {
			log.Printf("registerRunningProcess: failed to ensure issue %s is active: %v", issueID, err)
		} else {
			log.Printf("registerRunningProcess: normalized status for issue %s from %s to active", issueID, currentFolder)
		}
	}

	s.runningProcessesMutex.Lock()
	s.runningProcesses[issueID] = &RunningProcess{
		IssueID:   issueID,
		AgentID:   agentID,
		StartTime: startTime,
	}
	s.runningProcessesMutex.Unlock()

	// Publish agent started event
	startTimeParsed, _ := time.Parse(time.RFC3339, startTime)
	s.hub.Publish(NewEvent(EventAgentStarted, AgentStartedData{
		IssueID:   issueID,
		AgentID:   agentID,
		StartTime: startTimeParsed,
	}))
}

// unregisterRunningProcess removes a process from the running processes map
func (s *Server) unregisterRunningProcess(issueID string) {
	s.runningProcessesMutex.Lock()
	defer s.runningProcessesMutex.Unlock()
	delete(s.runningProcesses, issueID)
}

// isIssueRunning reports whether an issue currently has an active agent
func (s *Server) isIssueRunning(issueID string) bool {
	s.runningProcessesMutex.RLock()
	defer s.runningProcessesMutex.RUnlock()
	_, exists := s.runningProcesses[issueID]
	return exists
}

// getRunningProcessesHandler returns the list of currently running processes
func (s *Server) getRunningProcessesHandler(w http.ResponseWriter, r *http.Request) {
	s.runningProcessesMutex.RLock()
	processes := make([]*RunningProcess, 0, len(s.runningProcesses))
	for _, proc := range s.runningProcesses {
		processes = append(processes, proc)
	}
	s.runningProcessesMutex.RUnlock()

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"processes": processes,
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

		// Max issue enforcement temporarily disabled system-wide

		// Check currently running processes to calculate available slots
		s.runningProcessesMutex.RLock()
		currentlyRunning := len(s.runningProcesses)
		s.runningProcessesMutex.RUnlock()

		// Calculate available slots
		availableSlots := state.ConcurrentSlots - currentlyRunning
		if availableSlots <= 0 {
			log.Printf("Processor: All slots occupied (%d/%d), waiting for completion",
				currentlyRunning, state.ConcurrentSlots)
			time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
			continue
		}

		// Calculate how many issues to process (only use available slots)
		processCount := availableSlots
		if len(openIssues) < processCount {
			processCount = len(openIssues)
		}

		log.Printf("Processor: Found %d open issues, %d slots available (%d running), processing %d",
			len(openIssues), availableSlots, currentlyRunning, processCount)

		// Update ProcessorState.CurrentlyRunning for API visibility
		s.processorMutex.Lock()
		s.processorState.CurrentlyRunning = currentlyRunning
		s.processorMutex.Unlock()

		// Process issues in parallel up to concurrent slot limit
		for i := 0; i < processCount; i++ {
			issue := openIssues[i]
			agentID := "unified-resolver" // Default agent

			if err := s.triggerInvestigation(issue.ID, agentID, true); err != nil {
				log.Printf("Processor: Failed to trigger investigation for issue %s: %v", issue.ID, err)
			} else {
				log.Printf("Processor: Triggered investigation for issue %s", issue.ID)
			}
		}

		// Sleep before next iteration
		time.Sleep(time.Duration(state.RefreshInterval) * time.Second)
	}
}
