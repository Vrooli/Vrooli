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

// DefaultInvestigationPromptTemplate is the fallback prompt for investigation agents
// when prompt-manager is unavailable. Keep in sync with the SKILL.md file:
// scenarios/prompt-manager/store/skills/packs/core/agent-manager-process-investigation/SKILL.md
const DefaultInvestigationPromptTemplate = `## Investigation

Diagnose why the attached agent run(s) failed. Produce a structured report classifying root causes and recommending specific, actionable fixes.

### What you have

Context attachments below contain the key data — read them before exploring further:
- **Run overview**: status, timing, task description, runner configuration
- **Event timeline**: chronological tool calls, reasoning, and results
- **Agent setup paths**: file paths to the agent's prompt-manager configuration
- **Historical context**: recent runs with the same agent profile (success/fail patterns)
- **Run diff**: code changes made during the run (if any)

### What to do

1. Read the attached timeline and overview to understand what happened
2. Identify where things went wrong — errors, loops, wrong approaches, stalls
3. Classify each failure as one or both of:
   - **Environment/Tooling**: tools broken/missing/misconfigured, config errors, services down, wrong versions, permission issues
   - **Agent Setup**: prompt unclear or contradictory, missing guardrails, wrong tools listed, insufficient context, scope confusion
   - When a run identity is refused a lifecycle operation that the task explicitly required, classify **Both**: the server-side denial is intentional and the task must move the lifecycle action to an operator context.
4. If both apply, investigate Environment/Tooling first — a broken environment makes prompt analysis unreliable
5. For each finding: cite specific evidence (event numbers, file contents, command outputs), assess severity, and recommend a concrete fix naming the specific file and change needed

### Exploration

- Read agent prompt/instruction files listed in the agent-setup attachment
- Run diagnostic commands (which, version checks) to verify tools the agent tried to use
- Check configs and files referenced in error messages
- For standard/deep depth: do targeted exploration of the primary failure category
- For deep depth: thoroughly investigate all applicable categories
- **Do NOT modify any files** — investigation is read-only
- If a run identity was refused an operator-only lifecycle operation, do not retry it. Report that the task must move the operation to an operator context and investigate the task/guardrail mismatch.

### Output format

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
| ID | Finding | Evidence | Severity | Recommendation |
|---|---|---|---|---|
| E1 | ... | ... | ... | ... |

## Agent Setup Findings
| ID | Finding | Evidence | Severity | Recommendation |
|---|---|---|---|---|
| A1 | ... | ... | ... | ... |

## Recommendations Summary
| Priority | ID | Category | Recommendation | Expected Impact |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Risks and Caveats
- ...
` + "```\n\nIf a category has no findings, include the header with \"No findings in this category.\""

// DefaultApplyInvestigationPromptTemplate is the fallback prompt for apply agents
// when prompt-manager is unavailable. Keep in sync with the SKILL.md file:
// scenarios/prompt-manager/store/skills/packs/core/agent-manager-process-investigation-apply/SKILL.md
const DefaultApplyInvestigationPromptTemplate = `## Apply Investigation Fixes

Implement the recommendations from the attached investigation report. Produce a change report documenting what was applied, verified, and deferred.

### What you have

Context attachments contain the investigation report (in the investigation run's events/summary), the original failed run data, and any user-provided guidance.

### What to do

1. Find the investigation report in the attached investigation run events (look for the structured markdown report in the final assistant message)
2. Extract all recommendations, noting their category and priority
3. Apply **Environment/Tooling fixes first**, then Agent Setup fixes (each in priority order)
4. For each fix:
   - Read the target file to understand its current state
   - Make the minimal change needed
   - Verify: configs parse, paths exist, prompts are internally consistent
5. After all fixes: check that Environment/Tooling changes don't conflict with Agent Setup changes

### Rules

- Only implement recommendations from the investigation report — no extras
- All changes must be git-revertible
- Don't remove existing safety checks unless the investigation explicitly recommends it with justification
- If a recommendation is ambiguous, implement the narrower interpretation
- If a fix causes a new problem, stop and document it — don't "fix the fix"

### Output format

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
` + "```\n\nIf a category has no changes, include the header with \"No changes in this category.\""

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
