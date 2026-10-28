package handlers

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
)

// ManagedTeamReader uses Prompt Manager's existing caller-specific GetTeam
// decision. Authored team labels never substitute for its verified owner read.
type ManagedTeamReader interface {
	GetTeam(context.Context, string) (*teamsv1.TeamDetails, error)
}

func (h *AgentManagerConnectHandler) SetManagedTeamReader(reader ManagedTeamReader) {
	h.managedTeams = reader
}

func (h *AgentManagerConnectHandler) teamEffortActor(ctx context.Context, e *pb.EffortEnrollment) (supervision.EffortActor, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	deny := func() (supervision.EffortActor, error) {
		return supervision.EffortActor{}, connect.NewError(connect.CodePermissionDenied, errors.New("current team management authority required"))
	}
	if !ok || !p.IsVerified() || p.Kind != identity.ActorAgent || (!p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt)) || e == nil {
		return deny()
	}
	var member *pb.EffortSubject
	for _, subject := range e.Subjects {
		if subject.GetOwner() == "prompt-manager" && subject.GetKind() == "team-member" {
			if member != nil || subject.GetRole() != "orchestrator" {
				return deny()
			}
			member = subject
		}
	}
	if member == nil {
		return deny()
	}
	parts := strings.Split(member.Reference, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return deny()
	}
	if h.managedTeams == nil {
		return supervision.EffortActor{}, connect.NewError(connect.CodeUnavailable, errors.New("team owner reader unavailable"))
	}
	team, err := h.managedTeams.GetTeam(ctx, parts[0])
	if err != nil {
		return supervision.EffortActor{}, connect.NewError(connect.CodeUnavailable, errors.New("team owner read unavailable"))
	}
	if team == nil || team.Id != parts[0] || !team.ManagementAuthorized || team.Revision < 1 || team.Archived || team.Purpose != "delivery" || team.Lifetime != "finite" || team.Coordination.GetFields()["leadAgentId"].GetStringValue() != parts[1] {
		return deny()
	}
	declared := false
	for _, ref := range team.EffortRefs {
		declared = declared || ref == e.EffortRef
	}
	active := false
	for _, m := range team.Members {
		active = active || m.AgentId == parts[1] && m.Status == "active"
	}
	home := path.Clean(team.HomePath)
	if !declared || !active || home == "." || home == ".." || home != team.HomePath || path.IsAbs(home) || strings.HasPrefix(home, "../") || strings.Contains(home, "\\") {
		return deny()
	}
	return supervision.EffortActor{ID: p.Subject, Team: &supervision.TeamEffortAuthority{TeamID: team.Id, MemberRef: member.Reference, HomePath: home}}, nil
}
