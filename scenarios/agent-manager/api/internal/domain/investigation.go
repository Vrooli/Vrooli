// Package domain provides investigation-related domain types.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// =============================================================================
// INVESTIGATION TAG CONSTANTS
// =============================================================================

// Tag constants for identifying investigation runs.
// These are used across packages to identify investigation and apply runs.
const (
	// InvestigationTagPrefix is the prefix for investigation run tags.
	// Investigation runs have tags like "agent-manager-investigation" (exact match).
	InvestigationTagPrefix = "agent-manager-investigation"

	// InvestigationApplyTagSuffix is the suffix that distinguishes apply runs from investigation runs.
	// Apply runs have tags like "agent-manager-investigation-apply".
	InvestigationApplyTagSuffix = "-apply"

	// InvestigationTag is the exact tag for investigation runs (not apply).
	InvestigationTag = InvestigationTagPrefix

	// InvestigationApplyTag is the exact tag for apply investigation runs.
	InvestigationApplyTag = InvestigationTagPrefix + InvestigationApplyTagSuffix
)

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

	// ApplyPromptTemplate is the base instruction sent to apply investigation agents.
	// Plain text with NO variables/templating - dynamic content (investigation run ID,
	// CLI commands) is injected separately.
	ApplyPromptTemplate string `json:"applyPromptTemplate" db:"apply_prompt_template"`

	// DefaultDepth is the default investigation depth: "quick", "standard", or "deep"
	DefaultDepth InvestigationDepth `json:"defaultDepth" db:"default_depth"`

	// DefaultContext defines which context types are included by default
	DefaultContext InvestigationContextFlags `json:"defaultContext" db:"default_context"`
	// InvestigationTagAllowlist defines which run tags are eligible for Apply Fixes and recommendation extraction.
	InvestigationTagAllowlist []InvestigationTagRule `json:"investigationTagAllowlist" db:"investigation_tag_allowlist"`

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
	// FullLogs includes full run logs (can be very large)
	FullLogs bool `json:"fullLogs"`
}

// DefaultInvestigationContextFlags returns the default context flags.
func DefaultInvestigationContextFlags() InvestigationContextFlags {
	return InvestigationContextFlags{
		RunSummaries: true,
		RunEvents:    true,
		RunDiffs:     true,
		FullLogs:     false,
	}
}

// InvestigationTagRule defines a single allowlist rule for investigation tags.
type InvestigationTagRule struct {
	// Pattern is a glob or regex pattern used to match run tags.
	Pattern string `json:"pattern"`
	// IsRegex controls whether Pattern is treated as a regex. If false, Pattern is treated as a glob.
	IsRegex bool `json:"isRegex"`
	// CaseSensitive controls whether matching is case-sensitive.
	CaseSensitive bool `json:"caseSensitive"`
}

// DefaultInvestigationTagAllowlist returns the default tag allowlist rules.
func DefaultInvestigationTagAllowlist() []InvestigationTagRule {
	return []InvestigationTagRule{
		{
			Pattern:       "investigation",
			IsRegex:       false,
			CaseSensitive: false,
		},
		{
			Pattern:       "*-investigation",
			IsRegex:       false,
			CaseSensitive: false,
		},
	}
}

// NormalizeInvestigationTagAllowlist returns a usable allowlist (defaults if empty).
func NormalizeInvestigationTagAllowlist(rules []InvestigationTagRule) []InvestigationTagRule {
	if len(rules) == 0 {
		return DefaultInvestigationTagAllowlist()
	}
	return rules
}

// ValidateInvestigationTagAllowlist ensures regex patterns compile.
func ValidateInvestigationTagAllowlist(rules []InvestigationTagRule) error {
	for _, rule := range NormalizeInvestigationTagAllowlist(rules) {
		if strings.TrimSpace(rule.Pattern) == "" {
			continue
		}
		if rule.IsRegex {
			if _, err := compileInvestigationTagPattern(rule); err != nil {
				return err
			}
		}
	}
	return nil
}

// MatchesInvestigationTag returns true if tag matches any allowlist rule.
func MatchesInvestigationTag(tag string, rules []InvestigationTagRule) bool {
	for _, rule := range NormalizeInvestigationTagAllowlist(rules) {
		if strings.TrimSpace(rule.Pattern) == "" {
			continue
		}
		re, err := compileInvestigationTagPattern(rule)
		if err != nil {
			continue
		}
		if re.MatchString(tag) {
			return true
		}
	}
	return false
}

func compileInvestigationTagPattern(rule InvestigationTagRule) (*regexp.Regexp, error) {
	pattern := rule.Pattern
	if !rule.IsRegex {
		pattern = globToRegex(pattern)
	}
	if !rule.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid investigation tag pattern %q: %w", rule.Pattern, err)
	}
	return re, nil
}

