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

	"image-tools/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger  database.Pinger
	Service string
	Version string
	// ModelRuntime, when set, is a NON-critical check that reports whether every
	// enabled model is actually runnable (Python env provisioned + install-time
	// load-smoke passed). A failure degrades health (never 503) so an operator
	// sees "installed but broken" models on the standard probe. nil → not added.
	ModelRuntime func(ctx context.Context) error
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
	if d.ModelRuntime != nil {
		b = b.Check(apihealth.Func("model_runtime", d.ModelRuntime), apihealth.Optional)
	}
	return b.Handler()
}
