package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/promptmanager"
)

type InitiativeSnapshot struct {
	Name               string
	Title              string
	Description        string
	Mode               string
	PlanRef            *PlanRef
	Items              []string
	AcceptanceCriteria []string
}

type InitiativeReader interface {
	LoadInitiative(name string) (InitiativeSnapshot, error)
}

// InitiativeLister exposes the minimum fields needed to compute mode usage
// counts and the linked-initiative list for the operating-mode details page.
// Wired separately from InitiativeReader so existing test seams don't have
// to grow when only one of the two is needed.
type InitiativeLister interface {
	ListInitiatives() ([]InitiativeSummary, error)
}

type InitiativeSummary struct {
	Name    string
	Title   string
	Mode    string
	Status  string
	Updated string
}

type InitiativeModeUpdater interface {
	UpdateInitiativeMode(name, mode string) (InitiativeSnapshot, error)
}

type InitiativePlanRefBinder interface {
	BindInitiativePlanRef(name string, ref PlanRef) (InitiativeSnapshot, error)
}

type BacklogItemSnapshot struct {
	Title    string
	Status   string
	Priority int
	Effort   string
}

type BacklogReader interface {
	LoadBacklogItem(kind, name string) (BacklogItemSnapshot, error)
}

type BacklogCompletionResult struct {
	ItemRef    string `json:"item_ref"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
}

type BacklogMutationSource struct {
	Entrypoint     string
	InitiativeName string
	Mode           string
	Phase          string
	Round          int
	RunID          string
	RequestedBy    string
}

type BacklogMutator interface {
	MarkBacklogItemCompleted(ctx context.Context, kind, name string, source BacklogMutationSource) (BacklogCompletionResult, error)
}

type ProposalReconciler interface {
	ApplyBacklogSyncProposal(ctx context.Context, req ProposalReconcileRequest) (*ProposalApplyResult, error)
}

type ProposalReconcileRequest struct {
	InitiativeName      string
	Mode                string
	Round               int
	Phase               string
	RunID               string
	Proposal            json.RawMessage
	AcceptedMutationIDs []string
	DecidedBy           string
	DecidedAtRFC3339    string
}

type ProposalApplyResult struct {
	Outcomes []ProposalOutcome `json:"outcomes"`
	Applied  int               `json:"applied"`
	Failed   int               `json:"failed"`
	Skipped  int               `json:"skipped"`
	Created  int               `json:"created,omitempty"`
	Updated  int               `json:"updated,omitempty"`
}

type ProposalOutcome struct {
	MutationID string `json:"mutation_id"`
	Op         string `json:"op"`
	Target     string `json:"target,omitempty"`
	Applied    bool   `json:"applied"`
	Skipped    bool   `json:"skipped,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ActiveItemExecution struct {
	ItemRef     string `json:"item_ref"`
	ExecutionID string `json:"execution_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Status      string `json:"status,omitempty"`
}

type ItemExecutionController interface {
	ActiveExecutionsForInitiative(ctx context.Context, initiative InitiativeSnapshot) ([]ActiveItemExecution, error)
	CancelActiveExecutionsForInitiative(ctx context.Context, initiative InitiativeSnapshot) ([]ActiveItemExecution, error)
}

type AgentSpawner interface {
	SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error)
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

// RunMessageReader is an optional capability an AgentSpawner may also satisfy:
// returning the ordered assistant messages of a completed run. The resolution
// ladder's L0 (true-final-message detection) uses these to scan past a trailing
// subagent message and recover the real final answer. The concrete
// agent-manager client implements it; test doubles and spawners that don't need
// L0 simply omit it, and the refresher falls back to the run summary as the sole
// resolution candidate.
type RunMessageReader interface {
	GetRunMessages(ctx context.Context, runID string) ([]agentmanager.RunMessage, error)
}

// InitiativeActivitySpawner is the narrow seam the operating-mode service
// uses to launch initiative agent runs through the agentactivity tracker
// (which applies lane gating and emits typed errors like
// agentactivity.ErrLaneSaturated). Production wires this to
// *agentactivity.Service; tests can inject saturation failures here without
// dragging in the full activity service.
type InitiativeActivitySpawner interface {
	SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error)
}

type PromptCatalogResolver func(mode, phase string) (PromptCatalogEntry, bool)

// InitiativeLock is the narrow initiative-agent lock seam used by operating
// mode lifecycle orchestration. Production uses initiativelock.Lock; tests can
// inject failures at specific lifecycle points without touching file-lock
// internals.
type InitiativeLock interface {
	Acquire(initiativeName string, holder initiativelock.Holder) error
	AcquireOverride(initiativeName string, holder initiativelock.Holder) error
	Release(initiativeName, runID string) error
	Inspect(initiativeName string) (*initiativelock.Holder, error)
}

type Config struct {
	Store            *Store
	Overlay          *OverlayStore
	Lock             InitiativeLock
	Initiatives      InitiativeReader
	InitiativeLister InitiativeLister
	ModeUpdater      InitiativeModeUpdater
	PlanRefBinder    InitiativePlanRefBinder
	Backlog          BacklogReader
	BacklogMutator   BacklogMutator
	Reconciler       ProposalReconciler
	ItemExecutions   ItemExecutionController
	Agent            AgentSpawner
	Activity         InitiativeActivitySpawner
	PromptClient     promptmanager.Client
	PromptCatalog    PromptCatalogResolver
	Events           *eventlog.Emitter
	// Classifier is the L2 rung of the resolution ladder. When left nil, the
	// service wires the production ollama-backed classifier so real runs get
	// classifier recovery; tests inject a stub, or leave it nil and disable L2
	// via the mode's resolution policy. Set DisableClassifier to force L2 off.
	Classifier        FieldClassifier
	DisableClassifier bool
	ScenarioRoot      string
	Clock             func() time.Time
	RequestedByLabel  string
	PlanExecution     PlanExecutionClient
}

type Service struct {
	store         *Store
	overlay       *OverlayStore
	lock          InitiativeLock
	initiatives   InitiativeReader
	initLister    InitiativeLister
	modeUpdater   InitiativeModeUpdater
	planRefBinder InitiativePlanRefBinder
	backlog       BacklogReader
	backlogMut    BacklogMutator
	reconciler    ProposalReconciler
	itemExecs     ItemExecutionController
	agent         AgentSpawner
	activity      InitiativeActivitySpawner
	prompts       promptmanager.Client
	promptCatalog PromptCatalogResolver
	events        *eventlog.Emitter
	classifier    FieldClassifier
	scenarioRoot  string
	clock         func() time.Time
	requestedBy   string
	planExecution PlanExecutionClient
}

type StartPhaseRequest struct {
	InitiativeName string
	Phase          string
	Note           string
	Override       bool
	RequestedBy    string
}

// StartTargetPhaseRequest starts a mode round directly on a non-initiative
// target — the plan-first entry point. Mode names a registered plan-target
// mode; TargetRef is the target instance handle its adapter resolves (a
// plan-manager execution id/slug, or a plan-ref path). Phase defaults to the
// mode's start phase.
type StartTargetPhaseRequest struct {
	Mode        string
	TargetRef   string
	Phase       string
	Note        string
	Override    bool
	RequestedBy string
}

type SwitchModeRequest struct {
	InitiativeName             string
	Mode                       string
	CancelActiveItemExecutions bool
	RequestedBy                string
}

type CompleteItemsRequest struct {
	InitiativeName string
	Mode           string
	Round          int
	RunID          string
	ItemRefs       []string
	RequestedBy    string
}

type ApplyBacklogSyncRequest struct {
	InitiativeName      string
	Mode                string
	Round               int
	RunID               string
	AcceptedMutationIDs []string
	RequestedBy         string
}

type BacklogSyncResult struct {
	InitiativeName string                    `json:"initiative_name"`
	Mode           string                    `json:"mode"`
	Phase          string                    `json:"phase"`
	Round          int                       `json:"round"`
	RunID          string                    `json:"run_id,omitempty"`
	CompletedItems []BacklogCompletionResult `json:"completed_items,omitempty"`
	ProposalResult *ProposalApplyResult      `json:"proposal_result,omitempty"`
	Noop           bool                      `json:"noop,omitempty"`
}

type SwitchModeResult struct {
	InitiativeName           string                `json:"initiative_name"`
	FromMode                 string                `json:"from_mode"`
	ToMode                   string                `json:"to_mode"`
	CanceledItemExecutions   []ActiveItemExecution `json:"canceled_item_executions,omitempty"`
	ActiveItemExecutions     []ActiveItemExecution `json:"active_item_executions,omitempty"`
	RequiresCancellation     bool                  `json:"requires_cancellation,omitempty"`
	OperatingModeWorkspaceID string                `json:"operating_mode_workspace_id,omitempty"`
}

type ActiveItemExecutionsConflict struct {
	InitiativeName string                `json:"initiative_name"`
	FromMode       string                `json:"from_mode"`
	ToMode         string                `json:"to_mode"`
	Executions     []ActiveItemExecution `json:"active_item_executions"`
}

type ActiveOperatingModeRoundConflict struct {
	InitiativeName string        `json:"initiative_name"`
	FromMode       string        `json:"from_mode"`
	ToMode         string        `json:"to_mode"`
	Round          RoundEnvelope `json:"round"`
}

type Workspace struct {
	InitiativeName string                 `json:"initiative_name"`
	Mode           string                 `json:"mode"`
	Definition     WorkspaceMode          `json:"definition"`
	Lock           *initiativelock.Holder `json:"lock,omitempty"`
	Artifacts      []ArtifactSnapshot     `json:"artifacts"`
	Rounds         []RoundEnvelope        `json:"rounds"`
	Executions     []OperatingModeExecution `json:"executions"`
}

type WorkspaceMode struct {
	Mode         string              `json:"mode"`
	Label        string              `json:"label"`
	Description  string              `json:"description,omitempty"`
	TargetKind   string              `json:"target_kind"`
	Capabilities ModeCapabilities    `json:"capabilities"`
	Phases       []WorkspacePhase    `json:"phases"`
	Terminal     []string            `json:"terminal"`
	Transitions  map[string][]string `json:"transitions"`
	RunStrategy  string              `json:"run_strategy"`
}

type ModeCapabilities struct {
	SupportsPhases               bool `json:"supports_phases"`
	CanStartPhases               bool `json:"can_start_phases"`
	CanCompleteItems             bool `json:"can_complete_items"`
	CanApplyBacklogSyncProposals bool `json:"can_apply_backlog_sync_proposals"`
	RequiresAcceptanceCriteria   bool `json:"requires_acceptance_criteria"`
	SupportsArtifacts            bool `json:"supports_artifacts"`
	SupportsHandoffs             bool `json:"supports_handoffs"`
	UsesItemExecutionFlow        bool `json:"uses_item_execution_flow"`
}

type ModeCatalog struct {
	Modes []ModeCatalogEntry `json:"modes"`
}

type ModeCatalogEntry struct {
	Mode        string `json:"mode"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// Decision-support metadata. Validator guarantees the three lists are
	// non-empty per registered mode, so they intentionally do not use
	// omitempty (an empty list on the wire is a contract violation worth
	// surfacing). WhenInDoubtPickInstead is empty when the mode is itself
	// the safer default.
	BestFor                []string               `json:"best_for"`
	NotFor                 []string               `json:"not_for"`
	Tradeoffs              []string               `json:"tradeoffs"`
	WhenInDoubtPickInstead string                 `json:"when_in_doubt_pick_instead,omitempty"`
	UsageCount             int                    `json:"usage_count"`
	TargetKind             string                 `json:"target_kind"`
	RunStrategy            string                 `json:"run_strategy"`
	WorkspaceTabID         string                 `json:"workspace_tab_id"`
	Capabilities           ModeCapabilities       `json:"capabilities"`
	Default                bool                   `json:"default"`
	Switchable             bool                   `json:"switchable"`
	SupportsPhases         bool                   `json:"supports_phases"`
	Phases                 []ModeCatalogPhase     `json:"phases"`
	PhaseGraph             *ModeCatalogPhaseGraph `json:"phase_graph,omitempty"`
}

