package review

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/pathutil"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// buildReviewAttachments creates structured context attachments for the review
// agent so that BuildSplitPrompt can separate instructions (system prompt)
// from data (user message).
func buildReviewAttachments(deliverableContent string, changedPaths, affectedScenarios []string, gctResultsJSON, baselineDiffJSON, userRequest string) []*domainpb.ContextAttachment {
	var atts []*domainpb.ContextAttachment
	redactor := pathredact.NewForArtifactPath(".")
	deliverableContent = redactor.RedactString(deliverableContent)
	changedPaths = redactStrings(redactor, changedPaths)
	affectedScenarios = redactStrings(redactor, affectedScenarios)
	gctResultsJSON = redactor.RedactString(gctResultsJSON)
	baselineDiffJSON = redactor.RedactString(baselineDiffJSON)
	userRequest = redactor.RedactString(userRequest)

	atts = appendNoteAttachment(atts, "plan-content", "Deliverable Content", "Backlog deliverable (plan or conclusion)", deliverableContent, "markdown", "high")

	diffContent := fmt.Sprintf("Changed %d files across %d scenarios", len(changedPaths), len(affectedScenarios))
	if len(changedPaths) == 0 {
		diffContent += "\n\nNote: Zero tracked changes may indicate the execution agent ran without sandbox mode enabled. " +
			"In non-sandbox mode, changes are applied directly to the working tree and are not captured as a diff. " +
			"You should still verify the implementation by examining the codebase directly."
	}
	atts = appendNoteAttachment(atts, "diff-summary", "Diff Summary", "", diffContent, "text", "high")

	if len(changedPaths) > 0 {
		atts = appendNoteAttachment(atts, "changed-paths", "Changed File Paths", fmt.Sprintf("%d changed files", len(changedPaths)), strings.Join(changedPaths, "\n"), "text", "high")
	}

	if len(affectedScenarios) > 0 {
		atts = appendNoteAttachment(atts, "affected-scenarios", "Affected Scenarios", fmt.Sprintf("%d scenarios affected", len(affectedScenarios)), strings.Join(affectedScenarios, "\n"), "text", "medium")
	}

	if gctResultsJSON != "" {
		atts = appendNoteAttachment(atts, "gct-review-results", "GCT Review Results", "Automated review metrics per scenario", gctResultsJSON, "json", "high")
	}

	if baselineDiffJSON != "" {
		atts = appendNoteAttachment(atts, "baseline-diff-results", "Baseline Diff (New vs Pre-existing)", "Before/after baseline diff per scenario: regressions this item caused vs pre-existing failures", baselineDiffJSON, "json", "high")
	}

	if userRequest != "" {
		atts = appendNoteAttachment(atts, "user-request", "User Evidence Request", "Specific evidence request from human reviewer", userRequest, "text", "high")
	}

	return atts
}

func redactStrings(redactor pathredact.Redactor, values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = redactor.RedactString(value)
	}
	return out
}

func cloneChangedPaths(changedPathsByScenario map[string][]string) map[string][]string {
	if len(changedPathsByScenario) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(changedPathsByScenario))
	for scenarioName, paths := range changedPathsByScenario {
		cloned[scenarioName] = append([]string(nil), paths...)
		sort.Strings(cloned[scenarioName])
		cloned[scenarioName] = pathutil.UniqueSortedStrings(cloned[scenarioName])
	}
	return cloned
}

func appendNoteAttachment(
	atts []*domainpb.ContextAttachment,
	key, label, summary, content, format, priority string,
) []*domainpb.ContextAttachment {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return atts
	}
	return append(atts, &domainpb.ContextAttachment{
		Type:     "note",
		Key:      key,
		Label:    label,
		Summary:  summary,
		Content:  trimmed,
		Format:   format,
		Priority: priority,
	})
}

func MarshalScenarioGCTResults(results map[string]any) string {
	if len(results) == 0 {
		return ""
	}
	data, err := json.Marshal(results)
	if err != nil {
		return ""
	}
	return string(data)
}

func normalizeLiveRunStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

// mapRunStatusToRoundStatus converts an agent-manager run status to a terminal
// round status. Returns "" if the run is still in progress.
func mapRunStatusToRoundStatus(status string) RoundStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete":
		return RoundStatusComplete
	case "failed":
		return RoundStatusFailed
	case "cancelled":
		return RoundStatusFailed
	default:
		return "" // still in progress
	}
}
