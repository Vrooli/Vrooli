package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type AutomationProcessor struct {
	host *Server

	stateMu sync.RWMutex
	state   ProcessorState

	processedMu    sync.RWMutex
	processedCount int

	runningMu sync.RWMutex
	running   map[string]*trackedProcess

	loopMu     sync.Mutex
	loopCancel context.CancelFunc
}

type trackedProcess struct {
	info            *RunningProcess
	cancel          context.CancelFunc
	cancelRequested bool
	cancelReason    string
}

func NewAutomationProcessor(host *Server) *AutomationProcessor {
	p := &AutomationProcessor{
		host:    host,
		running: make(map[string]*trackedProcess),
	}
	p.state = ProcessorState{
		Active:            false,
		ConcurrentSlots:   2,
		RefreshInterval:   45,
		CurrentlyRunning:  0,
		MaxIssues:         0,
		MaxIssuesDisabled: true,
	}
	return p
}

func (p *AutomationProcessor) CurrentState() ProcessorState {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.state
}

func (p *AutomationProcessor) UpdateState(active *bool, slots *int, interval *int, maxIssues *int, maxIssuesDisabled *bool) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	if active != nil {
		p.state.Active = *active
	}
	if slots != nil && *slots > 0 {
		p.state.ConcurrentSlots = *slots
	}
	if interval != nil && *interval > 0 {
		p.state.RefreshInterval = *interval
	}
	if maxIssues != nil && *maxIssues >= 0 {
		p.state.MaxIssues = *maxIssues
	}
	if maxIssuesDisabled != nil {
		p.state.MaxIssuesDisabled = *maxIssuesDisabled
	}
}

func (p *AutomationProcessor) ResetCounter() {
	p.processedMu.Lock()
	defer p.processedMu.Unlock()
	p.processedCount = 0
}

func (p *AutomationProcessor) IncrementProcessedCount() int {
	p.processedMu.Lock()
	defer p.processedMu.Unlock()
	p.processedCount++
	return p.processedCount
}

func (p *AutomationProcessor) ProcessedCount() int {
	p.processedMu.RLock()
	defer p.processedMu.RUnlock()
	return p.processedCount
}

func (p *AutomationProcessor) RegisterRunningProcess(issueID, agentID, startTime string, cancel context.CancelFunc) {
	p.runningMu.Lock()
	defer p.runningMu.Unlock()

	if existing, ok := p.running[issueID]; ok {
		if existing.info == nil {
			existing.info = &RunningProcess{}
		}
		existing.info.IssueID = issueID
		existing.info.AgentID = agentID
		existing.info.StartTime = startTime
		existing.info.Status = AgentStatusRunning
		if cancel != nil {
			existing.cancel = cancel
			existing.cancelRequested = false
			existing.cancelReason = ""
		}
		p.updateRunningCountLocked()
		return
	}

	p.running[issueID] = &trackedProcess{
		info: &RunningProcess{
			IssueID:   issueID,
			AgentID:   agentID,
			StartTime: startTime,
			Status:    AgentStatusRunning,
		},
		cancel: cancel,
	}
	p.updateRunningCountLocked()
}

func (p *AutomationProcessor) UnregisterRunningProcess(issueID string) {
	p.runningMu.Lock()
	defer p.runningMu.Unlock()
	delete(p.running, issueID)
	p.updateRunningCountLocked()
}

func (p *AutomationProcessor) IsRunning(issueID string) bool {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	_, ok := p.running[issueID]
	return ok
}

func (p *AutomationProcessor) RunningProcesses() []*RunningProcess {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	out := make([]*RunningProcess, 0, len(p.running))
	for _, proc := range p.running {
		if proc.info != nil {
			copyInfo := *proc.info
			out = append(out, &copyInfo)
		}
	}
	return out
}

func (p *AutomationProcessor) CancelRunningProcess(issueID, reason string) bool {
	p.runningMu.Lock()
	defer p.runningMu.Unlock()

	proc, ok := p.running[issueID]
	if !ok {
		return false
	}
	if proc.cancelRequested {
		return true
	}
	proc.cancelRequested = true
	proc.cancelReason = reason
	if proc.info != nil {
		proc.info.Status = AgentStatusCancelling
	}
	if proc.cancel != nil {
		proc.cancel()
	}
	return true
}

func (p *AutomationProcessor) cancellationInfo(issueID string) (bool, string) {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	proc, ok := p.running[issueID]
	if !ok {
		return false, ""
	}
	return proc.cancelRequested, proc.cancelReason
}

func (p *AutomationProcessor) updateRunningCountLocked() {
	count := len(p.running)
	p.stateMu.Lock()
	p.state.CurrentlyRunning = count
	p.stateMu.Unlock()
}