// InitiativeRef is the compact view of an initiative attached to a mode in
// the operating-mode details page.
type InitiativeRef struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Updated string `json:"updated,omitempty"`
}

// ModeDetail is the projection returned by OperatingModeService.GetMode.
// It pairs the catalog entry (already merged with overlay) with the list of
// initiatives currently bound to the mode.
type ModeDetail struct {
	Entry             ModeCatalogEntry `json:"entry"`
	LinkedInitiatives []InitiativeRef  `json:"linked_initiatives"`
}

// ModeCatalogPhase is the wire shape for a single phase inside a mode's
// catalog entry. It is the projection of registry.PhaseDefinition + the
// phase's role in the mode's phase graph (start / terminal flags + sample
// flags). Only fields that the UI/CLI render are surfaced; transitions are
// carried separately on ModeCatalogPhaseGraph.
type ModeCatalogPhase struct {
	Phase string `json:"phase"`
	// PhaseKind classifies the phase (investigate / execute / review /
	// reconcile). Operations Center column placement, lane assignment, and
	// per-lane metrics all key off this axis.
	PhaseKind        string               `json:"phase_kind"`
	Label            string               `json:"label"`
	Title            string               `json:"title"`
	Purpose          string               `json:"purpose"`
	Trigger          string               `json:"trigger"`
	ProfileKey       string               `json:"profile_key"`
	WritesRepo       bool                 `json:"writes_repo"`
	RequiresCriteria bool                 `json:"requires_criteria,omitempty"`
	IsStart          bool                 `json:"is_start,omitempty"`
	IsTerminal       bool                 `json:"is_terminal,omitempty"`
	OutputArtifacts  []ArtifactDefinition `json:"output_artifacts,omitempty"`
	// Reads is the phase's declared input contract grouped by supplying
	// provider (generic base vs target adapter) — the Reads twin of
	// OutputContract, rendered from data.
	Reads                 PhaseReadsSummary          `json:"reads"`
	OutputContract        PhaseOutputContractSummary `json:"output_contract"`
	CatalogID             string                     `json:"catalog_id"`
	SkillID               string                     `json:"skill_id"`
	ActivityPurpose       string                     `json:"activity_purpose"`
	LockPurpose           string                     `json:"lock_purpose"`
	ResultBindings        []ResultBinding            `json:"result_bindings,omitempty"`
	SamplesReplanRate     bool                       `json:"samples_replan_rate,omitempty"`
	SamplesAcceptanceRate bool                       `json:"samples_acceptance_rate,omitempty"`
	// AutoStartAfter, when non-empty, lists the predecessor phase whose
	// completion auto-starts this phase via the round-refresher hook.
	// Length ≤ 1 in v1 (validator-enforced).
	AutoStartAfter []string `json:"auto_start_after,omitempty"`
	// ExecutedBy names the sub-mode that executes this phase (phase
	// delegation). Empty for regular phases. CLI/UI render the sub-mode's
	// flow inline under this phase; the backend stays the routing SSOT.
	ExecutedBy string `json:"executed_by,omitempty"`
	// Classification is the phase's classification-on-transition contract,
	// present when one of its outgoing edges derives its routing field from the
	// handoff instead of a directly-emitted field. Nil when every edge routes
	// on an emitted field. Renders as a built-in step (not an agent phase).
	Classification *PhaseClassificationSummary `json:"classification,omitempty"`
}

