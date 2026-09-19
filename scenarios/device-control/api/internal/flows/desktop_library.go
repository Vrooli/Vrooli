package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"device-control/internal/execution"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/targetmodel"
)

// Saved flow access retains actor and destination authority across source lease
// expiry. A saved revision is a procedure, never a new desktop grant.
type DesktopFlowScope struct {
	Actor      string                 `json:"actor"`
	DeviceID   string                 `json:"device_id"`
	Surface    targetmodel.SurfaceRef `json:"surface"`
	ContextKey string                 `json:"context_key"`
}

type SavedDesktopFlow struct {
	ID           string           `json:"id"`
	Version      int32            `json:"version"`
	Scope        DesktopFlowScope `json:"scope"`
	Source       DesktopRunScope  `json:"source"`
	SourceDigest string           `json:"source_digest"`
	Flow         execution.Flow   `json:"flow"`
	CreatedAt    string           `json:"created_at"`
}

func validDesktopFlowScope(s DesktopFlowScope) bool {
	return s.Actor != "" && s.DeviceID != "" && s.Surface.Validate() == nil && strings.TrimSpace(s.ContextKey) != "" && len(s.ContextKey) <= 256
}

func (r *SQLiteDesktopRuns) GetSaved(ctx context.Context, scope DesktopFlowScope, id string, version int32) (SavedDesktopFlow, error) {
	if !validDesktopFlowScope(scope) || id == "" || len(id) > 128 || version < 1 {
		return SavedDesktopFlow{}, ErrInvalidRequest
	}
	rows, err := r.db.QueryContext(ctx, `SELECT payload FROM device_control_desktop_saved_flows WHERE actor=? AND id=? AND version=?`, scope.Actor, id, version)
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return SavedDesktopFlow{}, err
		}
		return SavedDesktopFlow{}, ErrInvalidRequest
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return SavedDesktopFlow{}, err
	}
	var saved SavedDesktopFlow
	if json.Unmarshal([]byte(raw), &saved) != nil || saved.Scope != scope || saved.ID != id || saved.Version != version {
		return SavedDesktopFlow{}, ErrInvalidRequest
	}
	return saved, nil
}

// Promote loads immutable terminal evidence; callers cannot submit a fabricated
// result. Repairs preserve outcome checks and use atomic revision comparison.
func (r *SQLiteDesktopRuns) Promote(ctx context.Context, source DesktopRunScope, cohort, id string, expected int32) (SavedDesktopFlow, error) {
	if expected < 0 || expected == math.MaxInt32 || (id == "" && expected != 0) || (id != "" && expected < 1) || len(id) > 128 {
		return SavedDesktopFlow{}, ErrInvalidRequest
	}
	run, err := r.Get(ctx, source)
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	scope := DesktopFlowScope{Actor: source.Actor, DeviceID: source.DeviceID, Surface: source.Surface, ContextKey: cohort}
	if !validDesktopFlowScope(scope) || run.Disposition != "passed" || int(run.Confirmed) != len(run.Flow.Steps) || ValidateDesktopFlow(run.Flow) != nil || run.Flow.Steps[len(run.Flow.Steps)-1].Kind != "desktop-text-assert" {
		return SavedDesktopFlow{}, ErrInvalidRequest
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,version FROM device_control_desktop_saved_flows WHERE actor=? AND source_lease_id=? AND source_run_id=? AND context_key=?`, source.Actor, source.LeaseID, source.RunID, cohort)
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	if rows.Next() {
		var priorID string
		var priorVersion int32
		err := rows.Scan(&priorID, &priorVersion)
		rows.Close()
		if err != nil {
			return SavedDesktopFlow{}, err
		}
		if (id == "" && priorVersion != 1) || (id != "" && (id != priorID || expected+1 != priorVersion)) {
			return SavedDesktopFlow{}, fmt.Errorf("source already promoted with different revision intent")
		}
		return r.GetSaved(ctx, scope, priorID, priorVersion)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	if id == "" {
		id = uuid.NewString()
	} else {
		prior, err := r.GetSaved(ctx, scope, id, expected)
		if err != nil {
			return SavedDesktopFlow{}, err
		}
		if err := PreserveFlowChecks(prior.Flow, run.Flow); err != nil {
			return SavedDesktopFlow{}, err
		}
	}
	saved := SavedDesktopFlow{ID: id, Version: expected + 1, Scope: scope, Source: source, SourceDigest: run.Digest, Flow: run.Flow, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	payload, err := json.Marshal(saved)
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO device_control_desktop_saved_flows(actor,id,version,source_lease_id,source_run_id,context_key,payload) SELECT ?,?,?,?,?,?,? WHERE COALESCE((SELECT MAX(version) FROM device_control_desktop_saved_flows WHERE actor=? AND id=?),0)=?`, scope.Actor, id, saved.Version, source.LeaseID, source.RunID, cohort, string(payload), scope.Actor, id, expected)
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return SavedDesktopFlow{}, err
	}
	if n != 1 {
		return SavedDesktopFlow{}, fmt.Errorf("flow version conflict")
	}
	return saved, nil
}
