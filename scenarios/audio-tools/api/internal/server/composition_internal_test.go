package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"audio-tools/internal/duplex"
	"audio-tools/internal/httpc"
	measuretypes "audio-tools/internal/measures"
	modelpromotion "audio-tools/internal/modelpromotion"
	"audio-tools/internal/modules"
	trustfloor "audio-tools/internal/qualification"
	"audio-tools/internal/testutil/mocks"
	apiTestDB "github.com/vrooli/api-core/databasetest"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestCompose_MountsEveryDeclaredEndpoint is the composition drift gate.
// modules.AllEndpoints() is the static endpoint inventory used by docs and
// gateway registration; Compose is the only runtime mount point. A module that
// stops mounting a procedure, or an endpoint descriptor that is added without
// a module, fails here instead of surfacing as a 404 in production.
//
// The composition intentionally tolerates absent optional services (nil DB,
// nil chains, nil stores): handler constructors validate only their required
// logger/clock and defer dependency use to request time, so the route table can
// be built without a live environment.
func TestCompose_MountsEveryDeclaredEndpoint(t *testing.T) {
	srv := ComposeForBuild(CompositionDeps{
		Logger: &mocks.FakeLogger{},
		Env:    Env{TestIsolationActive: func() bool { return false }},
	})
	require.NotNil(t, srv)
	require.NotNil(t, srv.router)

	mounted := make([]string, 0, 32)
	err := srv.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil {
			return nil // regex-only routes have no template
		}
		mounted = append(mounted, template)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, mounted, "compose registered no routes")

	for _, endpoint := range modules.AllEndpoints() {
		require.Truef(t, routeServes(mounted, endpoint.Path),
			"endpoint %s %s is declared in modules.AllEndpoints but no module mounts it (mounted: %v)",
			endpoint.Method, endpoint.Path, mounted)
	}
}

func TestCompose_ProductEvidenceAndMeasuresUseReadContractsWithoutLabService(t *testing.T) {
	evidence := &productEvidenceReader{}
	counter := &productExperimentCounter{}
	srv := Compose(CompositionDeps{
		Logger:                &mocks.FakeLogger{},
		Env:                   Env{TestIsolationActive: func() bool { return false }},
		QualificationEvidence: evidence,
		ExperimentCounter:     counter,
	})

	modelResp := serveConnect(t, srv, "/vrooli.audio_tools.v1.models.ModelRegistryService/ResolveModel", `{"capability":"tts"}`)
	require.Equal(t, http.StatusOK, modelResp.Code, modelResp.Body.String())
	require.Contains(t, modelResp.Body.String(), "kokoro-82m-v1.0")

	measureResp := serveConnect(t, srv, "/vrooli.audio_tools.v1.measures.MeasuresService/CountExperiments", `{}`)
	require.Equal(t, http.StatusOK, measureResp.Code, measureResp.Body.String())
	require.Contains(t, measureResp.Body.String(), `"count":"7"`)
	require.True(t, counter.called, "product measure should resolve through its aggregate read contract")
}

func serveConnect(t *testing.T, srv *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	resp := httptest.NewRecorder()
	srv.router.ServeHTTP(resp, req)
	return resp
}

type productEvidenceReader struct{}

func (*productEvidenceReader) ListQualificationEvidence(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
	return nil, nil
}

var _ modelpromotion.QualificationEvidenceStore = (*productEvidenceReader)(nil)

type productExperimentCounter struct{ called bool }

func (c *productExperimentCounter) CountCreatedInRange(context.Context, time.Time, time.Time) (measuretypes.ExperimentCreatedCounts, error) {
	c.called = true
	return measuretypes.ExperimentCreatedCounts{Experiments: 7, Runs: 11}, nil
}

var _ measuretypes.ExperimentCounter = (*productExperimentCounter)(nil)

func TestCompose_ProductExcludesLabEndpoints(t *testing.T) {
	srv := Compose(CompositionDeps{
		Logger: &mocks.FakeLogger{},
		Env:    Env{TestIsolationActive: func() bool { return false }},
	})
	mounted := routeTemplates(t, srv)
	for _, endpoint := range modules.LabEndpoints() {
		require.Falsef(t, routeServes(mounted, endpoint.Path),
			"product composition mounted lab endpoint %s %s", endpoint.Method, endpoint.Path)
	}
	for _, endpoint := range modules.ProductEndpoints() {
		require.Truef(t, routeServes(mounted, endpoint.Path),
			"product composition omitted endpoint %s %s", endpoint.Method, endpoint.Path)
	}
}

