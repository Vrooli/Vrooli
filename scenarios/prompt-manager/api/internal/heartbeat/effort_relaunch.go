package heartbeat

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"prompt-manager/internal/store"
	"sync"
	"time"
)

// The supported FileTeamStore topology is one PM process per owner store.
// Serialize across runtime instances in that process; no multi-process/HA
// liveness certification is implied by the shared SQLite budget ledger.
var finiteEffortRelaunchMu sync.Mutex

// relaunchEffort admits the successor before recording terminal evidence or
// changing any Planner state. Its deterministic preparation key survives a
// crash between that commit and Planner saves; no deadline/budget is renewed.
// Native terminal evidence, not this method, settles the predecessor's slot.
func (f *FiniteLeaderRuntime) relaunchEffort(ctx context.Context, team, member string, terminal *Run) (*store.FiniteLeaderState, error) {
	finiteEffortRelaunchMu.Lock()
	defer finiteEffortRelaunchMu.Unlock()
	state, e := f.Executor.teamStore.ReadFiniteLeader(ctx, team, member)
	if e != nil {
		return state, e
	}
	cfg, e := f.Executor.teamStore.GetHeartbeatConfig(ctx, team, member)
	if e != nil {
		return state, e
	}
	if cfg == nil || cfg.FiniteLeader == nil || !cfg.FiniteLeader.KeepAlive || state.Completed != nil || !state.DispatchStarted || state.RunID != terminal.ID {
		return state, nil
	}
	qualified, e := f.prepareFiniteCaller(ctx, team, member, cfg)
	if e != nil {
		return state, e
	}
	ctx = qualified
	if e = f.eligible(ctx, team, member, cfg); e != nil {
		return state, e
	}
	consecutive := state.ConsecutiveRelaunches
	if healthyLeaderRun(terminal) {
		consecutive = 0
	}
	if consecutive >= livenessMaxRelaunches {
		return state, fmt.Errorf("finite liveness relaunch capped")
	}
	now := time.Now().UTC()
	if consecutive > 0 && len(state.RestartHistory) > 0 {
		last, err := time.Parse(time.RFC3339Nano, state.RestartHistory[len(state.RestartHistory)-1].RestartedAt)
		if next := last.Add(livenessRelaunchBackoff << (consecutive - 1)); err == nil && now.Before(next) {
			return state, fmt.Errorf("finite liveness relaunch backing off")
		}
	}
	candidate := &store.FiniteLeaderState{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("finite-successor:"+state.ID+":"+terminal.ID+":"+cfg.FiniteLeader.AcceptedRevision)).String(), Status: "queued", CreatedAt: now.Format(time.RFC3339Nano)}
	if e = f.reserveEffortLeader(ctx, cfg, candidate); e != nil {
		return state, e
	}
	// record owns queue completion and terminal evidence. It runs outside the
	// FileTeamStore finite-leader lock; failed record retains preparation capacity.
	if e = f.recordLocked(ctx, state, terminal); e != nil {
		return state, e
	}
	var out *store.FiniteLeaderState
	e = f.Executor.teamStore.WithFiniteLeader(ctx, team, member, func(currentCfg *store.HeartbeatConfig, current *store.FiniteLeaderState, save func() error) error {
		out = current
		if current.ID == candidate.ID {
			return nil
		} // completed same successor reconciliation
		if current.ID != state.ID || current.RunID != terminal.ID || current.Completed != nil || !current.DispatchStarted || currentCfg.FiniteLeader.AcceptedRevision != cfg.FiniteLeader.AcceptedRevision {
			return fmt.Errorf("finite successor changed concurrently; retain preparation")
		}
		current.RecordRestart(store.FiniteLeaderRestart{RunID: current.RunID, TaskID: current.TaskID, Revision: cfg.FiniteLeader.AcceptedRevision, EvidenceRef: "liveness-heartbeat:" + terminal.Status, RestartedAt: now.Format(time.RFC3339Nano)})
		current.ConsecutiveRelaunches = consecutive + 1
		current.ID, current.TaskID, current.RunID, current.CreatedAt, current.Error = candidate.ID, "", "", candidate.CreatedAt, ""
		current.TaskStarted, current.DispatchStarted = false, false
		current.Status = "queued"
		if err := save(); err != nil {
			return err
		}
		if memberOccupied(f.Queue.Status(team), member) {
			return nil
		}
		_, err := f.Queue.Enqueue(ctx, team, member, currentCfg.ProfileKey)
		return err
	})
	return out, e
}
