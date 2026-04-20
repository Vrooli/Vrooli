// Package orchestration: ResumeFromFailedRun creates a brand-new run that
// inherits the original task + profile of a failed/cancelled run and is
// seeded with that run's transcript and diff. This is the missing middle
// ground between Retry (fresh start, no context) and Continue (Codex session
// resume, fragile and only when SessionID is present): it lets the agent pick
// up where the prior attempt left off without redoing completed work.
package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ResumeFromFailedRun creates a new run that resumes the work of a failed
// or cancelled run.
func (o *Orchestrator) ResumeFromFailedRun(
	ctx context.Context,
	req ResumeFromFailedRunRequest,
) (*domain.Run, error) {
	failedRun, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	if allowed, reason := domain.CanResumeFromFailureRun(failedRun); !allowed {
		return nil, domain.NewValidationError("runId", reason)
	}

	originalTask, err := o.GetTask(ctx, failedRun.TaskID)
	if err != nil {
		return nil, err
	}

	attachments, err := o.buildResumeFromFailureAttachments(ctx, failedRun, originalTask, req.CustomContext)
	if err != nil {
		return nil, err
	}

	for _, id := range req.AttachmentIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:         "image",
			AttachmentID: id,
			Label:        "Uploaded image",
			Key:          "resume-image-" + id,
			Tags:         []string{"image", "resume"},
		})
	}

	prompt := buildResumePrompt(originalTask.Description, failedRun.ID, req.CustomContext)

	newTask, err := o.createInvestigationTask(ctx, "Resume "+originalTask.Title, prompt, attachments, originalTask.ProjectRoot)
	if err != nil {
		return nil, err
	}

	createReq := CreateRunRequest{
		TaskID:         newTask.ID,
		AgentProfileID: failedRun.AgentProfileID,
		Tag:            resumeTag(failedRun.Tag),
		Force:          true,
		SourceRunIDs:   []uuid.UUID{failedRun.ID},
	}

	newRun, err := o.CreateRun(ctx, createReq)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, newRun), nil
}

// resumeTag derives a tag for the resumed run that preserves traceability
// to the original attempt. When the original tag is unset it falls back to
// the empty string and CreateRun assigns a default.
func resumeTag(prevTag string) string {
	prevTag = strings.TrimSpace(prevTag)
	if prevTag == "" {
		return ""
	}
	if strings.HasSuffix(prevTag, "-resume") {
		return prevTag
	}
	return prevTag + "-resume"
}

// buildResumeFromFailureAttachments composes the context the resumed agent
// needs: the original task's attachments, plus a human-readable view of the
// prior attempt (overview, timeline, diff, history, user guidance).
func (o *Orchestrator) buildResumeFromFailureAttachments(
	ctx context.Context,
	failedRun *domain.Run,
	originalTask *domain.Task,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(originalTask.ContextAttachments)+5)
	attachments = append(attachments, originalTask.ContextAttachments...)

	short := shortID(failedRun.ID)
	prevKey := "prev-" + short

	var profile *domain.AgentProfile
	if failedRun.AgentProfileID != nil {
		profile, _ = o.GetProfile(ctx, *failedRun.AgentProfileID)
	}
	attachments = append(attachments, buildRunOverview(failedRun, originalTask, profile, prevKey))

	if o.events != nil {
		events, err := o.GetRunEvents(ctx, failedRun.ID, event.GetOptions{
			AfterSequence: -1,
			Limit:         investigationEventLimit,
		})
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, buildRunTimeline(events, failedRun, prevKey))
	}

	diff, err := o.GetRunDiff(ctx, failedRun.ID)
	if err == nil && diff != nil {
		diffJSON, err := marshalJSON(diff)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      fmt.Sprintf("prev-run-diff-%s", short),
			Label:    fmt.Sprintf("Previous Attempt Diff %s", short),
			Content:  diffJSON,
			Format:   "json",
			Priority: "high",
			Summary:  fmt.Sprintf("Code changes already present on disk from failed run %s (%d files, %d bytes)", short, failedRun.ChangedFiles, failedRun.TotalSizeBytes),
			Tags:     []string{"run", "diff", "resume"},
		})
	}

	if failedRun.AgentProfileID != nil {
		if att, ok := o.buildHistoricalContext(ctx, failedRun, prevKey); ok {
			attachments = append(attachments, att)
		}
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      "user-resume-context",
			Label:    "Additional Context for Resume",
			Content:  customContext,
			Format:   "markdown",
			Priority: "high",
			Summary:  "User-provided guidance for completing the remaining work",
			Tags:     []string{"user", "context", "resume"},
		})
	}

	return attachments, nil
}

// buildResumePrompt frames the resumed run for the agent: the original task
// goal, then explicit instructions to reuse work that's already on disk and
// only complete what's left.
func buildResumePrompt(originalPrompt string, failedRunID uuid.UUID, customContext string) string {
	var sb strings.Builder

	sb.WriteString(strings.TrimSpace(originalPrompt))
	sb.WriteString("\n\n---\n\n")

	sb.WriteString("## Prior Attempt\n\n")
	sb.WriteString(fmt.Sprintf("This task was previously attempted in run `%s`, which did not complete successfully. ", failedRunID))
	sb.WriteString("The attached context contains:\n\n")
	sb.WriteString("- **Previous Attempt Run Overview**: status, error, and execution metadata\n")
	sb.WriteString("- **Run Timeline**: the full event log of what the prior agent did\n")
	sb.WriteString("- **Previous Attempt Diff**: code changes that were made and may already be on disk\n\n")
	sb.WriteString("**Your job**:\n\n")
	sb.WriteString("1. Review the prior timeline and diff to understand what was already attempted.\n")
	sb.WriteString("2. Detect work that is already present on disk — do NOT redo it. Treat completed changes as done.\n")
	sb.WriteString("3. Identify the remaining work needed to satisfy the original task and complete it.\n")
	sb.WriteString("4. If the prior attempt left the workspace in a broken or inconsistent state, fix that first, then continue.\n")
	sb.WriteString("5. If the original task is no longer feasible (e.g. requirements have shifted or the prior attempt revealed a blocker), explain clearly and stop rather than guessing.\n\n")

	sb.WriteString("## Diagnostics\n\n")
	sb.WriteString("Use these CLI commands to fetch additional detail beyond the attached context:\n\n")
	sb.WriteString(fmt.Sprintf("```bash\nagent-manager run get %s\nagent-manager run events %s\nagent-manager run diff %s\n```\n", failedRunID, failedRunID, failedRunID))

	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("\n## User-Provided Guidance\n\n")
		sb.WriteString(strings.TrimSpace(customContext))
		sb.WriteString("\n")
	}

	return sb.String()
}
