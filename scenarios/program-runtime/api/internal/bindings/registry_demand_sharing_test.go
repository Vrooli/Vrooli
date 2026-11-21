package bindings

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/demand"
)

// sharedDemandFixture is the concurrent-safe counterpart of
// invocationDemandFixture. Each Acquire is a control-plane process spawn and a
// write to the runtime registry, so the counts are the contract.
type sharedDemandFixture struct {
	acquired atomic.Int64
	released atomic.Int64

	mu   sync.Mutex
	live int
	peak int
}

func (f *sharedDemandFixture) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	f.acquired.Add(1)
	f.mu.Lock()
	f.live++
	if f.live > f.peak {
		f.peak = f.live
	}
	f.mu.Unlock()
	return demand.Lease{LeaseID: req.LeaseID, Status: "active", ExpiresAt: time.Now().Add(req.TTL)}, nil
}

func (f *sharedDemandFixture) Renew(_ context.Context, id string, ttl time.Duration) (demand.Lease, error) {
	return demand.Lease{LeaseID: id, Status: "active", ExpiresAt: time.Now().Add(ttl)}, nil
}

func (f *sharedDemandFixture) Release(_ context.Context, id string, _ string) (demand.Lease, error) {
	f.released.Add(1)
	f.mu.Lock()
	f.live--
	f.mu.Unlock()
	return demand.Lease{LeaseID: id, Status: "released"}, nil
}

func (f *sharedDemandFixture) peakLive() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peak
}

// TestConcurrentInvocationsAtOneScenarioTakeOneLease pins the fix for the
// defect that silently blinded program briefings.
//
// A program's gather fan-out runs up to eight bindings at once. Each one used
// to acquire its own demand lease, which meant eight concurrent
// `vrooli scenario demand acquire` processes writing the same SQLite file; the
// losers came back as binding failures indistinguishable from the target being
// down, so a different two or three sources went dark on every run.
//
// Overlapping requests at one scenario are one statement of demand, so they
// must produce one lease.
func TestConcurrentInvocationsAtOneScenarioTakeOneLease(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	binding := registry.bindings[0]
	binding.Scenario = "fixture-target"

	fixture := &sharedDemandFixture{}
	registry.SetDemandLeaseClient(fixture)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
		return "http://fixture.invalid", nil
	})

	const callers = 8
	release := make(chan struct{})
	arrived := make(chan struct{}, callers)
	client := &http.Client{Transport: bindingRoundTripper(func(req *http.Request) (*http.Response, error) {
		// Hold every request in flight together so the invocations genuinely
		// overlap rather than queueing.
		arrived <- struct{}{}
		<-release
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}, nil
	})}

	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = registry.Execute(context.Background(), binding.GetId(), map[string]any{}, nil, false,
				InvocationMetadata{SessionID: "fixture-session"}, client)
		}()
	}
	for range callers {
		<-arrived
	}
	close(release)
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "caller %d", i)
	}
	require.Equal(t, int64(1), fixture.acquired.Load(), "overlapping requests at one scenario must share one lease")
	require.Equal(t, int64(1), fixture.released.Load(), "the shared lease must be released exactly once")
	require.Equal(t, 1, fixture.peakLive(), "never more than one lease live for one scenario")
}

// TestConcurrentInvocationsAtDifferentScenariosKeepSeparateLeases guards the
// other direction: sharing must not collapse demand for distinct targets.
func TestConcurrentInvocationsAtDifferentScenariosKeepSeparateLeases(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	fixture := &sharedDemandFixture{}
	registry.SetDemandLeaseClient(fixture)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
		return "http://fixture.invalid", nil
	})
	client := &http.Client{Transport: bindingRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}, nil
	})}

	binding := registry.bindings[0]
	for _, scenario := range []string{"source-ledger", "swarm-manager", "command-center"} {
		binding.Scenario = scenario
		_, err := registry.Execute(context.Background(), binding.GetId(), map[string]any{}, nil, false,
			InvocationMetadata{SessionID: "fixture-session"}, client)
		require.NoError(t, err)
	}
	require.Equal(t, int64(3), fixture.acquired.Load(), "distinct scenarios each need their own demand")
	require.Equal(t, int64(3), fixture.released.Load())
}
