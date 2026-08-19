// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check; Registry backs the (non-Critical) "providers"
// dependency that aggregates capability checkers so /health surfaces
// provider degradation without flipping readiness.
type Deps struct {
	Pinger         database.Pinger
	Registry       *capabilities.Registry
	Service        string
	Version        string
	BuildTime      string
	SourceIdentity string
}

// NewHandler returns a handler that reports overall health, service
// metadata, the connectivity of the database dependency, and (when a
// Registry is wired) a non-Critical providers rollup. The database
// check stays Critical so a failed ping flips status="unhealthy" with
// HTTP 503; providers being down only degrade the response, never the
// readiness flag — the API can still route to other tiers.
func NewHandler(d Deps) http.HandlerFunc {
	buildTime := strings.TrimSpace(d.BuildTime)
	if buildTime == "" {
		buildTime = "unknown"
	}
	sourceIdentity := strings.TrimSpace(d.SourceIdentity)
	if sourceIdentity == "" {
		sourceIdentity = "unknown"
	}
	b := apihealth.New(d.Service).
		Version(d.Version).
		Metric("build_time", func(time.Time) any { return buildTime }).
		Metric("source_identity", func(time.Time) any { return sourceIdentity }).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical)

	if d.Registry != nil {
		b = b.Check(apihealth.Func("providers", func(ctx context.Context) error {
			states := d.Registry.Resolve(ctx)
			var down []string
			for _, s := range states {
				// Skip the rollup pseudo-entry.
				if s.Def.ID == "audio-tools" {
					continue
				}
				if s.Status == capabilities.StatusUnavailable {
					down = append(down, s.Def.ID)
				}
			}
			if len(down) > 0 {
				return errors.New(strings.Join(down, ","))
			}
			return nil
		}), apihealth.Optional)
	}

	return b.Handler()
}
