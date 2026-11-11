package rehearsals

import (
	"context"
	"fmt"
	"time"
)

type State string

const (
	StateNeedsReview      State = "needs_review"
	StateApproved         State = "approved"
	StateChangesRequested State = "changes_requested"
	StateFailed           State = "failed"
)

type Decision string

const (
	DecisionApprove        Decision = "approve"
	DecisionRequestChanges Decision = "request_changes"
)

type Step struct {
	ID         string         `json:"id"`
	Operation  string         `json:"operation"`
	NodeType   string         `json:"node_type,omitempty"`
	Resource   string         `json:"resource,omitempty"`
	Action     string         `json:"action,omitempty"`
	Method     string         `json:"method"`
	Host       string         `json:"host"`
	URL        string         `json:"url"`
	Input      map[string]any `json:"input"`
	InputMatch map[string]any `json:"input_match,omitempty"`
}
type (
	Workflow struct {
		ID                 string               `json:"id"`
		Name               string               `json:"name"`
		Nodes              []WorkflowNode       `json:"nodes"`
		Connections        []Connection         `json:"connections"`
		EntryNodeIDs       []string             `json:"entry_node_ids"`
		SubworkflowEntryID string               `json:"subworkflow_entry_id,omitempty"`
		ErrorNodeID        string               `json:"error_node_id,omitempty"`
		ErrorWorkflowID    string               `json:"error_workflow_id,omitempty"`
		Subworkflows       map[string]*Workflow `json:"subworkflows,omitempty"`
		TriggerKinds       []TriggerKind        `json:"trigger_kinds"`
	}
	WorkflowNode struct {
		ID                          string            `json:"id"`
		Name                        string            `json:"name"`
		Type                        string            `json:"type"`
		Trigger                     TriggerKind       `json:"trigger,omitempty"`
		Request                     *Step             `json:"request,omitempty"`
		Condition                   *Condition        `json:"condition,omitempty"`
		FilterCondition             *Condition        `json:"filter_condition,omitempty"`
		SwitchRules                 []SwitchRule      `json:"switch_rules,omitempty"`
		OutputPath                  string            `json:"output_path,omitempty"`
		SetValues                   map[string]any    `json:"set_values,omitempty"`
		SetProjections              map[string]string `json:"set_projections,omitempty"`
		SetExpressions              map[string]string `json:"set_expressions,omitempty"`
		SetIncludeInput             bool              `json:"set_include_input,omitempty"`
		SplitField                  string            `json:"split_field,omitempty"`
		SubworkflowID               string            `json:"subworkflow_id,omitempty"`
		SafeCode                    string            `json:"safe_code,omitempty"`
		AggregateFields             []AggregateField  `json:"aggregate_fields,omitempty"`
		AggregateKeepMissing        bool              `json:"aggregate_keep_missing,omitempty"`
		AggregateMergeLists         bool              `json:"aggregate_merge_lists,omitempty"`
		AggregateDisableDotNotation bool              `json:"aggregate_disable_dot_notation,omitempty"`
		WaitSeconds                 uint32            `json:"wait_seconds,omitempty"`
		WaitAmountPath              string            `json:"wait_amount_path,omitempty"`
		WaitUnit                    string            `json:"wait_unit,omitempty"`
		LoopBatchSize               uint32            `json:"loop_batch_size,omitempty"`
		RSSURLPath                  string            `json:"rss_url_path,omitempty"`
		RequestReplacesInput        bool              `json:"request_replaces_input,omitempty"`
		ErrorOutput                 bool              `json:"error_output,omitempty"`
	}
	SwitchRule struct {
		Condition  Condition `json:"condition"`
		OutputPort int       `json:"output_port"`
	}
	Connection struct {
		From       string `json:"from"`
		To         string `json:"to"`
		OutputPort int    `json:"output_port"`
		InputPort  int    `json:"input_port"`
	}
	Condition struct {
		Path     string `json:"path"`
		Operator string `json:"operator"`
		Value    any    `json:"value"`
	}
	TriggerKind string

	TraceEvent struct {
		Sequence        uint32    `json:"sequence"`
		Operation       string    `json:"operation"`
		RequestSummary  string    `json:"request_summary"`
		ResponseSummary string    `json:"response_summary"`
		Status          string    `json:"status"`
		OccurredAt      time.Time `json:"occurred_at"`
	}
)

type AggregateField struct {
	InputPath  string `json:"input_path"`
	OutputName string `json:"output_name"`
}

const maxSyntheticItems = 1000

const (
	TriggerManual      TriggerKind = "manual"
	TriggerWebhook     TriggerKind = "webhook"
	TriggerScheduled   TriggerKind = "scheduled"
	TriggerPolling     TriggerKind = "polling"
	TriggerChat        TriggerKind = "chat"
	TriggerError       TriggerKind = "error"
	TriggerSubworkflow TriggerKind = "subworkflow"
)

type Rehearsal struct {
	ID                     string
	Title                  string
	SourceFormat           string
	SourceSHA256           string
	CredentialReplacements uint32
	SimulatorProfile       string
	SimulatorSourceSHA256  string
	State                  State
	Trace                  []TraceEvent
	FaultSummary           string
	RepairSummary          string
	HandoffMarkdown        string
	ReviewDecision         Decision
	ReviewNote             string
	CreatedAt              time.Time
	ReviewedAt             *time.Time
}
type CreateInput struct {
	Title               string
	WorkflowJSON        string
	TriggerNodeID       string
	SimulatorProfile    string
	SimulatorSourceJSON string
	InjectSeededFault   bool
	RequiredOutputPaths []string
	RepairOfID          string
}
type ReviewInput struct {
	ID       string
	Decision Decision
	Note     string
}

type Repository interface {
	Create(context.Context, Rehearsal) (Rehearsal, error)
	Get(context.Context, string) (Rehearsal, error)
	List(context.Context, int) ([]Rehearsal, error)
	RecordReview(context.Context, string, Decision, string, time.Time) (Rehearsal, error)
}
type Engine interface {
	Execute(context.Context, Workflow, string, Simulator, bool, time.Time) ([]TraceEvent, error)
}
type DetailedEngine interface {
	ExecuteDetailed(context.Context, Workflow, string, Simulator, bool, time.Time) (ExecutionResult, error)
}
type ExecutionResult struct {
	Trace  []TraceEvent
	Output map[string]any
}
type Simulator interface {
	Invoke(context.Context, Step, uint32, bool) (string, error)
}

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("rehearsal %q not found", e.ID) }

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrConflict struct{ Reason string }

func (e ErrConflict) Error() string { return e.Reason }
