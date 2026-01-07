package fix

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"test-genie/agentmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Service provides fix operations for test-genie scenarios.
type Service struct {
	agentSvc *agentmanager.AgentService
	store    *Store
	mu       sync.Mutex
}

// NewService creates a new fix service.
func NewService(agentSvc *agentmanager.AgentService) *Service {
	return &Service{
		agentSvc: agentSvc,
		store:    NewStore(),
	}
}

// SpawnRequest contains the parameters for spawning a fix agent.
type SpawnRequest struct {
	ScenarioName string
	Phases       []PhaseInfo
}

// SpawnResult contains the result of spawning a fix agent.
type SpawnResult struct {
	FixID  string
	RunID  string
	Tag    string
	Status Status
	Error  string
}

// Spawn creates a new fix agent for the specified scenario and phases.
func (s *Service) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if agent-manager is available
	if !s.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available")
	}

	// Check if there's already an active fix for this scenario
	if active := s.store.GetActiveForScenario(req.ScenarioName); active != nil {
		return nil, fmt.Errorf("a fix is already in progress for scenario %s (id: %s)", req.ScenarioName, active.ID)
	}

	// Generate IDs
	fixID := uuid.New().String()[:8]
	tag := fmt.Sprintf("fix-%s-%s", req.ScenarioName, fixID)

	// Create fix record
	record := &Record{
		ID:           fixID,
		ScenarioName: req.ScenarioName,
		Phases:       req.Phases,
		Status:       StatusPending,
		Tag:          tag,
		StartedAt:    time.Now(),
	}
	s.store.Create(record)

	// Spawn the agent in a goroutine
	go s.runFix(record)

	return &SpawnResult{
		FixID:  fixID,
		Tag:    tag,
		Status: StatusPending,
	}, nil
}

// runFix executes the fix agent and updates the record with results.
func (s *Service) runFix(record *Record) {
	ctx := context.Background()

	// Update status to running
	record.Status = StatusRunning
	s.store.Update(record)

	// Build the prompt
	scenarioPath := GetScenarioPath(record.ScenarioName)
	prompt := BuildPrompt(PromptConfig{
		ScenarioName: record.ScenarioName,
		ScenarioPath: scenarioPath,
		Phases:       record.Phases,
	})

	// Build safety preamble
	repoRoot := os.Getenv("VROOLI_ROOT")
	if repoRoot == "" {
		repoRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	preambleAttachment := agentmanager.GeneratePreambleAttachment(agentmanager.PreambleConfig{
		Scenario:       record.ScenarioName,
		RepoRoot:       repoRoot,
		MaxFiles:       50,
		MaxBytes:       1024 * 1024,
		NetworkEnabled: false,
	})

	// Create context attachments
	var attachments []*domainpb.ContextAttachment
	if preambleAttachment != nil {
		attachments = append(attachments, preambleAttachment)
	}

	// Add phase details as attachment
	var phaseDetails string
	for _, phase := range record.Phases {
		phaseDetails += fmt.Sprintf("- %s: %s\n", phase.Name, phase.Status)
		if phase.Error != "" {
			phaseDetails += fmt.Sprintf("  Error: %s\n", phase.Error)
		}
	}
	attachments = append(attachments, &domainpb.ContextAttachment{
		Type:    "note",
		Key:     "fix-phases",
		Label:   "Phases to Fix",
		Content: phaseDetails,
		Tags:    []string{"phases", "fix"},
	})

	// Create task and run via SpawnSingle
	task := &domainpb.Task{
		Title:              fmt.Sprintf("Test Fix - %s", record.ScenarioName),
		Description:        prompt,
		ScopePath:          scenarioPath,
		ProjectRoot:        scenarioPath,
		CreatedBy:          "test-genie-fix",
		ContextAttachments: attachments,
	}

	result, err := s.agentSvc.SpawnSingle(ctx, agentmanager.SpawnSingleRequest{
		Task: task,
		Tag:  record.Tag,
	})
	if err != nil {
		s.failFix(record, fmt.Sprintf("failed to spawn agent: %v", err))
		return
	}

	record.RunID = result.RunID
	s.store.Update(record)

	log.Printf("[fix] Started fix %s for scenario %s (runID: %s)", record.ID, record.ScenarioName, result.RunID)

	// Poll for completion
	s.pollForCompletion(ctx, record)
}

// pollForCompletion polls the agent run status until it reaches a terminal state.
func (s *Service) pollForCompletion(ctx context.Context, record *Record) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // 30 minute timeout for fix operations

	for {
		select {
		case <-ctx.Done():
			s.failFix(record, "context cancelled")
			return
		case <-timeout:
			s.failFix(record, "fix operation timed out")
			return
		case <-ticker.C:
			run, err := s.agentSvc.GetRun(ctx, record.RunID)
			if err != nil {
				log.Printf("[fix] Error getting run status for %s: %v", record.ID, err)
				continue
			}

			status := agentmanager.MapRunStatus(run.Status)

			switch status {
			case "completed":
				s.completeFix(record, run)
				return
			case "failed":
				errorMsg := "agent run failed"
				if run.ErrorMsg != "" {
					errorMsg = run.ErrorMsg
				}
				s.failFix(record, errorMsg)
				return
			case "stopped":
				record.Status = StatusCancelled
				now := time.Now()
				record.CompletedAt = &now
				s.store.Update(record)
				log.Printf("[fix] Fix %s was cancelled", record.ID)
				return
			}
		}
	}
}

// completeFix marks a fix as completed and stores the output.
func (s *Service) completeFix(record *Record, run *domainpb.Run) {
	record.Status = StatusCompleted
	now := time.Now()
	record.CompletedAt = &now

	if run.Summary != nil {
		record.Output = run.Summary.Description
	}

	s.store.Update(record)
	log.Printf("[fix] Fix %s completed successfully for scenario %s", record.ID, record.ScenarioName)
}

// failFix marks a fix as failed with an error message.
func (s *Service) failFix(record *Record, errorMsg string) {
	record.Status = StatusFailed
	record.Error = errorMsg
	now := time.Now()
	record.CompletedAt = &now
	s.store.Update(record)
	log.Printf("[fix] Fix %s failed for scenario %s: %s", record.ID, record.ScenarioName, errorMsg)
}

// Get retrieves a fix record by ID.
func (s *Service) Get(id string) (*Record, bool) {
	return s.store.Get(id)
}

// ListByScenario returns fix records for a scenario.
func (s *Service) ListByScenario(scenarioName string, limit int) []*Record {
	return s.store.ListByScenario(scenarioName, limit)
}

// GetActiveForScenario returns the active fix for a scenario, if any.
func (s *Service) GetActiveForScenario(scenarioName string) *Record {
	return s.store.GetActiveForScenario(scenarioName)
}

// Stop stops a running fix.
func (s *Service) Stop(ctx context.Context, id string) error {
	record, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("fix not found: %s", id)
	}

	if record.IsTerminal() {
		return fmt.Errorf("fix is already in terminal state: %s", record.Status)
	}

	if record.RunID == "" {
		return fmt.Errorf("fix has no associated run")
	}

	if err := s.agentSvc.StopRun(ctx, record.RunID); err != nil {
		return fmt.Errorf("failed to stop run: %w", err)
	}

	record.Status = StatusCancelled
	now := time.Now()
	record.CompletedAt = &now
	s.store.Update(record)

	return nil
}

// IsAgentAvailable checks if agent-manager is available.
func (s *Service) IsAgentAvailable(ctx context.Context) bool {
	return s.agentSvc.IsAvailable(ctx)
}
