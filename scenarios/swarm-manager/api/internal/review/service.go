package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
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

type PlanContentResolver func(ctx context.Context, kind, name, itemDir string) (string, error)

// ExecutionContext captures the finalized execution data needed to launch a
// review agent with the same context that automatic post-run checks use.
type ExecutionContext struct {
	BacklogKind            string
	BacklogName            string
	ItemTitle              string
	AffectedScenarios      []string
	ChangedPathsByScenario map[string][]string
	GCTResultsJSON         string
	// BaselineDiffJSON is the per-scenario before/after baseline diff (new vs
	// pre-existing failures) serialized as JSON, empty when no baseline was
	// captured. Lets the review agent prioritize regressions this item caused.
	BaselineDiffJSON string
}

// RoundTerminalHandler is invoked when a review round transitions to a
// terminal status (complete or failed). Implementations typically flip the
// backlog item's status from `in_review` to `review_pending` so the user can
// assess the review output and decide a terminal state via review-decide.
//
// Called synchronously after the round file is saved. Errors are logged but
// do not block review processing.
type RoundTerminalHandler func(ctx context.Context, kind, name string, round Round)

// ServiceConfig configures the review service dependencies.
type ServiceConfig struct {
	DataRoot             string
	AgentService         AgentSpawner
	PromptClient         promptmanager.Client
	ItemDirFn            func(kind, name string) string
	LoadItemTitle        func(kind, name string) (string, error)
	LoadExecutionContext func(ctx context.Context, executionID string) (*ExecutionContext, error)
	PlanContentResolver  PlanContentResolver
	// OnRoundTerminal fires when a review round transitions to complete/failed.
	// Used to flip the backlog item's status to review_pending.
	OnRoundTerminal RoundTerminalHandler
	// RoundMaxAge bounds how long a round may sit in `gathering` before the
	// poller treats its run as abandoned and finalizes it as failed (which
	// fires OnRoundTerminal so the item leaves in_review). Zero uses
	// DefaultRoundMaxAge.
	RoundMaxAge time.Duration
}

// DefaultRoundMaxAge is the wall-clock age past which a still-gathering review
// round is treated as abandoned (its run died without agent-manager reporting
// a terminal state). Chosen well above any healthy review run so a slow review
// isn't interrupted, but short enough that a crashed run can't strand an item
// in in_review for hours. Mirrors feedback.DefaultStuckMaxAge.
const DefaultRoundMaxAge = 30 * time.Minute

// Service provides review evidence management for completed executions.
type Service struct {
	dataRoot             string
	agentService         AgentSpawner
	inspector            RunInspector
	promptClient         promptmanager.Client
	eventLogger          EventLogger
	itemDirFn            func(kind, name string) string
	loadItemTitle        func(kind, name string) (string, error)
	loadExecutionContext func(ctx context.Context, executionID string) (*ExecutionContext, error)
	planContentResolver  PlanContentResolver
	onRoundTerminal      RoundTerminalHandler
	roundMaxAge          time.Duration
	clock                func() time.Time

	mu           sync.Mutex
	activeRounds map[string]activeRound // keyed by RunID
}

// NewService creates a new review service.
func NewService(cfg ServiceConfig) *Service {
	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	roundMaxAge := cfg.RoundMaxAge
	if roundMaxAge <= 0 {
		roundMaxAge = envDuration("SWARM_MANAGER_REVIEW_ROUND_MAX_AGE", DefaultRoundMaxAge)
	}
	svc := &Service{
		dataRoot:             cfg.DataRoot,
		agentService:         cfg.AgentService,
		promptClient:         pc,
		itemDirFn:            cfg.ItemDirFn,
		loadItemTitle:        cfg.LoadItemTitle,
		loadExecutionContext: cfg.LoadExecutionContext,
		planContentResolver:  cfg.PlanContentResolver,
		onRoundTerminal:      cfg.OnRoundTerminal,
		roundMaxAge:          roundMaxAge,
		clock:                time.Now,
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
func (s *Service) StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDir string, affectedScenarios []string, changedPathsByScenario map[string][]string, gctResultsJSON, baselineDiffJSON string) error {
	return s.startReview(ctx, startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            backlogKind,
		BacklogName:            backlogName,
		ItemTitle:              itemTitle,
		ItemDir:                itemDir,
		AffectedScenarios:      affectedScenarios,
		ChangedPathsByScenario: changedPathsByScenario,
		GCTResultsJSON:         gctResultsJSON,
		BaselineDiffJSON:       baselineDiffJSON,
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
	BaselineDiffJSON       string // Pre-serialized before/after baseline diff per scenario
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
	deliverableContent := s.loadReviewDeliverableContent(ctx, params.BacklogKind, params.BacklogName, params.ItemDir)

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
	attachments := buildReviewAttachments(deliverableContent, changedPaths, affectedScenarios, params.GCTResultsJSON, params.BaselineDiffJSON, "")

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
		PhaseKind:   string(agentactivity.LaneReview),
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
	s.trackActiveRound(runResult.RunID, params.BacklogKind, params.BacklogName, params.ItemDir, roundNum)

	if s.eventLogger != nil {
		s.eventLogger.EmitReviewStarted(params.ExecutionID, roundNum)
	}

	slog.Info("review round started", "round", roundNum, "execution_id", params.ExecutionID, "run_id", runResult.RunID)
	return nil
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
	return s.dataRoot + "/" + kindDir + "/" + name
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
		BaselineDiffJSON:       execCtx.BaselineDiffJSON,
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