// PhaseClassificationSummary is the catalog-side projection of a phase's
// classification-on-transition contract (registry TransitionClassification):
// the routing field derived at the edge, its closed enum, the handoff field it
// derives from, and an optional operator-facing description. It costs no agent
// round — the engine derives the field via the resolution ladder at the
// transition, abstaining to needs_attention rather than fabricating a route.
type PhaseClassificationSummary struct {
	Field       string   `json:"field"`
	Enum        []string `json:"enum"`
	From        string   `json:"from,omitempty"`
	Description string   `json:"description,omitempty"`
}

// PhaseOutputContractSummary is the flat catalog-side view of the registry's
// PhaseOutputContract. RequiredArtifacts is collapsed to a count because the
// per-artifact metadata is already available on the phase's OutputArtifacts.
type PhaseOutputContractSummary struct {
	RequiresStructuredResult bool `json:"requires_structured_result"`
	RequiresProgress         bool `json:"requires_progress"`
	RequiresPlanRef          bool `json:"requires_plan_ref"`
	RequiresVerdict          bool `json:"requires_verdict"`
	RequiresHandoff          bool `json:"requires_handoff"`
	RequiresBacklogSync      bool `json:"requires_backlog_sync"`
	RequiredArtifactCount    int  `json:"required_artifact_count"`
}

