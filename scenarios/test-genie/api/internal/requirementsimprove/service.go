package requirementsimprove

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

// Service provides requirements improve operations for test-genie scenarios.
type Service struct {
	agentSvc *agentmanager.AgentService
	store    *Store
	mu       sync.Mutex
}

// NewService creates a new requirements improve service.
func NewService(agentSvc *agentmanager.AgentService) *Service {
	return &Service{
		agentSvc: agentSvc,
		store:    NewStore(),
	}
}

// SpawnRequest contains the parameters for spawning a requirements improve agent.
type SpawnRequest struct {
	ScenarioName string
	Requirements []RequirementInfo
	ActionType   ActionType
}

// SpawnResult contains the result of spawning a requirements improve agent.
type SpawnResult struct {
	ImproveID string
	RunID     string
	Tag       string
	Status    Status
	Error     string
}

// Spawn creates a new improve agent for the specified scenario and requirements.
func (s *Service) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if agent-manager is available
	if !s.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available")
	}

	// Check if there's already an active improve for this scenario
	if active := s.store.GetActiveForScenario(req.ScenarioName); active != nil {
		return nil, fmt.Errorf("an improve operation is already in progress for scenario %s (id: %s)", req.ScenarioName, active.ID)
	}

	// Generate IDs
	improveID := uuid.New().String()[:8]
	tag := fmt.Sprintf("req-improve-%s-%s", req.ScenarioName, improveID)

	// Create improve record
	record := &Record{
		ID:           improveID,
		ScenarioName: req.ScenarioName,
		Requirements: req.Requirements,
		ActionType:   req.ActionType,
		Status:       StatusPending,
		Tag:          tag,
		StartedAt:    time.Now(),
	}
	s.store.Create(record)

	// Spawn the agent in a goroutine
	go s.runImprove(record)

	return &SpawnResult{
		ImproveID: improveID,
		Tag:       tag,
		Status:    StatusPending,
	}, nil
}

// runImprove executes the improve agent and updates the record with results.
func (s *Service) runImprove(record *Record) {
	ctx := context.Background()

	// Update status to running
	record.Status = StatusRunning
	s.store.Update(record)

	// Build the prompt
	scenarioPath := GetScenarioPath(record.ScenarioName)
	prompt := BuildPrompt(PromptConfig{
		ScenarioName: record.ScenarioName,
		ScenarioPath: scenarioPath,
		Requirements: record.Requirements,
		ActionType:   record.ActionType,
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

	// Add requirements summary as attachment
	var reqSummary string
	for _, req := range record.Requirements {
		reqSummary += fmt.Sprintf("- %s (%s): %s\n", req.ID, req.LiveStatus, req.Title)
	}
	attachments = append(attachments, &domainpb.ContextAttachment{
		Type:    "note",
		Key:     "requirements-to-improve",
		Label:   "Requirements to Improve",
		Content: reqSummary,
		Tags:    []string{"requirements", "improve"},
	})

	// Create task and run via SpawnSingle
	task := &domainpb.Task{
		Title:              fmt.Sprintf("Requirements Improve - %s", record.ScenarioName),
		Description:        prompt,
		ScopePath:          scenarioPath,
		ProjectRoot:        scenarioPath,
		CreatedBy:          "test-genie-requirements-improve",
		ContextAttachments: attachments,
	}

	result, err := s.agentSvc.SpawnSingle(ctx, agentmanager.SpawnSingleRequest{
		Task: task,
		Tag:  record.Tag,
	})
	if err != nil {
		s.failImprove(record, fmt.Sprintf("failed to spawn agent: %v", err))
		return
	}

	record.RunID = result.RunID
	s.store.Update(record)

	log.Printf("[requirements-improve] Started %s for scenario %s (runID: %s)", record.ID, record.ScenarioName, result.RunID)

	// Poll for completion
	s.pollForCompletion(ctx, record)
}

// pollForCompletion polls the agent run status until it reaches a terminal state.
func (s *Service) pollForCompletion(ctx context.Context, record *Record) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // 30 minute timeout for improve operations

	for {
		select {
		case <-ctx.Done():
			s.failImprove(record, "context cancelled")
			return
		case <-timeout:
			s.failImprove(record, "improve operation timed out")
			return
		case <-ticker.C:
			run, err := s.agentSvc.GetRun(ctx, record.RunID)
			if err != nil {
				log.Printf("[requirements-improve] Error getting run status for %s: %v", record.ID, err)
				continue
			}

			status := agentmanager.MapRunStatus(run.Status)

			switch status {
			case "completed":
				s.completeImprove(record, run)
				return
			case "failed":
				errorMsg := "agent run failed"
				if run.ErrorMsg != "" {
					errorMsg = run.ErrorMsg
				}
				s.failImprove(record, errorMsg)
				return
			case "stopped":
				record.Status = StatusCancelled
				now := time.Now()
				record.CompletedAt = &now
				s.store.Update(record)
				log.Printf("[requirements-improve] Improve %s was cancelled", record.ID)
				return
			}
		}
	}
}

// completeImprove marks an improve operation as completed and stores the output.
func (s *Service) completeImprove(record *Record, run *domainpb.Run) {
	record.Status = StatusCompleted
	now := time.Now()
	record.CompletedAt = &now

	if run.Summary != nil {
		record.Output = run.Summary.Description
	}

	s.store.Update(record)
	log.Printf("[requirements-improve] Improve %s completed successfully for scenario %s", record.ID, record.ScenarioName)
}

// failImprove marks an improve operation as failed with an error message.
func (s *Service) failImprove(record *Record, errorMsg string) {
	record.Status = StatusFailed
	record.Error = errorMsg
	now := time.Now()
	record.CompletedAt = &now
	s.store.Update(record)
	log.Printf("[requirements-improve] Improve %s failed for scenario %s: %s", record.ID, record.ScenarioName, errorMsg)
}

// Get retrieves an improve record by ID.
func (s *Service) Get(id string) (*Record, bool) {
	return s.store.Get(id)
}

// ListByScenario returns improve records for a scenario.
func (s *Service) ListByScenario(scenarioName string, limit int) []*Record {
	return s.store.ListByScenario(scenarioName, limit)
}

// GetActiveForScenario returns the active improve for a scenario, if any.
func (s *Service) GetActiveForScenario(scenarioName string) *Record {
	return s.store.GetActiveForScenario(scenarioName)
}

// Stop stops a running improve operation.
func (s *Service) Stop(ctx context.Context, id string) error {
	record, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("improve not found: %s", id)
	}

	if record.IsTerminal() {
		return fmt.Errorf("improve is already in terminal state: %s", record.Status)
	}

	if record.RunID == "" {
		return fmt.Errorf("improve has no associated run")
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
