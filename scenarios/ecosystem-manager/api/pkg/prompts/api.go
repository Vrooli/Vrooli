package prompts

import "github.com/ecosystem-manager/api/pkg/tasks"

// AssemblerAPI abstracts prompt assembly for testability.
// Implementations must be able to:
// - Assemble prompts for tasks from configurable templates
// - Provide access to the prompts directory for phase prompt loading
type AssemblerAPI interface {
	// AssemblePromptForTask generates the complete prompt for a task
	// by loading and composing template sections.
	AssemblePromptForTask(task tasks.TaskItem) (PromptAssembly, error)

	// GetPromptsDir returns the root directory containing prompt templates.
	// Used by components that need to load phase-specific prompts.
	GetPromptsDir() string
}

// Compile-time interface assertion
var _ AssemblerAPI = (*Assembler)(nil)
