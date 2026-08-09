package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const WorkflowSchemaVersionV1 = "agent-workflow/v1"

// ProfileSchemaVersionV1 discriminates scenario-owned profile declaration files
// in the unified .vrooli/agent-manager/ location. It is a file-format marker
// only: the reconcile reader peeks and strips it before the strict AgentProfile
// proto unmarshal, so it never leaks into the runtime profile entity, its DB
// row, or the public API surface.
const ProfileSchemaVersionV1 = "agent-profile/v1"

type WorkflowNodeKind string

const (
	WorkflowNodeRun      WorkflowNodeKind = "run"
	WorkflowNodeContinue WorkflowNodeKind = "continue"
	WorkflowNodeChild    WorkflowNodeKind = "child_workflow"
	WorkflowNodeWait     WorkflowNodeKind = "wait"
	WorkflowNodeBranch   WorkflowNodeKind = "branch"
	WorkflowNodeJoin     WorkflowNodeKind = "join"
	WorkflowNodeEnd      WorkflowNodeKind = "end"
)

type WorkflowBindingSource string

const (
	WorkflowBindingInput      WorkflowBindingSource = "workflow_input"
	WorkflowBindingAttempts   WorkflowBindingSource = "node_attempts"
	WorkflowBindingRunResult  WorkflowBindingSource = "run_result"
	WorkflowBindingStructured WorkflowBindingSource = "structured_result"
	WorkflowBindingHandoff    WorkflowBindingSource = "final_handoff"
	WorkflowBindingSignal     WorkflowBindingSource = "signal"
	WorkflowBindingCounter    WorkflowBindingSource = "counter"
	WorkflowBindingChild      WorkflowBindingSource = "child_workflow_output"
)

// WorkflowDefinition is scenario-authored desired state. It contains no
// runtime ids, activation fields, implicit profile, or mutable variable map.
type WorkflowDefinition struct {
	SchemaVersion string          `json:"schemaVersion"`
	Owner         string          `json:"owner"`
	Key           string          `json:"key"`
	Version       string          `json:"version"`
	Description   string          `json:"description,omitempty"`
	InputSchema   json.RawMessage `json:"inputSchema"`
	OutputSchema  json.RawMessage `json:"outputSchema"`
	EntryNode     string          `json:"entryNode"`
	Nodes         []WorkflowNode  `json:"nodes"`
	Edges         []WorkflowEdge  `json:"edges"`
	Budgets       WorkflowBudgets `json:"budgets"`
	// ExperimentEvaluator is mandatory for a workflow that arms a skill
	// experiment. It names an independent downstream evaluator rather than
	// inferring quality from treatment-run completion.
	ExperimentEvaluator *WorkflowExperimentEvaluator `json:"experimentEvaluator,omitempty"`
	// Trigger controls who may start this workflow and whether a workflow may
	// start another execution of itself. Its zero value is safe: all initiator
	// classes are allowed, while self-triggering is denied.
	Trigger  WorkflowTriggerPolicy `json:"trigger,omitempty"`
	Metadata map[string]string     `json:"metadata,omitempty"`
}

type WorkflowExperimentEvaluator struct {
	TreatmentNodeIDs        []string `json:"treatmentNodeIds"`
	EvaluatorNodeID         string   `json:"evaluatorNodeId"`
	VerdictPointer          string   `json:"verdictPointer"`
	AllowedVerdicts         []string `json:"allowedVerdicts"`
	SuccessVerdicts         []string `json:"successVerdicts"`
	RubricHash              string   `json:"rubricHash"`
	RubricAuthor            string   `json:"rubricAuthor"`
	EvaluatorPromptHash     string   `json:"evaluatorPromptHash"`
	IndependenceDeclaration string   `json:"independenceDeclaration"`
}

// WorkflowInitiator is the server-classified source of a workflow start.
// A request with no identity token is programmatic by design. A request that
// claims to be an agent but cannot verify its token is denied; it is never
// reclassified as programmatic.
type WorkflowInitiator string

const (
	WorkflowInitiatorHuman        WorkflowInitiator = "human"
	WorkflowInitiatorProgrammatic WorkflowInitiator = "programmatic"
	WorkflowInitiatorAgent        WorkflowInitiator = "agent"
)

// WorkflowSelfTriggerMode controls recursive starts of the same workflow.
type WorkflowSelfTriggerMode string

const (
	WorkflowSelfTriggerDeny  WorkflowSelfTriggerMode = "deny"
	WorkflowSelfTriggerAllow WorkflowSelfTriggerMode = "allow"
)

// WorkflowTriggerPolicy is authored in an agent-workflow/v1 declaration.
// An empty Initiators list means all initiators. An empty SelfTrigger mode
// means deny. maxDepth is required only when mode is allow.
type WorkflowTriggerPolicy struct {
	Initiators  []WorkflowInitiator       `json:"initiators,omitempty"`
	SelfTrigger WorkflowSelfTriggerPolicy `json:"selfTrigger,omitempty"`
}