// ModeCatalogPhaseGraph carries the mode's phase graph (start / terminal /
// transitions / accepted verdicts) so the UI can render a DAG without
// re-deriving state from the workspace endpoint. Omitted for item-level mode.
type ModeCatalogPhaseGraph struct {
	StartPhase       string                  `json:"start_phase"`
	Terminal         []string                `json:"terminal"`
	Transitions      []ModeCatalogTransition `json:"transitions"`
	AcceptedVerdicts []string                `json:"accepted_verdicts,omitempty"`
}

// ModeCatalogTransition is one edge of the phase graph. The condition is
// rendered server-side from the generic guard (kind + human label + the leaf
// field/value when applicable) so CLI and UI emit identical strings (e.g.
// "on replan_needed = true", "on progress.decision = continue", "always").
type ModeCatalogTransition struct {
	From          string `json:"from"`
	To            string `json:"to"`
	ConditionKind string `json:"condition_kind"`
	Label         string `json:"label"`
	Field         string `json:"field,omitempty"`
	Value         string `json:"value,omitempty"`
	// Classified marks an edge whose guard field is derived by
	// classification-on-transition rather than emitted directly by the round.
	Classified bool `json:"classified,omitempty"`
}

func summarizeContract(contract PhaseOutputContract) PhaseOutputContractSummary {
	return PhaseOutputContractSummary{
		RequiresStructuredResult: contract.RequiresStructuredResult,
		RequiresProgress:         contract.RequiresProgress,
		RequiresPlanRef:          contract.RequiresPlanRef,
		RequiresVerdict:          contract.RequiresVerdict,
		RequiresHandoff:          contract.RequiresHandoff,
		RequiresBacklogSync:      contract.RequiresBacklogSync,
		RequiredArtifactCount:    len(contract.RequiredArtifacts),
	}
}

