package review

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/workshop"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// AgentSpawner spawns agent-manager sessions.
type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

// RunInspector retrieves the current state of an agent run.
type RunInspector interface {
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
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

// activeRound tracks a gathering round so the poller knows which runs to check.
type activeRound struct {
	ItemDir  string
	RoundNum int
	RunID    string
}

// Service provides review evidence management for completed executions.
type Service struct {
	rootDir      string
	agentService AgentSpawner
	inspector    RunInspector
	promptClient promptmanager.Client
	eventLogger  EventLogger
	itemDirFn    func(kind, name string) string

	mu           sync.Mutex
	activeRounds map[string]activeRound // keyed by RunID
}

// NewService creates a new review service.
func NewService(cfg ServiceConfig) *Service {
	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	svc := &Service{
		rootDir:      cfg.RootDir,
		agentService: cfg.AgentService,
		promptClient: pc,
		itemDirFn:    cfg.ItemDirFn,
		activeRounds: make(map[string]activeRound),
	}
	// Type-assert for RunInspector capability (matches execution pattern).
	if inspector, ok := cfg.AgentService.(RunInspector); ok {
		svc.inspector = inspector
	}
	return svc
}

// SetEventLogger injects an optional event logger for analytics.
func (s *Service) SetEventLogger(e EventLogger) {
	s.eventLogger = e
}

// StartReviewForExecution is called by the execution service during finalization
// to spawn a review agent. It satisfies execution.ReviewServiceIntegration.
func (s *Service) StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDir string, affectedScenarios []string, changedPathsByScenario map[string][]string, gctResultsJSON string) error {
	return s.startReview(ctx, startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            backlogKind,
		BacklogName:            backlogName,
		ItemTitle:              itemTitle,
		ItemDir:                itemDir,
		AffectedScenarios:      affectedScenarios,
		ChangedPathsByScenario: changedPathsByScenario,
		GCTResultsJSON:         gctResultsJSON,
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
	GCTResultsJSON         string // Pre-serialized GCT review results per scenario
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

	// Load plan content for context.
	planContent := workshop.LoadPlanContent(params.ItemDir)

	// Build changed paths list.
	var changedPaths []string
	for _, paths := range params.ChangedPathsByScenario {
		changedPaths = append(changedPaths, paths...)
	}

	// Render instructions with structural variables only.
	instructionVars := map[string]string{
		"ITEM_FOLDER":  params.ItemDir,
		"ROUND_NUMBER": fmt.Sprintf("%03d", roundNum),
	}
	instructions, err := s.promptClient.ReadSkill(ctx, "swarm-manager-review", instructionVars, true)
	if err != nil {
		return fmt.Errorf("render review skill: %w", err)
	}

	// Build context attachments for data the agent needs to review.
	attachments := buildReviewAttachments(planContent, changedPaths, params.AffectedScenarios, params.GCTResultsJSON, "")

	// Create the round file in gathering state.
	round := Round{
		RoundNum:    roundNum,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ExecutionID: params.ExecutionID,
		Status:      RoundStatusGathering,
		Evidence:    []EvidenceItem{},
	}

	// Spawn the review agent with instructions as system prompt and data as context.
	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:               params.BacklogKind,
		Name:               params.BacklogName,
		Title:              "Review: " + params.ItemTitle,
		Description:        instructions,
		Prompt:             instructions,
		ScopePath:          ".",
		ProjectRoot:        ".",
		CreatedBy:          "swarm-manager:review",
		Purpose:            "review",
		ContextAttachments: attachments,
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

	// Track the gathering round for polling.
	s.trackActiveRound(runResult.RunID, params.ItemDir, roundNum)

	if s.eventLogger != nil {
		s.eventLogger.EmitReviewStarted(params.ExecutionID, roundNum)
	}

	slog.Info("review round started", "round", roundNum, "execution_id", params.ExecutionID, "run_id", runResult.RunID)
	return nil
}

// ListRounds returns all review evidence rounds for a backlog item.
// For any round still in gathering state, it checks the agent-manager run
// state inline and updates the round if the run has completed.
func (s *Service) ListRounds(kind, name string) ([]Round, error) {
	itemDir := s.resolveItemDir(kind, name)
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}

	if s.inspector == nil {
		return rounds, nil
	}

	for i := range rounds {
		round := &rounds[i]
		if round.Status != RoundStatusGathering || round.RunID == "" {
			continue
		}
		state, stateErr := s.inspector.GetRunState(context.Background(), round.RunID)
		if stateErr != nil {
			continue
		}
		nextStatus := mapRunStatusToRoundStatus(state.Status)
		if nextStatus == "" {
			continue
		}
		round.Status = nextStatus
		_ = SaveRound(itemDir, *round)

		// Remove from active tracking if present.
		s.mu.Lock()
		delete(s.activeRounds, round.RunID)
		s.mu.Unlock()
	}

	return rounds, nil
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

	// Spawn a targeted review agent with the user's request.
	if s.agentService != nil && s.agentService.IsEnabled() {
		go func() {
			spawnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Inject agent activity spec for the tracked agent service.
			spawnCtx = agentactivity.WithSpec(spawnCtx, agentactivity.Spec{
				OwnerType:   agentactivity.OwnerBacklog,
				OwnerKind:   kind,
				OwnerName:   name,
				ExecutionID: round.ExecutionID,
				Purpose:     agentactivity.PurposeReview,
				RequestedBy: "swarm-manager-ui",
				Metadata: map[string]string{
					"entrypoint": "review.request_more_evidence",
					"thread_id":  threadID,
				},
			})

			planContent := workshop.LoadPlanContent(itemDir)

			instructionVars := map[string]string{
				"ITEM_FOLDER":  itemDir,
				"ROUND_NUMBER": fmt.Sprintf("%03d", roundNum),
			}
			prompt, renderErr := s.promptClient.ReadSkill(spawnCtx, "swarm-manager-review", instructionVars, true)
			if renderErr != nil {
				slog.Error("render review skill for evidence request", "error", renderErr, "thread_id", threadID)
				return
			}

			reqAttachments := buildReviewAttachments(planContent, nil, nil, "", message)

			titlePreview := message
			if len(titlePreview) > 50 {
				titlePreview = titlePreview[:50]
			}

			result, spawnErr := s.agentService.SpawnBacklog(spawnCtx, agentmanager.BacklogSpawnRequest{
				Kind:               kind,
				Name:               name,
				Title:              "Evidence Request: " + titlePreview,
				Description:        prompt,
				Prompt:             prompt,
				ScopePath:          ".",
				ProjectRoot:        ".",
				CreatedBy:          "swarm-manager:review-request",
				Purpose:            "review",
				ContextAttachments: reqAttachments,
				Environment: map[string]string{
					"VROOLI_SPAWN_SOURCE": "swarm-manager-review-request",
				},
			})
			if spawnErr != nil {
				slog.Error("spawn review agent for evidence request", "error", spawnErr, "thread_id", threadID)
				return
			}

			// Update thread with agent RunID.
			r, loadErr := LoadRound(itemDir, roundNum)
			if loadErr != nil || r == nil {
				return
			}
			for i := range r.RequestThreads {
				if r.RequestThreads[i].ID == threadID {
					r.RequestThreads[i].RunID = result.RunID
					break
				}
			}
			_ = SaveRound(itemDir, *r)
			slog.Info("review evidence request agent spawned", "thread_id", threadID, "run_id", result.RunID)
		}()
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

// buildReviewAttachments creates structured context attachments for the review
// agent so that BuildSplitPrompt can separate instructions (system prompt)
// from data (user message).
func buildReviewAttachments(planContent string, changedPaths, affectedScenarios []string, gctResultsJSON, userRequest string) []*domainpb.ContextAttachment {
	var atts []*domainpb.ContextAttachment

	atts = append(atts, &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "plan-content",
		Label:    "Plan Content",
		Summary:  "Backlog item plan/spec",
		Content:  planContent,
		Format:   "markdown",
		Priority: "high",
	})

	diffContent := fmt.Sprintf("Changed %d files across %d scenarios", len(changedPaths), len(affectedScenarios))
	if len(changedPaths) == 0 {
		diffContent += "\n\nNote: Zero tracked changes may indicate the execution agent ran without sandbox mode enabled. " +
			"In non-sandbox mode, changes are applied directly to the working tree and are not captured as a diff. " +
			"You should still verify the implementation by examining the codebase directly."
	}
	atts = append(atts, &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "diff-summary",
		Label:    "Diff Summary",
		Content:  diffContent,
		Format:   "text",
		Priority: "high",
	})

	if len(changedPaths) > 0 {
		atts = append(atts, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "changed-paths",
			Label:    "Changed File Paths",
			Summary:  fmt.Sprintf("%d changed files", len(changedPaths)),
			Content:  strings.Join(changedPaths, "\n"),
			Format:   "text",
			Priority: "high",
		})
	}

	if len(affectedScenarios) > 0 {
		atts = append(atts, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "affected-scenarios",
			Label:    "Affected Scenarios",
			Summary:  fmt.Sprintf("%d scenarios affected", len(affectedScenarios)),
			Content:  strings.Join(affectedScenarios, "\n"),
			Format:   "text",
			Priority: "medium",
		})
	}

	if gctResultsJSON != "" {
		atts = append(atts, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "gct-review-results",
			Label:    "GCT Review Results",
			Summary:  "Automated review metrics per scenario",
			Content:  gctResultsJSON,
			Format:   "json",
			Priority: "high",
		})
	}

	if userRequest != "" {
		atts = append(atts, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "user-request",
			Label:    "User Evidence Request",
			Summary:  "Specific evidence request from human reviewer",
			Content:  userRequest,
			Format:   "text",
			Priority: "high",
		})
	}

	return atts
}

