package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/pathutil"
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

// ExecutionContext captures the finalized execution data needed to launch a
// review agent with the same context that automatic post-run checks use.
type ExecutionContext struct {
	BacklogKind            string
	BacklogName            string
	ItemTitle              string
	AffectedScenarios      []string
	ChangedPathsByScenario map[string][]string
	GCTResultsJSON         string
}

// ServiceConfig configures the review service dependencies.
type ServiceConfig struct {
	RootDir              string
	AgentService         AgentSpawner
	PromptClient         promptmanager.Client
	ItemDirFn            func(kind, name string) string
	LoadItemTitle        func(kind, name string) (string, error)
	LoadExecutionContext func(ctx context.Context, executionID string) (*ExecutionContext, error)
}

// activeRound tracks a gathering round so the poller knows which runs to check.
type activeRound struct {
	ItemDir  string
	RoundNum int
	RunID    string
}

// Service provides review evidence management for completed executions.
type Service struct {
	rootDir              string
	agentService         AgentSpawner
	inspector            RunInspector
	promptClient         promptmanager.Client
	eventLogger          EventLogger
	itemDirFn            func(kind, name string) string
	loadItemTitle        func(kind, name string) (string, error)
	loadExecutionContext func(ctx context.Context, executionID string) (*ExecutionContext, error)

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
		rootDir:              cfg.RootDir,
		agentService:         cfg.AgentService,
		promptClient:         pc,
		itemDirFn:            cfg.ItemDirFn,
		loadItemTitle:        cfg.LoadItemTitle,
		loadExecutionContext: cfg.LoadExecutionContext,
		activeRounds:         make(map[string]activeRound),
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

	itemTitle := s.resolveItemTitle(params.BacklogKind, params.BacklogName, params.ItemTitle)
	deliverableContent := loadReviewDeliverableContent(params.BacklogKind, params.ItemDir)

	// Build changed paths list.
	changedPaths := flattenChangedPaths(params.ChangedPathsByScenario)
	affectedScenarios := pathutil.UniqueSortedStrings(append([]string(nil), params.AffectedScenarios...))

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
	attachments := buildReviewAttachments(deliverableContent, changedPaths, affectedScenarios, params.GCTResultsJSON, "")

	// Create the round file in gathering state.
	round := Round{
		RoundNum:    roundNum,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ExecutionID: params.ExecutionID,
		Status:      RoundStatusGathering,
		Evidence:    []EvidenceItem{},
	}

	// Inject agent activity spec for the tracked agent service.
	ctx = agentactivity.WithSpec(ctx, agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   params.BacklogKind,
		OwnerName:   params.BacklogName,
		OwnerTitle:  itemTitle,
		ExecutionID: params.ExecutionID,
		Purpose:     agentactivity.PurposeReview,
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint":   "review.start",
			"round_number": fmt.Sprintf("%03d", roundNum),
		},
	})

	// Spawn the review agent with instructions as system prompt and data as context.
	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:               params.BacklogKind,
		Name:               params.BacklogName,
		Title:              "Review: " + itemTitle,
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
		if mapRunStatusToRoundStatus(state.Status) == "" {
			continue
		}
		*round = finalizeRoundFromRunState(*round, state)
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

			itemTitle := s.resolveItemTitle(kind, name, "")
			deliverableContent := loadReviewDeliverableContent(kind, itemDir)
			var changedPaths []string
			var affectedScenarios []string
			var gctResultsJSON string
			if round.ExecutionID != "" && s.loadExecutionContext != nil {
				if execCtx, ctxErr := s.loadExecutionContext(spawnCtx, round.ExecutionID); ctxErr != nil {
					slog.Warn("load execution review context for evidence request", "execution_id", round.ExecutionID, "error", ctxErr)
				} else if execCtx != nil {
					changedPaths = flattenChangedPaths(execCtx.ChangedPathsByScenario)
					affectedScenarios = pathutil.UniqueSortedStrings(append([]string(nil), execCtx.AffectedScenarios...))
					gctResultsJSON = execCtx.GCTResultsJSON
					itemTitle = s.resolveItemTitle(execCtx.BacklogKind, execCtx.BacklogName, execCtx.ItemTitle)
				}
			}

			// Inject agent activity spec for the tracked agent service.
			spawnCtx = agentactivity.WithSpec(spawnCtx, agentactivity.Spec{
				OwnerType:   agentactivity.OwnerBacklog,
				OwnerKind:   kind,
				OwnerName:   name,
				OwnerTitle:  itemTitle,
				ExecutionID: round.ExecutionID,
				Purpose:     agentactivity.PurposeReview,
				RequestedBy: "swarm-manager-ui",
				Metadata: map[string]string{
					"entrypoint": "review.request_more_evidence",
					"thread_id":  threadID,
				},
			})

			instructionVars := map[string]string{
				"ITEM_FOLDER":  itemDir,
				"ROUND_NUMBER": fmt.Sprintf("%03d", roundNum),
			}
			prompt, renderErr := s.promptClient.ReadSkill(spawnCtx, "swarm-manager-review", instructionVars, true)
			if renderErr != nil {
				slog.Error("render review skill for evidence request", "error", renderErr, "thread_id", threadID)
				return
			}

			reqAttachments := buildReviewAttachments(deliverableContent, changedPaths, affectedScenarios, gctResultsJSON, message)

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
				if errors.Is(spawnErr, agentactivity.ErrBacklogItemBusy) {
					slog.Info("evidence request skipped: agent already active", "kind", kind, "name", name, "thread_id", threadID)
				} else {
					slog.Error("spawn review agent for evidence request", "error", spawnErr, "thread_id", threadID)
				}
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
func (s *Service) TriggerReviewAgent(ctx context.Context, executionID string) error {
	params, err := s.buildStartReviewParamsFromExecution(ctx, executionID)
	if err != nil {
		return err
	}
	return s.startReview(ctx, params)
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
func buildReviewAttachments(deliverableContent string, changedPaths, affectedScenarios []string, gctResultsJSON, userRequest string) []*domainpb.ContextAttachment {
	var atts []*domainpb.ContextAttachment

	atts = appendNoteAttachment(atts, "plan-content", "Deliverable Content", "Backlog deliverable (plan or conclusion)", deliverableContent, "markdown", "high")

	diffContent := fmt.Sprintf("Changed %d files across %d scenarios", len(changedPaths), len(affectedScenarios))
	if len(changedPaths) == 0 {
		diffContent += "\n\nNote: Zero tracked changes may indicate the execution agent ran without sandbox mode enabled. " +
			"In non-sandbox mode, changes are applied directly to the working tree and are not captured as a diff. " +
			"You should still verify the implementation by examining the codebase directly."
	}
	atts = appendNoteAttachment(atts, "diff-summary", "Diff Summary", "", diffContent, "text", "high")

	if len(changedPaths) > 0 {
		atts = appendNoteAttachment(atts, "changed-paths", "Changed File Paths", fmt.Sprintf("%d changed files", len(changedPaths)), strings.Join(changedPaths, "\n"), "text", "high")
	}

	if len(affectedScenarios) > 0 {
		atts = appendNoteAttachment(atts, "affected-scenarios", "Affected Scenarios", fmt.Sprintf("%d scenarios affected", len(affectedScenarios)), strings.Join(affectedScenarios, "\n"), "text", "medium")
	}

	if gctResultsJSON != "" {
		atts = appendNoteAttachment(atts, "gct-review-results", "GCT Review Results", "Automated review metrics per scenario", gctResultsJSON, "json", "high")
	}

	if userRequest != "" {
		atts = appendNoteAttachment(atts, "user-request", "User Evidence Request", "Specific evidence request from human reviewer", userRequest, "text", "high")
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

		if mapRunStatusToRoundStatus(state.Status) == "" {
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

		*round = finalizeRoundFromRunState(*round, state)
		if saveErr := SaveRound(ar.ItemDir, *round); saveErr != nil {
			slog.Error("update review round status", "round", ar.RoundNum, "run_id", runID, "error", saveErr)
			continue
		}

		slog.Info("review round status updated", "round", ar.RoundNum, "run_id", runID, "status", round.Status)

		if s.eventLogger != nil && round.ExecutionID != "" {
			if round.Status == RoundStatusComplete {
				started, _ := time.Parse(time.RFC3339, round.GeneratedAt)
				duration := time.Since(started).Seconds()
				s.eventLogger.EmitReviewRoundCompleted(round.ExecutionID, round.RoundNum, len(round.Evidence), round.Classification, duration)
			} else if round.Status == RoundStatusFailed {
				s.eventLogger.EmitReviewFailed(round.ExecutionID, round.FailureReason, 0)
			}
		}

		s.mu.Lock()
		delete(s.activeRounds, runID)
		s.mu.Unlock()
	}
}

func (s *Service) buildStartReviewParamsFromExecution(ctx context.Context, executionID string) (startReviewParams, error) {
	if s.loadExecutionContext == nil {
		return startReviewParams{}, fmt.Errorf("execution context loader not configured")
	}

	execCtx, err := s.loadExecutionContext(ctx, executionID)
	if err != nil {
		return startReviewParams{}, fmt.Errorf("load execution review context: %w", err)
	}
	if execCtx == nil {
		return startReviewParams{}, fmt.Errorf("execution %s has no review context", executionID)
	}
	if strings.TrimSpace(execCtx.BacklogKind) == "" || strings.TrimSpace(execCtx.BacklogName) == "" {
		return startReviewParams{}, fmt.Errorf("execution %s is missing backlog identity", executionID)
	}

	return startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            execCtx.BacklogKind,
		BacklogName:            execCtx.BacklogName,
		ItemTitle:              execCtx.ItemTitle,
		ItemDir:                s.resolveItemDir(execCtx.BacklogKind, execCtx.BacklogName),
		AffectedScenarios:      append([]string(nil), execCtx.AffectedScenarios...),
		ChangedPathsByScenario: cloneChangedPaths(execCtx.ChangedPathsByScenario),
		GCTResultsJSON:         execCtx.GCTResultsJSON,
	}, nil
}

func (s *Service) resolveItemTitle(kind, name, fallback string) string {
	if title := strings.TrimSpace(fallback); title != "" {
		return title
	}
	if s.loadItemTitle != nil {
		if title, err := s.loadItemTitle(kind, name); err == nil {
			if trimmed := strings.TrimSpace(title); trimmed != "" {
				return trimmed
			}
		}
	}
	return name
}

func loadReviewDeliverableContent(kind, itemDir string) string {
	deliverable := backlog.DeliverableForKind(backlog.BacklogKind(strings.TrimSpace(kind)))
	return workshop.LoadPlanContentByName(itemDir, deliverable)
}

func flattenChangedPaths(changedPathsByScenario map[string][]string) []string {
	if len(changedPathsByScenario) == 0 {
		return nil
	}
	paths := make([]string, 0)
	for _, scenarioPaths := range changedPathsByScenario {
		paths = append(paths, scenarioPaths...)
	}
	return pathutil.UniqueSortedStrings(paths)
}

func cloneChangedPaths(changedPathsByScenario map[string][]string) map[string][]string {
	if len(changedPathsByScenario) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(changedPathsByScenario))
	for scenarioName, paths := range changedPathsByScenario {
		cloned[scenarioName] = append([]string(nil), paths...)
		sort.Strings(cloned[scenarioName])
		cloned[scenarioName] = pathutil.UniqueSortedStrings(cloned[scenarioName])
	}
	return cloned
}

func appendNoteAttachment(
	atts []*domainpb.ContextAttachment,
	key, label, summary, content, format, priority string,
) []*domainpb.ContextAttachment {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return atts
	}
	return append(atts, &domainpb.ContextAttachment{
		Type:     "note",
		Key:      key,
		Label:    label,
		Summary:  summary,
		Content:  trimmed,
		Format:   format,
		Priority: priority,
	})
}

func MarshalScenarioGCTResults(results map[string]any) string {
	if len(results) == 0 {
		return ""
	}
	data, err := json.Marshal(results)
	if err != nil {
		return ""
	}
	return string(data)
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
