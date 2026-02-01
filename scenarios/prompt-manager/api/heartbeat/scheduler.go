package heartbeat

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduledHeartbeat represents a scheduled heartbeat with its timer
type ScheduledHeartbeat struct {
	TeamID    string
	AgentID   string
	Schedule  string
	EntryID   cron.EntryID
	NextRun   time.Time
	cancelCtx context.CancelFunc
}

// Scheduler manages cron-based heartbeat execution
type Scheduler struct {
	mu          sync.RWMutex
	cron        *cron.Cron
	scheduled   map[string]*ScheduledHeartbeat // key: teamID/agentID
	executor    *Executor
	running     bool
	profileKey  string
	agentClient *AgentManagerClient
}

// NewScheduler creates a new heartbeat scheduler
func NewScheduler(executor *Executor, agentClient *AgentManagerClient) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		scheduled:   make(map[string]*ScheduledHeartbeat),
		executor:    executor,
		profileKey:  "prompt-manager-heartbeat",
		agentClient: agentClient,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Ensure profile exists at startup
	if err := s.ensureProfile(ctx); err != nil {
		log.Printf("Warning: Failed to ensure heartbeat profile: %v", err)
		// Don't fail startup, profile will be created on first heartbeat
	}

	s.cron.Start()
	s.running = true
	log.Println("Heartbeat scheduler started")

	return nil
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	// Cancel all scheduled heartbeats
	for key, sh := range s.scheduled {
		s.cron.Remove(sh.EntryID)
		if sh.cancelCtx != nil {
			sh.cancelCtx()
		}
		delete(s.scheduled, key)
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	log.Println("Heartbeat scheduler stopped")
}

// Schedule adds or updates a heartbeat schedule
func (s *Scheduler) Schedule(teamID, agentID, schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(teamID, agentID)

	// Remove existing schedule if any
	if existing, ok := s.scheduled[key]; ok {
		s.cron.Remove(existing.EntryID)
		if existing.cancelCtx != nil {
			existing.cancelCtx()
		}
		delete(s.scheduled, key)
	}

	// Parse and validate cron expression
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(schedule)
	if err != nil {
		return err
	}

	// Create context for this heartbeat
	ctx, cancel := context.WithCancel(context.Background())

	// Add to cron
	entryID, err := s.cron.AddFunc(schedule, func() {
		s.executeHeartbeat(ctx, teamID, agentID)
	})
	if err != nil {
		cancel()
		return err
	}

	// Get next run time
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next

	s.scheduled[key] = &ScheduledHeartbeat{
		TeamID:    teamID,
		AgentID:   agentID,
		Schedule:  schedule,
		EntryID:   entryID,
		NextRun:   nextRun,
		cancelCtx: cancel,
	}

	log.Printf("Scheduled heartbeat for %s/%s with schedule %s, next run: %s",
		teamID, agentID, schedule, nextRun.Format(time.RFC3339))

	return nil
}

// Unschedule removes a heartbeat schedule
func (s *Scheduler) Unschedule(teamID, agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(teamID, agentID)
	if sh, ok := s.scheduled[key]; ok {
		s.cron.Remove(sh.EntryID)
		if sh.cancelCtx != nil {
			sh.cancelCtx()
		}
		delete(s.scheduled, key)
		log.Printf("Unscheduled heartbeat for %s/%s", teamID, agentID)
	}
}

// GetNextRun returns the next scheduled execution time
func (s *Scheduler) GetNextRun(teamID, agentID string) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	if sh, ok := s.scheduled[key]; ok {
		entry := s.cron.Entry(sh.EntryID)
		return &entry.Next
	}
	return nil
}

// IsScheduled checks if a heartbeat is scheduled
func (s *Scheduler) IsScheduled(teamID, agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	_, ok := s.scheduled[key]
	return ok
}

// executeHeartbeat runs a heartbeat execution
func (s *Scheduler) executeHeartbeat(ctx context.Context, teamID, agentID string) {
	log.Printf("Executing heartbeat for %s/%s", teamID, agentID)

	result, err := s.executor.Execute(ctx, teamID, agentID, s.profileKey)
	if err != nil {
		log.Printf("Heartbeat execution failed for %s/%s: %v", teamID, agentID, err)
		return
	}

	log.Printf("Heartbeat execution completed for %s/%s, run ID: %s, status: %s",
		teamID, agentID, result.RunID, result.Status)
}

// ensureProfile ensures the heartbeat profile exists in agent-manager
func (s *Scheduler) ensureProfile(ctx context.Context) error {
	if s.agentClient == nil {
		return nil
	}

	req := &EnsureProfileRequest{
		ProfileKey:     s.profileKey,
		Defaults:       s.buildDefaultProfile(),
		UpdateExisting: false,
	}

	resp, err := s.agentClient.EnsureProfile(ctx, req)
	if err != nil {
		return err
	}

	if resp.Created {
		log.Printf("Created heartbeat profile: %s", s.profileKey)
	}

	return nil
}

// buildDefaultProfile returns the default profile for heartbeat execution
func (s *Scheduler) buildDefaultProfile() *AgentProfile {
	return &AgentProfile{
		Name:                 "Prompt Manager Heartbeat",
		ProfileKey:           s.profileKey,
		Description:          "Profile for team member heartbeat execution",
		RunnerType:           "RUNNER_TYPE_CODEX",
		ModelPreset:          "MODEL_PRESET_SMART",
		MaxTurns:             50,
		Timeout:              10 * time.Minute,
		AllowedTools:         []string{"read_file", "write_file", "execute_command"},
		SkipPermissionPrompt: true,
		RequiresSandbox:      false,
		RequiresApproval:     false,
		CreatedBy:            "prompt-manager",
	}
}

// makeKey creates a unique key for a team/agent combination
func makeKey(teamID, agentID string) string {
	return teamID + "/" + agentID
}