func TestComposePreparesCanonicalGPTLiveProviderWithoutMountingRoute(t *testing.T) {
	srv := Compose(CompositionDeps{
		Logger: &mocks.FakeLogger{},
		Env:    Env{TestIsolationActive: func() bool { return false }},
	})
	provider, ok := srv.duplexProvider.(*duplex.OpenAILiveProvider)
	require.True(t, ok, "production composition retains the provider dependency for future admitted route wiring")
	_, ok = provider.Keys.(*duplex.LazyAuthorityKeyResolver)
	require.True(t, ok, "provider must resolve the canonical credential authority lazily")
	for _, path := range routeTemplates(t, srv) {
		require.NotContains(t, path, "duplex", "GPT-Live API remains unmounted until eligibility and billing gates pass")
	}
}

type compositionCreditToken struct{}

func (compositionCreditToken) Token(context.Context) (string, error) { return "", nil }

type compositionConsentVerifier struct{}

func (compositionConsentVerifier) Verify(context.Context, string, string, string) error { return nil }

func TestComposeDuplexRouteRequiresExplicitAdmissionDependenciesAndCatalogEligibility(t *testing.T) {
	base := CompositionDeps{Logger: &mocks.FakeLogger{}, Env: Env{TestIsolationActive: func() bool { return false }}}
	assertDuplexRoute := func(t *testing.T, deps CompositionDeps, want bool) {
		t.Helper()
		for _, path := range routeTemplates(t, Compose(deps)) {
			if strings.Contains(path, "/duplex/") {
				require.True(t, want, "unexpected duplex route %s", path)
				return
			}
		}
		require.False(t, want, "duplex route was not composed")
	}
	assertDuplexRoute(t, base, false)

	journal, err := duplex.NewSQLJournal(apiTestDB.NewSQLite(t))
	require.NoError(t, err)
	credits, err := duplex.NewAccountCreditClient("http://lpbs.test", compositionCreditToken{}, httpc.DefaultDoer())
	require.NoError(t, err)
	fullyConfigured := base
	fullyConfigured.Stores.DuplexJournal = journal
	fullyConfigured.DuplexAccountCredits = credits
	fullyConfigured.DuplexConsentVerifier = compositionConsentVerifier{}
	fullyConfigured.DuplexMaxSessionBudgetMicros = 100_000
	// Even an injected predicate that says "eligible" cannot bypass the
	// canonical catalog. GPT-Live is still planned, so the production route
	// stays absent until the catalog declaration itself is promoted.
	fullyConfigured.DuplexProviderEligibility = func(provider, model string) bool {
		return provider == "openai-gpt-live" && model == "gpt-live-1.1"
	}
	assertDuplexRoute(t, fullyConfigured, false)

	fullyConfigured.DuplexProviderEligibility = func(string, string) bool { return false }
	assertDuplexRoute(t, fullyConfigured, false)
}

func TestPlannedGPTLiveCannotBeComposedDuringConcurrentEligibilityChanges(t *testing.T) {
	var eligible atomic.Bool
	d := CompositionDeps{
		Stores:                       CompositionStores{DuplexJournal: &duplex.SQLJournal{}},
		DuplexAccountCredits:         &duplex.AccountCreditClient{},
		DuplexConsentVerifier:        compositionConsentVerifier{},
		DuplexProviderEligibility:    func(string, string) bool { return eligible.Load() },
		DuplexMaxSessionBudgetMicros: 100_000,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10000; i++ {
			eligible.Store(i%2 == 0)
		}
	}()
	for i := 0; i < 10000; i++ {
		if canComposeGPTLive(d) {
			t.Fatal("injected eligibility bypassed the planned GPT-Live catalog status")
		}
	}
	<-done
}

func routeTemplates(t *testing.T, srv *Server) []string {
	t.Helper()
	mounted := make([]string, 0, 32)
	err := srv.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err == nil {
			mounted = append(mounted, template)
		}
		return nil
	})
	require.NoError(t, err)
	return mounted
}

// routeServes reports whether a mounted route template serves path. Connect
// mounts use PathPrefix (one procedure per method under the service prefix),
// while simple probes are exact routes.
func routeServes(templates []string, path string) bool {
	for _, template := range templates {
		if template == path {
			return true
		}
		if strings.HasSuffix(template, "/") && strings.HasPrefix(path, template) {
			return true
		}
	}
	return false
}
