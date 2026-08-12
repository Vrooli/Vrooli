package routing_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

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

	// Build the lister from the provider corpus fixtures (their NL descriptions
	// are exactly what the classifier routes on) and a type→provider_id index.
	// The corpus is test data, not a registration source — providers self-register
	// from their own search.json (Phase 2); see testdata/provider_corpus/README.md.
	seeds := loadProviderCorpus(t)
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
	// Stable descriptor order ⇒ byte-identical classifier prompt across runs.
	// The corpus loader returns a map; at temperature 0 the model is only
	// deterministic for a fixed prompt, and a gate this close to its 0.85
	// threshold must not coin-flip on Go map iteration order (production gets
	// a stable order from the registry).
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].GetProviderId() < descriptors[j].GetProviderId() })

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

// TestLearningsRoutingRecall pins the learnings-routing fix (web-search
// hardening Phase 4, 2026-06-10): external-world factual queries — software
// releases/versions/features, "what is the latest X" — must route to the
// `learning` type (web-search.learnings holds the already-verified answers),
// while pure project-code/config queries must NOT drag the learnings store
// into the fan-out. Routing is description-driven, so this gate measures
// whether the enriched web-search.learnings description (mirrored in
// testdata/provider_corpus/web-search.learnings.json from the production
// .vrooli/search.json) is concrete enough for the small classifier model.
// Live-Ollama-gated like TestClassifierRoutingRecall.
func TestLearningsRoutingRecall(t *testing.T) {
	if os.Getenv("SEARCH_HUB_SKIP_OLLAMA") != "" {
		t.Skip("SEARCH_HUB_SKIP_OLLAMA set")
	}
	clf := routing.NewOllamaClassifier()
	availCtx, availCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer availCancel()
	if !clf.Available(availCtx) {
		t.Skip("resource-ollama unavailable — learnings routing gate requires the local Ollama daemon + classifier model")
	}

	// Assert on the CLASSIFIER's own leaf choice, not the routed fan-out: the
	// router deliberately widens within sibling leaves on low confidence (recall over
	// precision), which would both mask a missed inclusion and trip the
	// exclusion half for reasons unrelated to the learnings description.
	// Profiles are sorted by provider id so the rendered prompt is byte-identical
	// across runs — the corpus loader returns a map, and at temperature 0 the
	// model's answer is deterministic only for a fixed prompt (production gets
	// a stable order from the registry's provider_id ordering).
	seeds := loadProviderCorpus(t)
	profiles := make([]routing.ProviderProfile, 0, len(seeds))
	hasLearnings := false
	idToType := map[string]string{}
	for _, d := range seeds {
		profiles = append(profiles, routing.ProviderProfile{
			ProviderID:  d.GetProviderId(),
			Type:        d.GetType(),
			Group:       d.GetProviderGroup(),
			Description: d.GetDescription(),
		})
		idToType[d.GetProviderId()] = d.GetType()
		if d.GetProviderId() == "web-search.learnings" {
			hasLearnings = true
		}
	}
	require.True(t, hasLearnings, "the corpus must carry the learnings provider fixture")
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ProviderID < profiles[j].ProviderID })

	classify := func(q string) map[string]struct{} {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		res, err := clf.Classify(ctx, q, profiles)
		require.NoErrorf(t, err, "classify %q", q)
		out := map[string]struct{}{}
		for _, providerID := range res.ProviderIDs {
			if typ := idToType[providerID]; typ != "" {
				out[typ] = struct{}{}
			}
		}
		return out
	}

	externalFactQueries := []string{
		"key features of Go 1.26",
		"what is the latest stable Rust version",
		"what changed in the newest Node.js LTS release",
	}
	for _, q := range externalFactQueries {
		types := classify(q)
		_, ok := types["learning"]
		require.Truef(t, ok, "external-fact query %q must include the learning type, classified %v", q, keys(types))
	}

	projectCodeQueries := []string{
		"where is the retry logic in the api-core http client",
		"which CLI command restarts a scenario",
	}
	for _, q := range projectCodeQueries {
		types := classify(q)
		_, ok := types["learning"]
		require.Falsef(t, ok, "project-code query %q must NOT classify to the learnings store, classified %v", q, keys(types))
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
