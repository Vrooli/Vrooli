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
	"errors"
	"fmt"
	"net/http"

	"vrooli-memory/internal/maintenance"

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
	Canopy        CanopyReporter
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
	if d.Canopy != nil {
		b = b.Check(canopyCheck(d.Canopy), apihealth.Optional)
	}
	return b.Handler()
}

// CanopyReporter is the seam onto the maintenance record. Health reads the
// engine's own recorded numbers through it rather than opening the database,
// which keeps raw SQL out of the handler layer.
type CanopyReporter interface {
	Latest(context.Context) (maintenance.Run, error)
}

// canopyCheck surfaces the compaction backlog. It is Optional on purpose: a
// frontier far above target does not stop recall, and reporting the scenario
// unhealthy would hand a quality signal to a liveness remediator.
//
// It reports the frontier size and target the compaction engine recorded on its
// last pass, and deliberately recomputes nothing. Eligibility is four guards
// deep - policy, pin, recency floor, and vector presence - and a surface that
// re-derives it drifts from the engine and reports numbers no operator can act
// on. The first version of this check did exactly that and read 273 against the
// engine's 16. Reading the recorded pair also keeps a thirty-second probe from
// loading every candidate.
func canopyCheck(reporter CanopyReporter) apihealth.Checker {
	return apihealth.CheckerFunc(func(ctx context.Context) apihealth.CheckResult {
		run, err := reporter.Latest(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apihealth.CheckResult{Name: "canopy", Connected: true, Database: "pending"}
			}
			return apihealth.CheckResult{Name: "canopy", Connected: false, Error: err}
		}
		c := run.Compaction
		if c.Status == "" || c.Status == "not_configured" {
			return apihealth.CheckResult{Name: "canopy", Connected: true, Database: "not_configured"}
		}
		detail := fmt.Sprintf("eligible_frontier=%d target=%d last_pass=%s", c.FrontierAfter, c.Target, c.Status)
		if c.Status == "failed" {
			return apihealth.CheckResult{
				Name: "canopy", Connected: true, Database: detail,
				Error: fmt.Errorf("last compaction pass failed: %s", c.Error),
			}
		}
		if c.FrontierAfter > c.Target {
			return apihealth.CheckResult{
				Name: "canopy", Connected: true, Database: detail + " status=lagging",
				Error: fmt.Errorf("compaction backlog: %d eligible nodes against target %d", c.FrontierAfter, c.Target),
			}
		}
		return apihealth.CheckResult{Name: "canopy", Connected: true, Database: detail + " status=ok"}
	})
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