func (p *AutomationProcessor) Start(ctx context.Context) {
	p.loopMu.Lock()
	if p.loopCancel != nil {
		p.loopMu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.loopCancel = cancel
	p.loopMu.Unlock()

	go p.run(runCtx)
}

func (p *AutomationProcessor) Stop() {
	p.loopMu.Lock()
	if p.loopCancel != nil {
		p.loopCancel()
		p.loopCancel = nil
	}
	p.loopMu.Unlock()
}

func (p *AutomationProcessor) run(ctx context.Context) {
	LogInfo("Processor loop started")

	for {
		select {
		case <-ctx.Done():
			LogInfo("Processor loop stopped")
			return
		default:
		}

		p.tick()

		sleep := p.sleepDuration()
		select {
		case <-ctx.Done():
			LogInfo("Processor loop stopped")
			return
		case <-time.After(sleep):
		}
	}
}

func (p *AutomationProcessor) sleepDuration() time.Duration {
	state := p.CurrentState()
	if state.RefreshInterval <= 0 {
		return 10 * time.Second
	}
	return time.Duration(state.RefreshInterval) * time.Second
}

func (p *AutomationProcessor) tick() {
	state := p.CurrentState()
	if !state.Active {
		return
	}

	p.host.clearExpiredRateLimitMetadata()

	openIssues, err := p.host.loadIssuesFromFolder("open")
	if err != nil {
		LogErrorErr("Failed to load open issues during processor loop", err)
		return
	}

	if len(openIssues) == 0 {
		return
	}

	currentlyRunning, availableSlots := p.processorSlots(state)
	if availableSlots <= 0 {
		LogInfo(
			"Processor waiting for available slots",
			"running", currentlyRunning,
			"slots", state.ConcurrentSlots,
		)
		return
	}

	p.updateRunningCount(currentlyRunning)
	p.scheduleInvestigations(openIssues, availableSlots, currentlyRunning)
}

func (p *AutomationProcessor) processorSlots(state ProcessorState) (int, int) {
	running := p.RunningProcesses()
	currentlyRunning := len(running)
	availableSlots := state.ConcurrentSlots - currentlyRunning
	if availableSlots < 0 {
		availableSlots = 0
	}
	return currentlyRunning, availableSlots
}

func (p *AutomationProcessor) updateRunningCount(currentlyRunning int) {
	p.stateMu.Lock()
	p.state.CurrentlyRunning = currentlyRunning
	p.stateMu.Unlock()
}

func (p *AutomationProcessor) scheduleInvestigations(openIssues []Issue, availableSlots, currentlyRunning int) {
	scheduled := 0
	for _, issue := range openIssues {
		if scheduled >= availableSlots {
			break
		}
		if !p.canProcessMoreIssues() {
			return
		}
		if p.shouldDeferIssue(issue) {
			continue
		}

		agentID := "unified-resolver"
		if issue.Metadata.Extra != nil {
			if id, ok := issue.Metadata.Extra["preferred_agent"]; ok && strings.TrimSpace(id) != "" {
				agentID = strings.TrimSpace(id)
			}
		}

		startTime := time.Now().UTC().Format(time.RFC3339)
		p.RegisterRunningProcess(issue.ID, agentID, startTime, nil)
		scheduled++

		sequence := currentlyRunning + scheduled
		go func(issue Issue, agentID string, sequence int) {
			defer p.UnregisterRunningProcess(issue.ID)

			if err := p.host.triggerInvestigation(issue.ID, agentID, true); err != nil {
				LogErrorErr("Failed to trigger investigation", err, "issue_id", issue.ID)
				return
			}

			LogInfo("Triggered investigation", "issue_id", issue.ID)
			p.IncrementProcessedCount()
		}(issue, agentID, sequence)
	}
}

func (p *AutomationProcessor) canProcessMoreIssues() bool {
	state := p.CurrentState()
	if state.MaxIssuesDisabled || state.MaxIssues == 0 {
		return true
	}

	processed := p.ProcessedCount()
	return processed < state.MaxIssues
}

func (p *AutomationProcessor) shouldDeferIssue(issue Issue) bool {
	if issue.Metadata.Extra == nil {
		return false
	}

	deadlineRaw := strings.TrimSpace(issue.Metadata.Extra["rate_limit_until"])
	if deadlineRaw == "" {
		return false
	}

	deadline, err := time.Parse(time.RFC3339, deadlineRaw)
	if err != nil {
		LogWarn("Clearing malformed rate limit metadata", "issue_id", issue.ID, "raw_value", deadlineRaw)
		p.host.clearRateLimitMetadata(issue.ID)
		return false
	}

	if time.Now().Before(deadline) {
		return true
	}

	// Expired window; clean up metadata so the issue can be picked up next tick.
	p.host.clearRateLimitMetadata(issue.ID)
	return false
}

func (s *Server) currentProcessorState() ProcessorState {
	return s.processor.CurrentState()
}

func (s *Server) updateProcessorState(active *bool, slots *int, interval *int, maxIssues *int, maxIssuesDisabled *bool) {
	s.processor.UpdateState(active, slots, interval, maxIssues, maxIssuesDisabled)
}

func (s *Server) resetIssueCounterHandler(w http.ResponseWriter, r *http.Request) {
	s.processor.ResetCounter()

	LogInfo("Processor issue counter reset")

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

func (s *Server) incrementProcessedCount() int {
	return s.processor.IncrementProcessedCount()
}

func (s *Server) getProcessorHandler(w http.ResponseWriter, r *http.Request) {
	state := s.currentProcessorState()
	issuesProcessed := s.processor.ProcessedCount()

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

	LogInfo(
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
		Data: map[string]interface{}{
			"processor": state,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getRunningProcessesHandler(w http.ResponseWriter, r *http.Request) {
	processes := s.processor.RunningProcesses()

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"processes": processes,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) stopRunningProcessHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	if !s.processor.CancelRunningProcess(issueID, "user_stop") {
		http.Error(w, "Agent is not running for the specified issue", http.StatusNotFound)
		return
	}

	LogInfo("Stop requested for running agent", "issue_id", issueID)

	response := ApiResponse{
		Success: true,
		Message: "Agent stop requested",
		Data: map[string]interface{}{
			"issue_id": issueID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) registerRunningProcess(issueID, agentID, startTime string, cancel context.CancelFunc) {
	if _, currentFolder, err := s.findIssueDirectory(issueID); err == nil && currentFolder != "active" {
		if err := s.moveIssue(issueID, "active"); err != nil {
			LogErrorErr("Failed to mark running issue as active", err, "issue_id", issueID)
		} else {
			LogInfo("Normalized issue status for running process", "issue_id", issueID, "previous_status", currentFolder)
		}
	}

	s.processor.RegisterRunningProcess(issueID, agentID, startTime, cancel)

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

func (s *Server) StartProcessor(ctx context.Context) {
	s.processor.Start(ctx)
}

func (s *Server) StartProcessorLoop() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	s.StartProcessor(ctx)
	return cancel
}

func (s *Server) getRateLimitStatusHandler(w http.ResponseWriter, r *http.Request) {
	openIssues, err := s.loadIssuesFromFolder("open")
	if err != nil {
		LogErrorErr("Failed to load open issues for rate limit status", err)
		openIssues = []Issue{}
	}

	var earliestReset time.Time
	rateLimitedCount := 0
	var rateLimitType string
	var rateLimitAgent string

	for _, issue := range openIssues {
		if issue.Metadata.Extra == nil {
			continue
		}

		resetTimeStr := issue.Metadata.Extra["rate_limit_until"]
		if resetTimeStr == "" {
			continue
		}

		resetTime, parseErr := time.Parse(time.RFC3339, resetTimeStr)
		if parseErr != nil {
			LogWarn(
				"Clearing stale rate limit metadata",
				"issue_id", issue.ID,
				"raw_reset_value", resetTimeStr,
			)
			if loaded, issueDir, _, loadErr := s.loadIssueWithStatus(issue.ID); loadErr == nil {
				if loaded.Metadata.Extra != nil {
					delete(loaded.Metadata.Extra, "rate_limit_until")
					delete(loaded.Metadata.Extra, "rate_limit_agent")
					s.writeIssueMetadata(issueDir, loaded)
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

		LogInfo("Rate limit window elapsed via status endpoint", "issue_id", issue.ID)
		s.clearRateLimitMetadata(issue.ID)
	}

	rateLimited := rateLimitedCount > 0
	var resetTimeValue string
	var secondsUntilReset int64

	if rateLimited {
		resetTimeValue = earliestReset.Format(time.RFC3339)
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
			"reset_time":           resetTimeValue,
			"seconds_until_reset":  secondsUntilReset,
			"waiting_issues_count": rateLimitedCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) clearExpiredRateLimitMetadata() {
	issues, err := s.loadIssuesFromFolder("open")
	if err != nil {
		LogErrorErr("Failed to inspect open issues for rate limit metadata", err)
		return
	}

	for _, issue := range issues {
		if issue.Metadata.Extra == nil {
			continue
		}

		resetTimeStr := issue.Metadata.Extra["rate_limit_until"]
		if resetTimeStr == "" {
			continue
		}

		resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
		if err != nil {
			LogWarn(
				"Clearing stale rate limit metadata",
				"issue_id", issue.ID,
				"raw_reset_value", resetTimeStr,
			)
			s.clearRateLimitMetadata(issue.ID)
			continue
		}

		if time.Now().Before(resetTime) {
			continue
		}

		LogInfo("Rate limit expiry detected", "issue_id", issue.ID)
		s.clearRateLimitMetadata(issue.ID)
	}
}

func (s *Server) clearRateLimitMetadata(issueID string) {
	loaded, issueDir, _, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		return
	}

	if loaded.Metadata.Extra == nil {
		return
	}

	delete(loaded.Metadata.Extra, "rate_limit_until")
	delete(loaded.Metadata.Extra, "rate_limit_agent")

	s.writeIssueMetadata(issueDir, loaded)
}
