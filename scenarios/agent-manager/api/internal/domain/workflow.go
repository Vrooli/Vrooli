package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const WorkflowSchemaVersionV1 = "agent-workflow/v1"

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
	ProfileKey     string                 `json:"profileKey,omitempty"`
	RoleRef        string                 `json:"roleRef,omitempty"`
	PromptTemplate string                 `json:"promptTemplate"`
	ResultSpec     *ResultSpec            `json:"resultSpec,omitempty"`
	Bindings       []WorkflowInputBinding `json:"bindings,omitempty"`
	MaxTurns       int                    `json:"maxTurns,omitempty"`
	TimeoutSeconds int                    `json:"timeoutSeconds,omitempty"`
}

type WorkflowContinueNode struct {
	ConversationFromNode string                 `json:"conversationFromNode"`
	PromptTemplate       string                 `json:"promptTemplate"`
	Bindings             []WorkflowInputBinding `json:"bindings,omitempty"`
	ResultSpec           *ResultSpec            `json:"resultSpec,omitempty"`
	MaxTurns             int                    `json:"maxTurns,omitempty"`
	TimeoutSeconds       int                    `json:"timeoutSeconds,omitempty"`
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

type WorkflowBranchNode struct {
	Expression string `json:"expression"`
	Parallel   bool   `json:"parallel,omitempty"`
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

type WorkflowDiagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}
