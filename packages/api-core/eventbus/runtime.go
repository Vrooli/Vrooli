package eventbus

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/demand"
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
func AutomaticRuntime(ctx context.Context, next http.Handler) (http.Handler, func()) {
	target := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO"))
	requestID := rand.Text()
	return automaticRuntime(ctx, next, target, strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE")), func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	}, func(ctx context.Context) (*demand.Hold, error) {
		return demand.AcquireLifetime(ctx, demand.Client{}, demand.AcquireRequest{
			Scenario: "vrooli-events", ConsumerID: "eventbus:" + target, Kind: demand.KindDependency,
			RequestID: requestID, TTL: 10 * time.Minute,
		}, "event runtime stopped")
	})
}

func automaticRuntime(ctx context.Context, next http.Handler, target, baseURL string, resolve func(context.Context) (string, error), acquire func(context.Context) (*demand.Hold, error)) (http.Handler, func()) {
	if target == "" || target == "vrooli-events" {
		return next, func() {}
	}
	cache := NewCache()
	client, setEndpoint := newDynamicClient(baseURL)
	var stop func()
	if client.Enabled() {
		stop = StartRefresher(ctx, client, cache, RefreshConfig{})
	} else {
		stop = startDiscoveryRefresher(ctx, client, setEndpoint, cache, RefreshConfig{}, resolve, acquire)
	}
	handler := Middleware(MiddlewareConfig{
		Target: target, Reporter: client, ReceiptPolicy: cache,
		Operation:         func(r *http.Request) string { return r.Method + " " + r.URL.Path },
		Projection:        automaticProjection,
		Correlation:       VerifiedCorrelation,
		SourceFromRequest: automaticSource,
	})(next)
	return runtimeHealthHeaders(cache, handler), stop
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
func startDiscoveryRefresher(ctx context.Context, client Client, setEndpoint func(string), cache *Cache, cfg RefreshConfig, resolve func(context.Context) (string, error), acquire func(context.Context) (*demand.Hold, error)) func() {
	if cache == nil || client.Enabled() || resolve == nil {
		return func() {}
	}
	ctx, cancelRuntime := context.WithCancel(ctx)
	done := make(chan struct{})
	cfg = cfg.normalized()
	go func() {
		defer close(done)
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
				workCtx := ctx
				var hold *demand.Hold
				if acquire != nil {
					hold, err = acquire(ctx)
					if err == nil {
						workCtx = hold.Context()
					}
				}
				if err == nil {
					setEndpoint(base)
					stop := StartRefresher(workCtx, client, cache, cfg)
					<-workCtx.Done()
					stop()
					setEndpoint("")
					if hold != nil {
						_ = hold.Close()
					}
				}
			}
			if ctx.Err() != nil {
				return
			}
			wait = cfg.Jitter(backoff)
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}()
	return func() { cancelRuntime(); <-done }
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
