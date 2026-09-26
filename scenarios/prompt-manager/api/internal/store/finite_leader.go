package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"prompt-manager/internal/teamconfig"

	"github.com/google/uuid"
)

// One PM lifecycle owns a runtime root. Share serialization with configuration
// writes, including routed views, so disable/retirement linearize with dispatch.
var finiteLeaderMu sync.Mutex

// FiniteLeaderState is the heartbeat's admission receipt, not an effort ledger
// or a queue. The exact identity remains after terminal owner observations.
type FiniteLeaderState struct {
	Version           int                      `json:"version"`
	TeamID            string                   `json:"teamId"`
	AgentID           string                   `json:"agentId"`
	Binding           teamconfig.FiniteLeader  `json:"binding"`
	ProfileKey        string                   `json:"profileKey"`
	ID                string                   `json:"id"`
	TaskID            string                   `json:"taskId,omitempty"`
	RunID             string                   `json:"runId,omitempty"`
	TaskStarted       bool                     `json:"taskStarted"`
	DispatchStarted   bool                     `json:"dispatchStarted"`
	CreatedAt         string                   `json:"createdAt"`
	Status            string                   `json:"status"`
	Error             string                   `json:"error,omitempty"`
	Completed         *FiniteLeaderCompletion  `json:"completed,omitempty"`
	CompletionHistory []FiniteLeaderCompletion `json:"completionHistory,omitempty"`
	RestartHistory    []FiniteLeaderRestart    `json:"restartHistory,omitempty"`
}

// FiniteLeaderCompletion is the retained completion receipt for a finite
// effort. It is a lifecycle fact, not the effort's outcome ledger; the
// evidence reference names the owner artifact that holds the outcome.
type FiniteLeaderCompletion struct {
	ReceiptID   string `json:"receiptId"`
	Revision    string `json:"revision"`
	EvidenceRef string `json:"evidenceRef"`
	CompletedAt string `json:"completedAt"`
}

// FiniteLeaderRestart records an explicit fresh-run recovery boundary. The
// accepted effort revision remains stable; only the consumed owner-run
// reservation is superseded after its exact run is known to be terminal.
type FiniteLeaderRestart struct {
	RunID       string `json:"runId"`
	TaskID      string `json:"taskId,omitempty"`
	Revision    string `json:"revision"`
	EvidenceRef string `json:"evidenceRef"`
	RestartedAt string `json:"restartedAt"`
}

// Completion and pause are distinct: a completion receipt is terminal until an
// explicit reopen, while a quota pause resumes on the owner's reset event.
var (
	ErrFiniteLeaderCompleted     = errors.New("finite leader effort is completed")
	ErrFiniteLeaderStaleRevision = errors.New("finite leader revision is not the accepted revision")
)

func validFiniteReference(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && len(value) <= 1024 && !strings.ContainsAny(value, "\x00\r\n")
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

// CompleteFiniteLeader records a revision-checked, idempotent completion
// receipt. Completion is terminal: it never resumes on a timer or reset event.
// Active and parked owner runs stay accounted until their owner settles them.
func (s *FileTeamStore) CompleteFiniteLeader(ctx context.Context, teamID, agentID, revision, evidenceRef string) (*FiniteLeaderCompletion, bool, error) {
	s = s.forContext(ctx)
	var out *FiniteLeaderCompletion
	changed := false
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err := s.WithFiniteLeader(ctx, teamID, agentID, func(cfg *HeartbeatConfig, state *FiniteLeaderState, save func() error) error {
		if !validFiniteReference(revision) || !validFiniteReference(evidenceRef) {
			return fmt.Errorf("finite leader completion requires an exact revision and retained evidence reference")
		}
		if revision != cfg.FiniteLeader.AcceptedRevision {
			return ErrFiniteLeaderStaleRevision
		}
		if state.Completed != nil {
			if state.Completed.Revision == revision && state.Completed.EvidenceRef == evidenceRef {
				out = state.Completed
				return nil
			}
			return ErrFiniteLeaderCompleted
		}
		state.Completed = &FiniteLeaderCompletion{ReceiptID: uuid.NewString(), Revision: revision, EvidenceRef: evidenceRef, CompletedAt: now}
		out = state.Completed
		changed = true
		return save()
	})
	return out, changed, err
}

// RestartFiniteLeader explicitly supersedes a terminal dispatched owner run
// that did not produce a completion receipt. It preserves the old identity and
// evidence reference, then clears only the reusable reservation. The accepted
// effort revision remains unchanged because this is run recovery, not a new
// product acceptance decision.
func (s *FileTeamStore) RestartFiniteLeader(ctx context.Context, teamID, agentID, revision, evidenceRef, ownerStatus string) error {
	s = s.forContext(ctx)
	return s.WithFiniteLeader(ctx, teamID, agentID, func(cfg *HeartbeatConfig, state *FiniteLeaderState, save func() error) error {
		if !validFiniteReference(revision) || !validFiniteReference(evidenceRef) {
			return fmt.Errorf("finite leader restart requires an exact revision and retained evidence reference")
		}
		if revision != cfg.FiniteLeader.AcceptedRevision {
			return ErrFiniteLeaderStaleRevision
		}
		if state.Completed != nil {
			return ErrFiniteLeaderCompleted
		}
		if !state.DispatchStarted || state.RunID == "" || !isTerminalFiniteLeaderStatus(ownerStatus) {
			return fmt.Errorf("finite leader restart requires a terminal dispatched owner run")
		}
		state.Status = ownerStatus
		state.RestartHistory = append(state.RestartHistory, FiniteLeaderRestart{
			RunID: state.RunID, TaskID: state.TaskID, Revision: revision,
			EvidenceRef: evidenceRef, RestartedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
		state.ID = ""
		state.TaskID = ""
		state.RunID = ""
		state.TaskStarted = false
		state.DispatchStarted = false
		state.CreatedAt = ""
		state.Status = "idle"
		state.Error = ""
		return save()
	})
}

func isTerminalFiniteLeaderStatus(status string) bool {
	switch status {
	case "RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED", "complete", "failed", "cancelled":
		return true
	default:
		return false
	}
}

// ReopenFiniteLeader is the explicit authorized operation that clears a
// completion receipt. A replacement revision must differ from the completed
// revision; the prior receipt is retained in history with its evidence.
func (s *FileTeamStore) ReopenFiniteLeader(ctx context.Context, teamID, agentID, revision, evidenceRef string) error {
	s = s.forContext(ctx)
	return s.WithFiniteLeader(ctx, teamID, agentID, func(_ *HeartbeatConfig, state *FiniteLeaderState, save func() error) error {
		if !validFiniteReference(revision) || !validFiniteReference(evidenceRef) {
			return fmt.Errorf("finite leader reopen requires an exact replacement revision and retained evidence reference")
		}
		if state.Completed == nil {
			return fmt.Errorf("finite leader effort is not completed")
		}
		if revision == state.Completed.Revision {
			return ErrFiniteLeaderStaleRevision
		}
		state.CompletionHistory = append(state.CompletionHistory, *state.Completed)
		state.Completed = nil
		return save()
	})
}
