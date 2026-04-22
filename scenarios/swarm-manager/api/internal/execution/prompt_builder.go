package execution

import (
	"crypto/sha256"
	"encoding/hex"
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
	DeliverablePath    string // primary workshop artifact path (plan.md or conclusion.md)
	DeliverableContent string // full primary workshop artifact text (empty if missing)
	ReviewFeedback     string // review summary for fixup runs
	FollowUpNote       string // user-provided context for follow-up/custom runs
	IdeaHandoff        *handoff.Package
	SuggestedSkills    []string
}

func buildProcessingTitle(item backlogItem) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch item.Kind {
	case "fix":
		return "Apply fix: " + label
	case "execute":
		return "Execute task: " + label
	case "chore":
		return "Run chore: " + label
	default:
		return "Generate scenario: " + label
	}
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
		b.WriteString("Use brief.md as the ecosystem-manager task notes when creating the downstream task.\n")
		b.WriteString("Preserve the handoff origin metadata on the ecosystem-manager task so later loops can trace back to swarm-manager.\n")
		b.WriteString("When creating the downstream task, pass: --handoff-dir, --origin-source swarm-manager, --origin-backlog-item, and --origin-item-folder.\n")
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
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if warning.ScenarioName != "" {
			b.WriteString(fmt.Sprintf("- warning [%s] %s: %s", warning.Code, warning.ScenarioName, warning.Message))
		} else {
			b.WriteString(fmt.Sprintf("- warning [%s]: %s", warning.Code, warning.Message))
		}
	}
	for _, scenario := range finalization.Scenarios {
		if scenario.Restart.Status != "" && scenario.Restart.Status != FinalizationStatusCompleted {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s restart: %s", scenario.ScenarioName, scenario.Restart.LastError))
		}
		if scenario.Health.Status != "" && scenario.Health.Status != FinalizationStatusCompleted {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s health: %s", scenario.ScenarioName, scenario.Health.Details))
		}
		if scenario.Review.Result == nil {
			if scenario.Review.SkipReason == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s review: %s", scenario.ScenarioName, scenario.Review.SkipReason))
			continue
		}
		if strings.TrimSpace(scenario.Review.Result.Summary) != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s review summary: %s", scenario.ScenarioName, scenario.Review.Result.Summary))
		}
		for _, dim := range scenario.Review.Result.Dimensions {
			if dim.Status == "green" || dim.Status == "skipped" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			if dim.Details != "" {
				b.WriteString(fmt.Sprintf("- %s %s (%s): %s", scenario.ScenarioName, dim.Name, dim.Status, dim.Details))
			} else {
				b.WriteString(fmt.Sprintf("- %s %s (%s)", scenario.ScenarioName, dim.Name, dim.Status))
			}
		}
	}
	return b.String()
}

func deliverableForKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "research":
		return "conclusion.md"
	default:
		return "plan.md"
	}
}

func deliverablePromptTag(kind string) string {
	switch deliverableForKind(kind) {
	case "conclusion.md":
		return "research-conclusion"
	default:
		return "implementation-plan"
	}
}

func missingDeliverableReason(kind, deliverablePath string) string {
	switch deliverableForKind(kind) {
	case "conclusion.md":
		return fmt.Sprintf("no research conclusion (%s) exists — run workshop first", deliverablePath)
	default:
		return fmt.Sprintf("no implementation plan (%s) exists — run workshop first", deliverablePath)
	}
}

func promptRevision(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func (s *Service) buildIdeaHandoffPackage(item backlogItem, itemDir string, preflight ProcessPreflight) (*handoff.Package, error) {
	if strings.TrimSpace(item.Kind) != "idea" {
		return nil, nil
	}
	targetScenarioID, _ := resolveTargetScenario(item)
	return handoff.BuildIdeaPackage(handoff.BuildRequest{
		BacklogKind:             item.Kind,
		BacklogName:             item.Name,
		BacklogTitle:            item.Title,
		BacklogDescription:      item.Description,
		ItemFolder:              itemDir,
		DeliverableFileName:     deliverableForKind(item.Kind),
		TargetScenario:          targetScenarioID,
		Operation:               preflight.SuggestedOperation,
		SuggestedSteerProfileID: preflight.SuggestedSteerProfileID,
		AcceptanceAllow:         item.AcceptanceAllow,
		AcceptanceDeny:          item.AcceptanceDeny,
	})
}
