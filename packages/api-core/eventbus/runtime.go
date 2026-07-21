package eventbus

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
)

// AutomaticRuntime is installed once by api-core/server.Run. It deliberately
// has no scenario configuration: target identity comes from VROOLI_SCENARIO,
// source/correlation from verified request provenance, and a missing Events
// service leaves the business handler untouched.
func AutomaticRuntime(next http.Handler) http.Handler {
	target := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO"))
	if target == "" || target == "vrooli-events" {
		return next
	}
	cache := NewCache()
	client, setEndpoint := newDynamicClient(strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE")))
	if client.Enabled() {
		StartRefresher(context.Background(), client, cache, RefreshConfig{})
	} else {
		// Discovery is deliberately asynchronous. A scenario must never delay
		// startup or a business request while Events is down or not yet started.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
			if err == nil && strings.TrimSpace(base) != "" {
				setEndpoint(base)
				StartRefresher(context.Background(), client, cache, RefreshConfig{})
			}
		}()
	}
	return Middleware(MiddlewareConfig{Target: target, Reporter: client, ReceiptPolicy: cache,
		Operation:         func(r *http.Request) string { return r.Method + " " + r.URL.Path },
		Projection:        automaticProjection,
		Correlation:       VerifiedCorrelation,
		SourceFromRequest: automaticSource,
	})(next)
}

// automaticProjection decodes only a bounded JSON response object. The policy
// cache strips every unlisted key, so this never becomes implicit id capture.
func automaticProjection(_ *http.Request, status int, body []byte) (map[string]any, bool) {
	if status < 200 || status >= 400 || len(body) == 0 {
		return nil, false
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false
	}
	return response, true
}

func automaticSource(r *http.Request) string {
	p := provenance.FromContext(r.Context())
	if p.Invocation.Scenario != "" {
		return p.Invocation.Scenario
	}
	return "system"
}
