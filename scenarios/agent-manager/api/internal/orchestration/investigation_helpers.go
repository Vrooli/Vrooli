// Package orchestration: investigation helpers retain resume-only context and
// recurrence formatting outside the bounded workflow-dispatch implementation.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"

	"github.com/google/uuid"
)

func recurrenceAttachment(runID uuid.UUID, prior []findings.Finding) domain.ContextAttachment {
	var content strings.Builder
	content.WriteString("### Prior findings and recurrence\n\n")
	for _, finding := range prior {
		applied := "no recorded decision"
		if finding.Decision != "" {
			applied = "decision=" + finding.Decision
		}
		fingerprint := finding.Fingerprint
		if len(fingerprint) > 12 {
			fingerprint = fingerprint[:12]
		}
		fmt.Fprintf(&content, "- fingerprint=%s occurrences=%d %s: %s\n", fingerprint, finding.Occurrences, applied, finding.Recommendation)
	}
	return domain.ContextAttachment{Type: "note", Key: "prior-findings-" + shortID(runID), Label: "Prior Findings " + shortID(runID), Content: content.String(), Format: "markdown", Priority: "medium", Summary: fmt.Sprintf("%d prior finding(s), ordered by recurrence", len(prior)), Tags: []string{"findings", "recurrence", "investigation"}}
}

func (o *Orchestrator) buildHistoricalContext(ctx context.Context, currentRun *domain.Run, short string) (domain.ContextAttachment, bool) {
	if currentRun.AgentProfileID == nil {
		return domain.ContextAttachment{}, false
	}
	recentRuns, err := o.ListRuns(ctx, RunListOptions{ListOptions: ListOptions{Limit: 10}, AgentProfileID: currentRun.AgentProfileID})
	if err != nil || len(recentRuns) == 0 {
		return domain.ContextAttachment{}, false
	}
	var history []*domain.Run
	for _, run := range recentRuns {
		if run.ID != currentRun.ID && !strings.Contains(run.Tag, "investigation") {
			history = append(history, run)
		}
	}
	if len(history) == 0 {
		return domain.ContextAttachment{}, false
	}
	var sb strings.Builder
	sb.WriteString("### Recent Runs (same agent profile)\n\n| Run ID | Date | Tag | Status | Duration | Error |\n|--------|------|-----|--------|----------|-------|\n")
	var successCount, failCount int
	for _, run := range history {
		duration := "—"
		if run.StartedAt != nil && run.EndedAt != nil {
			duration = run.EndedAt.Sub(*run.StartedAt).Round(time.Second).String()
		}
		errMsg := run.ErrorMsg
		if errMsg == "" {
			errMsg = "—"
		} else if len(errMsg) > 60 {
			errMsg = errMsg[:60] + "..."
		}
		tag := run.Tag
		if len(tag) > 30 {
			tag = tag[:30] + "..."
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s |\n", shortID(run.ID), run.CreatedAt.Format("Jan 02 15:04"), tag, run.Status, duration, errMsg)
		if run.Status == domain.RunStatusComplete {
			successCount++
		} else if run.Status == domain.RunStatusFailed {
			failCount++
		}
	}
	total := len(history)
	fmt.Fprintf(&sb, "\n**Pattern**: %d of last %d runs succeeded, %d failed\n", successCount, total, failCount)
	if failCount > 0 && successCount > 0 {
		sb.WriteString("*Compare successful vs failed runs to identify what changed.*\n")
	} else if failCount == total {
		sb.WriteString("*All recent runs failed — likely a persistent issue rather than a transient one.*\n")
	}
	return domain.ContextAttachment{Type: "note", Key: fmt.Sprintf("run-history-%s", short), Label: fmt.Sprintf("Historical Context %s", short), Content: sb.String(), Format: "markdown", Priority: "medium", Summary: fmt.Sprintf("%d recent runs: %d succeeded, %d failed", total, successCount, failCount), Tags: []string{"run", "history", "investigation"}}, true
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	return string(data), err
}

func shortID(id uuid.UUID) string {
	value := id.String()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}
func taskShortID() string { return shortID(uuid.New()) }