type WorkspacePhase struct {
	Phase string `json:"phase"`
	// PhaseKind classifies the phase (investigate / execute / review /
	// reconcile). Mirrors the catalog field of the same name; surfaced on
	// the workspace endpoint so the UI can render lane-aware affordances
	// without re-fetching the catalog.
	PhaseKind        string               `json:"phase_kind"`
	ActivityPurpose  string               `json:"activity_purpose"`
	ProfileKey       string               `json:"profile_key"`
	WritesRepo       bool                 `json:"writes_repo"`
	OutputArtifacts  []ArtifactDefinition `json:"output_artifacts,omitempty"`
	RequiresCriteria bool                 `json:"requires_criteria,omitempty"`
	Startable        bool                 `json:"startable"`
	Reason           string               `json:"reason,omitempty"`
	Next             bool                 `json:"next,omitempty"`
	// AutoStartAfter mirrors the catalog field. UI surfaces use it to
	// render an "auto-starts after X" badge instead of an operator-action
	// button.
	AutoStartAfter []string `json:"auto_start_after,omitempty"`
	// ExecutedBy mirrors the catalog field: the sub-mode that executes this
	// phase via delegation, empty for regular phases.
	ExecutedBy string `json:"executed_by,omitempty"`
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("operatingmode: Store is required")
	}
	if cfg.Lock == nil {
		return nil, errors.New("operatingmode: Lock is required")
	}
	if cfg.Initiatives == nil {
		return nil, errors.New("operatingmode: InitiativeReader is required")
	}
	if err := ValidatePromptCatalog(cfg.PromptCatalog); err != nil {
		return nil, err
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	requestedBy := strings.TrimSpace(cfg.RequestedByLabel)
	if requestedBy == "" {
		requestedBy = "swarm-manager"
	}
	classifier := cfg.Classifier
	if classifier == nil && !cfg.DisableClassifier {
		classifier = newOllamaFieldClassifier()
	}
	return &Service{
		store:         cfg.Store,
		overlay:       cfg.Overlay,
		lock:          cfg.Lock,
		initiatives:   cfg.Initiatives,
		initLister:    cfg.InitiativeLister,
		modeUpdater:   cfg.ModeUpdater,
		planRefBinder: cfg.PlanRefBinder,
		backlog:       cfg.Backlog,
		backlogMut:    cfg.BacklogMutator,
		reconciler:    cfg.Reconciler,
		itemExecs:     cfg.ItemExecutions,
		agent:         cfg.Agent,
		activity:      cfg.Activity,
		prompts:       cfg.PromptClient,
		promptCatalog: cfg.PromptCatalog,
		events:        cfg.Events,
		classifier:    classifier,
		scenarioRoot:  strings.TrimSpace(cfg.ScenarioRoot),
		clock:         clk,
		requestedBy:   requestedBy,
		planExecution: cfg.PlanExecution,
	}, nil
}

