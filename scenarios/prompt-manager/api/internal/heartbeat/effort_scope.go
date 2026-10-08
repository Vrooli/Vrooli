package heartbeat

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

// AcceptedEffortReader is the canonical work owner's protected acceptance
// read. It must verify the actual accepted content, not interpret quotes,
// WorkReferences.Verified or an editable acceptedRevision label as acceptance.
// Exact deployment setup must select this owner route; none is inferred.
type AcceptedEffortReader interface {
	CheckAcceptedEffort(context.Context, string, string, string, string) error
}
type FiniteScopeTeams interface {
	Get(context.Context, string) (*store.Team, error)
	GetMembers(context.Context, string) ([]store.TeamMemberRelation, error)
	GetHeartbeatConfig(context.Context, string, string) (*store.HeartbeatConfig, error)
}

// NativeProfiles must read the current complete native profile contract. An
// editable PM profile label cannot qualify native permissions before PM effects.
type NativeProfileReader interface {
	CheckNativeProfile(context.Context, string, string) error
}
type FiniteEffortScope struct {
	Teams    FiniteScopeTeams
	Accepted AcceptedEffortReader
	Profiles NativeProfileReader
}

func (s *FiniteEffortScope) CheckEffortScope(ctx context.Context, p effortauthority.Policy) error {
	if s == nil || s.Teams == nil || s.Accepted == nil || s.Profiles == nil {
		return effortauthority.ErrRefused
	}
	if e := s.Accepted.CheckAcceptedEffort(ctx, p.Owner, p.Effort, p.Revision, p.ContentDigest); e != nil {
		return effortauthority.ErrRefused
	}
	team, e := s.Teams.Get(ctx, p.Team)
	if e != nil || team == nil || validateTeamEnabled(team) != nil || effortauthority.Digest(team.Contract()) != p.TeamDigest {
		return effortauthority.ErrRefused
	}
	if team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed || team.Execution.QueuePolicy != teamconfig.QueuePolicySerialized || team.Execution.MaxConcurrentRuns != 1 {
		return effortauthority.ErrRefused
	}
	cfg, e := s.Teams.GetHeartbeatConfig(ctx, p.Team, team.Coordination.LeadAgentID)
	if e != nil || cfg == nil || cfg.FiniteLeader == nil || !cfg.Enabled || cfg.FiniteLeader.Retired || cfg.FiniteLeader.EffortRef != p.Effort || cfg.FiniteLeader.AcceptedRevision != p.Revision || effortauthority.Digest(cfg.FiniteLeader) != p.BindingDigest || p.Profiles[cfg.ProfileKey] == "" {
		return effortauthority.ErrRefused
	}
	for key, digest := range p.Profiles {
		if s.Profiles.CheckNativeProfile(ctx, key, digest) != nil {
			return effortauthority.ErrRefused
		}
	}
	roster, e := s.Teams.GetMembers(ctx, p.Team)
	if e != nil {
		return effortauthority.ErrRefused
	}
	for _, id := range p.Members {
		active := false
		for _, m := range roster {
			if m.AgentID == id && (m.Status == "" || m.Status == store.MemberStatusActive) {
				active = true
			}
		}
		if !active || team.OperatingContract == nil {
			return effortauthority.ErrRefused
		}
		if _, ok := team.OperatingContract.Members[id]; !ok {
			return effortauthority.ErrRefused
		}
	}
	return nil
}