type WorkflowSelfTriggerPolicy struct {
	Mode     WorkflowSelfTriggerMode `json:"mode,omitempty"`
	MaxDepth int                     `json:"maxDepth,omitempty"`
}

func (p WorkflowTriggerPolicy) Allows(initiator WorkflowInitiator) bool {
	if len(p.Initiators) == 0 {
		return true
	}
	for _, allowed := range p.Initiators {
		if allowed == initiator {
			return true
		}
	}
	return false
}

func (p WorkflowTriggerPolicy) SelfTriggerMode() WorkflowSelfTriggerMode {
	if p.SelfTrigger.Mode == "" {
		return WorkflowSelfTriggerDeny
	}
	return p.SelfTrigger.Mode
}

type WorkflowNode struct {
	ID       string                `json:"id"`
	Kind     WorkflowNodeKind      `json:"kind"`
	Run      *WorkflowRunNode      `json:"run,omitempty"`
	Continue *WorkflowContinueNode `json:"continue,omitempty"`
	Child    *WorkflowChildNode    `json:"childWorkflow,omitempty"`
	Wait     *WorkflowWaitNode     `json:"wait,omitempty"`
	Branch   *WorkflowBranchNode   `json:"branch,omitempty"`
	Join     *WorkflowJoinNode     `json:"join,omitempty"`
	End      *WorkflowEndNode      `json:"end,omitempty"`
}

type WorkflowRunNode struct {
	ProfileKey string `json:"profileKey,omitempty"`
	RoleRef    string `json:"roleRef,omitempty"`
	// ScopePathTemplate renders from this node's declared bindings and limits
	// the task workspace for a fresh run. An empty value retains the workflow
	// default scope (the project root).
	ScopePathTemplate string                 `json:"scopePathTemplate,omitempty"`
	Tag               string                 `json:"tag,omitempty"`
	Force             bool                   `json:"force,omitempty"`
	PromptTemplate    string                 `json:"promptTemplate,omitempty"`
	PromptRef         *WorkflowPromptRef     `json:"promptRef,omitempty"`
	PromptProvenance  *WorkflowPromptSource  `json:"promptProvenance,omitempty"`
	ResultSpec        *ResultSpec            `json:"resultSpec,omitempty"`
	Bindings          []WorkflowInputBinding `json:"bindings,omitempty"`
	MaxTurns          int                    `json:"maxTurns,omitempty"`
	TimeoutSeconds    int                    `json:"timeoutSeconds,omitempty"`
}

type WorkflowContinueNode struct {
	ConversationFromNode string                 `json:"conversationFromNode"`
	PromptTemplate       string                 `json:"promptTemplate,omitempty"`
	PromptRef            *WorkflowPromptRef     `json:"promptRef,omitempty"`
	PromptProvenance     *WorkflowPromptSource  `json:"promptProvenance,omitempty"`
	Bindings             []WorkflowInputBinding `json:"bindings,omitempty"`
	ResultSpec           *ResultSpec            `json:"resultSpec,omitempty"`
	MaxTurns             int                    `json:"maxTurns,omitempty"`
	TimeoutSeconds       int                    `json:"timeoutSeconds,omitempty"`
}

// WorkflowPromptRef references a prompt-manager skill as an alternative to an
// inline promptTemplate. It resolves at reconcile time: the resolved content is
// embedded into promptTemplate and the resolution is pinned in the revision, so
// a later skill change produces a new revision rather than a silent behavior
// change. Exactly one of promptTemplate or promptRef may be authored.
type WorkflowPromptRef struct {
	SkillID   string            `json:"skillId"`
	Variables map[string]string `json:"variables,omitempty"`
	WithScope bool              `json:"withScope,omitempty"`
	// ExperimentID deliberately arms this node against a running experiment. When
	// empty (the default), resolution is pinned/deterministic — a running
	// experiment never silently varies the node's prompt across reconciles.
	ExperimentID string `json:"experimentId,omitempty"`
}

// WorkflowPromptSource is the digest-pinned provenance of a promptRef that has
// been resolved. It records which skill and revision produced the embedded
// prompt so the resolution is auditable; the resolved-at instant is the
// revision's CreatedAt and is intentionally not stored here to keep the digest
// deterministic across re-resolution of an unchanged skill.
type WorkflowPromptSource struct {
	SkillID     string `json:"skillId"`
	Revision    int    `json:"revision,omitempty"`
	VariantID   string `json:"variantId,omitempty"`
	ContentHash string `json:"contentHash"`
	// ExperimentID records the armed experiment when the read was
	// experiment-selected, enabling outcome attribution at run finalization.
	ExperimentID string `json:"experimentId,omitempty"`
	// Variables and WithScope preserve the exact prompt-manager read contract so
	// staleness checks compare the pinned rendered prompt with like-for-like
	// current source content.
	Variables map[string]string `json:"variables,omitempty"`
	WithScope bool              `json:"withScope,omitempty"`
}

