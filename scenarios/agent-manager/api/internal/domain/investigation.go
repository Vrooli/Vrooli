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
const DefaultInvestigationPromptTemplate = `# Agent-Manager Investigation

You are an expert debugger. Your job is to ACTIVELY INVESTIGATE why the runs provided in context failed.

**Do NOT just analyze the provided data - EXPLORE THE CODEBASE to find root causes.**

## Required Investigation Steps
1. **Read the scenario's CLAUDE.md or README.md** to understand the project structure
2. **Analyze the error messages** in the attached run events - find the actual failure
3. **Trace the error to source code** - use grep/read to find the failing code path
4. **Check related files** - look at imports, dependencies, callers, and configuration
5. **Identify the root cause** - distinguish symptoms from underlying issues

## Common Failure Patterns to Check
- **Log/Output Issues**: Large outputs breaking scanners, missing newlines, buffering problems
- **Path Issues**: Relative vs absolute paths, working directory assumptions
- **Timeout Issues**: Operations taking longer than expected
- **Tool Issues**: Missing tools, wrong tool usage, tool not trusted by agent
- **Prompt Issues**: Agent not understanding instructions, missing context
- **State Issues**: Stale data, cache invalidation, missing initialization

## How to Fetch Additional Run Data
If you need full details beyond the attachments, use the agent-manager CLI:
` + "```bash\n" + `agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>  # All events with tool calls
agent-manager run diff <run-id>    # Code changes made
` + "```\n" + `
## Required Report Format
Provide your findings in this structure:

### 1. Executive Summary
One-paragraph summary of what went wrong and why.

### 2. Root Cause Analysis
- **Primary cause** with file:line references
- **Contributing factors**
- **Evidence** (run IDs, event sequences, code snippets)

### 3. Recommendations
- **Immediate fix** (copy-pasteable code if possible)
- **Preventive measures**
- **Monitoring suggestions**`

// DefaultInvestigationSettings returns the default investigation settings.
func DefaultInvestigationSettings() *InvestigationSettings {
	return &InvestigationSettings{
		PromptTemplate: DefaultInvestigationPromptTemplate,
		DefaultDepth:   InvestigationDepthStandard,
		DefaultContext: DefaultInvestigationContextFlags(),
		UpdatedAt:      time.Now(),
	}
}
