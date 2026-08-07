// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"context"
	"database/sql"
	"net/http"

	"vrooli-memory/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger        database.Pinger
	Service       string
	Version       string
	MaintenanceDB *sql.DB
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
func NewHandler(d Deps) http.HandlerFunc {
	b := apihealth.New(d.Service).
		Version(d.Version).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical)
	if d.MaintenanceDB != nil {
		b = b.Check(maintenanceCheck(d.MaintenanceDB), apihealth.Optional)
	}
	return b.Handler()
}

// maintenanceCheck deliberately remains optional: the process is healthy
// while its first startup tick is still running. Once a run exists, its ID is
// exposed in the standard dependency's database field so probes can prove
// which durable run the health response reflects without changing the common
// health wire contract.
func maintenanceCheck(db *sql.DB) apihealth.Checker {
	return apihealth.CheckerFunc(func(ctx context.Context) apihealth.CheckResult {
		var id, completed string
		err := db.QueryRowContext(ctx, `SELECT id,completed_at FROM maintenance_runs ORDER BY started_at DESC,id DESC LIMIT 1`).Scan(&id, &completed)
		if err == sql.ErrNoRows {
			return apihealth.CheckResult{Name: "maintenance", Connected: true, Database: "pending"}
		}
		if err != nil {
			return apihealth.CheckResult{Name: "maintenance", Connected: false, Error: err}
		}
		return apihealth.CheckResult{Name: "maintenance", Connected: true, Database: id + " completed_at=" + completed}
	})
}
