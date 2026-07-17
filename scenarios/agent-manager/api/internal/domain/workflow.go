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
	SchemaVersion string            `json:"schemaVersion"`
	Owner         string            `json:"owner"`
	Key           string            `json:"key"`
	Version       string            `json:"version"`
	Description   string            `json:"description,omitempty"`
	InputSchema   json.RawMessage   `json:"inputSchema"`
	OutputSchema  json.RawMessage   `json:"outputSchema"`
	EntryNode     string            `json:"entryNode"`
	Nodes         []WorkflowNode    `json:"nodes"`
	Edges         []WorkflowEdge    `json:"edges"`
	Budgets       WorkflowBudgets   `json:"budgets"`
	Metadata      map[string]string `json:"metadata,omitempty"`
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
	ProfileKey       string                 `json:"profileKey,omitempty"`
	RoleRef          string                 `json:"roleRef,omitempty"`
	Tag              string                 `json:"tag,omitempty"`
	Force            bool                   `json:"force,omitempty"`
	PromptTemplate   string                 `json:"promptTemplate,omitempty"`
	PromptRef        *WorkflowPromptRef     `json:"promptRef,omitempty"`
	PromptProvenance *WorkflowPromptSource  `json:"promptProvenance,omitempty"`
	ResultSpec       *ResultSpec            `json:"resultSpec,omitempty"`
	Bindings         []WorkflowInputBinding `json:"bindings,omitempty"`
	MaxTurns         int                    `json:"maxTurns,omitempty"`
	TimeoutSeconds   int                    `json:"timeoutSeconds,omitempty"`
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
	Name          string                `json:"name"`
	Source        WorkflowBindingSource `json:"source"`
	Selector      string                `json:"selector,omitempty"`
	Order         string                `json:"order,omitempty"`
	Limit         int                   `json:"limit"`
	MaxBytes      int                   `json:"maxBytes"`
	RenderAs      string                `json:"renderAs"`
	MissingPolicy string                `json:"missingPolicy"`
}

type WorkflowEdge struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Condition     string `json:"condition,omitempty"`
	MaxTraversals int    `json:"maxTraversals,omitempty"`
}

type WorkflowBudgets struct {
	WallTimeSeconds int     `json:"wallTimeSeconds"`
	MaxTurns        int     `json:"maxTurns"`
	MaxTokens       int     `json:"maxTokens"`
	MaxCostUSD      float64 `json:"maxCostUsd"`
	MaxNodeAttempts int     `json:"maxNodeAttempts"`
	MaxChildren     int     `json:"maxChildren"`
	MaxConcurrency  int     `json:"maxConcurrency"`
	MaxRecursion    int     `json:"maxRecursion"`
	MaxRetries      int     `json:"maxRetries"`
	MaxWaitSeconds  int     `json:"maxWaitSeconds"`
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