func (s *Service) ResolveRoundActionMode(initiativeName, rawMode string) (Mode, error) {
	trimmed := strings.TrimSpace(rawMode)
	if trimmed != "" {
		return requireRoundActionMode(Mode(trimmed))
	}
	if s.initiatives == nil {
		return "", errors.New("operatingmode: InitiativeReader is required")
	}
	init, err := s.initiatives.LoadInitiative(strings.TrimSpace(initiativeName))
	if err != nil {
		return "", err
	}
	mode, err := requireRoundActionMode(Mode(init.Mode))
	if err != nil {
		return "", fmt.Errorf("round actions require an explicit non-default mode or an initiative currently using one: %w", err)
	}
	return mode, nil
}

// ActiveRoundSummary is the wire shape for the first non-terminal round of
// an initiative-scoped operating mode. It is the seam through which the
// graph projection learns which initiatives are mid-phase without coupling
// to RoundEnvelope.
type ActiveRoundSummary struct {
	Mode   string `json:"mode"`
	Phase  string `json:"phase"`
	Round  int    `json:"round"`
	Status string `json:"status"`
	RunID  string `json:"run_id,omitempty"`
}

// ActiveRoundsByInitiative returns the first non-terminal round per
// initiative, keyed by initiative name. Initiatives with no active round
// (or in item-level mode) are absent from the map. The implementation does
// one pass over the initiatives list and one rounds-directory read per
// initiative bound to a non-default mode — N+1 only in initiatives, never
// in rounds. Initiative-exclusive locking guarantees at most one
// non-terminal round per initiative, so the first match is canonical.
func (s *Service) ActiveRoundsByInitiative(_ context.Context) (map[string]ActiveRoundSummary, error) {
	out := map[string]ActiveRoundSummary{}
	if s == nil || s.initLister == nil || s.store == nil {
		return out, nil
	}
	all, err := s.initLister.ListInitiatives()
	if err != nil {
		return nil, err
	}
	for _, init := range all {
		mode := NormalizeMode(init.Mode)
		if mode == ModeItemLevel {
			continue
		}
		def, err := DefinitionFor(mode)
		if err != nil {
			// Unknown mode on an initiative is a registry/data drift; skip
			// the initiative rather than fail the whole projection.
			continue
		}
		rounds, err := s.store.ListRounds(init.Name, def.Mode)
		if err != nil {
			return nil, fmt.Errorf("list rounds for %q: %w", init.Name, err)
		}
		for _, round := range rounds {
			if !isRoundActive(round) {
				continue
			}
			out[init.Name] = ActiveRoundSummary{
				Mode:   string(round.Mode),
				Phase:  string(round.Phase),
				Round:  round.Round,
				Status: string(round.Status),
				RunID:  strings.TrimSpace(round.RunID),
			}
			break
		}
	}
	return out, nil
}

func requireRoundActionMode(mode Mode) (Mode, error) {
	raw := strings.TrimSpace(string(mode))
	if raw == "" {
		return "", errors.New("mode is required for operating-mode round actions")
	}
	def, err := DefinitionFor(Mode(raw))
	if err != nil {
		return "", err
	}
	if def.Mode == ModeItemLevel {
		return "", errors.New("item-level mode round actions are owned by the existing backlog execution flow")
	}
	return def.Mode, nil
}
