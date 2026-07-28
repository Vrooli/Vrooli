package execution

import (
	"fmt"
	"strings"

	"swarm-manager/internal/handoff"
)

// executionPromptParams holds all inputs for building a unified execution prompt.
type executionPromptParams struct {
	Kind               string // backlog item kind (idea, fix, execute, etc.)
	Name               string // backlog item name
	Title              string // human-readable title
	ItemFolder         string // absolute path to the backlog item directory
	RunType            string // process, fixup, followup, custom
	DeliverablePath    string // rendered canonical plan_ref path
	DeliverableContent string // full primary workshop artifact text (empty if missing)
	ReviewFeedback     string // review summary for fixup runs
	FollowUpNote       string // user-provided context for follow-up/custom runs
	IdeaHandoff        *handoff.Package
	SuggestedSkills    []string
}

// buildExecutionPrompt constructs a single unified prompt for all execution
// run types. The prompt uses XML tags to clearly delineate context sections.
func buildExecutionPrompt(p executionPromptParams) string {
	var b strings.Builder

	// Execution context header — always present.
	b.WriteString("<execution-context>\n")
	b.WriteString(fmt.Sprintf("Backlog item: %s/%s\n", p.Kind, p.Name))
	if strings.TrimSpace(p.Title) != "" {
		b.WriteString(fmt.Sprintf("Title: %s\n", p.Title))
	}
	b.WriteString(fmt.Sprintf("Item folder: %s\n", p.ItemFolder))
	b.WriteString(fmt.Sprintf("Run type: %s\n", p.RunType))
	b.WriteString("</execution-context>\n")

	// Suggested skills — recommended by the creating agent.
	if len(p.SuggestedSkills) > 0 {
		b.WriteString("\n<suggested-skills>\n")
		b.WriteString("Before executing, read these skills for required context:\n")
		for _, skill := range p.SuggestedSkills {
			b.WriteString(fmt.Sprintf("  prompt-manager skill read %s\n", skill))
		}
		b.WriteString("</suggested-skills>\n")
	}

	// Review feedback — only for fixup runs.
	if strings.TrimSpace(p.ReviewFeedback) != "" {
		b.WriteString("\n<review-feedback>\n")
		b.WriteString(p.ReviewFeedback)
		b.WriteString("\n</review-feedback>\n")
	}

	// Follow-up context — for follow-up and custom runs with user-provided notes.
	if strings.TrimSpace(p.FollowUpNote) != "" {
		b.WriteString("\n<follow-up-context>\n")
		b.WriteString(p.FollowUpNote)
		b.WriteString("\n</follow-up-context>\n")
	}

	// Primary workshop deliverable — always present when available.
	if strings.TrimSpace(p.DeliverableContent) != "" {
		tag := deliverablePromptTag(p.Kind)
		b.WriteString(fmt.Sprintf("\n<%s path=\"%s\">\n", tag, p.DeliverablePath))
		b.WriteString(p.DeliverableContent)
		b.WriteString(fmt.Sprintf("\n</%s>\n", tag))
	}

	if p.IdeaHandoff != nil {
		b.WriteString("\n<idea-handoff>\n")
		b.WriteString(fmt.Sprintf("Handoff directory: %s\n", p.IdeaHandoff.Dir))
		b.WriteString(fmt.Sprintf("Brief path: %s\n", p.IdeaHandoff.BriefPath))
		b.WriteString(fmt.Sprintf("Manifest path: %s\n", p.IdeaHandoff.ManifestPath))
		b.WriteString(fmt.Sprintf("Source index path: %s\n", p.IdeaHandoff.SourceIndexPath))
		b.WriteString("Treat the rendered plan as the execution contract and use brief.md only as supporting context.\n")
		b.WriteString("Record the true frontier and provenance in the swarm-manager execution handoff so later work can trace back to this item.\n")
		b.WriteString("Execute the next bounded plan slice through the declared swarm-manager workflow; do not create a second queue.\n")
		b.WriteString("</idea-handoff>\n")

		if strings.TrimSpace(p.IdeaHandoff.BriefMarkdown) != "" {
			b.WriteString(fmt.Sprintf("\n<idea-handoff-brief path=\"%s\">\n", p.IdeaHandoff.BriefPath))
			b.WriteString(p.IdeaHandoff.BriefMarkdown)
			b.WriteString("\n</idea-handoff-brief>\n")
		}
	}

	return b.String()
}

// buildFinalizationFeedback formats multi-scenario finalization output into a
// readable prompt block for fixup/follow-up runs.
func buildFinalizationFeedback(finalization *Finalization) string {
	if finalization == nil {
		return ""
	}
	var b strings.Builder
	if strings.TrimSpace(finalization.AggregateSummary) != "" {
		b.WriteString(finalization.AggregateSummary)
	}
	for _, warning := range finalization.Warnings {
		if warning.ScenarioName != "" {
			appendFeedbackLine(&b, fmt.Sprintf("- warning [%s] %s: %s", warning.Code, warning.ScenarioName, warning.Message))
		} else {
			appendFeedbackLine(&b, fmt.Sprintf("- warning [%s]: %s", warning.Code, warning.Message))
		}
	}
	for _, scenario := range finalization.Scenarios {
		appendScenarioFeedback(&b, scenario)
	}
	return b.String()
}

// appendFeedbackLine writes line to b, prefixing a newline when b already has
// content so the joined output stays line-separated.
func appendFeedbackLine(b *strings.Builder, line string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(line)
}

// appendScenarioFeedback formats the restart/health/review lines for a single
// scenario into the shared feedback builder.
func appendScenarioFeedback(b *strings.Builder, scenario ScenarioFinalization) {
	if scenario.Restart.Status != "" && scenario.Restart.Status != FinalizationStatusCompleted {
		appendFeedbackLine(b, fmt.Sprintf("- %s restart: %s", scenario.ScenarioName, scenario.Restart.LastError))
	}
	if scenario.Health.Status != "" && scenario.Health.Status != FinalizationStatusCompleted {
		appendFeedbackLine(b, fmt.Sprintf("- %s health: %s", scenario.ScenarioName, scenario.Health.Details))
	}
	if scenario.Review.Result == nil {
		if scenario.Review.SkipReason != "" {
			appendFeedbackLine(b, fmt.Sprintf("- %s review: %s", scenario.ScenarioName, scenario.Review.SkipReason))
		}
		return
	}
	if strings.TrimSpace(scenario.Review.Result.Summary) != "" {
		appendFeedbackLine(b, fmt.Sprintf("- %s review summary: %s", scenario.ScenarioName, scenario.Review.Result.Summary))
	}
	for _, dim := range scenario.Review.Result.Dimensions {
		if dim.Status == "green" || dim.Status == "skipped" {
			continue
		}
		if dim.Details != "" {
			appendFeedbackLine(b, fmt.Sprintf("- %s %s (%s): %s", scenario.ScenarioName, dim.Name, dim.Status, dim.Details))
		} else {
			appendFeedbackLine(b, fmt.Sprintf("- %s %s (%s)", scenario.ScenarioName, dim.Name, dim.Status))
		}
	}
}

func deliverablePromptTag(kind string) string {
	return "implementation-plan"
}

func missingDeliverableReason(kind, deliverablePath string) string {
	return "no implementation plan_ref exists — author a plan through plan.author before queueing"
}
