// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"context"
	"net/http"
	"time"

	"notification-hub/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger       database.Pinger
	Identity     interface{ Reachable(context.Context) error }
	Service      string
	Version      string
	TrustPosture string
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
func NewHandler(d Deps) http.HandlerFunc {
	checker := apihealth.New(d.Service)
	if d.Identity != nil {
		checker = checker.Check(apihealth.Func("owneridentity", d.Identity.Reachable), apihealth.Critical)
	}
	return checker.
		Version(d.Version).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical).
		Metric("trust_posture", func(time.Time) any { return d.TrustPosture }).
		Handler()
}
