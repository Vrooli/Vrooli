package flows

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"device-control/internal/execution"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/targetmodel"
)

// DesktopRunScope is supplied by owner admission, never a public actor field.
// Lease identity prevents caller-chosen run IDs from colliding across sessions.
type DesktopRunScope struct {
	Actor            string                 `json:"actor"`
	DeviceID         string                 `json:"device_id"`
	Surface          targetmodel.SurfaceRef `json:"surface"`
	DesktopSessionID string                 `json:"desktop_session_id"`
	LeaseID          string                 `json:"lease_id"`
	RunID            string                 `json:"run_id"`
}

// Terminal evidence retains the candidate and the helper-confirmed prefix.
// It does not imply promotion or grant permission to execute the candidate.
type DesktopRun struct {
	SavedRevision *DesktopFlowRevision `json:"saved_revision,omitempty"`
	Scope         DesktopRunScope      `json:"scope"`
	Digest        string               `json:"digest"`
	Flow          execution.Flow       `json:"flow"`
	Disposition   string               `json:"disposition"`
	Confirmed     uint32               `json:"confirmed"`
}

type DesktopFlowRevision struct {
	ID         string `json:"id"`
	Version    int32  `json:"version"`
	ContextKey string `json:"context_key"`
}

type DesktopRuns interface {
	Put(context.Context, DesktopRun) error
	Get(context.Context, DesktopRunScope) (DesktopRun, error)
	Promote(context.Context, DesktopRunScope, string, string, int32) (SavedDesktopFlow, error)
	GetSaved(context.Context, DesktopFlowScope, string, int32) (SavedDesktopFlow, error)
}

//go:embed desktop_runs.sql
var desktopRunsSchema string

type SQLiteDesktopRuns struct{ db AnchorDB }

func NewSQLiteDesktopRuns(ctx context.Context, db AnchorDB) (*SQLiteDesktopRuns, error) {
	if db == nil {
		return nil, ErrInvalidRequest
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(func() string { return desktopRunsSchema })); err != nil {
		return nil, err
	}
	return &SQLiteDesktopRuns{db: db}, nil
}

func validDesktopRunScope(s DesktopRunScope) bool {
	return s.Actor != "" && len(s.Actor) <= 1024 && s.DeviceID != "" && len(s.DeviceID) <= 256 && s.Surface.Validate() == nil && s.DesktopSessionID != "" && len(s.DesktopSessionID) <= 256 && s.LeaseID != "" && len(s.LeaseID) <= 128 && s.RunID != "" && len(s.RunID) <= 120 && !strings.ContainsRune(s.RunID, ':')
}

func (r *SQLiteDesktopRuns) Put(ctx context.Context, run DesktopRun) error {
	digest, err := hex.DecodeString(run.Digest)
	if err != nil || len(digest) != 32 || !validDesktopRunScope(run.Scope) || ValidateDesktopFlow(run.Flow) != nil || int(run.Confirmed) > len(run.Flow.Steps) || (run.Disposition != "passed" && run.Disposition != "incomplete") || (run.Disposition == "passed" && int(run.Confirmed) != len(run.Flow.Steps)) {
		return ErrInvalidRequest
	}
	payload, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if _, err = r.db.ExecContext(ctx, `INSERT INTO device_control_desktop_flow_runs(actor,lease_id,run_id,payload) VALUES (?,?,?,?) ON CONFLICT(actor,lease_id,run_id) DO NOTHING`, run.Scope.Actor, run.Scope.LeaseID, run.Scope.RunID, string(payload)); err != nil {
		return err
	}
	stored, err := r.read(ctx, run.Scope)
	if err != nil {
		return err
	}
	if stored != string(payload) {
		return errors.New("desktop run evidence conflict")
	}
	return nil
}

func (r *SQLiteDesktopRuns) read(ctx context.Context, scope DesktopRunScope) (string, error) {
	if !validDesktopRunScope(scope) {
		return "", ErrInvalidRequest
	}
	rows, err := r.db.QueryContext(ctx, `SELECT payload FROM device_control_desktop_flow_runs WHERE actor=? AND lease_id=? AND run_id=?`, scope.Actor, scope.LeaseID, scope.RunID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", ErrInvalidRequest
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return "", err
	}
	return raw, nil
}

func (r *SQLiteDesktopRuns) Get(ctx context.Context, scope DesktopRunScope) (DesktopRun, error) {
	raw, err := r.read(ctx, scope)
	if err != nil {
		return DesktopRun{}, err
	}
	var run DesktopRun
	if json.Unmarshal([]byte(raw), &run) != nil || run.Scope != scope {
		return DesktopRun{}, ErrInvalidRequest
	}
	return run, nil
}
