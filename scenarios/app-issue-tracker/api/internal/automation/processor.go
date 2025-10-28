package automation

import (
	"context"
	"strings"
	"sync"
	"time"

	"app-issue-tracker-api/internal/agents"
	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/metadata"
)

// Host defines the operations the processor requires from the embedding server.
type Host interface {
	ClearExpiredRateLimitMetadata()
	LoadIssuesFromFolder(folder string) ([]issuespkg.Issue, error)
	ClearRateLimitMetadata(issueID string)
	TriggerInvestigation(issueID, agentID string, autoResolve bool) error
	CleanupOldTranscripts()
}

// Processor manages automated issue investigations.
type Processor struct {
	host Host

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

func NewProcessor(host Host) *Processor {
	p := &Processor{
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

func (p *Processor) CurrentState() ProcessorState {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.state
}

func (p *Processor) UpdateState(active *bool, slots *int, interval *int, maxIssues *int, maxIssuesDisabled *bool) {
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

func (p *Processor) ResetCounter() {
	p.processedMu.Lock()
	defer p.processedMu.Unlock()
	p.processedCount = 0
}

func (p *Processor) IncrementProcessedCount() int {
	p.processedMu.Lock()
	defer p.processedMu.Unlock()
	p.processedCount++
	return p.processedCount
}

func (p *Processor) ProcessedCount() int {
	p.processedMu.RLock()
	defer p.processedMu.RUnlock()
	return p.processedCount
}

func (p *Processor) RegisterRunningProcess(issueID, agentID, startTime string, cancel context.CancelFunc) {
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

func (p *Processor) UnregisterRunningProcess(issueID string) {
	p.runningMu.Lock()
	defer p.runningMu.Unlock()
	delete(p.running, issueID)
	p.updateRunningCountLocked()
}

func (p *Processor) IsRunning(issueID string) bool {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	_, ok := p.running[issueID]
	return ok
}

func (p *Processor) RunningProcesses() []*RunningProcess {
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

func (p *Processor) CancelRunningProcess(issueID, reason string) bool {
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

func (p *Processor) CancellationInfo(issueID string) (bool, string) {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	proc, ok := p.running[issueID]
	if !ok {
		return false, ""
	}
	return proc.cancelRequested, proc.cancelReason
}

func (p *Processor) updateRunningCountLocked() {
	count := len(p.running)
	p.stateMu.Lock()
	p.state.CurrentlyRunning = count
	p.stateMu.Unlock()
}

func (p *Processor) Start(ctx context.Context) {
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

func (p *Processor) Stop() {
	p.loopMu.Lock()
	if p.loopCancel != nil {
		p.loopCancel()
		p.loopCancel = nil
	}
	p.loopMu.Unlock()
}

func (p *Processor) run(ctx context.Context) {
	logging.LogInfo("Processor loop started")

	for {
		select {
		case <-ctx.Done():
			logging.LogInfo("Processor loop stopped")
			return
		default:
		}

		p.tick()

		sleep := p.sleepDuration()
		select {
		case <-ctx.Done():
			logging.LogInfo("Processor loop stopped")
			return
		case <-time.After(sleep):
		}
	}
}

func (p *Processor) sleepDuration() time.Duration {
	state := p.CurrentState()
	if state.RefreshInterval <= 0 {
		return 10 * time.Second
	}
	return time.Duration(state.RefreshInterval) * time.Second
}

func (p *Processor) tick() {
	state := p.CurrentState()
	if !state.Active {
		return
	}

	p.host.ClearExpiredRateLimitMetadata()
	p.host.CleanupOldTranscripts()

	openIssues, err := p.host.LoadIssuesFromFolder("open")
	if err != nil {
		logging.LogErrorErr("Failed to load open issues during processor loop", err)
		return
	}

	if len(openIssues) == 0 {
		return
	}

	currentlyRunning, availableSlots := p.processorSlots(state)
	if availableSlots <= 0 {
		logging.LogInfo(
			"Processor waiting for available slots",
			"running", currentlyRunning,
			"slots", state.ConcurrentSlots,
		)
		return
	}

	p.updateRunningCount(currentlyRunning)
	p.scheduleInvestigations(openIssues, availableSlots, currentlyRunning)
}

func (p *Processor) processorSlots(state ProcessorState) (int, int) {
	running := p.RunningProcesses()
	currentlyRunning := len(running)
	availableSlots := state.ConcurrentSlots - currentlyRunning
	if availableSlots < 0 {
		availableSlots = 0
	}
	return currentlyRunning, availableSlots
}

func (p *Processor) updateRunningCount(currentlyRunning int) {
	p.stateMu.Lock()
	p.state.CurrentlyRunning = currentlyRunning
	p.stateMu.Unlock()
}

func (p *Processor) scheduleInvestigations(openIssues []issuespkg.Issue, availableSlots, currentlyRunning int) {
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

		agentID := agents.UnifiedResolverID
		if issue.Metadata.Extra != nil {
			if id, ok := issue.Metadata.Extra[metadata.PreferredAgentKey]; ok && strings.TrimSpace(id) != "" {
				agentID = strings.TrimSpace(id)
			}
		}

		// TriggerInvestigation will handle the registration and unregistration
		// of the running process, so we don't need to wrap it in a goroutine here.
		// The investigation runs asynchronously inside TriggerInvestigation.
		if err := p.host.TriggerInvestigation(issue.ID, agentID, true); err != nil {
			logging.LogErrorErr("Failed to trigger investigation", err, "issue_id", issue.ID)
			continue
		}

		scheduled++
		logging.LogInfo("Scheduled investigation", "issue_id", issue.ID, "sequence", currentlyRunning+scheduled)
	}
}

func (p *Processor) canProcessMoreIssues() bool {
	state := p.CurrentState()
	if state.MaxIssuesDisabled || state.MaxIssues == 0 {
		return true
	}

	processed := p.ProcessedCount()
	return processed < state.MaxIssues
}

func (p *Processor) shouldDeferIssue(issue issuespkg.Issue) bool {
	if issue.Metadata.Extra == nil {
		return false
	}

	deadlineRaw := strings.TrimSpace(issue.Metadata.Extra[metadata.RateLimitUntilKey])
	if deadlineRaw == "" {
		return false
	}

	deadline, err := time.Parse(time.RFC3339, deadlineRaw)
	if err != nil {
		logging.LogWarn("Clearing malformed rate limit metadata", "issue_id", issue.ID, "raw_value", deadlineRaw)
		p.host.ClearRateLimitMetadata(issue.ID)
		return false
	}

	if time.Now().Before(deadline) {
		return true
	}

	// Expired window; clean up metadata so the issue can be picked up next tick.
	p.host.ClearRateLimitMetadata(issue.ID)
	return false
}

// ProcessorState captures current automation configuration and metrics.
type ProcessorState struct {
	Active            bool `json:"active"`
	ConcurrentSlots   int  `json:"concurrent_slots"`
	RefreshInterval   int  `json:"refresh_interval"`
	CurrentlyRunning  int  `json:"currently_running"`
	MaxIssues         int  `json:"max_issues"`
	MaxIssuesDisabled bool `json:"max_issues_disabled"`
}

// RunningProcess describes an active agent run tracked by the processor.
type RunningProcess struct {
	IssueID   string `json:"issue_id"`
	AgentID   string `json:"agent_id"`
	StartTime string `json:"start_time"`
	Status    string `json:"status,omitempty"`
}

const (
	AgentStatusRunning    = "running"
	AgentStatusCancelling = "cancelling"
	AgentStatusCompleted  = "completed"
	AgentStatusFailed     = "failed"
	AgentStatusCancelled  = "cancelled"

	AgentStatusExtraKey          = "agent_last_status"
	AgentStatusTimestampExtraKey = "agent_last_status_at"
)
