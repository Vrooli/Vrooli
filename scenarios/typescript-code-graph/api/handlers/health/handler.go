// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
//
// In typescript-code-graph the response is also augmented with the
// Node sidecar's lifecycle status — the sidecar is this scenario's
// single biggest failure mode and the health probe is the canonical
// surface for operators to see it. The augmentation is done as a
// JSON-merging response wrapper so the api-core schema stays
// authoritative for everything else.
package health

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"typescript-code-graph/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// SidecarStatusProvider is the narrow seam the health handler uses to
// read the supervisor's current state. *sidecar.Supervisor satisfies
// it; tests wire a stub directly.
type SidecarStatusProvider interface {
	Status() string
}

// statusOnly is a tiny adapter for plumbing typed sidecar.Status into
// the string-based seam without dragging the sidecar package into the
// health handler's test fixtures.
type statusOnly func() string

func (f statusOnly) Status() string { return f() }

// FuncProvider wraps a closure returning a status string. Used by
// main.go to bridge *sidecar.Supervisor's typed Status enum into the
// generic string seam.
func FuncProvider(f func() string) SidecarStatusProvider { return statusOnly(f) }

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check; Sidecar (optional) backs the sidecar_status field.
type Deps struct {
	Pinger  database.Pinger
	Service string
	Version string
	// Sidecar is optional — when nil, the response omits sidecar_status
	// so the template-shape handler tests stay green.
	Sidecar SidecarStatusProvider
}

// NewHandler returns a handler that reports overall health, service
// metadata, the connectivity of the database dependency, and the
// sidecar's lifecycle status when a SidecarStatusProvider is wired.
// A failed database ping flips the response to status="unhealthy" with
// HTTP 503; the sidecar status is reported informationally and does
// not by itself flip the overall status (a degraded sidecar is still
// recoverable; operators want to see the actual reason via the
// dedicated field rather than a generic 503).
func NewHandler(d Deps) http.HandlerFunc {
	inner := apihealth.New(d.Service).
		Version(d.Version).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical).
		Handler()

	if d.Sidecar == nil {
		return inner
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Capture the inner handler's response so we can splice in the
		// sidecar fields. The api-core handler only writes JSON; a buffer
		// is both cheap and lossless.
		rec := &bufferingRecorder{header: http.Header{}, code: http.StatusOK}
		inner(rec, r)

		// Decode → augment → re-encode. If the inner body is somehow
		// not JSON (defensive), pass it through untouched.
		var payload map[string]any
		if err := json.Unmarshal(rec.body.Bytes(), &payload); err == nil {
			payload["sidecar_status"] = d.Sidecar.Status()
			out, _ := json.Marshal(payload)
			for k, vs := range rec.header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("Content-Length", "")
			w.WriteHeader(rec.code)
			_, _ = w.Write(out)
			return
		}

		// Fallback: pass through untouched.
		for k, vs := range rec.header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.code)
		_, _ = w.Write(rec.body.Bytes())
	}
}

// bufferingRecorder captures status/headers/body so we can post-process
// the JSON response from api-core/health before sending it. Lighter
// than httptest.ResponseRecorder (no testing dependency).
type bufferingRecorder struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (r *bufferingRecorder) Header() http.Header { return r.header }

func (r *bufferingRecorder) WriteHeader(code int) { r.code = code }

func (r *bufferingRecorder) Write(b []byte) (int, error) { return r.body.Write(b) }
