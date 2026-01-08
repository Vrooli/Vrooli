// Package orchestration provides helpers for investigation runs.
//
// Investigation runs are normal runs tagged for analysis of other runs.
// They use a dedicated profile key and standard task/run creation flow.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

const (
	investigationTag           = "agent-manager-investigation"
	investigationApplyTag      = "agent-manager-investigation-apply"
	investigationProfileKey    = "agent-manager-investigation"
	investigationEventLimit    = 500
	investigationReportTimeout = 10 * time.Minute
)

// CreateInvestigationRun creates a new investigation run for the given run IDs.
func (o *Orchestrator) CreateInvestigationRun(
	ctx context.Context,
	runIDs []uuid.UUID,
	customContext string,
) (*domain.Run, error) {
	if len(runIDs) == 0 {
		return nil, domain.NewValidationError("runIds", "at least one run ID is required")
	}

	attachments, err := o.buildInvestigationAttachments(ctx, runIDs, customContext)
	if err != nil {
		return nil, err
	}

	prompt := buildInvestigationPrompt(runIDs, customContext)
	task, err := o.createInvestigationTask(ctx, "Investigation", prompt, attachments)
	if err != nil {
		return nil, err
	}

	return o.createInvestigationRun(ctx, task.ID, investigationTag)
}

// CreateInvestigationApplyRun creates a new run that applies investigation recommendations.
func (o *Orchestrator) CreateInvestigationApplyRun(
	ctx context.Context,
	investigationRunID uuid.UUID,
	customContext string,
) (*domain.Run, error) {
	run, err := o.GetRun(ctx, investigationRunID)
	if err != nil {
		return nil, err
	}
	if run.Tag != investigationTag {
		return nil, domain.NewValidationError("investigationRunId", "run is not an investigation run")
	}

	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	attachments, err := o.buildApplyAttachments(ctx, run, task, customContext)
	if err != nil {
		return nil, err
	}

	prompt := buildApplyPrompt(investigationRunID, customContext)
	applyTask, err := o.createInvestigationTask(ctx, "Apply Investigation", prompt, attachments)
	if err != nil {
		return nil, err
	}

	return o.createInvestigationRun(ctx, applyTask.ID, investigationApplyTag)
}

func (o *Orchestrator) createInvestigationTask(
	ctx context.Context,
	titlePrefix string,
	prompt string,
	attachments []domain.ContextAttachment,
) (*domain.Task, error) {
	now := time.Now()
	projectRoot := strings.TrimSpace(o.config.DefaultProjectRoot)
	task := &domain.Task{
		ID:                 uuid.New(),
		Title:              fmt.Sprintf("%s %s", titlePrefix, taskShortID()),
		Description:        prompt,
		ScopePath:          ".",
		ProjectRoot:        projectRoot,
		Status:             domain.TaskStatusQueued,
		ContextAttachments: attachments,
		CreatedAt:          now,
		UpdatedAt:          now,
		CreatedBy:          "agent-manager",
	}

	return o.CreateTask(ctx, task)
}

func (o *Orchestrator) createInvestigationRun(ctx context.Context, taskID uuid.UUID, tag string) (*domain.Run, error) {
	sandboxConfig := &domain.SandboxConfig{NoLock: true}
	if o.config.DefaultSandboxConfig != nil {
		clone := *o.config.DefaultSandboxConfig
		clone.NoLock = true
		sandboxConfig = &clone
	}

	return o.CreateRun(ctx, CreateRunRequest{
		TaskID:        taskID,
		ProfileRef:    investigationProfileRef(),
		Tag:           tag,
		Force:         true,
		SandboxConfig: sandboxConfig,
	})
}

func (o *Orchestrator) buildInvestigationAttachments(
	ctx context.Context,
	runIDs []uuid.UUID,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(runIDs)*2+1)
	for _, runID := range runIDs {
		run, err := o.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}

		runJSON, err := marshalJSON(run)
		if err != nil {
			return nil, err
		}

		short := shortID(runID)
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("run-summary-%s", short),
			fmt.Sprintf("Run Summary %s", short),
			runJSON,
			[]string{"run", "summary", "investigation"},
		))

		if o.events != nil {
			events, err := o.GetRunEvents(ctx, runID, event.GetOptions{Limit: investigationEventLimit})
			if err != nil {
				return nil, err
			}
			eventsJSON, err := marshalJSON(events)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, noteAttachment(
				fmt.Sprintf("run-events-%s", short),
				fmt.Sprintf("Run Events %s", short),
				eventsJSON,
				[]string{"run", "events", "investigation"},
			))
		}

		diff, err := o.GetRunDiff(ctx, runID)
		if err == nil && diff != nil {
			diffJSON, err := marshalJSON(diff)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, noteAttachment(
				fmt.Sprintf("run-diff-%s", short),
				fmt.Sprintf("Run Diff %s", short),
				diffJSON,
				[]string{"run", "diff", "investigation"},
			))
		}
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, noteAttachment(
			"user-context",
			"Additional Context",
			customContext,
			[]string{"user", "context"},
		))
	}

	return attachments, nil
}

