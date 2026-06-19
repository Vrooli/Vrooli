package heartbeat

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"prompt-manager/store"
	"prompt-manager/teamconfig"

	"github.com/robfig/cron/v3"
)

// HeartbeatExecutor defines the execution behavior used by the scheduler.
type HeartbeatExecutor interface {
	Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error)
}

// HeartbeatConfigStore defines access to heartbeat configurations for scheduling.
type HeartbeatConfigStore interface {
	GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, error)
}

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
	mu            sync.RWMutex
	cron          *cron.Cron
	scheduled     map[string]*ScheduledHeartbeat // key: teamID/agentID
	executor      HeartbeatExecutor
	running       bool
	profileKey    string
	agentClient   AgentClient
	configStore   HeartbeatConfigStore
	teamExecStore *TeamExecutionStore
	controlStore  *HeartbeatControlStore
}

// NewScheduler creates a new heartbeat scheduler
func NewScheduler(executor HeartbeatExecutor, agentClient AgentClient, configStore HeartbeatConfigStore, teamExecStore *TeamExecutionStore) *Scheduler {
	return &Scheduler{
		cron:          cron.New(),
		scheduled:     make(map[string]*ScheduledHeartbeat),
		executor:      executor,
		profileKey:    "prompt-manager-heartbeat",
		agentClient:   agentClient,
		configStore:   configStore,
		teamExecStore: teamExecStore,
	}
}

// SetControlStore attaches the heartbeat engagement guard. The scheduler
// treats a paused gate as "do not schedule/start" without mutating heartbeat
// configs, so resume can restore the same enabled configs later.
func (s *Scheduler) SetControlStore(controlStore *HeartbeatControlStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.controlStore = controlStore
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

	if s.controlStore != nil {
		if paused, err := s.controlStore.AllowStart(context.Background(), teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				log.Printf("Heartbeat schedule held for %s/%s: %s", teamID, agentID, paused.Message)
				return nil
			}
			return err
		}
	}

	key := makeKey(teamID, agentID)

	// Remove existing schedule if any
	if existing, ok := s.scheduled[key]; ok {
		s.cron.Remove(existing.EntryID)
		if existing.cancelCtx != nil {
			existing.cancelCtx()
		}
		delete(s.scheduled, key)
	}

	// Parse and validate cron expression (standard 5-field: minute hour dom month dow)
	_, err := cronParser.Parse(schedule)
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

// cronParser is the shared parser matching the cron instance (standard 5-field).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// GetNextRuns returns up to count future execution times for the given heartbeat,
// computed by iterating the cron schedule forward from now.
func (s *Scheduler) GetNextRuns(teamID, agentID string, count int) []time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	sh, ok := s.scheduled[key]
	if !ok || count <= 0 {
		return nil
	}

	schedule, err := cronParser.Parse(sh.Schedule)
	if err != nil {
		return nil
	}

	runs := make([]time.Time, 0, count)
	t := time.Now()
	for i := 0; i < count; i++ {
		t = schedule.Next(t)
		if t.IsZero() {
			break
		}
		runs = append(runs, t)
	}
	return runs
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

	if s.controlStore != nil {
		if paused, err := s.controlStore.AllowStart(ctx, teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				log.Printf("Heartbeat execution skipped for %s/%s: %s", teamID, agentID, paused.Message)
				return
			}
			log.Printf("Heartbeat execution gate failed for %s/%s: %v", teamID, agentID, err)
			return
		}
	}

	// Start with empty profileKey; Execute() resolves the default based on
	// the team's runtime mode when the key is empty.
	var profileKey string
	if s.configStore != nil {
		config, err := s.configStore.GetHeartbeatConfig(ctx, teamID, agentID)
		if err != nil {
			log.Printf("Heartbeat execution aborted for %s/%s: %v", teamID, agentID, err)
			return
		}
		if config == nil {
			log.Printf("Heartbeat execution aborted for %s/%s: config not found", teamID, agentID)
			return
		}
		if !config.Enabled {
			log.Printf("Heartbeat execution skipped for %s/%s: config disabled", teamID, agentID)
			return
		}
		profileKey = config.ProfileKey
	}

	// Route through the team execution store so the configured queue policy is enforced.
	if s.teamExecStore != nil {
		result, err := s.teamExecStore.Enqueue(ctx, teamID, agentID, profileKey)
		if err != nil {
			if IsMemberAlreadyQueued(err) {
				log.Printf("Heartbeat skipped for %s/%s: already queued or running", teamID, agentID)
				return
			}
			log.Printf("Heartbeat execution failed for %s/%s: %v", teamID, agentID, err)
			return
		}
		log.Printf("Heartbeat enqueued for %s/%s: status=%s, position=%d",
			teamID, agentID, result.Status, result.Position)
		return
	}

	// Fallback: direct execution (no team execution store)
	result, err := s.executor.Execute(ctx, teamID, agentID, profileKey)
	if err != nil {
		log.Printf("Heartbeat execution failed for %s/%s: %v", teamID, agentID, err)
		return
	}

	log.Printf("Heartbeat execution completed for %s/%s, run ID: %s, status: %s",
		teamID, agentID, result.RunID, result.Status)
}