type WorkflowChildNode struct {
	WorkflowKey string                 `json:"workflowKey"`
	Version     string                 `json:"version,omitempty"`
	Bindings    []WorkflowInputBinding `json:"bindings,omitempty"`
	MaxDepth    int                    `json:"maxDepth"`
}

type WorkflowWaitNode struct {
	Signal         string          `json:"signal"`
	PayloadSchema  json.RawMessage `json:"payloadSchema,omitempty"`
	TimeoutSeconds int             `json:"timeoutSeconds"`
	OnTimeout      string          `json:"onTimeout,omitempty"`
}

// WorkflowBranchNode marks a routing node. A non-parallel branch selects among
// its outgoing edges by their CEL conditions; a parallel branch forks into its
// members. Routing lives entirely on the edges, so a branch carries no
// expression of its own.
type WorkflowBranchNode struct {
	Parallel bool `json:"parallel,omitempty"`
}

type WorkflowJoinNode struct {
	Strategy string `json:"strategy"`
	Quorum   int    `json:"quorum,omitempty"`
}

type WorkflowEndNode struct {
	Status   string                 `json:"status"`
	Bindings []WorkflowInputBinding `json:"bindings,omitempty"`
}

type WorkflowInputBinding struct {
	Name           string                `json:"name"`
	Source         WorkflowBindingSource `json:"source"`
	Selector       string                `json:"selector,omitempty"`
	Order          string                `json:"order,omitempty"`
	Limit          int                   `json:"limit"`
	MaxBytes       int                   `json:"maxBytes"`
	RenderAs       string                `json:"renderAs"`
	WrapTag        string                `json:"wrapTag,omitempty"`
	Lang           string                `json:"lang,omitempty"`
	Overflow       string                `json:"overflow,omitempty"`
	ItemTag        string                `json:"itemTag,omitempty"`
	ItemMaxBytes   int                   `json:"itemMaxBytes,omitempty"`
	EvictionPolicy string                `json:"evictionPolicy,omitempty"`
	KeepFirst      int                   `json:"keepFirst,omitempty"`
	MissingPolicy  string                `json:"missingPolicy"`
}

type WorkflowEdge struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Condition     string `json:"condition,omitempty"`
	MaxTraversals int    `json:"maxTraversals,omitempty"`
}

type WorkflowBudgets struct {
	WallTimeSeconds   int   `json:"wallTimeSeconds"`
	MaxTurns          int   `json:"maxTurns"`
	MaxTokens         int   `json:"maxTokens"`
	MaxChargeMicroUSD int64 `json:"maxChargeMicroUsd"`
	MaxNodeAttempts   int   `json:"maxNodeAttempts"`
	MaxChildren       int   `json:"maxChildren"`
	MaxConcurrency    int   `json:"maxConcurrency"`
	MaxRecursion      int   `json:"maxRecursion"`
	MaxRetries        int   `json:"maxRetries"`
	MaxWaitSeconds    int   `json:"maxWaitSeconds"`
}

// WorkflowRevision is Agent Manager's immutable runtime projection of one
// scenario-owned definition revision.
type WorkflowRevision struct {
	ID              uuid.UUID          `json:"id" db:"id"`
	Owner           string             `json:"owner" db:"owner"`
	Key             string             `json:"key" db:"workflow_key"`
	SemanticVersion string             `json:"semanticVersion" db:"semantic_version"`
	Digest          string             `json:"digest" db:"digest"`
	Definition      WorkflowDefinition `json:"definition" db:"definition_json"`
	SourcePath      string             `json:"sourcePath" db:"source_path"`
	SourceHash      string             `json:"sourceHash" db:"source_hash"`
	SourceUpdatedAt time.Time          `json:"sourceUpdatedAt" db:"source_updated_at"`
	Active          bool               `json:"active" db:"active"`
	CreatedAt       time.Time          `json:"createdAt" db:"created_at"`
	// PromptStale is a live status projection, deliberately excluded from the
	// immutable revision row and digest-addressed definition.
	PromptStale bool `json:"promptStale,omitempty" db:"-"`
}

// Diagnostic severities. An empty severity is treated as an error so every
// diagnostic that predates the severity field keeps blocking registration.
const (
	DiagnosticSeverityError   = "error"
	DiagnosticSeverityWarning = "warning"
)

type WorkflowDiagnostic struct {
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

// IsError reports whether the diagnostic blocks registration. Only an explicit
// warning severity is non-blocking; anything else (including unset) blocks.
func (d WorkflowDiagnostic) IsError() bool { return d.Severity != DiagnosticSeverityWarning }

// HasBlockingDiagnostic reports whether any diagnostic in the set blocks
// registration. A definition carrying only warnings still earns a digest.
func HasBlockingDiagnostic(diagnostics []WorkflowDiagnostic) bool {
	for _, d := range diagnostics {
		if d.IsError() {
			return true
		}
	}
	return false
}
