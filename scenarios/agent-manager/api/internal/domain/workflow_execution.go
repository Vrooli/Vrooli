package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WorkflowExecutionStatus string

const (
	WorkflowExecutionPending         WorkflowExecutionStatus = "pending"
	WorkflowExecutionRunning         WorkflowExecutionStatus = "running"
	WorkflowExecutionWaiting         WorkflowExecutionStatus = "waiting"
	WorkflowExecutionCancelling      WorkflowExecutionStatus = "cancelling"
	WorkflowExecutionSucceeded       WorkflowExecutionStatus = "succeeded"
	WorkflowExecutionBlocked         WorkflowExecutionStatus = "blocked"
	WorkflowExecutionAbstained       WorkflowExecutionStatus = "abstained"
	WorkflowExecutionBudgetExhausted WorkflowExecutionStatus = "budget_exhausted"
	WorkflowExecutionFailed          WorkflowExecutionStatus = "failed"
	WorkflowExecutionCancelled       WorkflowExecutionStatus = "cancelled"
)

func (s WorkflowExecutionStatus) Terminal() bool {
	switch s {
	case WorkflowExecutionSucceeded, WorkflowExecutionBlocked, WorkflowExecutionAbstained, WorkflowExecutionBudgetExhausted, WorkflowExecutionFailed, WorkflowExecutionCancelled:
		return true
	default:
		return false
	}
}

type WorkflowTerminalReason struct {
	Code       string `json:"code"`
	Message    string `json:"message,omitempty"`
	Retryable  bool   `json:"retryable,omitempty"`
	BudgetName string `json:"budgetName,omitempty"`
}

type WorkflowBudgetUsage struct {
	Turns  int `json:"turns"`
	Tokens int `json:"tokens"`
	// ChargeMicroUSD is authoritative metered charge only. Unpriced usage and
	// historical estimates cannot exhaust a monetary budget.
	ChargeMicroUSD int64 `json:"chargeMicroUsd"`
	// CostUSD is retained for readable historical workflow records.
	CostUSD      float64 `json:"costUsd"`
	NodeAttempts int     `json:"nodeAttempts"`
	Children     int     `json:"children"`
	Retries      int     `json:"retries"`
}

type WorkflowExecution struct {
	ID                uuid.UUID               `json:"id"`
	Owner             string                  `json:"owner"`
	WorkflowKey       string                  `json:"workflowKey"`
	DefinitionDigest  string                  `json:"definitionDigest"`
	Status            WorkflowExecutionStatus `json:"status"`
	CurrentNodeID     string                  `json:"currentNodeId"`
	Input             json.RawMessage         `json:"input"`
	Output            json.RawMessage         `json:"output,omitempty"`
	TerminalReason    *WorkflowTerminalReason `json:"terminalReason,omitempty"`
	BudgetUsage       WorkflowBudgetUsage     `json:"budgetUsage"`
	EdgeTraversals    map[string]int          `json:"edgeTraversals"`
	Version           int64                   `json:"version"`
	IdempotencyKey    string                  `json:"idempotencyKey"`
	ParentExecutionID *uuid.UUID              `json:"parentExecutionId,omitempty"`
	ParentAttemptID   *uuid.UUID              `json:"parentAttemptId,omitempty"`
	Depth             int                     `json:"depth"`
	CreatedAt         time.Time               `json:"createdAt"`
	UpdatedAt         time.Time               `json:"updatedAt"`
	EndedAt           *time.Time              `json:"endedAt,omitempty"`
}

type WorkflowAttemptStrategy string

const (
	WorkflowAttemptFreshRun WorkflowAttemptStrategy = "fresh_run"
	WorkflowAttemptContinue WorkflowAttemptStrategy = "continue"
	WorkflowAttemptChild    WorkflowAttemptStrategy = "child_workflow"
)

type WorkflowAttemptStatus string

const (
	WorkflowAttemptDispatchPending WorkflowAttemptStatus = "dispatch_pending"
	WorkflowAttemptDispatched      WorkflowAttemptStatus = "dispatched"
	WorkflowAttemptWaiting         WorkflowAttemptStatus = "waiting"
	WorkflowAttemptCompleted       WorkflowAttemptStatus = "completed"
	WorkflowAttemptFailed          WorkflowAttemptStatus = "failed"
)