func (o *Orchestrator) buildApplyAttachments(
	ctx context.Context,
	investigationRun *domain.Run,
	task *domain.Task,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(task.ContextAttachments)+3)
	attachments = append(attachments, task.ContextAttachments...)

	short := shortID(investigationRun.ID)
	runJSON, err := marshalJSON(investigationRun)
	if err != nil {
		return nil, err
	}
	attachments = append(attachments, noteAttachment(
		fmt.Sprintf("investigation-run-summary-%s", short),
		fmt.Sprintf("Investigation Run Summary %s", short),
		runJSON,
		[]string{"run", "investigation"},
	))

	if o.events != nil {
		events, err := o.GetRunEvents(ctx, investigationRun.ID, event.GetOptions{Limit: investigationEventLimit})
		if err != nil {
			return nil, err
		}
		eventsJSON, err := marshalJSON(events)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("investigation-run-events-%s", short),
			fmt.Sprintf("Investigation Run Events %s", short),
			eventsJSON,
			[]string{"run", "events", "investigation"},
		))
	}

	diff, err := o.GetRunDiff(ctx, investigationRun.ID)
	if err == nil && diff != nil {
		diffJSON, err := marshalJSON(diff)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("investigation-run-diff-%s", short),
			fmt.Sprintf("Investigation Run Diff %s", short),
			diffJSON,
			[]string{"run", "diff", "investigation"},
		))
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, noteAttachment(
			"user-context",
			"Additional Context",
			customContext,
			[]string{"user", "context"},
		))
	}

	return attachments, nil
}

func buildInvestigationPrompt(runIDs []uuid.UUID, customContext string) string {
	var sb strings.Builder
	sb.WriteString("# Agent-Manager Investigation\n\n")
	sb.WriteString("You are investigating agent-manager runs to diagnose issues and recommend improvements.\n\n")

	sb.WriteString("## Runs Under Investigation\n")
	for _, id := range runIDs {
		sb.WriteString(fmt.Sprintf("- %s\n", id))
	}
	sb.WriteString("\n")

	sb.WriteString("## How to Fetch Full Run Data\n")
	sb.WriteString("If you need full details beyond the attachments, use the agent-manager CLI:\n")
	sb.WriteString("- agent-manager run get <run-id>\n")
	sb.WriteString("- agent-manager run events <run-id>\n")
	sb.WriteString("- agent-manager run diff <run-id>\n\n")

	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(customContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Report\n")
	sb.WriteString("- Summary of root cause(s)\n")
	sb.WriteString("- Evidence (reference run IDs and event sequences)\n")
	sb.WriteString("- Recommendations\n")

	return sb.String()
}

func buildApplyPrompt(investigationRunID uuid.UUID, customContext string) string {
	var sb strings.Builder
	sb.WriteString("# Apply Investigation Recommendations\n\n")
	sb.WriteString("You are applying recommendations from a prior investigation run.\n\n")
	sb.WriteString(fmt.Sprintf("Investigation Run ID: %s\n\n", investigationRunID))

	sb.WriteString("## How to Fetch Full Investigation Data\n")
	sb.WriteString("If you need full details beyond the attachments, use the agent-manager CLI:\n")
	sb.WriteString(fmt.Sprintf("- agent-manager run get %s\n", investigationRunID))
	sb.WriteString(fmt.Sprintf("- agent-manager run events %s\n", investigationRunID))
	sb.WriteString(fmt.Sprintf("- agent-manager run diff %s\n\n", investigationRunID))

	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(customContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. Review the investigation report output.\n")
	sb.WriteString("2. Apply the recommended fixes explicitly called out by the user context.\n")
	sb.WriteString("3. If a fix requires code changes, implement them.\n")
	sb.WriteString("4. Summarize what you changed and why.\n")

	return sb.String()
}

func investigationProfileRef() *ProfileRef {
	return &ProfileRef{
		ProfileKey: investigationProfileKey,
		Defaults:   defaultInvestigationProfile(),
	}
}

func defaultInvestigationProfile() *domain.AgentProfile {
	return &domain.AgentProfile{
		Name:                 "Agent-Manager Investigation",
		ProfileKey:           investigationProfileKey,
		Description:          "Agent profile for agent-manager investigations",
		RunnerType:           domain.RunnerTypeCodex,
		ModelPreset:          domain.ModelPresetSmart,
		MaxTurns:             50,
		Timeout:              investigationReportTimeout,
		AllowedTools:         []string{"read_file", "list_files", "execute_command", "analyze_code"},
		SkipPermissionPrompt: true,
		RequiresSandbox:      true,
		RequiresApproval:     false,
		CreatedBy:            "agent-manager",
	}
}

func noteAttachment(key, label, content string, tags []string) domain.ContextAttachment {
	return domain.ContextAttachment{
		Type:    "note",
		Key:     key,
		Label:   label,
		Content: content,
		Tags:    tags,
	}
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func shortID(id uuid.UUID) string {
	value := id.String()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func taskShortID() string {
	return shortID(uuid.New())
}