// -----------------------------------------------------------------------------
// Active round tracking & polling
// -----------------------------------------------------------------------------

func (s *Service) trackActiveRound(runID, itemDir string, roundNum int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRounds[runID] = activeRound{
		ItemDir:  itemDir,
		RoundNum: roundNum,
		RunID:    runID,
	}
}

// RefreshGatheringRounds polls agent-manager for the status of all tracked
// gathering rounds and updates their on-disk status when the run completes.
func (s *Service) RefreshGatheringRounds(ctx context.Context) {
	if s.inspector == nil {
		return
	}

	s.mu.Lock()
	snapshot := make(map[string]activeRound, len(s.activeRounds))
	for k, v := range s.activeRounds {
		snapshot[k] = v
	}
	s.mu.Unlock()

	for runID, ar := range snapshot {
		state, err := s.inspector.GetRunState(ctx, runID)
		if err != nil {
			continue // transient error, retry next tick
		}

		nextStatus := mapRunStatusToRoundStatus(state.Status)
		if nextStatus == "" {
			continue // still running
		}

		round, loadErr := LoadRound(ar.ItemDir, ar.RoundNum)
		if loadErr != nil || round == nil {
			s.mu.Lock()
			delete(s.activeRounds, runID)
			s.mu.Unlock()
			continue
		}

		if round.Status == RoundStatusComplete || round.Status == RoundStatusFailed {
			// Already terminal (e.g. agent wrote the round file itself).
			s.mu.Lock()
			delete(s.activeRounds, runID)
			s.mu.Unlock()
			continue
		}

		round.Status = nextStatus
		if saveErr := SaveRound(ar.ItemDir, *round); saveErr != nil {
			slog.Error("update review round status", "round", ar.RoundNum, "run_id", runID, "error", saveErr)
			continue
		}

		slog.Info("review round status updated", "round", ar.RoundNum, "run_id", runID, "status", nextStatus)

		if s.eventLogger != nil && round.ExecutionID != "" {
			if nextStatus == RoundStatusComplete {
				started, _ := time.Parse(time.RFC3339, round.GeneratedAt)
				duration := time.Since(started).Seconds()
				s.eventLogger.EmitReviewRoundCompleted(round.ExecutionID, round.RoundNum, len(round.Evidence), round.Classification, duration)
			} else if nextStatus == RoundStatusFailed {
				s.eventLogger.EmitReviewFailed(round.ExecutionID, "agent run failed", 0)
			}
		}

		s.mu.Lock()
		delete(s.activeRounds, runID)
		s.mu.Unlock()
	}
}

