package gates

import (
	"context"
	"database/sql"
)

type calibrationSchemaDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// createCalibrationEvidenceSchema owns the temporary evidence tables used by
// calibration fixtures. These tables mirror the gate read contract; they are
// fixture schema, not production migrations.
func createCalibrationEvidenceSchema(ctx context.Context, db calibrationSchemaDB, withViewports bool) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE reconcile_evidence (id TEXT PRIMARY KEY, scenario TEXT NOT NULL, document_kind TEXT NOT NULL, component_id TEXT NOT NULL, example_name TEXT NOT NULL, state_id TEXT NOT NULL, ax_node_json TEXT NOT NULL, measurement_json TEXT NOT NULL, checked_at TEXT NOT NULL)`); err != nil {
		return err
	}
	if withViewports {
		if _, err := db.ExecContext(ctx, `CREATE TABLE reconcile_evidence_viewports (evidence_id TEXT PRIMARY KEY, viewport_id TEXT NOT NULL, viewport_width INTEGER NOT NULL, viewport_height INTEGER NOT NULL)`); err != nil {
			return err
		}
	}
	return nil
}
