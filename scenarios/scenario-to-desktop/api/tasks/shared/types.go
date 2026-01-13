package shared

import (
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
)

// TaskInput provides all data needed to build a prompt.
// This is shared between task handlers and the service.
type TaskInput struct {
	// Task is the investigation record (contains task configuration).
	Task *domain.Investigation

	// Pipeline is the pipeline run being investigated/fixed.
	Pipeline *pipeline.Status

	// Request contains the original task request with configuration.
	Request *domain.CreateTaskRequest

	// For fix tasks: the source investigation's findings.
	SourceFindings *string

	// For fix iterations: the current iteration number.
	Iteration int

	// For fix iterations: results from previous iterations.
	PreviousIterations []domain.FixIterationRecord
}

// PromptResult contains the generated prompt and context attachments.
type PromptResult struct {
	// Prompt is the base prompt text sent to the agent.
	Prompt string

	// Attachments are the context attachments for the agent.
	Attachments []*domainpb.ContextAttachment
}

// AgentResult contains the result from agent execution.
type AgentResult struct {
	// RunID is the agent-manager run ID.
	RunID string

	// Success indicates if the run completed successfully.
	Success bool

	// Output is the agent's output/findings.
	Output string

	// ErrorMessage contains error details if the run failed.
	ErrorMessage string

	// DurationSeconds is the total execution time.
	DurationSeconds int

	// TokensUsed is the number of tokens consumed.
	TokensUsed int32

	// CostEstimate is the estimated cost of the run.
	CostEstimate float64
}

// Fix loop termination statuses.
const (
	FixStatusSuccess       = "success"
	FixStatusMaxIterations = "max_iterations"
	FixStatusAgentGaveUp   = "agent_gave_up"
	FixStatusUserStopped   = "user_stopped"
	FixStatusTimeout       = "timeout"
)
