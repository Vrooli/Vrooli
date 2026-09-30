// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"context"
	"encoding/json"
	"net/http"

	"backdrop-studio/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger  database.Pinger
	Service string
	Version string
	// Fingerprint identifies the source tree this process was built from, so a
	// caller can tell a missing feature from a stale binary without guessing.
	// Injected rather than read globally so a handler test can pin it.
	Fingerprint string
	// SeedVersion is the catalog version this binary ships. An install whose
	// applied version is lower has not restarted since the upgrade.
	SeedVersion int
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
//
// The reported version carries the build fingerprint as semver build metadata
// ("1.0.0+a1b2c3d4e5f6"), so the one field every health consumer already reads
// answers "which source tree is this?" without a second request.
func NewHandler(d Deps) http.HandlerFunc {
	return apihealth.New(d.Service).
		Version(versionWithFingerprint(d.Version, d.Fingerprint)).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical).
		Handler()
}

const fingerprintDisplayLength = 12

func versionWithFingerprint(version, fingerprint string) string {
	if fingerprint == "" {
		return version
	}
	if len(fingerprint) > fingerprintDisplayLength {
		fingerprint = fingerprint[:fingerprintDisplayLength]
	}
	return version + "+" + fingerprint
}

// BuildReport is the machine-readable freshness answer. The integration lane
// compares Fingerprint against one it computes from the working tree and
// refuses to render anything on a mismatch, because a stale binary has twice
// produced audit findings that were simply false.
type BuildReport struct {
	Service            string `json:"service"`
	Version            string `json:"version"`
	Fingerprint        string `json:"fingerprint"`
	SeedVersion        int    `json:"seed_version"`
	AppliedSeedVersion int    `json:"applied_seed_version"`
}

// NewBuildHandler serves the freshness report. AppliedSeedVersion is resolved
// per request rather than cached, so an operator watching an upgrade sees it
// move.
func NewBuildHandler(d Deps, applied func(context.Context) (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := BuildReport{
			Service:     d.Service,
			Version:     versionWithFingerprint(d.Version, d.Fingerprint),
			Fingerprint: d.Fingerprint,
			SeedVersion: d.SeedVersion,
		}
		if applied != nil {
			if version, err := applied(r.Context()); err == nil {
				report.AppliedSeedVersion = version
			}
		}
		writeJSON(w, report)
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
