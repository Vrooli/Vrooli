// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
//
// Beyond liveness, health reports backup posture (OT-P0-010): when any
// target is overdue or its last run failed, an Optional "backups" check
// fails, flipping the overall status to "degraded" (readiness stays true,
// HTTP 200) and emitting a posture event for platform monitoring.
package health

import (
	"context"
	"errors"
	"net/http"

	"data-backup-manager/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// BackupPosture reports whether any registered target is overdue or its last
// run failed. The detail string is surfaced as the "backups" dependency error.
//
// seam: implemented by an adapter over runs.Service in main.go; tests inject a
// fake.
type BackupPosture interface {
	OverdueOrFailed(ctx context.Context) (degraded bool, detail string, err error)
}

// PostureEventSink receives a posture-degraded signal so platform monitoring
// (infra-health / system-monitor) can act on overdue/failed backups.
//
// seam: production wires a log-backed sink in main.go; tests assert emission.
type PostureEventSink interface {
	BackupPostureDegraded(ctx context.Context, detail string)
}

// Deps wires the seams the health handler needs. Service and Version are
// reported in the response envelope; Pinger backs the "database" dependency
// check. Posture and Events are optional — when Posture is nil the handler
// reports liveness only (the template behavior).
type Deps struct {
	Pinger  database.Pinger
	Service string
	Version string
	Posture BackupPosture
	Events  PostureEventSink
}

// NewHandler returns a handler that reports overall health, service metadata,
// the connectivity of the database dependency (Critical), and — when a
// BackupPosture is wired — the backup posture (Optional: a failure degrades
// rather than downs the service).
func NewHandler(d Deps) http.HandlerFunc {
	b := apihealth.New(d.Service).
		Version(d.Version).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical)

	if d.Posture != nil {
		b = b.Check(apihealth.Func("backups", func(ctx context.Context) error {
			degraded, detail, err := d.Posture.OverdueOrFailed(ctx)
			if err != nil {
				return err
			}
			if degraded {
				if d.Events != nil {
					d.Events.BackupPostureDegraded(ctx, detail)
				}
				return errors.New(detail)
			}
			return nil
		}), apihealth.Optional)
	}

	return b.Handler()
}