// mapRunStatusToRoundStatus converts an agent-manager run status to a terminal
// round status. Returns "" if the run is still in progress.
func mapRunStatusToRoundStatus(status string) RoundStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete":
		return RoundStatusComplete
	case "failed":
		return RoundStatusFailed
	case "cancelled":
		return RoundStatusFailed
	default:
		return "" // still in progress
	}
}

// StartBackgroundWorker polls gathering rounds on a 5-second interval until
// the stop channel is closed.
func (s *Service) StartBackgroundWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.RefreshGatheringRounds(context.Background())
		}
	}
}

// RecoverActiveRounds scans all backlog items for rounds in gathering state
// and re-populates the in-memory tracking map. Call this at startup so that
// rounds spawned before a restart are still polled.
func (s *Service) RecoverActiveRounds() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, kindDir := range []string{"ideas", "research", "fix", "execute", "chore"} {
		baseDir := s.rootDir + "/" + kindDir
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			itemDir := baseDir + "/" + entry.Name()
			rounds, loadErr := LoadRounds(itemDir)
			if loadErr != nil {
				continue
			}
			for _, round := range rounds {
				if round.Status == RoundStatusGathering && round.RunID != "" {
					s.activeRounds[round.RunID] = activeRound{
						ItemDir:  itemDir,
						RoundNum: round.RoundNum,
						RunID:    round.RunID,
					}
				}
			}
		}
	}

	if len(s.activeRounds) > 0 {
		slog.Info("recovered active review rounds", "count", len(s.activeRounds))
	}
}
