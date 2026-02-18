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
// Prompt templates are managed externally via prompt-manager skills;
// the PromptTemplate and ApplyPromptTemplate fields are populated at
// read time by the orchestration layer (not stored in the DB).
type InvestigationSettings struct {
	// PromptTemplate is the base instruction sent to investigation agents.
	// Populated from prompt-manager skill "agent-manager-process-investigation"
	// at read time; not stored in the database.
	PromptTemplate string `json:"promptTemplate"`

	// ApplyPromptTemplate is the base instruction sent to apply investigation agents.
	// Populated from prompt-manager skill "agent-manager-process-investigation-apply"
	// at read time; not stored in the database.
	ApplyPromptTemplate string `json:"applyPromptTemplate"`

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
// This template classifies failures as Environment/Tooling or Agent Setup through active exploration.
// Dynamic data (depth, runs, scenarios) is provided as separate context attachments.
const DefaultInvestigationPromptTemplate = `# Agent-Manager Process Investigation

Diagnose why an agent run failed by classifying the root cause as Environment/Tooling, Agent Setup, or both, through active codebase exploration. Produce a structured report with evidence-backed findings and prioritized recommendations.

## Failure Categories

| Category | Definition | Example Signals |
|---|---|---|
| **Environment/Tooling** | Tools, configs, paths, or services are broken, missing, or misconfigured | Tool errors, missing files, service unreachable, permission denied, command not found, wrong versions |
| **Agent Setup** | The agent's prompt, context, or instructions are malformed, insufficient, or conflicting | Agent looping on contradictory instructions, misinterpreting task scope, ignoring available tools, missing guardrails, prompt ambiguity |

When both categories apply, **investigate Environment/Tooling first** -- a broken environment makes Agent Setup analysis unreliable.

## Scope Boundaries

**In scope:** Classifying failures, exploring codebase/configs/tools, analyzing prompts for conflicts/gaps, structured recommendations.
**Out of scope:** Fixing problems (apply skill's job), completing the failed task, architecture redesign.

## Inputs

Context attachments are provided automatically (metadata, run summaries, events, diffs, custom context).

For additional data, use the agent-manager CLI:
` + "```bash\n" + `agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>   # All events with tool calls
agent-manager run diff <run-id>     # Code changes made
` + "```\n" + `
## Investigation Workflow

### Phase 1: Categorize (all depths)
1. **Build timeline**: Extract chronological sequence of agent actions from run events
2. **Extract failure signals**: Identify where progress stopped, reversed, or looped
3. **Classify signals**: Map each signal to Environment/Tooling, Agent Setup, or both
4. **Determine primary category**: Environment-blocking signal? Investigate Environment/Tooling first

### Phase 2: Deep Investigation (standard and deep only)
**Environment/Tooling**: Verify tool availability, check configs, test commands, check file existence/permissions, look for missing deps.
**Agent Setup**: Analyze prompts/instructions for conflicts, gaps, ambiguity, missing guardrails.

### Phase 3: Synthesize
For each finding: evidence, root cause, severity (Critical/Major/Gap/Minor), recommendation.

## Signal Classification Table

| Signal | Primary Category |
|---|---|
| Tool returns error/not found | Environment/Tooling |
| File missing or inaccessible | Environment/Tooling |
| Service unreachable | Environment/Tooling |
| Permission denied | Environment/Tooling |
| Config syntax invalid | Environment/Tooling |
| Agent retries same failing approach | Agent Setup |
| Agent contradicts its instructions | Agent Setup |
| Agent misinterprets task scope | Agent Setup |
| Agent ignores available tools | Agent Setup |
| Prompt contains contradictory guidance | Agent Setup |

## Safety Guardrails

**DO actively explore**: Read files (check size first -- skip files over 1MB unless essential), run diagnostic commands with timeouts, test tool availability, inspect configurations.

**DO NOT**: Modify any files or state, re-run commands that caused the original failure without safeguards, run unbounded searches without timeouts.

## Required Report Format

` + "```markdown\n" + `# Investigation Report

## Categorization Summary
- **Primary category**: [Environment/Tooling | Agent Setup | Both]
- **Confidence**: [High | Medium | Low]
- **Severity**: [Critical | Major | Gap | Minor]

## Timeline
| # | Event | Action | Result | Category Signal |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Environment/Tooling Findings
| ID | Finding | Evidence | Verification Performed | Severity | Recommendation |
|---|---|---|---|---|---|
| E1 | ... | ... | ... | ... | ... |

## Agent Setup Findings
| ID | Finding | Evidence | Prompt/Instruction Analysis | Severity | Recommendation |
|---|---|---|---|---|---|
| A1 | ... | ... | ... | ... | ... |

## Recommendations Summary
| Priority | ID | Category | Recommendation | Expected Impact |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Risks and Caveats
- ...
` + "```"

// DefaultApplyInvestigationPromptTemplate is the default prompt for apply investigation agents.
// This template implements fixes from investigation reports organized by category.
// Dynamic data (investigation run ID, CLI commands) is injected separately by the orchestrator.
const DefaultApplyInvestigationPromptTemplate = `# Apply Investigation Recommendations

Implement fixes from an investigation report, organized by category, with per-change verification.

## Scope Boundaries

**In scope:** Implementing investigation recommendations, fixing configs/paths/tools, modifying prompts/instructions, verifying fixes.
**Out of scope:** Re-investigating, adding unrecommended improvements, completing original task.

## Inputs

Context attachments are provided automatically (investigation run summary, events, original run data, custom context).

For additional data, use the agent-manager CLI:
` + "```bash\n" + `agent-manager run get <investigation-run-id>      # Full investigation run details
agent-manager run events <investigation-run-id>   # All events from investigation
agent-manager run diff <investigation-run-id>     # Any code changes investigation made
` + "```\n" + `
## Apply Workflow

### Step 1: Parse and Prioritize
Parse investigation report, extract recommendations by category, order by priority. Apply Environment/Tooling fixes first.

### Step 2: Apply Environment/Tooling Fixes
For each recommendation: read target file/config, plan minimal change, implement, verify.
Common fixes: fix config syntax/values, correct paths, create missing files, fix tool settings.

### Step 3: Apply Agent Setup Fixes
For each recommendation: read target prompt/instruction file, plan minimal change, implement, verify.
Common fixes: add guardrails, clarify instructions, resolve contradictions, add examples.

### Step 4: Cross-Category Verification
Check that Environment/Tooling fixes don't conflict with Agent Setup changes. Verify prompt changes reference tools/configs that now exist correctly.

### Step 5: Produce Change Report
Document all changes using the report format below.

## Fix Order Decision

| Situation | Action |
|---|---|
| Both categories have recommendations | Apply Environment/Tooling first, then Agent Setup |
| Only one category | Apply and verify each in priority order |
| A fix depends on another fix | Apply the dependency first regardless of category |

## Verification Decision Table

| Change Type | Verification Method |
|---|---|
| Config file change | Parse/validate the file format |
| Path correction | Verify the path exists and is accessible |
| Missing file creation | Verify file exists with expected content |
| Prompt wording change | Read the full prompt and check for internal consistency |
| Guardrail addition | Verify it doesn't conflict with existing instructions |

## Safety Guardrails

- Only implement listed recommendations
- Verify each change individually
- Don't remove existing safety checks (only add to them)
- Git-revertible changes only
- If unclear, use conservative interpretation
- If fix causes new problem, stop and document -- don't "fix the fix"

## Required Report Format

` + "```markdown\n" + `# Apply Investigation Report

## Summary
- **Recommendations received**: [count]
- **Applied successfully**: [count]
- **Not applied**: [count]
- **Verification failures**: [count]

## Environment/Tooling Changes
| ID | Recommendation | File | Change | Verification | Status |
|---|---|---|---|---|---|
| E1 | ... | ... | ... | ... | Applied |

## Agent Setup Changes
| ID | Recommendation | File | Change | Verification | Status |
|---|---|---|---|---|---|
| A1 | ... | ... | ... | ... | Applied |

## Not Applied
| ID | Recommendation | Reason |
|---|---|---|
| ... | ... | ... |

## Cross-Category Verification
- **Conflicts found**: [Yes/No]
- **Details**: ...

## Follow-Up Actions
- ...
` + "```"

// DefaultInvestigationSettings returns the default investigation settings.
// Prompt templates are populated separately by the orchestration layer from
// prompt-manager skills (with hardcoded constants as fallback).
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
