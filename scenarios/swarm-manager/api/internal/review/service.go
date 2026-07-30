package review

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/transitionrunner"
)

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

// ReviewContract is the immutable, item-authored material the reviewer must
// assess. It deliberately has no backlog dependency so review remains a
// projection package rather than a second owner of backlog state.
type ReviewContract struct {
	Description string `json:"description"`
	Criteria    any    `json:"criteria"`
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

// RoundEvidenceRecorder projects a completed round into the canonical evidence
// ledger. The round file remains the recovery input; a recorder failure is
// returned to transitionrunner so its idempotent apply journal can retry.
type RoundEvidenceRecorder func(ctx context.Context, kind, name string, round Round) error

// EvidenceVerificationRecorder writes an explicit operator verification or
// revocation observation. The boolean in Round remains a readable projection
// only; the ledger is authoritative once migration parity is proven.
type EvidenceVerificationRecorder func(ctx context.Context, kind, name string, round Round, evidence EvidenceItem, verified bool, actor, reason string) error

// EvidenceVerificationProjection returns ledger verification state. The bool
// reports whether a completed parity audit permits it to replace file flags.
type EvidenceVerificationProjection func(ctx context.Context, kind, name string, round Round) (verified map[string]bool, authoritative bool, err error)

// ServiceConfig configures the review service dependencies.
type ServiceConfig struct {
	DataRoot string
	// RunInspector polls agent-run state for gathering rounds. Optional —
	// without it, stale rounds fall back to age-based recovery.
	RunInspector         RunInspector
	PromptClient         promptmanager.Client
	ItemDirFn            func(kind, name string) string
	LoadItemTitle        func(kind, name string) (string, error)
	LoadReviewContract   func(kind, name string) (ReviewContract, error)
	LoadExecutionContext func(ctx context.Context, executionID string) (*ExecutionContext, error)
	PlanContentResolver  PlanContentResolver
	// OnRoundTerminal fires when a review round transitions to complete/failed.
	// Used to flip the backlog item's status to review_pending.
	OnRoundTerminal        RoundTerminalHandler
	RoundTerminalObserver  RoundTerminalObserver
	EvidenceRecorder       RoundEvidenceRecorder
	VerificationRecorder   EvidenceVerificationRecorder
	VerificationProjection EvidenceVerificationProjection
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
	dataRoot               string
	transitionRunner       *transitionrunner.Runner
	inspector              RunInspector
	promptClient           promptmanager.Client
	eventLogger            EventLogger
	itemDirFn              func(kind, name string) string
	loadItemTitle          func(kind, name string) (string, error)
	loadReviewContract     func(kind, name string) (ReviewContract, error)
	loadExecutionContext   func(ctx context.Context, executionID string) (*ExecutionContext, error)
	planContentResolver    PlanContentResolver
	onRoundTerminal        RoundTerminalHandler
	roundTerminalObserver  RoundTerminalObserver
	evidenceRecorder       RoundEvidenceRecorder
	verificationRecorder   EvidenceVerificationRecorder
	verificationProjection EvidenceVerificationProjection
	roundMaxAge            time.Duration
	clock                  func() time.Time
}

// SetTransitionRunner installs the shared lifecycle owner for declared review
// workflows. The review package retains only snapshot construction and result
// projection into the operator-facing round ledger.
func (s *Service) SetTransitionRunner(runner *transitionrunner.Runner) { s.transitionRunner = runner }

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
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
		dataRoot:               cfg.DataRoot,
		inspector:              cfg.RunInspector,
		promptClient:           pc,
		itemDirFn:              cfg.ItemDirFn,
		loadItemTitle:          cfg.LoadItemTitle,
		loadReviewContract:     cfg.LoadReviewContract,
		loadExecutionContext:   cfg.LoadExecutionContext,
		planContentResolver:    cfg.PlanContentResolver,
		onRoundTerminal:        cfg.OnRoundTerminal,
		roundTerminalObserver:  cfg.RoundTerminalObserver,
		evidenceRecorder:       cfg.EvidenceRecorder,
		verificationRecorder:   cfg.VerificationRecorder,
		verificationProjection: cfg.VerificationProjection,
		roundMaxAge:            roundMaxAge,
		clock:                  time.Now,
	}
	return svc
}

// SetEvidenceRecorder installs the canonical ledger projection after database
// initialization; it is separate from the backlog terminal callback.
func (s *Service) SetEvidenceRecorder(recorder RoundEvidenceRecorder) {
	if s != nil {
		s.evidenceRecorder = recorder
	}
}

func (s *Service) SetEvidenceVerificationRecorder(recorder EvidenceVerificationRecorder) {
	if s != nil {
		s.verificationRecorder = recorder
	}
}

func (s *Service) SetEvidenceVerificationProjection(projection EvidenceVerificationProjection) {
	if s != nil {
		s.verificationProjection = projection
	}
}

