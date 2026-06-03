package routing_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/providers"
	"search-hub/internal/routing"
)

// routingFixture mirrors testdata/routing_queries.json.
type routingFixture struct {
	Cases []struct {
		Query    string   `json:"query"`
		Expected []string `json:"expected"`
	} `json:"cases"`
	Uncertain []string `json:"uncertain"`
}

// emptyResultsDoer answers every fan-out call with an empty result set, so the
// recall gate measures *routing* (which providers were chosen) without needing
// the live federation up. The routed provider set is read back from the
// response groups.
type emptyResultsDoer struct{}

func (emptyResultsDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"results":[]}`)),
		Header:     make(http.Header),
	}, nil
}

// TestClassifierRoutingRecall is the make-or-break gate from plan §6 #1: it runs
// the REAL local-Ollama classifier over the REAL seed provider descriptions and
// asserts routing recall ≥ 0.85 against a labeled query→types fixture (recall
// over precision — uncertain queries widen). It is skipped when the Ollama
// daemon is unavailable (e.g. CI without the resource), so the deterministic
// widen-policy unit tests remain the always-on guarantee.
func TestClassifierRoutingRecall(t *testing.T) {
	if os.Getenv("SEARCH_HUB_SKIP_OLLAMA") != "" {
		t.Skip("SEARCH_HUB_SKIP_OLLAMA set")
	}
	clf := routing.NewOllamaClassifier()
	availCtx, availCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer availCancel()
	if !clf.Available(availCtx) {
		t.Skip("resource-ollama unavailable — routing recall gate requires the local Ollama daemon + classifier model")
	}

	// Build the lister from the real shipped seeds (their NL descriptions are
	// exactly what the classifier routes on) and a type→provider_id index.
	seeds := providers.Seeds()
	descriptors := make([]*registryv1.ProviderDescriptor, 0, len(seeds))
	idToType := map[string]string{}
	resolverURLs := map[string]string{}
	for id, d := range seeds {
		descriptors = append(descriptors, d)
		idToType[id] = d.GetType()
		if hj := d.GetEndpoint().GetHttpJson(); hj != nil {
			resolverURLs[hj.GetScenarioId()] = "http://" + hj.GetScenarioId() + ".test"
		}
	}

	r := routing.NewRouter(routing.Deps{
		Lister:     &fakeLister{providers: descriptors},
		Resolver:   staticResolver{urls: resolverURLs},
		Doer:       emptyResultsDoer{},
		Classifier: clf,
	})

	fx := loadRoutingFixture(t)

	var totalExpected, recalled, totalRouted, correctRouted int
	for _, c := range fx.Cases {
		routed := routedTypes(t, r, c.Query, idToType)
		for _, want := range c.Expected {
			totalExpected++
			if _, ok := routed[want]; ok {
				recalled++
			} else {
				t.Logf("MISS  %-60q expected %v, routed %v", c.Query, c.Expected, keys(routed))
			}
		}
		for rt := range routed {
			totalRouted++
			if contains(c.Expected, rt) {
				correctRouted++
			}
		}
	}

	require.Positive(t, totalExpected)
	recall := float64(recalled) / float64(totalExpected)
	precision := 0.0
	if totalRouted > 0 {
		precision = float64(correctRouted) / float64(totalRouted)
	}
	t.Logf("routing recall=%.3f (%d/%d)  precision=%.3f (%d/%d)  [precision is reported, not gated]",
		recall, recalled, totalExpected, precision, correctRouted, totalRouted)
	require.GreaterOrEqualf(t, recall, 0.85, "routing recall %.3f below the 0.85 gate", recall)

	// Uncertain queries must widen (route to something) rather than drop to zero.
	for _, q := range fx.Uncertain {
		routed := routedTypes(t, r, q, idToType)
		require.NotEmptyf(t, routed, "uncertain query %q dropped to no providers (should widen)", q)
	}
}

func routedTypes(t *testing.T, r *routing.Router, query string, idToType map[string]string) map[string]struct{} {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, err := r.Query(ctx, &routingv1.QueryRequest{Query: query})
	require.NoErrorf(t, err, "query %q", query)
	out := map[string]struct{}{}
	for _, g := range resp.GetGroups() {
		if typ, ok := idToType[g.GetProviderId()]; ok {
			out[typ] = struct{}{}
		}
	}
	return out
}

func loadRoutingFixture(t *testing.T) routingFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/routing_queries.json")
	require.NoError(t, err)
	var fx routingFixture
	require.NoError(t, json.Unmarshal(raw, &fx))
	require.NotEmpty(t, fx.Cases)
	return fx
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