func globToRegex(pattern string) string {
	var builder strings.Builder
	builder.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			builder.WriteString(".*")
		case '?':
			builder.WriteString(".")
		default:
			builder.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	builder.WriteString("$")
	return builder.String()
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

// DefaultApplyInvestigationPromptTemplate is the default prompt for apply investigation agents.
// This template focuses on SAFELY IMPLEMENTING fixes recommended by a prior investigation.
// Dynamic data (investigation run ID, CLI commands) is injected separately by the orchestrator.
const DefaultApplyInvestigationPromptTemplate = `# Apply Investigation Recommendations

## CRITICAL: What You Are Doing
**You are applying fixes recommended by a prior investigation run.** Your job is to implement
changes safely, verify they work, and document what you changed.

Think of yourself as a careful surgeon implementing a treatment plan, not a detective investigating the problem.
- ✅ CORRECT: "I'll modify the prompt to add a file size check as recommended"
- ❌ WRONG: "Let me investigate why the agent failed" (that's already been done)

**DO NOT:**
- Make changes beyond what the investigation recommended
- Skip verification steps
- Introduce new features or refactors not in the recommendations
- Apply fixes blindly without understanding the context
- Re-investigate the original problem (focus on implementing fixes)

**DO:**
- Review the investigation report thoroughly before making changes
- Implement only the specific fixes recommended
- Verify each change works as expected
- Document what you changed and why
- Be conservative - when in doubt, make smaller changes

## Safety Protocol

### 1. Understand Before Acting
Read the full investigation report first. Understand:
- What behavioral failure occurred
- What root cause was identified
- What specific fixes are recommended

### 2. Minimal Changes Only
Only implement what's recommended. Do not:
- Add "nice to have" improvements
- Refactor surrounding code
- Fix unrelated issues you notice
- Expand scope beyond recommendations

### 3. Verify Changes Work
After each change:
- Confirm the file was modified correctly
- Check for syntax errors
- Verify the change addresses the recommendation

### 4. Rollback Awareness
Know that your changes can be reverted via git. If something seems wrong:
- Stop and document the issue
- Don't try to "fix the fix"

## Required Steps

### Step 1: Review Investigation Report
Read the investigation report attachments carefully. Extract:
- The specific recommendations to implement
- The priority/order of recommendations
- Any warnings or caveats mentioned

### Step 2: Plan Changes
Before making any edits, list out:
- Which files need to be modified
- What specific changes each file needs
- The order of changes (dependencies)

### Step 3: Implement Each Recommendation
For each recommendation:
1. **Locate** the file(s) to modify
2. **Read** the current content to understand context
3. **Plan** the minimal change needed
4. **Implement** the change
5. **Verify** the change is correct

### Step 4: Document Changes
Create a summary of:
- What recommendations were applied
- What files were modified
- What specific changes were made
- Any recommendations NOT applied and why

## How to Fetch Investigation Data
If you need full details beyond the attachments, use the agent-manager CLI:
` + "```bash\n" + `agent-manager run get <investigation-run-id>      # Full investigation run details
agent-manager run events <investigation-run-id>   # All events from investigation
agent-manager run diff <investigation-run-id>     # Any code changes investigation made
` + "```\n" + `
## Common Fix Categories

### Prompt/Instruction Changes
- Modify CLAUDE.md or instruction files
- Add explicit warnings or constraints
- Clarify ambiguous instructions
- Add examples of correct behavior

### Guardrail Additions
- Add file size checks before reading
- Add output length limits
- Add loop detection/limits
- Add explicit scope boundaries

### Code Fixes
- Fix error handling
- Add validation
- Improve logging
- Fix resource cleanup

### Configuration Updates
- Adjust timeouts
- Modify tool permissions
- Update limits or thresholds

## ⚠️ SAFETY WARNINGS

**DO NOT:**
- Delete or overwrite large sections of code without understanding them
- Modify files outside the scenario being fixed
- Change system configuration files
- Remove existing safety checks (only add to them)
- Make changes that could affect other agents or scenarios

**IF UNSURE:**
- Document the uncertainty in your report
- Apply only the changes you're confident about
- Leave questionable recommendations for manual review

## Required Report Format

### 1. Summary
Brief overview of what was accomplished.

### 2. Changes Applied
For each change:
- **Recommendation**: Which recommendation this addresses
- **File**: Path to modified file
- **Change**: Description of what was changed
- **Verification**: How you verified it's correct

### 3. Recommendations Not Applied
For any recommendations not implemented:
- **Recommendation**: What wasn't applied
- **Reason**: Why (unclear, out of scope, requires manual review, etc.)
- **Suggestion**: What should happen next

### 4. Follow-up Actions
Any additional steps needed:
- Manual verification required
- Tests to run
- Documentation to update`

// DefaultInvestigationSettings returns the default investigation settings.
func DefaultInvestigationSettings() *InvestigationSettings {
	return &InvestigationSettings{
		PromptTemplate:            DefaultInvestigationPromptTemplate,
		ApplyPromptTemplate:       DefaultApplyInvestigationPromptTemplate,
		DefaultDepth:              InvestigationDepthStandard,
		DefaultContext:            DefaultInvestigationContextFlags(),
		InvestigationTagAllowlist: DefaultInvestigationTagAllowlist(),
		UpdatedAt:                 time.Now(),
	}
}