// ensureProfile ensures both heartbeat profiles exist in agent-manager.
func (s *Scheduler) ensureProfile(ctx context.Context) error {
	if s.agentClient == nil {
		return nil
	}

	profiles := []struct {
		key         string
		runtimeMode string
	}{
		{DefaultProfileKeyCodex, teamconfig.RuntimeModeMultiProcess},
		{DefaultProfileKeyClaudeCode, teamconfig.RuntimeModeSingleProcess},
	}
	for _, p := range profiles {
		req := &EnsureProfileRequest{
			ProfileKey:     p.key,
			Defaults:       BuildDefaultProfileForRuntimeMode(p.key, p.runtimeMode),
			UpdateExisting: true,
		}
		resp, err := s.agentClient.EnsureProfile(ctx, req)
		if err != nil {
			return err
		}
		if resp.Created {
			log.Printf("Created heartbeat profile: %s", p.key)
		}
	}

	return nil
}

// Default profile key constants.
const (
	// DefaultProfileKeyCodex is used for multi-process teams (Codex runner).
	DefaultProfileKeyCodex = "prompt-manager-heartbeat"
	// DefaultProfileKeyClaudeCode is used for single-process teams (Claude Code runner).
	DefaultProfileKeyClaudeCode = "prompt-manager-heartbeat-cc"
)

// DefaultProfileKeyForRuntimeMode returns the appropriate default profile key
// for the given runtime mode. Single-process teams get Claude Code; everything
// else gets Codex.
func DefaultProfileKeyForRuntimeMode(runtimeMode string) string {
	if runtimeMode == teamconfig.RuntimeModeSingleProcess {
		return DefaultProfileKeyClaudeCode
	}
	return DefaultProfileKeyCodex
}

// BuildDefaultProfile returns the default Codex agent profile for the given key.
// Exported so the executor can embed defaults in CreateRun requests.
// For runtime-aware resolution, use BuildDefaultProfileForRuntimeMode instead.
func BuildDefaultProfile(profileKey string) *AgentProfile {
	return buildCodexProfile(profileKey)
}

// BuildDefaultProfileForRuntimeMode returns the correct default AgentProfile
// for the given profile key and runtime mode. Known default keys always map to
// their specific runner; unknown custom keys use the runtime mode as a heuristic.
func BuildDefaultProfileForRuntimeMode(profileKey, runtimeMode string) *AgentProfile {
	switch profileKey {
	case DefaultProfileKeyClaudeCode:
		return buildClaudeCodeProfile(profileKey)
	case DefaultProfileKeyCodex:
		return buildCodexProfile(profileKey)
	default:
		// Custom keys: fall back to runtime-mode heuristic.
		if runtimeMode == teamconfig.RuntimeModeSingleProcess {
			return buildClaudeCodeProfile(profileKey)
		}
		return buildCodexProfile(profileKey)
	}
}

func buildCodexProfile(profileKey string) *AgentProfile {
	return &AgentProfile{
		Name:                 "Prompt Manager Heartbeat",
		ProfileKey:           profileKey,
		Description:          "Profile for multi-process team heartbeat execution",
		RunnerType:           "RUNNER_TYPE_CODEX",
		ModelPreset:          "MODEL_PRESET_SMART",
		MaxTurns:             50,
		Timeout:              DurationToProtojson(10 * time.Minute),
		AllowedTools:         []string{"read_file", "write_file", "execute_command"},
		SkipPermissionPrompt: true,
		SandboxConfig:        &SandboxConfig{Mode: "SANDBOX_MODE_PROTECTED"},
		CreatedBy:            "prompt-manager",
	}
}

func buildClaudeCodeProfile(profileKey string) *AgentProfile {
	return &AgentProfile{
		Name:                 "Prompt Manager Heartbeat (Claude Code)",
		ProfileKey:           profileKey,
		Description:          "Profile for single-process team heartbeat execution",
		RunnerType:           "RUNNER_TYPE_CLAUDE_CODE",
		ModelPreset:          "MODEL_PRESET_SMART",
		MaxTurns:             200,
		Timeout:              DurationToProtojson(30 * time.Minute),
		AllowedTools:         []string{},
		SkipPermissionPrompt: true,
		SandboxConfig:        &SandboxConfig{Mode: "SANDBOX_MODE_PROTECTED"},
		CreatedBy:            "prompt-manager",
	}
}

// makeKey creates a unique key for a team/agent combination
func makeKey(teamID, agentID string) string {
	return teamID + "/" + agentID
}
