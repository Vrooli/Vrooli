package routing

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type cacheObservationDoer struct {
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
}

func (d *cacheObservationDoer) Do(req *http.Request) (*http.Response, error) {
	d.mu.Lock()
	d.calls++
	if d.calls == 1 {
		close(d.started)
	}
	d.mu.Unlock()
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-d.release:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":[{"id":"one","title":"one","score":0.9}]}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
}

func TestProviderSingleflightKeepsResolverCacheObservationPerCaller(t *testing.T) {
	resolver := discovery.NewResolver(discovery.ResolverConfig{
		CacheTTL: time.Minute,
		CommandRunner: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("4321\n"), nil
		},
	})
	doer := &cacheObservationDoer{started: make(chan struct{}), release: make(chan struct{})}
	router := &Router{
		deps:          Deps{Resolver: &discoveryURLResolver{resolver: resolver}, Doer: doer, Now: time.Now, PerProviderTimeout: time.Second},
		providerCalls: make(map[string]*providerCall),
	}
	provider := &registryv1.ProviderDescriptor{
		ProviderId: "test-provider",
		Type:       "command",
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId:   "test-peer",
			Path:         "/search",
			Method:       registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}}}`,
		}}},
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score"},
	}

	firstCtx, firstObservation := discovery.ObserveResolverCache(context.Background())
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		router.callProvider(firstCtx, provider, "same query", 1, "")
	}()
	<-doer.started

	secondCtx, secondObservation := discovery.ObserveResolverCache(context.Background())
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		router.callProvider(secondCtx, provider, "same query", 1, "")
	}()
	waitForProviderWaiters(t, router, 2)
	close(doer.release)
	<-firstDone
	<-secondDone

	firstHits, firstMisses := firstObservation.Snapshot()
	secondHits, secondMisses := secondObservation.Snapshot()
	if firstHits != 0 || firstMisses != 1 || secondHits != 1 || secondMisses != 0 {
		t.Fatalf("first query cache counts hits:%d misses:%d; second hits:%d misses:%d", firstHits, firstMisses, secondHits, secondMisses)
	}
	doer.mu.Lock()
	defer doer.mu.Unlock()
	if doer.calls != 1 {
		t.Fatalf("shared provider HTTP calls = %d, want 1", doer.calls)
	}
}

func waitForProviderWaiters(t *testing.T, router *Router, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		router.providerCallMu.Lock()
		for _, call := range router.providerCalls {
			if call.waiters >= want {
				router.providerCallMu.Unlock()
				return
			}
		}
		router.providerCallMu.Unlock()
		runtime.Gosched()
	}
	t.Fatalf("provider call did not reach %d waiters", want)
}

type discoveryURLResolver struct{ resolver *discovery.Resolver }

func (r *discoveryURLResolver) ResolveScenarioURL(ctx context.Context, scenario string) (string, error) {
	return r.resolver.ResolveScenarioURLDefault(ctx, scenario)
}
