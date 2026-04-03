package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/workshop"
)

// AgentSpawner spawns agent-manager sessions.
type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

// EventLogger records review events for analytics.
type EventLogger interface {
	EmitReviewStarted(executionID string, roundNumber int)
	EmitReviewEvidenceVerified(executionID, evidenceID string)
	EmitReviewRequestCreated(executionID, requestID, description string)
	EmitReviewRoundCompleted(executionID string, roundNumber, evidenceCount int, classification string, durationSecs float64)
	EmitReviewFailed(executionID, reason string, durationSecs float64)
}

// BacklogItemDirResolver resolves the on-disk directory for a backlog item.
type BacklogItemDirResolver interface {
	ItemDir(kind, name string) string
}

// ServiceConfig configures the review service dependencies.
type ServiceConfig struct {
	RootDir      string
	AgentService AgentSpawner
	PromptClient promptmanager.Client
	ItemDirFn    func(kind, name string) string
}

// Service provides review evidence management for completed executions.
type Service struct {
	rootDir      string
	agentService AgentSpawner
	promptClient promptmanager.Client
	eventLogger  EventLogger
	itemDirFn    func(kind, name string) string
}

// NewService creates a new review service.
func NewService(cfg ServiceConfig) *Service {
	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	return &Service{
		rootDir:      cfg.RootDir,
		agentService: cfg.AgentService,
		promptClient: pc,
		itemDirFn:    cfg.ItemDirFn,
	}
}

// SetEventLogger injects an optional event logger for analytics.
func (s *Service) SetEventLogger(e EventLogger) {
	s.eventLogger = e
}

// StartReviewForExecution is called by the execution service during finalization
// to spawn a review agent. It satisfies execution.ReviewServiceIntegration.
func (s *Service) StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDir string, affectedScenarios []string, changedPathsByScenario map[string][]string) error {
	return s.startReview(ctx, startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            backlogKind,
		BacklogName:            backlogName,
		ItemTitle:              itemTitle,
		ItemDir:                itemDir,
		AffectedScenarios:      affectedScenarios,
		ChangedPathsByScenario: changedPathsByScenario,
	})
}

// startReviewParams packages the data needed to begin evidence gathering.
type startReviewParams struct {
	ExecutionID            string
	BacklogKind            string
	BacklogName            string
	ItemTitle              string
	ItemDir                string
	AffectedScenarios      []string
	ChangedPathsByScenario map[string][]string
}

// startReview creates a review round and spawns the review agent.
func (s *Service) startReview(ctx context.Context, params startReviewParams) error {
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return fmt.Errorf("agent service not available")
	}

	roundNum, err := NextRoundNumber(params.ItemDir)
	if err != nil {
		return fmt.Errorf("determine next round: %w", err)
	}

	// Load plan content for the prompt.
	planContent := workshop.LoadPlanContent(params.ItemDir)

	// Build changed paths summary.
	var changedPaths []string
	for _, paths := range params.ChangedPathsByScenario {
		changedPaths = append(changedPaths, paths...)
	}

	// Build the prompt via prompt-manager skill.
	variables := map[string]string{
		"ITEM_FOLDER":        params.ItemDir,
		"PLAN_CONTENT":       planContent,
		"DIFF_SUMMARY":       fmt.Sprintf("Changed %d files across %d scenarios", len(changedPaths), len(params.AffectedScenarios)),
		"CHANGED_PATHS":      strings.Join(changedPaths, ", "),
		"AFFECTED_SCENARIOS": strings.Join(params.AffectedScenarios, ", "),
		"ROUND_NUMBER":       fmt.Sprintf("%03d", roundNum),
	}

	promptContent, err := s.promptClient.ReadSkill(ctx, "swarm-manager-review", variables, true)
	if err != nil {
		return fmt.Errorf("render review skill: %w", err)
	}

	// Create the round file in gathering state.
	round := Round{
		RoundNum:    roundNum,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ExecutionID: params.ExecutionID,
		Status:      RoundStatusGathering,
		Evidence:    []EvidenceItem{},
	}

	// Spawn the review agent.
	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        params.BacklogKind,
		Name:        params.BacklogName,
		Title:       "Review: " + params.ItemTitle,
		Description: promptContent,
		Prompt:      promptContent,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager:review",
		Purpose:     "review",
		Environment: map[string]string{
			"VROOLI_SPAWN_SOURCE": "swarm-manager-review",
		},
	})
	if err != nil {
		return fmt.Errorf("spawn review agent: %w", err)
	}

	round.RunID = runResult.RunID
	if err := SaveRound(params.ItemDir, round); err != nil {
		return fmt.Errorf("save review round: %w", err)
	}

	if s.eventLogger != nil {
		s.eventLogger.EmitReviewStarted(params.ExecutionID, roundNum)
	}

	slog.Info("review round started", "round", roundNum, "execution_id", params.ExecutionID, "run_id", runResult.RunID)
	return nil
}

