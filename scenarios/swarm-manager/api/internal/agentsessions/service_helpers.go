package agentsessions

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
)

func skillIDForKind(kind Kind) string {
	switch kind {
	case KindMetaOrchestration:
		return SkillMetaOrchestrator
	case KindSwarmOperations:
		return SkillSwarmOperations
	case KindWorkflowAuthoring:
		return SkillWorkflowAuthoring
	default:
		return ""
	}
}

func sessionEnvironment(session Session) map[string]string {
	return map[string]string{
		EnvSessionID:   session.ID,
		EnvSessionKind: string(session.Kind),
		EnvSpawnSource: "session/" + session.ID,
	}
}

func sessionActivitySpec(session Session, interaction agentactivity.InteractionType) agentactivity.Spec {
	purpose := agentactivity.Purpose(session.Kind)
	return agentactivity.Spec{
		OwnerType:  agentactivity.OwnerSession,
		OwnerKind:  string(session.Kind),
		OwnerName:  session.ID,
		OwnerTitle: session.Title,
		Purpose:    purpose,
		Metadata: map[string]string{
			"entrypoint":       "agent_sessions." + string(session.Kind),
			"session_id":       session.ID,
			"skill_id":         session.SkillID,
			"interaction_type": string(interaction),
			"session_kind":     string(session.Kind),
			"swarm_source":     "session/" + session.ID,
		},
	}
}

func attributionForContext(ctx context.Context) *Attribution {
	attr := AttributionFromProvenance(identity.FromContext(ctx))
	if attr.Type == "" {
		attr.Type = AttributionOperator
	}
	return &attr
}

func sessionCreatedBy(session Session) string {
	if session.CreatedBy == nil {
		return "swarm-manager"
	}
	if session.CreatedBy.Type == AttributionAgent && session.CreatedBy.RunID != "" {
		return "agent:" + session.CreatedBy.RunID
	}
	return string(session.CreatedBy.Type)
}

func sessionStatusFromRunState(status string) Status {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting", "running":
		return StatusRunning
	case "needs_review":
		return StatusWaitingForUser
	case "complete", "completed":
		return StatusComplete
	case "failed":
		return StatusFailed
	case "cancelled", "canceled":
		return StatusCanceled
	default:
		return ""
	}
}

func mapStoreError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apierr.NotFound("agent session not found")
	}
	if errors.Is(err, ErrValidation) {
		return apierr.BadRequest("%s", err.Error())
	}
	return err
}

func mapSpawnError(err error) error {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return apierr.Unavailable("agent-manager is unavailable")
	}
	if errors.Is(err, agentmanager.ErrRequestFailed) {
		return apierr.Wrap(apierr.ErrBadGateway, http.StatusBadGateway, err.Error())
	}
	return err
}

func nowRFC3339() string {
	return nowUTC().Format(time.RFC3339)
}
