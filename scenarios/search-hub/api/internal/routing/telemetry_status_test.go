package routing_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

type statusDoer struct {
	body        string
	err         error
	statusCalls int
	searchCalls int
}

type parallelStatusDoer struct {
	mu        sync.Mutex
	delay     time.Duration
	active    int
	maxActive int
}

func (d *parallelStatusDoer) Do(req *http.Request) (*http.Response, error) {
	if !strings.HasSuffix(req.URL.Path, "/status") {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[]}`)), Header: make(http.Header)}, nil
	}
	d.mu.Lock()
	d.active++
	if d.active > d.maxActive {
		d.maxActive = d.active
	}
	d.mu.Unlock()
	time.Sleep(d.delay)
	d.mu.Lock()
	d.active--
	d.mu.Unlock()
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"point_count":1}`)), Header: make(http.Header)}, nil
}

func (d *statusDoer) Do(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Path, "/status") {
		d.statusCalls++
	} else {
		d.searchCalls++
	}
	if d.err != nil {
		return nil, d.err
	}
	body := d.body
	if body == "" {
		body = `{"results":[]}`
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func providerWithStatus(body string) *registryv1.ProviderDescriptor {
	p := cliHealthCommands()
	p.StatusEndpoint = httpJSON("cli-health", "/status", "{}")
	return p
}

// recordingRecorder captures the samples the router emits so telemetry wiring
// is testable without the metrics store.
type recordingRecorder struct {
	samples []routing.TelemetrySample
}

func (r *recordingRecorder) Record(_ context.Context, s routing.TelemetrySample) {
	r.samples = append(r.samples, s)
}

func TestQueryRecordsTelemetry(t *testing.T) {
	rec := &recordingRecorder{}
	r := routing.NewRouter(routing.Deps{
		Lister:   threeProviderLister(),
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
		Recorder: rec,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", All: true})
	require.NoError(t, err)
	require.Len(t, rec.samples, 1, "one query records exactly one telemetry sample")

	s := rec.samples[0]
	require.NotEmpty(t, s.QueryHash)
	require.NotContains(t, s.QueryHash, "restart", "query text is hashed, never stored raw")
	require.ElementsMatch(t, []string{"command", "component", "record"}, s.RoutedTypes)
	// cli-health(1) + ui-health(1) + swarm-manager(1) = 3 hits across the fan-out.
	require.Equal(t, 3, s.ResultCount)
	require.Equal(t, 1, s.ProviderResults["cli-health.commands"].HitCount)
	require.GreaterOrEqual(t, s.ProviderResults["cli-health.commands"].LatencyMs, int64(0))
	require.Equal(t, resp.GetLatencyMs(), s.LatencyMs)
	require.False(t, s.Reranked)
	require.Equal(t, "explicit_all", s.RoutingMode)
	require.Equal(t, 3, s.EligibleProviderCount)
	require.Equal(t, 3, s.SelectedProviderCount)
	require.Equal(t, 3, s.SelectedLeafCount)
	require.Zero(t, s.WidenedLeafCount)
	require.False(t, s.FanoutWidthBoundReached)
	require.Zero(t, s.WithheldExternalCount)
	require.Zero(t, s.QueuedProviderCount)
	require.Empty(t, s.ResponseDegradeReason)
}

func TestQueryRecordsZeroResultAndDegraded(t *testing.T) {
	rec := &recordingRecorder{}
	// One provider, resolvable but the HTTP call errors ⇒ degraded, zero hits.
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     routeDoer{byURL: map[string]cannedResponse{}}, // no canned response ⇒ transport error
		Recorder: rec,
	})

	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.Len(t, rec.samples, 1)
	require.Equal(t, 0, rec.samples[0].ResultCount)
	require.True(t, rec.samples[0].Degraded)
	provider := rec.samples[0].ProviderResults["cli-health.commands"]
	require.True(t, provider.Degraded)
	require.Equal(t, "other", provider.DegradeReason)
	require.Equal(t, "provider_leg,zero_result", rec.samples[0].ResponseDegradeReason)
}

func TestResponseDegradeReasonClassifiesCoexistingCauses(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "down", Degraded: true}}
	require.Equal(t, "classifier,reranker_absent,provider_leg,zero_result",
		routing.ResponseDegradeReason(true, true, groups, 0))
}