// ListRounds returns all review evidence rounds for a backlog item.
func (s *Service) ListRounds(kind, name string) ([]Round, error) {
	itemDir := s.resolveItemDir(kind, name)
	return LoadRounds(itemDir)
}

// GetRound returns a specific review round by number.
func (s *Service) GetRound(kind, name string, roundNum int) (*Round, error) {
	itemDir := s.resolveItemDir(kind, name)
	return LoadRound(itemDir, roundNum)
}

// VerifyEvidence toggles the verified flag on an evidence item.
func (s *Service) VerifyEvidence(kind, name string, roundNum int, evidenceID string, verified bool, executionID string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.Evidence {
		if round.Evidence[i].ID == evidenceID {
			round.Evidence[i].Verified = verified
			if verified {
				round.Evidence[i].VerifiedAt = time.Now().UTC().Format(time.RFC3339)
			} else {
				round.Evidence[i].VerifiedAt = ""
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("evidence item %s not found in round %d", evidenceID, roundNum)
	}

	if err := SaveRound(itemDir, *round); err != nil {
		return fmt.Errorf("save round: %w", err)
	}

	if s.eventLogger != nil && executionID != "" {
		s.eventLogger.EmitReviewEvidenceVerified(executionID, evidenceID)
	}
	return nil
}

// RequestMoreEvidence creates a new request thread and optionally spawns a
// targeted review agent run.
func (s *Service) RequestMoreEvidence(ctx context.Context, kind, name string, roundNum int, message string, evidenceID string) (string, error) {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return "", fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return "", fmt.Errorf("round %d not found", roundNum)
	}

	threadID := "rt-" + idgen.Generate()
	thread := RequestThread{
		ID:         threadID,
		EvidenceID: evidenceID,
		Status:     "pending",
		Messages: []RequestMessage{
			{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	round.RequestThreads = append(round.RequestThreads, thread)
	if err := SaveRound(itemDir, *round); err != nil {
		return "", fmt.Errorf("save round: %w", err)
	}

	if s.eventLogger != nil && round.ExecutionID != "" {
		s.eventLogger.EmitReviewRequestCreated(round.ExecutionID, threadID, message)
	}

	return threadID, nil
}

// ContinueRequest appends a user message to an existing request thread.
func (s *Service) ContinueRequest(kind, name string, roundNum int, threadID, message string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.RequestThreads {
		if round.RequestThreads[i].ID == threadID {
			round.RequestThreads[i].Messages = append(round.RequestThreads[i].Messages, RequestMessage{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("thread %s not found in round %d", threadID, roundNum)
	}

	return SaveRound(itemDir, *round)
}

// DismissRequest marks a request thread as dismissed.
func (s *Service) DismissRequest(kind, name string, roundNum int, threadID string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.RequestThreads {
		if round.RequestThreads[i].ID == threadID {
			round.RequestThreads[i].Status = "dismissed"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("thread %s not found in round %d", threadID, roundNum)
	}

	return SaveRound(itemDir, *round)
}

// TriggerReviewAgent manually triggers a review agent for an execution.
// Used when the user wants to re-run or initiate evidence gathering.
func (s *Service) TriggerReviewAgent(ctx context.Context, kind, name, executionID string, affectedScenarios []string) error {
	itemDir := s.resolveItemDir(kind, name)
	return s.startReview(ctx, startReviewParams{
		ExecutionID:       executionID,
		BacklogKind:       kind,
		BacklogName:       name,
		ItemTitle:         name,
		ItemDir:           itemDir,
		AffectedScenarios: affectedScenarios,
	})
}

func (s *Service) resolveItemDir(kind, name string) string {
	if s.itemDirFn != nil {
		return s.itemDirFn(kind, name)
	}
	// Fallback: construct from root dir (mirrors backlog store pattern).
	kindDir := kind + "s"
	if kind == "fix" || kind == "research" {
		kindDir = kind
	}
	return s.rootDir + "/" + kindDir + "/" + name
}
