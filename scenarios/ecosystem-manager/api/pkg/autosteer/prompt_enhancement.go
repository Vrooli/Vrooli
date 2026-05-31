package autosteer

import (
	"fmt"
	"sort"
	"strings"
)

// AutoSteerPlaceholder is replaced by the steering section when present in templates.
const AutoSteerPlaceholder = "{{AUTO_STEER_SECTION}}"

// InjectSteeringSection inserts the steering block at the placeholder if present, otherwise appends it.
func InjectSteeringSection(prompt string, steeringSection string) string {
	trimmed := strings.TrimSpace(steeringSection)

	if strings.Contains(prompt, AutoSteerPlaceholder) {
		return strings.ReplaceAll(prompt, AutoSteerPlaceholder, trimmed)
	}

	if trimmed == "" {
		return strings.ReplaceAll(prompt, AutoSteerPlaceholder, "")
	}

	if strings.TrimSpace(prompt) == "" {
		return trimmed
	}

	return strings.TrimRight(prompt, "\n") + "\n\n" + trimmed
}

// PromptEnhancer renders the controller's prompt section: the selected skill's
// instructions plus the objective/diagnosis context.
type PromptEnhancer struct {
	promptLoader *PromptLoader
}

// NewPromptEnhancer creates a new prompt enhancer.
// Does not fail if prompt-manager unavailable - operates in degraded mode.
func NewPromptEnhancer() *PromptEnhancer {
	return &PromptEnhancer{
		promptLoader: NewPromptLoader(nil),
	}
}

// IsAvailable returns whether prompt-manager is reachable.
func (p *PromptEnhancer) IsAvailable() bool {
	if p == nil || p.promptLoader == nil {
		return false
	}
	return p.promptLoader.IsAvailable()
}

// GetPromptLoader returns the underlying prompt loader for external access.
func (p *PromptEnhancer) GetPromptLoader() *PromptLoader {
	if p == nil {
		return nil
	}
	return p.promptLoader
}

// GenerateSkillSetSection renders a standalone section for a steering skill set.
// XML content is sourced from prompt-manager via api-core discovery through PromptLoader.
func (p *PromptEnhancer) GenerateSkillSetSection(skillIDs []string, withScope bool, scope string) string {
	if p == nil {
		return ""
	}
	if len(skillIDs) == 0 {
		return ""
	}
	combined, err := p.promptLoader.ReadSkillsWithScope(skillIDs, withScope, scope)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(combined)
}

// GenerateControllerSection renders the controller's steering block: the
// currently selected skill's instructions, why it was chosen, and the open
// diagnosis the agent should close.
func (p *PromptEnhancer) GenerateControllerSection(state *ProfileExecutionState, profile *AutoSteerProfile) string {
	if state == nil || profile == nil {
		return ""
	}

	if strings.TrimSpace(state.CurrentSkill) == "" {
		return "\n## Auto Steer Status\nThe objective is met — no actionable findings remain. Finalize the task.\n"
	}

	var output strings.Builder

	// Selected skill instructions first.
	skillContent := p.GenerateSkillSetSection([]string{state.CurrentSkill}, true, "")
	if strings.TrimSpace(skillContent) != "" {
		output.WriteString("\n")
		output.WriteString(skillContent)
		output.WriteString("\n\n")
	}

	output.WriteString("### Controller Decision\n\n")
	output.WriteString(fmt.Sprintf("Iteration %d of %d (max). Profile objective: **%s**.\n\n",
		state.Iteration, profile.Budget.MaxIterations, profile.Name))
	if targets := objectiveTargetSummary(profile); targets != "" {
		output.WriteString(fmt.Sprintf("Done when: %s.\n\n", targets))
	}
	output.WriteString(fmt.Sprintf("Selected skill **%s** — %s\n\n", state.CurrentSkill, state.CurrentRationale))

	// Open diagnosis, heaviest dimensions first.
	if len(state.Findings.DimensionScore) > 0 {
		output.WriteString("**Open findings by dimension (weighted score):**\n\n")
		for _, dim := range state.Findings.HeaviestDimensions() {
			output.WriteString(fmt.Sprintf("- %s: %.1f (%d findings)\n",
				dim, state.Findings.DimensionScore[dim], state.Findings.DimensionCount[dim]))
		}
		output.WriteString("\n")
	} else {
		output.WriteString("No open findings recorded.\n\n")
	}

	output.WriteString("### Important Reminders\n\n")
	output.WriteString("1. Focus this run on closing the selected dimension's findings; do not regress others\n")
	output.WriteString("2. Protect operational targets and passing tests — no regressions\n")
	output.WriteString("3. The controller will re-audit after this run and choose the next skill from the result\n\n")

	return output.String()
}

// PreviewControllerSection renders a static, audit-free preview of what the
// controller will do for a profile: its objective targets and the allowed-skill
// set it will select from. Used by the prompt-preview endpoint, which must not
// trigger a live test-genie audit.
func PreviewControllerSection(profile *AutoSteerProfile) string {
	if profile == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Auto Steer (controller preview)\n\n")
	b.WriteString(fmt.Sprintf("Objective profile **%s** — the controller will audit the target, then iteratively select the skill that best closes the heaviest open finding cluster.\n\n", profile.Name))
	if targets := objectiveTargetSummary(profile); targets != "" {
		b.WriteString(fmt.Sprintf("Done when: %s.\n\n", targets))
	}
	if len(profile.AllowedSkills) > 0 {
		b.WriteString("Eligible skills: ")
		b.WriteString(strings.Join(profile.AllowedSkills, ", "))
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("Budget: up to %d iterations.\n", profile.Budget.MaxIterations))
	return b.String()
}

// objectiveTargetSummary renders a profile's targets for display.
func objectiveTargetSummary(profile *AutoSteerProfile) string {
	if profile == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if sev := strings.TrimSpace(profile.Objective.Targets.MaxOpenSeverity); sev != "" {
		parts = append(parts, "max open severity "+sev)
	}
	if pct := profile.Objective.Targets.OperationalTargetsPct; pct > 0 {
		parts = append(parts, fmt.Sprintf("operational targets ≥ %.0f%%", pct))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
