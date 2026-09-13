package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"prompt-manager/internal/teamconfig"
)

// One PM lifecycle owns a runtime root. Share serialization with configuration
// writes, including routed views, so disable/retirement linearize with dispatch.
var finiteLeaderMu sync.Mutex

// FiniteLeaderState is the heartbeat's admission receipt, not an effort ledger
// or a queue. The exact identity remains after terminal owner observations.
type FiniteLeaderState struct {
	Version         int                     `json:"version"`
	TeamID          string                  `json:"teamId"`
	AgentID         string                  `json:"agentId"`
	Binding         teamconfig.FiniteLeader `json:"binding"`
	ProfileKey      string                  `json:"profileKey"`
	ID              string                  `json:"id"`
	TaskID          string                  `json:"taskId,omitempty"`
	RunID           string                  `json:"runId,omitempty"`
	TaskStarted     bool                    `json:"taskStarted"`
	DispatchStarted bool                    `json:"dispatchStarted"`
	CreatedAt       string                  `json:"createdAt"`
	Status          string                  `json:"status"`
	Error           string                  `json:"error,omitempty"`
}

func sameFiniteBinding(a, b *teamconfig.FiniteLeader) bool {
	if a == nil || b == nil {
		return a == b
	}
	x, y := *a, *b
	x.Retired, y.Retired = false, false
	return reflect.DeepEqual(x, y)
}

func validateFiniteLeaderUpdate(old, next *HeartbeatConfig) error {
	if err := next.FiniteLeader.Validate(next.ProfileKey, next.Supervision); err != nil {
		return err
	}
	if old == nil || old.FiniteLeader == nil {
		// Existing ordinary dispatches have no durable uncertainty fence. Migration
		// uses a new disabled member after its predecessor is settled by its owner.
		if old != nil && next.FiniteLeader != nil && (old.Enabled || old.LastExecution != nil) {
			return fmt.Errorf("finite leader migration requires an unused disabled heartbeat")
		}
		return nil
	}
	if !sameFiniteBinding(old.FiniteLeader, next.FiniteLeader) || old.ProfileKey != next.ProfileKey {
		return fmt.Errorf("finite leader identity and profile are immutable; retain the original owner binding")
	}
	if old.FiniteLeader.Retired && !next.FiniteLeader.Retired {
		return fmt.Errorf("finite leader retirement is irreversible")
	}
	return nil
}

func (s *FileTeamStore) finiteLeaderPath(effortRef string) string {
	return filepath.Join(s.runtimeDataRoot, "finite-leaders", fmt.Sprintf("%x.json", sha256.Sum256([]byte(effortRef))))
}

// WithFiniteLeader serializes reservation, dispatch and binding controls. Save
// publishes via the normal atomic runtime store before each external effect.
// Callbacks must not write heartbeat configuration while holding this boundary.
func (s *FileTeamStore) WithFiniteLeader(ctx context.Context, teamID, agentID string, fn func(*HeartbeatConfig, *FiniteLeaderState, func() error) error) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.WithFiniteLeader(ctx, teamID, agentID, fn)
	}
	finiteLeaderMu.Lock()
	defer finiteLeaderMu.Unlock()
	cfg, err := s.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		return err
	}
	if cfg == nil || cfg.FiniteLeader == nil {
		return fmt.Errorf("finite leader binding unavailable")
	}
	if err := cfg.FiniteLeader.Validate(cfg.ProfileKey, cfg.Supervision); err != nil {
		return err
	}
	path := s.finiteLeaderPath(cfg.FiniteLeader.EffortRef)
	state, err := LoadJSON[FiniteLeaderState](path)
	if errors.Is(err, os.ErrNotExist) {
		state, err = &FiniteLeaderState{Version: 1, TeamID: teamID, AgentID: agentID, Binding: *cfg.FiniteLeader, ProfileKey: cfg.ProfileKey, Status: "idle"}, nil
	}
	if err != nil {
		return err
	}
	if state.Version != 1 || state.TeamID != teamID || state.AgentID != agentID || !sameFiniteBinding(&state.Binding, cfg.FiniteLeader) || state.ProfileKey != cfg.ProfileKey {
		return fmt.Errorf("finite leader reservation conflicts with the exact effort binding; retain owner identity")
	}
	if (state.TaskStarted && state.ID == "") || (state.DispatchStarted && (!state.TaskStarted || state.TaskID == "")) || (state.RunID != "" && !state.DispatchStarted) {
		return fmt.Errorf("finite leader reservation has inconsistent dispatch identity; preserve for owner recovery")
	}
	return fn(cfg, state, func() error { return SaveJSON(path, state) })
}

// ReadFiniteLeader is a pure runtime read; it never reconciles or admits work.
func (s *FileTeamStore) ReadFiniteLeader(ctx context.Context, teamID, agentID string) (*FiniteLeaderState, error) {
	var out *FiniteLeaderState
	err := s.WithFiniteLeader(ctx, teamID, agentID, func(_ *HeartbeatConfig, state *FiniteLeaderState, _ func() error) error {
		out = state
		return nil
	})
	return out, err
}

// RecordFiniteLeaderExecution updates ordinary heartbeat health under the same
// control lock. Observation never rewrites enabled, retirement or the binding.
func (s *FileTeamStore) RecordFiniteLeaderExecution(ctx context.Context, teamID, agentID, identity string, result *HeartbeatExecResult) (bool, error) {
	s = s.forContext(ctx)
	changed := false
	err := s.WithFiniteLeader(ctx, teamID, agentID, func(cfg *HeartbeatConfig, state *FiniteLeaderState, _ func() error) error {
		if state.ID != identity || state.RunID != result.RunID {
			return fmt.Errorf("finite leader heartbeat observation has a different identity")
		}
		if reflect.DeepEqual(cfg.LastExecution, result) {
			return nil
		}
		previous := cfg.LastExecution
		cfg.LastExecution = result
		if result.EndedAt != "" && (previous == nil || previous.EndedAt == "" || previous.StartedAt != result.StartedAt) {
			cfg.RecordHeartbeatStatus(result.Status)
		}
		cfg.RefreshHealth()
		cfg.UpdateTimestamp()
		if err := SaveJSON(filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json"), cfg); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}
