// DOC: docs/concepts/ARCHITECTURE.md#review-evidence
// DOC: docs/internal/SEAMS.md#review-evidence
//
// Package review provides types and file I/O for the post-execution evidence
// gathering system. A review agent produces typed evidence (screenshots, API
// test results, CLI output captures, etc.) that helps humans decide whether to
// archive or follow up on a completed backlog item.
package review

import (
	"context"
	"encoding/json"
)

// EvidenceType enumerates the kinds of evidence a review agent can produce.
type EvidenceType string

const (
	EvidenceTypeScreenshot        EvidenceType = "screenshot"
	EvidenceTypeAPITest           EvidenceType = "api_test"
	EvidenceTypeCLIOutput         EvidenceType = "cli_output"
	EvidenceTypeConfigDiff        EvidenceType = "config_diff"
	EvidenceTypeWorkflowRecording EvidenceType = "workflow_recording"
	EvidenceTypeCustom            EvidenceType = "custom"
)

// RoundStatus tracks the review round lifecycle.
type RoundStatus string

const (
	RoundStatusPending   RoundStatus = "pending"
	RoundStatusGathering RoundStatus = "gathering"
	RoundStatusComplete  RoundStatus = "complete"
	RoundStatusFailed    RoundStatus = "failed"
)

// Round captures one pass of evidence gathering by the review agent.
type Round struct {
	RoundNum        int         `json:"round"`
	GeneratedAt     string      `json:"generated_at"`
	ExecutionID     string      `json:"execution_id"`
	Status          RoundStatus `json:"status"`
	FailureReason   string      `json:"failure_reason,omitempty"`
	AgentAssessment string      `json:"agent_assessment,omitempty"`
	Classification  string      `json:"classification,omitempty"`
	// RegressionIntroduced is set true by the review agent when
	// baseline-diff-results showed a regression this item caused that the
	// agent could not disprove. Distinct from pre-existing failures, which
	// must never set this flag.
	RegressionIntroduced bool            `json:"regression_introduced,omitempty"`
	Notes                []string        `json:"notes,omitempty"`
	Evidence             []EvidenceItem  `json:"evidence"`
	RequestThreads       []RequestThread `json:"request_threads,omitempty"`
	// ImprovementSuggestions recommends durable automations to replace one-off evidence.
	ImprovementSuggestions []ImprovementSuggestion `json:"improvement_suggestions,omitempty"`
	// Disposition is the review's bounded recommendation for the common Plan
	// Workshop loop. It is not a terminal decision and never mutates work.
	Disposition *Disposition `json:"disposition,omitempty"`
	// AgentWorkflowSnapshot is the immutable review request this round was
	// started from. It is persisted because the transition runner rebuilds the
	// input at apply time to detect mid-run edits, and the GCT results and
	// baseline diffs it contains cannot be re-derived cheaply or identically.
	AgentWorkflowSnapshot json.RawMessage `json:"agent_workflow_snapshot,omitempty"`
	// RunID is the agent-manager run ID for the review agent session.
	RunID string `json:"run_id,omitempty"`
	// CurrentRunStatus is the live agent-manager status for an in-flight review run.
	// It is populated at read time so the UI can distinguish "still gathering"
	// from "waiting in needs_review for manual approval".
	CurrentRunStatus string `json:"current_run_status,omitempty"`
	// OpWorkflowID and OpExecutionID link a runner-owned round to the declarative
	// operation execution that produced it. When OpExecutionID is set the round is
	// RUNNER-OWNED: the operation runner's completion bridge finalizes it (via the
	// commit-review-round handler), so the legacy
	// review poller defers and never re-drives it from agent-run state.
	OpWorkflowID             string `json:"op_workflow_id,omitempty"`
	OpExecutionID            string `json:"op_execution_id,omitempty"`
	AgentWorkflowExecutionID string `json:"agent_workflow_execution_id,omitempty"`
	AgentWorkflowDefinition  string `json:"agent_workflow_definition_digest,omitempty"`
	AgentWorkflowVersion     string `json:"agent_workflow_entity_version,omitempty"`
	AgentWorkflowApplyState  string `json:"agent_workflow_apply_state,omitempty"`
	AgentWorkflowAppliedAt   string `json:"agent_workflow_applied_at,omitempty"`
}

// Disposition keeps review recommendations typed and portable without giving
// a review round authority to create or apply follow-up work.
type Disposition struct {
	Kind       string `json:"kind"`
	Rationale  string `json:"rationale"`
	Confidence string `json:"confidence"`
	Scope      string `json:"scope,omitempty"`
}

// RoundTerminalObserver projects completed evidence to another operator
// surface. The round file remains the historical source for review detail.
type RoundTerminalObserver func(ctx context.Context, kind, name string, round Round)

// RunnerOwned reports whether the round's terminal transition is owned by the
// operation runner (started through the reroute) rather than the legacy poller.
func (r Round) RunnerOwned() bool {
	return r.OpExecutionID != ""
}

// WorkflowOwned reports a round whose terminal result is applied through the
// declared Agent Manager workflow boundary rather than legacy run polling.
func (r Round) WorkflowOwned() bool {
	return r.AgentWorkflowExecutionID != ""
}

// EvidenceItem is a single piece of proof that work was done correctly.
type EvidenceItem struct {
	ID                string               `json:"id"`
	Type              EvidenceType         `json:"type"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	CapturePath       string               `json:"capture_path,omitempty"`
	Verified          bool                 `json:"verified"`
	VerifiedAt        string               `json:"verified_at,omitempty"`
	BeforeCapturePath string               `json:"before_capture_path,omitempty"`
	TestResults       []EvidenceTestResult `json:"test_results,omitempty"`
}

// EvidenceTestResult is a structured test outcome within an evidence item.
type EvidenceTestResult struct {
	Name          string `json:"name"`
	Passed        bool   `json:"passed"`
	OutputSummary string `json:"output_summary,omitempty"`
}

// RequestThread is a multi-turn conversation about needing more evidence.
type RequestThread struct {
	ID         string           `json:"id"`
	EvidenceID string           `json:"evidence_id,omitempty"`
	Status     string           `json:"status"` // pending, fulfilled, dismissed
	Messages   []RequestMessage `json:"messages"`
	CreatedAt  string           `json:"created_at"`
	// RunID is the agent-manager run ID for the targeted evidence request.
	RunID                    string `json:"run_id,omitempty"`
	AgentWorkflowExecutionID string `json:"agent_workflow_execution_id,omitempty"`
	AgentWorkflowDefinition  string `json:"agent_workflow_definition_digest,omitempty"`
	AgentWorkflowVersion     string `json:"agent_workflow_entity_version,omitempty"`
	AgentWorkflowApplyState  string `json:"agent_workflow_apply_state,omitempty"`
}

// ImprovementSuggestion recommends a durable automation to replace one-off evidence.
type ImprovementSuggestion struct {
	Category    string `json:"category"` // test_coverage, visual_capture, health_check, ci_workflow, standards_rule, other
	Description string `json:"description"`
	EvidenceID  string `json:"evidence_id,omitempty"`
	Priority    string `json:"priority"` // high, medium, low
}

// RequestMessage is a single turn in a request thread.
type RequestMessage struct {
	Role             string   `json:"role"` // user, assistant
	Content          string   `json:"content"`
	Timestamp        string   `json:"timestamp"`
	AddedEvidenceIDs []string `json:"added_evidence_ids,omitempty"`
}