type WorkflowNodeAttempt struct {
	ID             uuid.UUID               `json:"id"`
	ExecutionID    uuid.UUID               `json:"executionId"`
	NodeID         string                  `json:"nodeId"`
	Ordinal        int                     `json:"ordinal"`
	Strategy       WorkflowAttemptStrategy `json:"strategy"`
	Status         WorkflowAttemptStatus   `json:"status"`
	IdempotencyKey string                  `json:"idempotencyKey"`
	InputSnapshot  json.RawMessage         `json:"inputSnapshot"`
	PromptSnapshot string                  `json:"promptSnapshot"`
	// Experiment provenance is captured before a dispatch_pending attempt can
	// transition to dispatched. It survives retry and recovery with the attempt.
	ExperimentID     string     `json:"experimentId,omitempty"`
	VariantID        string     `json:"variantId,omitempty"`
	PromptHash       string     `json:"promptHash,omitempty"`
	RunID            *uuid.UUID `json:"runId,omitempty"`
	ConversationID   string     `json:"conversationId,omitempty"`
	SourceAttemptID  *uuid.UUID `json:"sourceAttemptId,omitempty"`
	ChildExecutionID *uuid.UUID `json:"childExecutionId,omitempty"`
	ErrorCode        string     `json:"errorCode,omitempty"`
	RawOutput        string     `json:"rawOutput,omitempty"`
	ValidationError  string     `json:"validationError,omitempty"`
	Version          int64      `json:"version"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	// ProfileIdentity is a derived operator projection from the pinned
	// definition. It is never persisted as a second source of authored truth.
	ProfileIdentity string `json:"-"`
}

type WorkflowJournalKind string

const (
	WorkflowJournalInput       WorkflowJournalKind = "workflow_input"
	WorkflowJournalAttempt     WorkflowJournalKind = "node_attempt"
	WorkflowJournalRunResult   WorkflowJournalKind = "run_result"
	WorkflowJournalStructured  WorkflowJournalKind = "structured_result"
	WorkflowJournalHandoff     WorkflowJournalKind = "final_handoff"
	WorkflowJournalSignal      WorkflowJournalKind = "signal"
	WorkflowJournalCounter     WorkflowJournalKind = "counter"
	WorkflowJournalWait        WorkflowJournalKind = "wait"
	WorkflowJournalWaitTimeout WorkflowJournalKind = "wait_timeout"
	WorkflowJournalCancel      WorkflowJournalKind = "cancel"
	WorkflowJournalRetry       WorkflowJournalKind = "retry"
	WorkflowJournalResume      WorkflowJournalKind = "resume"
	WorkflowJournalChild       WorkflowJournalKind = "child_workflow"
	WorkflowJournalJoin        WorkflowJournalKind = "join"
	WorkflowJournalCleanup     WorkflowJournalKind = "cleanup"
	// WorkflowJournalDiagnostic records deterministic binding clamps and
	// evictions without placing the diagnostic in prompt content alone.
	WorkflowJournalDiagnostic WorkflowJournalKind = "binding_diagnostic"
)

type WorkflowJournalEntry struct {
	ID          uuid.UUID           `json:"id"`
	ExecutionID uuid.UUID           `json:"executionId"`
	Sequence    int64               `json:"sequence"`
	Kind        WorkflowJournalKind `json:"kind"`
	NodeID      string              `json:"nodeId,omitempty"`
	AttemptID   *uuid.UUID          `json:"attemptId,omitempty"`
	Payload     json.RawMessage     `json:"payload"`
	CreatedAt   time.Time           `json:"createdAt"`
}

// WorkflowLifecycleEvent is the safe broadcast projection of a committed
// journal transition. Content bodies intentionally remain in durable storage.
type WorkflowLifecycleEvent struct {
	ExecutionID          uuid.UUID
	DefinitionDigest     string
	Status               WorkflowExecutionStatus
	NodeID               string
	Strategy             WorkflowAttemptStrategy
	ProfileIdentity      string
	RunID                *uuid.UUID
	ConversationID       string
	SourceAttemptID      *uuid.UUID
	JournalSequence      int64
	JournalKind          WorkflowJournalKind
	JournalPayloadDigest string
	BudgetUsage          WorkflowBudgetUsage
	TerminalReason       *WorkflowTerminalReason
}
