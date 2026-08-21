package graph

import (
	"context"
	"strings"
)

// CommandReferenceValidator validates one Vrooli-owned command reference
// without executing it. Production delegates to CLI Health; tests inject fakes.
type CommandReferenceValidator interface {
	ValidateCommandReference(context.Context, CommandReferenceRequest) (CommandReferenceResult, error)
}

type CommandReferenceRequest struct {
	CommandText string
}

type CommandReferenceResult struct {
	Verdict         string
	ValidationLevel string
	Issues          []CommandIssue
	Suggestions     []string
	Guidance        []string
}

type CommandIssue struct {
	Code    string
	Message string
}

// ApplyCommandReferenceDiagnostics folds CLI Health command-reference validation
// into skill graph health. It is intentionally scoped to skill-authored
// Vrooli-owned command usages; external tools are handled by code-usage scoring.
func ApplyCommandReferenceDiagnostics(ctx context.Context, g Graph, scores []HealthScore, validator CommandReferenceValidator) []HealthScore {
	if validator == nil || len(scores) == 0 {
		return scores
	}
	nodeType := map[string]NodeType{}
	for _, n := range g.Nodes {
		nodeType[n.ID] = n.Type
	}
	scoreByNode := map[string]int{}
	for i := range scores {
		scoreByNode[scores[i].NodeID] = i
	}
	for _, e := range g.Edges {
		if e.Kind != EdgeCodeUsage || e.Category != CodeScenarioCLI || nodeType[e.From] != NodeSkill {
			continue
		}
		idx, ok := scoreByNode[e.From]
		if !ok {
			continue
		}
		result, err := validator.ValidateCommandReference(ctx, CommandReferenceRequest{CommandText: e.ToCommandText()})
		if err != nil {
			scores[idx].Messages = append(scores[idx].Messages, HealthMessage{
				Key:            "skill.command.validation_unknown",
				Severity:       severityWarning,
				Summary:        "Skill command validation unavailable",
				Detail:         "CLI Health could not validate `" + e.ValueForMessage() + "`: " + err.Error(),
				Recommendation: "Re-run graph health after CLI Health is available.",
			})
			continue
		}
		switch strings.ToLower(result.Verdict) {
		case "valid", "skipped":
			continue
		case "partial":
			scores[idx].Messages = append(scores[idx].Messages, HealthMessage{
				Key:            "skill.command.partial",
				Severity:       severityWarning,
				Summary:        "Skill command has partial validation coverage",
				Detail:         "CLI Health confirmed `" + e.ValueForMessage() + "` exists, but could not fully validate its arguments.",
				Recommendation: commandReferenceMessage(result),
			})
		default:
			scores[idx].Messages = append(scores[idx].Messages, HealthMessage{
				Key:            "skill.command.invalid",
				Severity:       severityCritical,
				Summary:        "Skill references an invalid current command",
				Detail:         "CLI Health rejected `" + e.ValueForMessage() + "`.",
				Recommendation: commandReferenceMessage(result),
			})
			if scores[idx].Score > 0.2 {
				scores[idx].Score = 0.2
			}
			if scores[idx].Factors == nil {
				scores[idx].Factors = map[string]float64{}
			}
			scores[idx].Factors["command-reference"] = 0
		}
	}
	return scores
}

func (e Edge) ToCommandText() string {
	if strings.TrimSpace(e.CommandText) != "" {
		return strings.TrimSpace(e.CommandText)
	}
	if strings.TrimSpace(e.To) != "" && strings.HasPrefix(e.To, "cli:") {
		return strings.TrimPrefix(e.To, "cli:")
	}
	return strings.TrimSpace(strings.Join([]string{e.Command, e.Subcommand}, " "))
}

func (e Edge) ValueForMessage() string {
	if cmd := e.ToCommandText(); cmd != "" {
		return cmd
	}
	return strings.TrimPrefix(e.To, "cli:")
}

func commandReferenceMessage(result CommandReferenceResult) string {
	var parts []string
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion != "" {
			parts = append(parts, "suggestion: "+suggestion)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		parts = append(parts, strings.TrimSpace(result.Verdict+" "+result.ValidationLevel))
	}
	return strings.Join(parts, "; ")
}
