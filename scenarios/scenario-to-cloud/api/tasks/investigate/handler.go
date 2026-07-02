package investigate

import (
	"context"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

// Handler implements TaskHandler for investigation tasks.
type Handler struct{}

// NewHandler creates a new investigation handler.
func NewHandler() *Handler {
	return &Handler{}
}

// TaskType returns the task type this handler processes.
func (h *Handler) TaskType() domain.TaskType {
	return domain.TaskTypeInvestigate
}

// BuildPromptAndContext creates the prompt and attachments for investigation.
func (h *Handler) BuildPromptAndContext(ctx context.Context, input shared.TaskInput) (shared.PromptResult, error) {
	return BuildPromptAndContext(input)
}

// AgentTag returns the tag for investigation runs.
func (h *Handler) AgentTag() string {
	return "scenario-to-cloud-investigation"
}

// ShouldContinue always returns false for investigations (non-iterative).
func (h *Handler) ShouldContinue(ctx context.Context, task *domain.Investigation, result *shared.AgentResult) (bool, string) {
	// Investigations are single-shot, never continue
	return false, ""
}