// SetRoundTerminalObserver configures the optional projection into the common
// operator loop. It deliberately supplements, rather than replaces, the
// backlog lifecycle callback.
func (s *Service) SetRoundTerminalObserver(observer RoundTerminalObserver) {
	if s != nil {
		s.roundTerminalObserver = observer
	}
}

func (s *Service) notifyRoundTerminal(ctx context.Context, kind, name string, round Round) {
	if s.roundTerminalObserver != nil {
		s.roundTerminalObserver(ctx, kind, name, round)
	}
}

// SetEventLogger injects an optional event logger for analytics.
func (s *Service) SetEventLogger(e EventLogger) {
	s.eventLogger = e
}

// StartReviewForExecution is called by the execution service during finalization
// to spawn a review agent. It satisfies execution.ReviewServiceIntegration.
func (s *Service) StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDescription, itemDir string, acceptanceCriteria any, machineEvidence []EvidenceItem, affectedScenarios []string, changedPathsByScenario map[string][]string, gctResultsJSON, baselineDiffJSON string) error {
	return s.startReview(ctx, startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            backlogKind,
		BacklogName:            backlogName,
		ItemTitle:              itemTitle,
		ItemDescription:        itemDescription,
		AcceptanceCriteria:     acceptanceCriteria,
		MachineEvidence:        machineEvidence,
		ItemDir:                itemDir,
		AffectedScenarios:      affectedScenarios,
		ChangedPathsByScenario: changedPathsByScenario,
		GCTResultsJSON:         gctResultsJSON,
		BaselineDiffJSON:       baselineDiffJSON,
	})
}

// startReviewParams packages the data needed to begin evidence gathering.
type startReviewParams struct {
	ExecutionID            string              `json:"executionId"`
	BacklogKind            string              `json:"backlogKind"`
	BacklogName            string              `json:"backlogName"`
	ItemTitle              string              `json:"itemTitle"`
	ItemDescription        string              `json:"itemDescription"`
	PlanContent            string              `json:"planContent"`
	AcceptanceCriteria     any                 `json:"acceptanceCriteria"`
	MachineEvidence        []EvidenceItem      `json:"machineEvidence,omitempty"`
	ItemDir                string              `json:"itemDir"`
	AffectedScenarios      []string            `json:"affectedScenarios"`
	ChangedPathsByScenario map[string][]string `json:"changedPathsByScenario"`
	GCTResultsJSON         string              `json:"gctResultsJSON"`   // Pre-serialized GCT review results per scenario
	BaselineDiffJSON       string              `json:"baselineDiffJSON"` // Pre-serialized before/after baseline diff per scenario
}

// startReview creates a review round then invokes the declared independent-review
// workflow with an immutable execution snapshot. Swarm retains only the local
// review ledger and terminal operator-gate application.
func (s *Service) startReview(ctx context.Context, params startReviewParams) error {
	if s.transitionRunner == nil {
		return fmt.Errorf("transition runner is not configured")
	}
	if s.loadReviewContract != nil && params.AcceptanceCriteria == nil && params.ItemDescription == "" {
		contract, err := s.loadReviewContract(params.BacklogKind, params.BacklogName)
		if err != nil {
			return fmt.Errorf("load review contract: %w", err)
		}
		params.ItemDescription, params.AcceptanceCriteria = contract.Description, contract.Criteria
	}
	if strings.TrimSpace(params.PlanContent) == "" && s.planContentResolver != nil {
		content, err := s.planContentResolver(ctx, params.BacklogKind, params.BacklogName, params.ItemDir)
		if err != nil {
			return fmt.Errorf("load review plan content: %w", err)
		}
		params.PlanContent = content
	}
	return s.startReviewTransition(ctx, params)
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

	contract := ReviewContract{}
	if s.loadReviewContract != nil {
		contract, err = s.loadReviewContract(execCtx.BacklogKind, execCtx.BacklogName)
		if err != nil {
			return startReviewParams{}, fmt.Errorf("load review contract: %w", err)
		}
	}
	return startReviewParams{
		ExecutionID:            executionID,
		BacklogKind:            execCtx.BacklogKind,
		BacklogName:            execCtx.BacklogName,
		ItemTitle:              execCtx.ItemTitle,
		ItemDescription:        contract.Description,
		AcceptanceCriteria:     contract.Criteria,
		ItemDir:                s.resolveItemDir(execCtx.BacklogKind, execCtx.BacklogName),
		AffectedScenarios:      append([]string(nil), execCtx.AffectedScenarios...),
		ChangedPathsByScenario: cloneChangedPaths(execCtx.ChangedPathsByScenario),
		GCTResultsJSON:         execCtx.GCTResultsJSON,
		BaselineDiffJSON:       execCtx.BaselineDiffJSON,
	}, nil
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

// MarshalScenarioGCTResults preserves the canonical GCT result projection in
// the immutable review input without adding a second attachment protocol.
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
