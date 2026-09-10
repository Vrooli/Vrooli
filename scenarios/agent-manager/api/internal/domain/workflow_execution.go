package domain

import (
	"encoding/json"
	"fmt"
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
	// AccountingComplete is false for legacy records or any child lacking a terminal receipt.
	AccountingComplete bool `json:"accountingComplete"`
	Turns              int  `json:"turns"`
	Tokens             int  `json:"tokens"`
	// ChargeMicroUSD is authoritative marginal charge. A verified subscription
	// or local basis contributes zero; unpriced usage and historical estimates
	// cannot exhaust a monetary budget.
	ChargeMicroUSD int64 `json:"chargeMicroUsd"`
	// ChargeMeasured requires explicit child-run charge evidence: metered
	// amounts, or zero subscription/local amounts matching the saved run basis.
	// It is persisted so the API can publish an honest receipt after restart.
	ChargeMeasured bool `json:"chargeMeasured"`
	// CostUSD is retained for readable historical workflow records.
	CostUSD      float64 `json:"costUsd"`
	NodeAttempts int     `json:"nodeAttempts"`
	Children     int     `json:"children"`
	Retries      int     `json:"retries"`
}

// WorkflowEngagementGrant is an owner-issued aggregate allowance attached to
// one workflow execution. Zero leaves that dimension at the workflow's own
// declared limit. A grant can only narrow a declaration and is persisted so
// restart/recovery cannot silently restore a larger allowance.
type WorkflowEngagementGrant struct {
	MaxTurns           int   `json:"maxTurns,omitempty"`
	MaxTokens          int   `json:"maxTokens,omitempty"`
	MaxChargeMicroUSD  int64 `json:"maxChargeMicroUsd,omitempty"`
	MaxWallTimeSeconds int   `json:"maxWallTimeSeconds,omitempty"`
	MaxNodeAttempts    int   `json:"maxNodeAttempts,omitempty"`
	MaxChildren        int   `json:"maxChildren,omitempty"`
	MaxConcurrency     int   `json:"maxConcurrency,omitempty"`
	MaxRecursion       int   `json:"maxRecursion,omitempty"`
	MaxRetries         int   `json:"maxRetries,omitempty"`
	// RetryLimitSet distinguishes an explicit exhausted allowance from legacy
	// omitted zero, which inherits the workflow declaration.
	RetryLimitSet  bool `json:"retryLimitSet,omitempty"`
	MaxWaitSeconds int  `json:"maxWaitSeconds,omitempty"`
	// AllowedEffects is an owner-issued ceiling inherited by every child run.
	AllowedEffects []string `json:"allowedEffects,omitempty"`
}

func (g WorkflowEngagementGrant) Present() bool {
	return g.MaxTurns > 0 || g.MaxTokens > 0 || g.MaxChargeMicroUSD > 0 || g.MaxWallTimeSeconds > 0 ||
		g.MaxNodeAttempts > 0 || g.MaxChildren > 0 || g.MaxConcurrency > 0 || g.MaxRecursion > 0 ||
		g.MaxRetries > 0 || g.MaxWaitSeconds > 0
}

func (g WorkflowEngagementGrant) Validate() error {
	if !g.Present() || g.MaxTokens <= 0 || g.MaxWallTimeSeconds <= 0 {
		return fmt.Errorf("engagement grant requires positive max tokens and wall time")
	}
	for _, v := range []int{g.MaxTurns, g.MaxTokens, g.MaxWallTimeSeconds, g.MaxNodeAttempts, g.MaxChildren, g.MaxConcurrency, g.MaxRecursion, g.MaxRetries, g.MaxWaitSeconds} {
		if v < 0 {
			return fmt.Errorf("engagement grant limits cannot be negative")
		}
	}
	if g.MaxChargeMicroUSD < 0 {
		return fmt.Errorf("engagement grant charge limit cannot be negative")
	}
	return nil
}

type WorkflowExecution struct {
	ID                uuid.UUID                `json:"id"`
	Owner             string                   `json:"owner"`
	WorkflowKey       string                   `json:"workflowKey"`
	DefinitionDigest  string                   `json:"definitionDigest"`
	Status            WorkflowExecutionStatus  `json:"status"`
	CurrentNodeID     string                   `json:"currentNodeId"`
	Input             json.RawMessage          `json:"input"`
	Output            json.RawMessage          `json:"output,omitempty"`
	TerminalReason    *WorkflowTerminalReason  `json:"terminalReason,omitempty"`
	BudgetUsage       WorkflowBudgetUsage      `json:"budgetUsage"`
	EngagementGrant   *WorkflowEngagementGrant `json:"engagementGrant,omitempty"`
	EdgeTraversals    map[string]int           `json:"edgeTraversals"`
	Version           int64                    `json:"version"`
	IdempotencyKey    string                   `json:"idempotencyKey"`
	ParentExecutionID *uuid.UUID               `json:"parentExecutionId,omitempty"`
	ParentAttemptID   *uuid.UUID               `json:"parentAttemptId,omitempty"`
	Depth             int                      `json:"depth"`
	ApprovalDigest    string                   `json:"approvalDigest,omitempty"`
	GrantDigest       string                   `json:"grantDigest,omitempty"`
	CreatedAt         time.Time                `json:"createdAt"`
	UpdatedAt         time.Time                `json:"updatedAt"`
	EndedAt           *time.Time               `json:"endedAt,omitempty"`
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
