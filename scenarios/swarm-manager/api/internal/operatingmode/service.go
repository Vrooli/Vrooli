package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
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
	Items              []string
	AcceptanceCriteria []string
}

type InitiativeReader interface {
	LoadInitiative(name string) (InitiativeSnapshot, error)
}

type InitiativeModeUpdater interface {
	UpdateInitiativeMode(name, mode string) (InitiativeSnapshot, error)
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
	Lock             InitiativeLock
	Initiatives      InitiativeReader
	ModeUpdater      InitiativeModeUpdater
	Backlog          BacklogReader
	BacklogMutator   BacklogMutator
	Reconciler       ProposalReconciler
	ItemExecutions   ItemExecutionController
	Agent            AgentSpawner
	Activity         *agentactivity.Service
	PromptClient     promptmanager.Client
	PromptCatalog    PromptCatalogResolver
	Events           *eventlog.Emitter
	ScenarioRoot     string
	Clock            func() time.Time
	RequestedByLabel string
}

type Service struct {
	store         *Store
	lock          InitiativeLock
	initiatives   InitiativeReader
	modeUpdater   InitiativeModeUpdater
	backlog       BacklogReader
	backlogMut    BacklogMutator
	reconciler    ProposalReconciler
	itemExecs     ItemExecutionController
	agent         AgentSpawner
	activity      *agentactivity.Service
	prompts       promptmanager.Client
	promptCatalog PromptCatalogResolver
	events        *eventlog.Emitter
	scenarioRoot  string
	clock         func() time.Time
	requestedBy   string
}

type StartPhaseRequest struct {
	InitiativeName string
	Phase          string
	Note           string
	Override       bool
	RequestedBy    string
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
}

type WorkspaceMode struct {
	Mode         string              `json:"mode"`
	Label        string              `json:"label"`
	ScopeKind    string              `json:"scope_kind"`
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
	Mode           string             `json:"mode"`
	Label          string             `json:"label"`
	ScopeKind      string             `json:"scope_kind"`
	RunStrategy    string             `json:"run_strategy"`
	WorkspaceTabID string             `json:"workspace_tab_id"`
	Capabilities   ModeCapabilities   `json:"capabilities"`
	Default        bool               `json:"default"`
	Switchable     bool               `json:"switchable"`
	SupportsPhases bool               `json:"supports_phases"`
	Phases         []ModeCatalogPhase `json:"phases"`
}

type ModeCatalogPhase struct {
	Phase            string `json:"phase"`
	ProfileKey       string `json:"profile_key"`
	WritesRepo       bool   `json:"writes_repo"`
	RequiresCriteria bool   `json:"requires_criteria,omitempty"`
}

type WorkspacePhase struct {
	Phase            string               `json:"phase"`
	ActivityPurpose  string               `json:"activity_purpose"`
	ProfileKey       string               `json:"profile_key"`
	WritesRepo       bool                 `json:"writes_repo"`
	OutputArtifacts  []ArtifactDefinition `json:"output_artifacts,omitempty"`
	RequiresCriteria bool                 `json:"requires_criteria,omitempty"`
	Startable        bool                 `json:"startable"`
	Reason           string               `json:"reason,omitempty"`
	Next             bool                 `json:"next,omitempty"`
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
	return &Service{
		store:         cfg.Store,
		lock:          cfg.Lock,
		initiatives:   cfg.Initiatives,
		modeUpdater:   cfg.ModeUpdater,
		backlog:       cfg.Backlog,
		backlogMut:    cfg.BacklogMutator,
		reconciler:    cfg.Reconciler,
		itemExecs:     cfg.ItemExecutions,
		agent:         cfg.Agent,
		activity:      cfg.Activity,
		prompts:       cfg.PromptClient,
		promptCatalog: cfg.PromptCatalog,
		events:        cfg.Events,
		scenarioRoot:  strings.TrimSpace(cfg.ScenarioRoot),
		clock:         clk,
		requestedBy:   requestedBy,
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
