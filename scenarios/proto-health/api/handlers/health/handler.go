// Package health provides the /health endpoint.
package health

import (
	"net/http"
	"strconv"
	"time"

	"proto-health/internal/database"
	"proto-health/internal/httpx"

	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger           database.Pinger
	Service          string
	Version          string
	DescriptorSource *descriptorimage.Source
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
func NewHandler(d Deps) http.HandlerFunc {
	started := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		statusCode := http.StatusOK
		resp := &healthv1.Response{
			Status:        "healthy",
			Service:       d.Service,
			Timestamp:     now.UTC().Format(time.RFC3339),
			Readiness:     true,
			Version:       d.Version,
			UptimeSeconds: now.Sub(started).Seconds(),
			Dependencies: map[string]*healthv1.DependencyStatus{
				"database": {Connected: true},
			},
		}

		if err := d.Pinger.PingContext(r.Context()); err != nil {
			statusCode = http.StatusServiceUnavailable
			resp.Status = "unhealthy"
			resp.Readiness = false
			resp.Dependencies["database"] = &healthv1.DependencyStatus{
				Connected: false,
				Error:     err.Error(),
			}
		}
		if d.DescriptorSource != nil {
			snapshot, snapshotErr := d.DescriptorSource.Snapshot()
			if snapshot != nil {
				w.Header().Set("X-Proto-Descriptor-Digest", snapshot.Digest)
				w.Header().Set("X-Proto-Descriptor-Generation", strconv.FormatUint(snapshot.Generation, 10))
				w.Header().Set("X-Proto-Descriptor-Loaded-At", snapshot.LoadedAt.Format(time.RFC3339Nano))
			}
			if reloadErr := d.DescriptorSource.LastReloadError(); reloadErr != nil {
				w.Header().Set("X-Proto-Descriptor-Reload-Error", reloadErr.Error())
			} else if snapshotErr != nil {
				w.Header().Set("X-Proto-Descriptor-Reload-Error", snapshotErr.Error())
			}
		}

		httpx.WriteProto(w, statusCode, resp)
	}
}
