package agentmanager

import (
	"context"
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// RunState captures externally visible lifecycle state for a run.
type RunState struct {
	RunID         string
	TaskID        string
	Status        string
	StartedAt     string
	FinishedAt    string
	ErrorMsg      string
	SandboxID     string
	Summary       string
	TokensUsed    int32
	TurnsUsed     int32
	CostEstimate  float64
	ChangedFiles  int32
	ContextTokens int32
}

type RunEventsOptions struct {
	AfterSequence int64
	Limit         int32
}

// RunDiff captures the changed files for a sandboxed run.
type RunDiff struct {
	RunID       string
	SandboxID   string
	GeneratedAt string
	Files       []RunDiffFile
}

// RunDiffFile captures one changed file from a run diff.
type RunDiffFile struct {
	Path       string
	ChangeType string
}

// GetRunState resolves run state from agent-manager.
func (s *AgentService) GetRunState(ctx context.Context, runID string) (RunState, error) {
	if !s.enabled {
		return RunState{}, ErrNotAvailable
	}

	run, err := s.client.GetRun(ctx, runID)
	if err != nil {
		return RunState{}, err
	}

	state := RunState{
		RunID:     strings.TrimSpace(run.Id),
		TaskID:    strings.TrimSpace(run.TaskId),
		Status:    normalizeRunStatus(run.Status),
		ErrorMsg:  strings.TrimSpace(run.ErrorMsg),
		SandboxID: strings.TrimSpace(run.GetSandboxId()),
	}
	if run.StartedAt != nil {
		state.StartedAt = run.StartedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.EndedAt != nil {
		state.FinishedAt = run.EndedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.Summary != nil {
		if run.Summary.Description != "" {
			state.Summary = strings.TrimSpace(run.Summary.Description)
		}
		state.TokensUsed = run.Summary.TokensUsed
		state.TurnsUsed = run.Summary.TurnsUsed
		state.CostEstimate = run.Summary.CostEstimate
		state.ContextTokens = run.Summary.ContextTokens
	}
	state.ChangedFiles = run.ChangedFiles
	return state, nil
}

func (s *AgentService) GetRunEvents(ctx context.Context, runID string, opts RunEventsOptions) ([]*domainpb.RunEvent, bool, error) {
	if !s.enabled {
		return nil, false, ErrNotAvailable
	}
	resp, err := s.client.GetRunEvents(ctx, runID, opts)
	if err != nil {
		return nil, false, err
	}
	return resp.Events, resp.HasMore, nil
}

// GetRunDiff resolves changed files for a sandboxed run.
func (s *AgentService) GetRunDiff(ctx context.Context, runID string) (RunDiff, error) {
	if !s.enabled {
		return RunDiff{}, ErrNotAvailable
	}

	run, err := s.client.GetRun(ctx, runID)
	if err != nil {
		return RunDiff{}, err
	}
	diff, err := s.client.GetRunDiff(ctx, runID)
	if err != nil {
		return RunDiff{}, err
	}

	result := RunDiff{
		RunID:     strings.TrimSpace(diff.RunId),
		SandboxID: strings.TrimSpace(run.GetSandboxId()),
	}
	if diff.GeneratedAt != nil {
		result.GeneratedAt = diff.GeneratedAt.AsTime().UTC().Format(time.RFC3339)
	}
	for _, file := range diff.Files {
		result.Files = append(result.Files, RunDiffFile{
			Path:       strings.TrimSpace(file.Path),
			ChangeType: strings.TrimSpace(file.ChangeType),
		})
	}
	return result, nil
}

// ContinueRun sends a follow-up message to an existing run.
func (s *AgentService) ContinueRun(ctx context.Context, runID string, message string) error {
	if !s.enabled {
		return ErrNotAvailable
	}
	return s.client.ContinueRun(ctx, runID, message)
}

// StopRun requests cancellation of an in-flight run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return ErrNotAvailable
	}
	return s.client.StopRun(ctx, runID)
}

// ApproveRun releases a run held at needs_review, merging its sandbox overlay
// into the working tree. The Baseline Modes pre-merge hold calls this after the
// shadow restore point has captured the clean working tree.
func (s *AgentService) ApproveRun(ctx context.Context, runID, actor, commitMsg string) error {
	if !s.enabled {
		return ErrNotAvailable
	}
	return s.client.ApproveRun(ctx, runID, actor, commitMsg)
}

func normalizeRunStatus(status domainpb.RunStatus) string {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING:
		return "pending"
	case domainpb.RunStatus_RUN_STATUS_STARTING:
		return "starting"
	case domainpb.RunStatus_RUN_STATUS_RUNNING:
		return "running"
	case domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return "needs_review"
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return "complete"
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}
