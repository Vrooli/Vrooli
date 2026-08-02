package eventbus

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
)

const (
	RuntimeStateHeader       = "X-Vrooli-Events-Runtime-State"
	RuntimeArmedHeader       = "X-Vrooli-Events-Armed"
	RuntimePolicyCountHeader = "X-Vrooli-Events-Policy-Count"
	RuntimeLastRefreshHeader = "X-Vrooli-Events-Last-Refresh"
)

// AutomaticRuntime is installed once by api-core/server.Run. It deliberately
// has no scenario configuration: target identity comes from VROOLI_SCENARIO,
// source/correlation from verified request provenance, and a missing Events
// service leaves the business handler untouched.
func AutomaticRuntime(next http.Handler) http.Handler {
	target := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO"))
	return automaticRuntime(next, target, strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE")), func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	})
}

func automaticRuntime(next http.Handler, target, baseURL string, resolve func(context.Context) (string, error)) http.Handler {
	if target == "" || target == "vrooli-events" {
		return next
	}
	cache := NewCache()
	client, setEndpoint := newDynamicClient(baseURL)
	if client.Enabled() {
		StartRefresher(context.Background(), client, cache, RefreshConfig{})
	} else {
		startDiscoveryRefresher(context.Background(), client, setEndpoint, cache, RefreshConfig{}, resolve)
	}
	handler := Middleware(MiddlewareConfig{
		Target: target, Reporter: client, ReceiptPolicy: cache,
		Operation:         func(r *http.Request) string { return r.Method + " " + r.URL.Path },
		Projection:        automaticProjection,
		Correlation:       VerifiedCorrelation,
		SourceFromRequest: automaticSource,
	})(next)
	return runtimeHealthHeaders(cache, handler)
}

func runtimeHealthHeaders(cache *Cache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := cache.RuntimeState(time.Now())
		w.Header().Set(RuntimeStateHeader, state.State)
		w.Header().Set(RuntimeArmedHeader, strconv.FormatBool(state.Armed))
		w.Header().Set(RuntimePolicyCountHeader, strconv.Itoa(state.PolicyCount))
		if !state.LastRefresh.IsZero() {
			w.Header().Set(RuntimeLastRefreshHeader, state.LastRefresh.Format(time.RFC3339Nano))
		}
		next.ServeHTTP(w, r)
	})
}

// startDiscoveryRefresher uses the refresher's existing backoff and jitter to
// arm a dynamically addressed client. Discovery stays wholly asynchronous: a
// missing Events scenario cannot delay startup or any business request.
func startDiscoveryRefresher(ctx context.Context, client Client, setEndpoint func(string), cache *Cache, cfg RefreshConfig, resolve func(context.Context) (string, error)) {
	if cache == nil || client.Enabled() || resolve == nil {
		return
	}
	cfg = cfg.normalized()
	go func() {
		wait, backoff := time.Duration(0), cfg.MinBackoff
		for {
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			discoveryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			base, err := resolve(discoveryCtx)
			cancel()
			if err == nil && strings.TrimSpace(base) != "" {
				setEndpoint(base)
				StartRefresher(ctx, client, cache, cfg)
				return
			}
			wait = cfg.Jitter(backoff)
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}()
}

// automaticProjection decodes a bounded JSON response object. Connect clients
// may legitimately negotiate binary protobuf: in that case a declared policy
// still receives an empty candidate and emits a receipt, while its explicit
// projection remains empty. Dropping the whole receipt would make provenance
// depend on a caller's wire encoding.
func automaticProjection(request *http.Request, status int, body []byte) (map[string]any, bool) {
	if status < 200 || status >= 400 || len(body) == 0 {
		return nil, false
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		if requestProtocol(request) == "connect" {
			return map[string]any{}, true
		}
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
