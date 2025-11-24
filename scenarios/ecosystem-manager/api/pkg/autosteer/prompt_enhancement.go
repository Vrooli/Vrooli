package autosteer

import (
	"fmt"
	"strings"
)

// PromptEnhancer generates mode-specific prompt enhancements for Auto Steer
type PromptEnhancer struct {
	modeInstructions *ModeInstructions
}

// NewPromptEnhancer creates a new prompt enhancer
func NewPromptEnhancer(phasesDir string) *PromptEnhancer {
	return &PromptEnhancer{
		modeInstructions: NewModeInstructions(phasesDir),
	}
}

// GenerateModeSection renders a standalone section for a specific mode (no Auto Steer framing).
func (p *PromptEnhancer) GenerateModeSection(mode SteerMode) string {
	if p == nil {
		return ""
	}
	return p.renderModeContent(mode)
}

// renderModeContent returns the markdown for a mode with success criteria and tools appended.
func (p *PromptEnhancer) renderModeContent(mode SteerMode) string {
	data, err := p.modeInstructions.loadPrompt(mode)
	if err != nil {
		return fmt.Sprintf("Auto Steer instructions unavailable for %s: %v", mode, err)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(data.Instructions))

	if len(data.SuccessCriteria) > 0 {
		b.WriteString("\n\n**Success Criteria:**\n")
		for _, criterion := range data.SuccessCriteria {
			b.WriteString(fmt.Sprintf("- %s\n", criterion))
		}
	}

	if len(data.ToolRecommendations) > 0 {
		b.WriteString("\n**Recommended Tools:**\n")
		for _, tool := range data.ToolRecommendations {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	return strings.TrimSpace(b.String())
}

// GenerateAutoSteerSection generates the Auto Steer section for agent prompts
func (p *PromptEnhancer) GenerateAutoSteerSection(
	state *ProfileExecutionState,
	profile *AutoSteerProfile,
	evaluator *ConditionEvaluator,
) string {
	if state == nil || profile == nil {
		return ""
	}

	if state.CurrentPhaseIndex >= len(profile.Phases) {
		return "\n## Auto Steer Status\nAll phases completed. Task should be finalized.\n"
	}

	currentPhase := profile.Phases[state.CurrentPhaseIndex]

	var output strings.Builder

	// Mode-specific instructions first (phase markdown owns the heading)
	modeContent := p.renderModeContent(currentPhase.Mode)
	if strings.TrimSpace(modeContent) != "" {
		output.WriteString("\n")
		output.WriteString(modeContent)
		output.WriteString("\n\n")
	}

	// Phase progress
	output.WriteString("### Phase Progress\n\n")
	output.WriteString(fmt.Sprintf("Phase %d of %d\n", state.CurrentPhaseIndex+1, len(profile.Phases)))
	output.WriteString(fmt.Sprintf("Iteration: %d of %d (max)\n\n", state.CurrentPhaseIteration, currentPhase.MaxIterations))

	// Stop conditions with progress
	if len(currentPhase.StopConditions) > 0 {
		output.WriteString("**Stop Conditions:**\n\n")
		output.WriteString(p.modeInstructions.FormatConditionProgress(
			currentPhase.StopConditions,
			state.Metrics,
			evaluator,
		))
		output.WriteString("\n")
	}

	// Context from previous phases
	if len(state.PhaseHistory) > 0 {
		output.WriteString("### Completed Phases\n\n")
		output.WriteString("The following phases have been completed:\n\n")

		for i, phaseExec := range state.PhaseHistory {
			output.WriteString(fmt.Sprintf("**Phase %d: %s** (%d iterations)\n",
				i+1,
				strings.ToUpper(string(phaseExec.Mode)),
				phaseExec.Iterations,
			))

			// Show key improvements from this phase
			improvements := p.getKeyImprovements(phaseExec)
			if len(improvements) > 0 {
				output.WriteString("- Improvements: ")
				output.WriteString(strings.Join(improvements, ", "))
				output.WriteString("\n")
			}

			output.WriteString(fmt.Sprintf("- Completed: %s\n", phaseExec.StopReason))
			output.WriteString("\n")
		}
	}

	// Quality gates
	if len(profile.QualityGates) > 0 {
		output.WriteString("### Quality Gates\n\n")
		output.WriteString("The following quality gates are enforced:\n\n")

		for _, gate := range profile.QualityGates {
			status := "✓"
			if result, err := evaluator.Evaluate(gate.Condition, state.Metrics); err != nil || !result {
				status = "✗"
			}

			output.WriteString(fmt.Sprintf("%s **%s**: %s\n", status, gate.Name, gate.Message))
		}
		output.WriteString("\n")
	}

	// Important reminders
	output.WriteString("### Important Reminders\n\n")
	output.WriteString("1. Stay aligned with this phase’s focus; if you defer target advancement, state why and keep existing targets passing\n")
	output.WriteString("2. Continue in this phase until stop conditions are met or max iterations reached\n")
	output.WriteString("3. Protect operational targets and passing tests—no regressions\n")
	output.WriteString("4. Document changes and reasoning for the next phase\n")
	output.WriteString("\n")

	return output.String()
}

// getKeyImprovements extracts significant improvements from a phase execution
func (p *PromptEnhancer) getKeyImprovements(phaseExec PhaseExecution) []string {
	improvements := []string{}

	// Calculate deltas
	deltas := calculateMetricDeltas(phaseExec.StartMetrics, phaseExec.EndMetrics)

	// Define significant thresholds for different metrics
	thresholds := map[string]float64{
		"operational_targets_percentage": 5.0,  // 5% improvement
		"accessibility_score":            10.0, // 10 points
		"ui_test_coverage":               10.0,
		"unit_test_coverage":             10.0,
		"integration_test_coverage":      10.0,
		"tidiness_score":                 5.0,
		"cyclomatic_complexity_avg":      -2.0, // Reduction is good
	}

	for metric, delta := range deltas {
		threshold, hasThreshold := thresholds[metric]
		if !hasThreshold {
			continue
		}

		// Check if improvement meets threshold
		isSignificant := false
		if metric == "cyclomatic_complexity_avg" {
			// Lower is better for complexity
			isSignificant = delta <= threshold
		} else {
			isSignificant = delta >= threshold
		}

		if isSignificant {
			improvements = append(improvements, p.formatImprovement(metric, delta))
		}
	}

	return improvements
}

// formatImprovement formats a metric improvement for display
func (p *PromptEnhancer) formatImprovement(metric string, delta float64) string {
	// Format metric name nicely
	metricName := strings.ReplaceAll(metric, "_", " ")
	metricName = strings.Title(metricName)

	// Format delta with sign and appropriate precision
	sign := "+"
	if delta < 0 {
		sign = ""
	}

	// Special handling for complexity (lower is better)
	if metric == "cyclomatic_complexity_avg" {
		if delta < 0 {
			return fmt.Sprintf("%s reduced by %.1f", metricName, -delta)
		}
		return fmt.Sprintf("%s increased by %.1f", metricName, delta)
	}

	return fmt.Sprintf("%s %s%.1f", metricName, sign, delta)
}

// GeneratePhaseTransitionMessage generates a message for phase transitions
func (p *PromptEnhancer) GeneratePhaseTransitionMessage(
	oldPhase SteerPhase,
	newPhase SteerPhase,
	phaseNumber int,
	totalPhases int,
) string {
	return fmt.Sprintf(`
## Phase Transition

You have completed the **%s** phase and are now entering the **%s** phase.

**Progress:** Phase %d of %d

**Previous Phase Summary:**
- Mode: %s
- Focus: Completed as planned

**New Phase Focus:**
%s

Continue building on the work from previous phases while focusing on the new objectives.
`,
		strings.ToUpper(string(oldPhase.Mode)),
		strings.ToUpper(string(newPhase.Mode)),
		phaseNumber,
		totalPhases,
		strings.ToUpper(string(oldPhase.Mode)),
		p.modeInstructions.GetInstructions(newPhase.Mode),
	)
}

// GenerateCompletionMessage generates a message when all phases are complete
func (p *PromptEnhancer) GenerateCompletionMessage(profile *AutoSteerProfile, state *ProfileExecutionState) string {
	var output strings.Builder

	output.WriteString("\n## Auto Steer Profile Complete!\n\n")
	output.WriteString(fmt.Sprintf("All %d phases of the **%s** profile have been successfully completed.\n\n",
		len(profile.Phases),
		profile.Name,
	))

	output.WriteString("### Phase Summary\n\n")
	for i, phaseExec := range state.PhaseHistory {
		output.WriteString(fmt.Sprintf("**Phase %d: %s**\n",
			i+1,
			strings.ToUpper(string(phaseExec.Mode)),
		))
		output.WriteString(fmt.Sprintf("- Iterations: %d\n", phaseExec.Iterations))
		output.WriteString(fmt.Sprintf("- Stop Reason: %s\n", phaseExec.StopReason))

		improvements := p.getKeyImprovements(phaseExec)
		if len(improvements) > 0 {
			output.WriteString("- Key Improvements:\n")
			for _, improvement := range improvements {
				output.WriteString(fmt.Sprintf("  - %s\n", improvement))
			}
		}
		output.WriteString("\n")
	}

	output.WriteString("### Next Steps\n\n")
	output.WriteString("The scenario has been improved across multiple dimensions. ")
	output.WriteString("Consider:\n")
	output.WriteString("1. Final validation that all operational targets pass\n")
	output.WriteString("2. Manual testing of critical user flows\n")
	output.WriteString("3. Review of code changes and documentation\n")
	output.WriteString("4. Preparation for deployment or handoff\n\n")

	return output.String()
}
