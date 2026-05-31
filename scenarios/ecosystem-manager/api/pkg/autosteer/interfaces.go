package autosteer

// This file defines internal component interfaces for dependency injection and testing.
// These interfaces enable unit testing of the controller without database or I/O.

// PromptEnhancerAPI defines the contract for generating controller prompt sections.
// Enables unit testing of ExecutionOrchestrator without filesystem access.
type PromptEnhancerAPI interface {
	// GenerateSkillSetSection renders a standalone section for a specific skill set.
	GenerateSkillSetSection(skillIDs []string, withScope bool, scope string) string

	// GenerateControllerSection generates the full controller section for agent
	// prompts: the selected skill's instructions plus the objective context.
	GenerateControllerSection(state *ProfileExecutionState, profile *AutoSteerProfile) string
}

// Compile-time interface assertion
var _ PromptEnhancerAPI = (*PromptEnhancer)(nil)
