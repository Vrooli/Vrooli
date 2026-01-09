// Package domain provides investigation-related domain types.
package domain

import "time"

// =============================================================================
// INVESTIGATION SETTINGS
// =============================================================================

// InvestigationSettings holds configuration for investigation agents.
// This is a singleton - only one row exists in the database.
type InvestigationSettings struct {
	// PromptTemplate is the base instruction sent to investigation agents.
	// Plain text with NO variables/templating - all dynamic content is provided
	// as separate context attachments.
	PromptTemplate string `json:"promptTemplate" db:"prompt_template"`

	// DefaultDepth is the default investigation depth: "quick", "standard", or "deep"
	DefaultDepth InvestigationDepth `json:"defaultDepth" db:"default_depth"`

	// DefaultContext defines which context types are included by default
	DefaultContext InvestigationContextFlags `json:"defaultContext" db:"default_context"`

	// UpdatedAt is when these settings were last modified
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// InvestigationDepth controls how thorough the investigation should be.
type InvestigationDepth string

const (
	// InvestigationDepthQuick performs rapid analysis with minimal exploration.
	InvestigationDepthQuick InvestigationDepth = "quick"
	// InvestigationDepthStandard performs balanced analysis with targeted exploration.
	InvestigationDepthStandard InvestigationDepth = "standard"
	// InvestigationDepthDeep performs thorough exploration of the codebase.
	InvestigationDepthDeep InvestigationDepth = "deep"
)

// IsValid checks if the investigation depth is valid.
func (d InvestigationDepth) IsValid() bool {
	switch d {
	case InvestigationDepthQuick, InvestigationDepthStandard, InvestigationDepthDeep, "":
		return true
	default:
		return false
	}
}

// InvestigationContextFlags defines which context types to include in investigations.
type InvestigationContextFlags struct {
	// RunSummaries includes run summary data (always lightweight)
	RunSummaries bool `json:"runSummaries"`
	// RunEvents includes run events (can be large but essential for debugging)
	RunEvents bool `json:"runEvents"`
	// RunDiffs includes code changes made during runs
	RunDiffs bool `json:"runDiffs"`
	// ScenarioDocs includes scenario documentation (CLAUDE.md, README)
	ScenarioDocs bool `json:"scenarioDocs"`
	// FullLogs includes full run logs (can be very large)
	FullLogs bool `json:"fullLogs"`
}

// DefaultInvestigationContextFlags returns the default context flags.
func DefaultInvestigationContextFlags() InvestigationContextFlags {
	return InvestigationContextFlags{
		RunSummaries: true,
		RunEvents:    true,
		RunDiffs:     true,
		ScenarioDocs: true,
		FullLogs:     false,
	}
}

// =============================================================================
// DETECTED SCENARIO
// =============================================================================

// DetectedScenario represents a scenario detected from run data.
// Used when presenting investigation creation options to the user.
type DetectedScenario struct {
	// Name is the scenario name (e.g., "agent-manager")
	Name string `json:"name"`
	// ProjectRoot is the full path to the scenario
	ProjectRoot string `json:"projectRoot"`
	// KeyFiles are important files found in the scenario (CLAUDE.md, README.md, etc.)
	KeyFiles []string `json:"keyFiles"`
	// RunCount is how many of the selected runs are from this scenario
	RunCount int `json:"runCount"`
}

// =============================================================================
// DEFAULT PROMPT TEMPLATE
// =============================================================================

// DefaultInvestigationPromptTemplate is the default prompt for investigation agents.
// This template focuses on BEHAVIORAL ANALYSIS of the agent, not technical debugging.
// Dynamic data (depth, runs, scenarios) is provided as separate context attachments.
const DefaultInvestigationPromptTemplate = `# Agent-Manager Investigation

## CRITICAL: What You Are Investigating
**You are investigating THE AGENT'S BEHAVIOR, NOT the technical problem it was working on.**

Think of yourself as an internal affairs investigator, not a detective trying to solve the same case.
- ✅ CORRECT: "The agent made a bad decision when it tried to read a 50MB log file"
- ❌ WRONG: "The bufio.Scanner buffer limit is too small" (this is the problem the agent was investigating)

**DO NOT:**
- Continue or redo the task the failed agent was working on
- Investigate the same technical problem the agent was investigating
- Try to fix code issues the agent discovered (that's for a separate apply run)
- Reproduce actions that caused the agent to fail (you might fail the same way!)

**DO:**
- Analyze the agent's decision-making process
- Identify where the agent got confused or made mistakes
- Find patterns in agent behavior that led to failure
- Recommend prompt/instruction improvements to prevent similar failures

## Your Mission
You are an expert in AI agent behavior analysis. Your job is to understand WHY THE AGENT BEHAVED THE WAY IT DID.
Analyze the agent's tool usage, reasoning, and decision points to identify behavioral failures.

## Required Investigation Steps
1. **Review the run events chronologically** - understand the sequence of agent actions
2. **Identify the decision point** - find where the agent made a choice that led to failure
3. **Analyze the agent's reasoning** - look at what information the agent had and what it concluded
4. **Check for confusion signals** - repeated attempts, backtracking, tool misuse, ignoring instructions
5. **Identify behavioral root cause** - why did the agent make this decision? Missing context? Bad prompt? Misunderstanding?

## Common Agent Behavioral Failures
- **Scope Creep**: Agent went beyond its assigned task or investigated the wrong thing
- **Dangerous Reproduction**: Agent reproduced an action that broke a previous agent (self-harm)
- **Tool Misuse**: Agent used tools incorrectly (unbounded searches, reading huge files, etc.)
- **Instruction Blindness**: Agent ignored explicit warnings or instructions in its prompt
- **Context Confusion**: Agent confused what it was supposed to investigate vs. what it was given as data
- **Infinite Loops**: Agent kept retrying the same failing approach
- **Missing Safeguards**: Agent didn't check file sizes, output lengths, or resource limits before acting

## ⚠️ SAFETY WARNING
**DO NOT reproduce actions from the failed run's event log.** If the agent failed because it:
- Read a huge file → DO NOT read that file yourself
- Ran an unbounded search → DO NOT run that search yourself
- Made an API call that hung → DO NOT make that call yourself

You can ANALYZE the events without REPRODUCING them. Look at what happened, don't re-do it.

## How to Fetch Additional Run Data
If you need full details beyond the attachments, use the agent-manager CLI:
` + "```bash\n" + `agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>   # All events with tool calls
agent-manager run diff <run-id>     # Code changes made
` + "```\n" + `
## Required Report Format
Provide your findings in this structure:

### 1. Executive Summary
One-paragraph summary of what behavioral mistake the agent made and why.

### 2. Behavioral Analysis
- **Decision Timeline**: Key decision points where the agent went wrong (cite event sequence numbers)
- **Root Behavioral Cause**: Why did the agent make this decision? (e.g., ambiguous instructions, missing guardrails, context confusion)
- **Contributing Factors**: What else made this failure more likely?

### 3. Recommendations
- **Prompt/Instruction Changes**: How should the agent's instructions be modified?
- **Guardrails to Add**: What safety checks should be added to prevent similar failures?
- **Training Data**: Should this failure pattern be documented for future reference?`

// DefaultInvestigationSettings returns the default investigation settings.
func DefaultInvestigationSettings() *InvestigationSettings {
	return &InvestigationSettings{
		PromptTemplate: DefaultInvestigationPromptTemplate,
		DefaultDepth:   InvestigationDepthStandard,
		DefaultContext: DefaultInvestigationContextFlags(),
		UpdatedAt:      time.Now(),
	}
}