func TestQueryNoRecorderIsNoop(t *testing.T) {
	// No Recorder wired ⇒ Query still succeeds (telemetry is optional).
	r := routing.NewRouter(routing.Deps{
		Lister:   threeProviderLister(),
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
}

func TestStatusReportsProviderHealthAndModels(t *testing.T) {
	// swarm-manager is unreachable; cli-health + ui-health resolve.
	resolver := staticResolver{
		urls: map[string]string{"cli-health": "http://cli-health.test", "ui-health": "http://ui-health.test"},
		errs: map[string]error{"swarm-manager": errors.New("not running")},
	}
	rrk := &fakeReranker{err: errors.New("reranker model down")} // Available ⇒ false
	r := routing.NewRouter(routing.Deps{
		Lister:   threeProviderLister(),
		Resolver: resolver,
		Doer:     threeProviderDoer(),
		Reranker: rrk,
	})

	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, st.GetProviders(), 3)

	byID := map[string]*routingv1.ProviderHealth{}
	for _, h := range st.GetProviders() {
		byID[h.GetProviderId()] = h
	}
	require.True(t, byID["cli-health.commands"].GetReachable())
	require.False(t, byID["cli-health.commands"].GetDegraded())
	require.False(t, byID["swarm-manager.records"].GetReachable())
	require.True(t, byID["swarm-manager.records"].GetDegraded())
	require.Contains(t, byID["swarm-manager.records"].GetReachability(), "unreachable")

	require.False(t, st.GetClassifierAvailable(), "the interactive LLM classifier is retired")
	require.False(t, st.GetRerankerAvailable())
}

func TestStatusProbesProvidersConcurrently(t *testing.T) {
	p1 := providerWithStatus("")
	p1.ProviderId = "cli-health.commands"
	p2 := providerWithStatus("")
	p2.ProviderId = "ui-health.surfaces"
	doer := &parallelStatusDoer{delay: 120 * time.Millisecond}
	r := routing.NewRouter(routing.Deps{
		Lister: &fakeLister{providers: []*registryv1.ProviderDescriptor{p1, p2}},
		Resolver: staticResolver{urls: map[string]string{
			"cli-health": "http://cli-health.test",
			"ui-health":  "http://ui-health.test",
		}},
		Doer: doer,
	})
	start := time.Now()
	_, err := r.Status(context.Background())
	require.NoError(t, err)
	require.Less(t, time.Since(start), 210*time.Millisecond, "status must be bounded by the slowest provider probe")
	doer.mu.Lock()
	maxActive := doer.maxActive
	doer.mu.Unlock()
	require.Equal(t, 2, maxActive, "both provider probes should overlap")
}

func TestStatusReportsProbedIndexAgeAndPointCount(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	doer := &statusDoer{body: `{"last_indexed_at":"2026-08-12T07:30:00Z","point_count":42}`}
	p := providerWithStatus(doer.body)
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{p}},
		Resolver: staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}},
		Doer:     doer,
		Now:      func() time.Time { return now },
	})

	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, st.GetProviders(), 1)
	h := st.GetProviders()[0]
	require.Equal(t, "endpoint resolved", h.GetReachability())
	require.Equal(t, "30m0s", h.GetIndexAge())
	require.Equal(t, int64(42), h.GetPointCount())
	require.Equal(t, "2026-08-12T07:30:00Z", h.GetLastIndexedAt().AsTime().Format(time.RFC3339))
	require.Equal(t, 1, doer.statusCalls)
}

func TestStatusReportsExplicitIndexAgeAbsence(t *testing.T) {
	tests := []struct {
		name string
		body string
		err  error
		want string
	}{
		{name: "no timestamp is unreported", body: `{"point_count":9}`, want: "unreported: status response has no usable declared index timestamp"},
		{name: "probe timeout is unreported", err: context.DeadlineExceeded, want: "unreported: status probe failed: context deadline exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := providerWithStatus(tt.body)
			r := routing.NewRouter(routing.Deps{
				Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{p}},
				Resolver: staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}},
				Doer:     &statusDoer{body: tt.body, err: tt.err},
			})
			st, err := r.Status(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, st.GetProviders()[0].GetIndexAge())
		})
	}
}

func TestStatusReportsMissingStatusEndpointAsNotApplicable(t *testing.T) {
	p := providerWithStatus("")
	p.StatusEndpoint = nil
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{p}},
		Resolver: staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}},
		Doer:     &statusDoer{},
	})

	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, "not_applicable: provider has no status_endpoint", st.GetProviders()[0].GetIndexAge())
}

func TestStatusProbeDoesNotRunOnQueryPath(t *testing.T) {
	doer := &statusDoer{}
	p := providerWithStatus("")
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{p}},
		Resolver: staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}},
		Doer:     doer,
	})

	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "probe", All: true})
	require.NoError(t, err)
	require.Zero(t, doer.statusCalls)
	require.Equal(t, 1, doer.searchCalls)
}

func TestStatusNoModelsWiredReportsUnavailable(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.False(t, st.GetClassifierAvailable(), "no classifier wired ⇒ unavailable")
	require.False(t, st.GetRerankerAvailable(), "no reranker wired ⇒ unavailable")
	require.Len(t, st.GetProviders(), 1)
	require.True(t, st.GetProviders()[0].GetReachable())
}

func TestStatusListerErrorPropagates(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{err: errors.New("db down")},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	_, err := r.Status(context.Background())
	require.Error(t, err, "a registry read failure is a real error (unlike per-provider failures)")
}
