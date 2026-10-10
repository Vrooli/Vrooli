package heartbeat

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	"prompt-manager/internal/store"
	"reflect"
)

type serialEpisodeReader interface {
	ReadSerialHandoff(context.Context, effortauthority.Binding, string) (effortauthority.SerialHandoff, error)
}

// watchSerialIfDispatched intercepts every retained serial-leader read/trigger
// before ordinary observe/relaunch callbacks. Initial root admission remains
// the existing exact PM route; an uncertain dispatched root is held, not replayed.
func (f *FiniteLeaderRuntime) watchSerialIfDispatched(ctx context.Context, team, member string) (*store.FiniteLeaderState, bool, error) {
	if f.Efforts == nil {
		return nil, false, nil
	}
	binding, configured := f.Efforts.Bindings[team+"/"+member]
	if !configured {
		return nil, false, nil
	}
	if f.Efforts.Authority == nil {
		return nil, true, effortauthority.ErrRefused
	}
	policy, err := f.Efforts.Authority.CheckBinding(ctx, binding)
	if err != nil {
		return nil, true, err
	}
	if len(policy.SerialEdges) == 0 {
		return nil, false, nil
	}
	state, err := f.Executor.teamStore.ReadFiniteLeader(ctx, team, member)
	if err != nil {
		return state, true, err
	}
	if !state.DispatchStarted {
		return state, false, nil
	}
	cfg, err := f.Executor.teamStore.GetHeartbeatConfig(ctx, team, member)
	if err != nil {
		return state, true, err
	}
	qualified, err := f.prepareFiniteCaller(ctx, team, member, cfg)
	if err != nil {
		return state, true, err
	}
	if err = f.eligible(qualified, team, member, cfg); err != nil {
		return state, true, err
	}
	var out *store.FiniteLeaderState
	err = f.Executor.teamStore.WithFiniteLeader(qualified, team, member, func(current *store.HeartbeatConfig, s *store.FiniteLeaderState, save func() error) error {
		out = s
		if !s.DispatchStarted || s.ID != state.ID || s.TaskID != state.TaskID || s.RunID != state.RunID {
			return effortauthority.ErrRefused
		}
		// The store serializes disabled/retired binding changes; the protected owner
		// rechecks original scope, policy and deadline before any projection save.
		if _, err := f.prepareFiniteCaller(qualified, team, member, current); err != nil {
			return err
		}
		episodes, err := f.readSerialEpisodes(qualified, binding, policy, current, s)
		if err != nil {
			return err
		}
		latest := episodes[len(episodes)-1]
		if reflect.DeepEqual(s.SerialEpisodes, episodes) && s.SerialCurrentRunID == latest.RunID && s.SerialCurrentStatus == latest.Status {
			return nil
		}
		p, err := f.Efforts.Authority.CheckBinding(qualified, binding)
		if err != nil || effortauthority.Digest(p) != effortauthority.Digest(policy) || !f.Efforts.now().Before(policy.Deadline) || qualified.Err() != nil {
			return effortauthority.ErrRefused
		}
		s.SerialEpisodes = episodes
		s.SerialCurrentRunID = latest.RunID
		s.SerialCurrentStatus = latest.Status
		return save()
	})
	return out, true, err
}
func (f *FiniteLeaderRuntime) readSerialEpisodes(ctx context.Context, b effortauthority.Binding, p effortauthority.Policy, cfg *store.HeartbeatConfig, state *store.FiniteLeaderState) ([]store.FiniteLeaderSerialEpisode, error) {
	reader, ok := f.Efforts.Authority.(serialEpisodeReader)
	if !ok || p.Validate() != nil || p.MaxStarts < 1 || p.MaxStarts > 8 || f.Executor.agentClient == nil {
		return nil, effortauthority.ErrRefused
	}
	key := "finite-leader-" + state.ID
	slot, err := f.Efforts.Authority.ReadReservation(ctx, b, key)
	if err != nil || slot.Intent.Effect != "run.create" || slot.Intent.ParentRunID != "" || slot.Intent.Profile != cfg.ProfileKey || slot.Intent.Member != state.AgentID || (state.RunID != "" && slot.RunID != state.RunID) {
		return nil, effortauthority.ErrRefused
	}
	episodes := []store.FiniteLeaderSerialEpisode{}
	seenRuns := map[string]bool{}
	seenKeys := map[string]bool{}
	for n := 0; n < p.MaxStarts; n++ {
		i := slot.Intent
		if !slot.NativeBound || slot.RunID == "" || seenRuns[slot.RunID] || seenKeys[key] || i.IdempotencyKey != key || slot.IntentDigest != effortauthority.Digest(i) || i.PolicyID != p.ID || i.Epoch != p.Epoch || i.Client != p.Client || i.Repository != p.Repository || i.Effort != p.Effort || i.Revision != p.Revision || i.ContentDigest != p.ContentDigest || i.Team != p.Team || i.ProfileDigest == "" || i.ProfileDigest != p.Profiles[i.Profile] || i.SourceRunID != "" {
			return nil, effortauthority.ErrRefused
		}
		if _, err := uuid.Parse(slot.RunID); err != nil {
			return nil, effortauthority.ErrRefused
		}
		run, err := f.Executor.agentClient.GetRun(ctx, slot.RunID)
		if err != nil || run == nil || run.ID != slot.RunID || run.TaskID == "" || !knownSerialRunStatus(run.Status) || IsTerminalStatus(run.Status) != slot.Terminal {
			return nil, effortauthority.ErrRefused
		}
		// Every fresh episode retains the root task while AM owns an immutable
		// per-run task projection; no new model task is inferred here.
		if run.TaskID != state.TaskID || run.ParentRunID != i.ParentRunID {
			return nil, effortauthority.ErrRefused
		}
		seenRuns[slot.RunID] = true
		seenKeys[key] = true
		episodes = append(episodes, store.FiniteLeaderSerialEpisode{ReservationKey: key, RunID: slot.RunID, TaskID: run.TaskID, Member: i.Member, Profile: i.Profile, ParentRunID: i.ParentRunID, Status: run.Status, Terminal: slot.Terminal})
		if !slot.Terminal || n+1 == p.MaxStarts {
			return episodes, nil
		}
		nextKey := "serial-" + slot.RunID
		handoff, err := reader.ReadSerialHandoff(ctx, b, nextKey)
		if err != nil || handoff.SourceKey != key || handoff.SourceRunID != slot.RunID || handoff.Next.IdempotencyKey != nextKey || handoff.Next.Effect != "run.child" || handoff.Next.ParentRunID != slot.RunID || handoff.Next.SourceRunID != "" {
			return nil, effortauthority.ErrRefused
		}
		edge := false
		for _, e := range p.SerialEdges {
			if e.FromMember == i.Member && e.FromProfile == i.Profile && e.ToMember == handoff.Next.Member && e.ToProfile == handoff.Next.Profile {
				edge = true
			}
		}
		if !edge {
			return nil, effortauthority.ErrRefused
		}
		next, err := f.Efforts.Authority.ReadReservation(ctx, b, nextKey)
		if err != nil || effortauthority.Digest(next.Intent) != effortauthority.Digest(handoff.Next) {
			return nil, effortauthority.ErrRefused
		}
		key, slot = nextKey, next
	}
	return nil, effortauthority.ErrRefused
}

