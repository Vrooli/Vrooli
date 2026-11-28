package orchestration_test

import (
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// newLifecycleRunFixture supplies the fields every repository-backed lifecycle
// case needs. Status, phase, mode, session, and IDs stay at each call site
// because they define the behavior under test; callers add sandbox-specific
// authority, while terminal timestamps follow the selected status.
func newLifecycleRunFixture(id, taskID, profileID uuid.UUID, mode domain.RunMode, status domain.RunStatus, phase domain.RunPhase, sessionID string) *domain.Run {
	now := time.Now()
	run := &domain.Run{
		ID: id, TaskID: taskID, AgentProfileID: &profileID, Tag: id.String(),
		RunMode: mode, Status: status, Phase: phase, SessionID: sessionID,
		StartedAt: &now, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState: domain.ApprovalStateNone, CreatedAt: now, UpdatedAt: now,
	}
	if status.IsTerminal() {
		endedAt := now
		run.EndedAt = &endedAt
	}
	return run
}
