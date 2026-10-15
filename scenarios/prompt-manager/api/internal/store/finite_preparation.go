package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"prompt-manager/internal/teamconfig"
)

// TeamCreationProof records a verified actor, independently of authored team
// members, labels and harness headers. Legacy/unverified creation has no proof.
type TeamCreationProof struct {
	Kind    identity.ActorKind  `json:"kind"`
	Subject string              `json:"subject"`
	Owner   string              `json:"owner"`
	Source  identity.AuthSource `json:"source"`
}

func verifiedTeamCreator(ctx context.Context) *TeamCreationProof {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsVerified() || (!p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt)) {
		return nil
	}
	return &TeamCreationProof{Kind: p.Kind, Subject: p.Subject, Owner: p.OwnerSubject(), Source: p.Source}
}

// callerIsVerifiedTeamCreator reports whether the current verified caller is the
// recorded creator of team id and that identity is still live (unexpired, per
// verifiedTeamCreator). The creator updating its own team needs no separate
// operation grant; any other actor does. Fail-closed: a missing/expired proof,
// an unknown team, or a legacy team with no creation proof returns false so the
// normal operation gate still applies. A different agent's subject never matches.
func (s *FileTeamStore) callerIsVerifiedTeamCreator(ctx context.Context, id string) bool {
	creator := verifiedTeamCreator(ctx)
	if creator == nil {
		return false
	}
	team, err := s.Get(ctx, id)
	if err != nil || team == nil || team.CreationProof == nil {
		return false
	}
	return *team.CreationProof == *creator
}

// FinitePreparationAmendment is part of the same atomic heartbeat snapshot as
// the replacement binding. It grants no execution or acceptance authority.
type FinitePreparationAmendment struct {
	Previous    teamconfig.FiniteLeader `json:"previous"`
	Replacement teamconfig.FiniteLeader `json:"replacement"`
	Actor       TeamCreationProof       `json:"actor"`
	EvidenceRef string                  `json:"evidenceRef"`
	RecordedAt  string                  `json:"recordedAt"`
}

var ErrFinitePreparationConflict = errors.New("finite preparation requires a disabled, undispatched, unretired draft with the expected revision")

func unreservedPreparation(state *FiniteLeaderState) bool {
	return state != nil && state.Status == "idle" && state.ID == "" && state.TaskID == "" && state.RunID == "" && !state.TaskStarted && !state.DispatchStarted && state.Completed == nil && len(state.CompletionHistory) == 0 && len(state.RestartHistory) == 0 && len(state.SerialEpisodes) == 0 && state.SerialCurrentRunID == "" && state.SerialCurrentStatus == ""
}

// reconcilePreparationState projects an unused historical idle state through
// the durable amendment chain. Reads remain read-only. The next reservation
// persists the projection; an occupied/uncertain owner is never rewritten.
func reconcilePreparationState(cfg *HeartbeatConfig, state *FiniteLeaderState) {
	if !unreservedPreparation(state) || state.ProfileKey != cfg.ProfileKey {
		return
	}
	for _, amendment := range cfg.PreparationAmendments {
		if sameFiniteBinding(&state.Binding, &amendment.Previous) {
			state.Binding = amendment.Replacement
		}
	}
}

// AmendFinitePreparation changes content references only, never effort,
// leader/profile identity, activation, liveness policy or an existing run.
// Current verified creators need their write capability; other actors need a
// capable human or an exact reviewed operation grant. No legacy proof is inferred.
func (s *FileTeamStore) AmendFinitePreparation(ctx context.Context, teamID, agentID, expected string, next *teamconfig.FiniteLeader, evidence string) error {
	s = s.forContext(ctx)
	return s.WithFiniteLeader(ctx, teamID, agentID, func(cfg *HeartbeatConfig, state *FiniteLeaderState, _ func() error) error {
		team, err := s.Get(ctx, teamID)
		if err != nil {
			return err
		}
		p, err := authn.RequireCapability(ctx, "prompt-manager:write")
		if err != nil {
			return err
		}
		creator := verifiedTeamCreator(ctx)
		if creator == nil {
			return authn.ErrOperation
		}
		_, bounded := authn.RequestOperationAuthorityFromContext(ctx)
		if p.Grant != nil || bounded || team.CreationProof == nil || *team.CreationProof != *creator {
			if _, err = authn.RequireOperation(ctx, "prompt-manager:write", "team.prepare", "prompt-manager:team/"+teamID); err != nil {
				return err
			}
		}
		if !validFiniteReference(expected) || !validFiniteReference(evidence) || next == nil {
			return fmt.Errorf("amend requires exact expectedRevision, binding and retained evidenceRef")
		}
		if err = next.Validate(cfg.ProfileKey); err != nil {
			return err
		}
		if team.Purpose != "delivery" || team.Lifetime != "finite" || team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed || team.Coordination.LeadAgentID != agentID || !slices.Contains(team.EffortRefs, cfg.FiniteLeader.EffortRef) || team.Enabled || team.Archived || cfg.Enabled || cfg.FiniteLeader.Retired || !unreservedPreparation(state) || cfg.LastExecution != nil || cfg.LastSuccessfulExecution != nil || cfg.LastManualTriggerAt != "" {
			return ErrFinitePreparationConflict
		}
		allowed := *cfg.FiniteLeader
		allowed.AcceptedRevision = next.AcceptedRevision
		allowed.CoordinatorPromptRef = next.CoordinatorPromptRef
		allowed.SourceRefs = next.SourceRefs
		if !reflect.DeepEqual(&allowed, next) {
			return fmt.Errorf("amend cannot change effort, retirement or liveness policy")
		}
		// Lost-response replay returns the existing receipt without another write.
		for _, a := range cfg.PreparationAmendments {
			if a.Previous.AcceptedRevision == expected && reflect.DeepEqual(&a.Replacement, next) && a.EvidenceRef == evidence && a.Actor == *creator && reflect.DeepEqual(cfg.FiniteLeader, next) {
				return nil
			}
		}
		if cfg.FiniteLeader.AcceptedRevision != expected || next.AcceptedRevision == expected {
			return ErrFiniteLeaderStaleRevision
		}
		if len(cfg.PreparationAmendments) >= 32 {
			return fmt.Errorf("preparation amendment history limit reached; retain owner evidence")
		}
		cfg.PreparationAmendments = append(cfg.PreparationAmendments, FinitePreparationAmendment{Previous: *cfg.FiniteLeader, Replacement: *next, Actor: *creator, EvidenceRef: evidence, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		cfg.FiniteLeader = next
		cfg.UpdateTimestamp()
		return SaveJSON(filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json"), cfg)
	})
}
