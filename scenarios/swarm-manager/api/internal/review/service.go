package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
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
	operationStarter     OperationStarter
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

// startReview creates a review round and starts the review-round operation
// through the operation runner. The runner resolves the bound backlog-review mode
// and spawns the agent through the operating-mode engine's chokepoint; the round's
// completion arrives through the runner completion bridge, which fires the
// commit-review-round handler (see opshandlers.go). The round is written BEFORE
// the start (commit-before-async) so a start failure leaves a resolvable record;
// its run id is stamped after start so the completion handler correlates back to
// it. The round is NOT tracked for legacy polling — it is runner-owned, and the
// completion bridge + refresh driver own its terminal transition (see the
// OpExecutionID guard in polling.go).
func (s *Service) startReview(ctx context.Context, params startReviewParams) error {
	if s.operationStarter == nil {
		return fmt.Errorf("review operation runner not available")
	}

	roundNum, err := NextRoundNumber(params.ItemDir)
	if err != nil {
		return fmt.Errorf("determine next round: %w", err)
	}

	// Create the round file in gathering state before the async start.
	round := Round{
		RoundNum:    roundNum,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ExecutionID: params.ExecutionID,
		Status:      RoundStatusGathering,
		Evidence:    []EvidenceItem{},
	}
	if err := SaveRound(params.ItemDir, round); err != nil {
		return fmt.Errorf("save review round: %w", err)
	}

	res, err := s.operationStarter.StartReviewOperation(ctx, OperationStartRequest{
		Operation:      opReviewRound,
		TargetKind:     targetKindBacklogItem,
		TargetID:       params.BacklogKind + "/" + params.BacklogName,
		IdempotencyKey: fmt.Sprintf("review-%s-r%d", params.ExecutionID, roundNum),
		RequestedBy:    "swarm-manager",
	})
	if err != nil {
		return fmt.Errorf("start review operation: %w", err)
	}

	// Stamp the run association + operation execution refs on the round so the
	// completion handler correlates back and the poller defers to the runner.
	round.RunID = res.RunID
	round.OpWorkflowID = res.WorkflowID
	round.OpExecutionID = res.ExecutionID
	if err := SaveRound(params.ItemDir, round); err != nil {
		return fmt.Errorf("save review round run association: %w", err)
	}

	if s.eventLogger != nil {
		s.eventLogger.EmitReviewStarted(params.ExecutionID, roundNum)
	}

	slog.Info("review round started", "round", roundNum, "execution_id", params.ExecutionID,
		"run_id", res.RunID, "op_execution", res.ExecutionID)
	return nil
}

// Operation + target identifiers the review reroute pins. The operation version
// pins the exact contract version the system binding pins.
const (
	opReviewRound         = "review-round"
	opEvidenceRequest     = "evidence-request"
	targetKindBacklogItem = "backlog-item"
)

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