func knownSerialRunStatus(status string) bool {
	if IsTerminalStatus(status) || IsParkedStatus(status) {
		return true
	}
	switch status {
	case "pending", "starting", "running", "needs_review", "RUN_STATUS_PENDING", "RUN_STATUS_STARTING", "RUN_STATUS_RUNNING", "RUN_STATUS_NEEDS_REVIEW":
		return true
	}
	return false
}

const finiteSerialLeaderGuidance = "\n\nFinite serial episode owner guidance:\n" +
	"- Keep this episode inside the accepted effort, original task, immutable scope and original deadline. Perform useful bounded work; do not wait inside a run or invoke run identity, child creation, park, wake or authority reads.\n" +
	"- To request the next bounded episode, emit only this strict final codec output JSON: {\"finite_serial\":{\"version\":1,\"next_profile\":\"<exact allowed edge target>\",\"title\":\"...\",\"instruction\":\"...\",\"summary\":\"...\"}}. Select only the exact allowed target profile for your current owner-selected profile.\n" +
	"- Do not supply identities, run IDs, task IDs, reservation keys, grants or extended budgets. The native owner derives and binds those facts; this model output is task data, not caller proof.\n" +
	"- The native owner proceeds only after exact unit terminal proof, under the original conserved limits. A missing handoff ends this episode; it is not effort acceptance and does not authorize a replacement leader.\n" +
	"- Preserve the accepted receiving package, record changed/verified/remaining/unverified work and name external blockers. An owner refusal holds this route; never invent an alternate grant or launcher.\n"

func finitePromptGuidance(ctx context.Context) (string, error) {
	if caller, ok := effortCaller(ctx); ok {
		p, err := caller.authority.Authority.CheckBinding(ctx, caller.binding)
		if err != nil {
			return "", err
		}
		if len(p.SerialEdges) > 0 {
			raw, _ := json.Marshal(p.SerialEdges)
			return finiteSerialLeaderGuidance + "Owner-selected exact serial profile/member edges (data, not authority):\n" + string(raw), nil
		}
	}
	return finiteLeaderGuidance, nil
}

// An old terminal parent is not permission to reset the retained serial chain.
// A replacement commission is a separate exact owner installation, not Reopen.
func (f *FiniteLeaderRuntime) refuseSerialReset(ctx context.Context, team, member string) error {
	if f.Efforts == nil {
		return nil
	}
	b, configured := f.Efforts.Bindings[team+"/"+member]
	if !configured {
		return nil
	}
	if f.Efforts.Authority == nil {
		return effortauthority.ErrRefused
	}
	p, err := f.Efforts.Authority.CheckBinding(ctx, b)
	if err != nil || len(p.SerialEdges) > 0 {
		return effortauthority.ErrRefused
	}
	return nil
}
