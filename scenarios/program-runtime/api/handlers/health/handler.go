// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"program-runtime/internal/database"

	apihealth "github.com/vrooli/api-core/health"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger               database.Pinger
	Service              string
	Version              string
	SkippedManifestCount func() int
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
func NewHandler(d Deps) http.HandlerFunc {
	base := apihealth.New(d.Service).
		Version(d.Version).
		Check(apihealth.Func("database", func(ctx context.Context) error {
			return d.Pinger.PingContext(ctx)
		}), apihealth.Critical).Handler()
	if d.SkippedManifestCount == nil {
		return base
	}
	return func(w http.ResponseWriter, r *http.Request) {
		capture := &responseCapture{header: make(http.Header), status: http.StatusOK}
		base.ServeHTTP(capture, r)
		var payload map[string]any
		if err := json.Unmarshal(capture.body.Bytes(), &payload); err != nil {
			copyResponse(w, capture)
			return
		}
		metrics, ok := payload["metrics"].(map[string]any)
		if !ok {
			metrics = make(map[string]any)
		}
		metrics["skipped_manifests"] = d.SkippedManifestCount()
		payload["metrics"] = metrics
		capture.body.Reset()
		encoded, err := json.Marshal(payload)
		if err != nil {
			copyResponse(w, capture)
			return
		}
		_, _ = capture.body.Write(encoded)
		copyResponse(w, capture)
	}
}

type responseCapture struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (c *responseCapture) Header() http.Header { return c.header }

func (c *responseCapture) WriteHeader(status int) { c.status = status }

func (c *responseCapture) Write(body []byte) (int, error) { return c.body.Write(body) }

func copyResponse(w http.ResponseWriter, c *responseCapture) {
	for key, values := range c.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(c.status)
	_, _ = w.Write(c.body.Bytes())
}
