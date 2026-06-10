package routing_test

import (
	"context"
	"strings"
	"testing"

	"search-hub/internal/routing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"github.com/stretchr/testify/require"
)

// scopeMixLister/Resolver/Doer back the OT-P2-002 tests: one project provider
// (cli-health.commands) + one SCOPE_EXTERNAL provider (web-search.live).
func scopeMixSetup(projectHits, externalHits string) (*fakeLister, staticResolver, routeDoer) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health": "http://cli-health.test",
		"web-search": "http://web-search.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search":         {status: 200, body: projectHits},
		"http://web-search.test/vrooli.web_search.v1.livesearch.LiveSearchService/Search": {status: 200, body: externalHits},
	}}
	return lister, resolver, doer
}

const (
	oneCommandHit = `{"results":[{"name":"scenario restart","description":"Restart","score":0.9}]}`
	noHits        = `{"results":[]}`
	oneWebHit     = `{"results":[{"url":"https://x","title":"X","snippet":"s","score":0.5}]}`
)

// TestAutoRouteExternalFlagOffPreservesWithhold is the regression guard: with
// the flag OFF (default), a confidently web-shaped query STILL withholds the
// external provider — current behavior is unchanged.
func TestAutoRouteExternalFlagOffPreservesWithhold(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: true}}
	lister, resolver, doer := scopeMixSetup(oneCommandHit, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf,
		// AutoRouteExternal defaults to false.
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "latest news on X", Explain: true})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live", "flag OFF must never auto-hit external, even web-shaped")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "withheld 1 external")
}

// TestAutoRouteExternalFoldsInWebShaped: flag ON + web-shaped + above threshold
// folds the external provider into the fan-out.
func TestAutoRouteExternalFoldsInWebShaped(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: true}}
	lister, resolver, doer := scopeMixSetup(oneCommandHit, oneWebHit)
	rec := &recordingRecorder{}
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf,
		Recorder: rec, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "latest news on X", Explain: true})
	require.NoError(t, err)
	require.Contains(t, resp.GetCorporaSearched(), "web-search.live", "web-shaped + opt-in folds external in")
	require.Contains(t, resp.GetCorporaSearched(), "cli-health.commands")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "auto-routed 1 external")
	require.Len(t, rec.samples, 1)
	require.True(t, rec.samples[0].AutoRoutedExternal, "telemetry records the auto-route")
	require.False(t, rec.samples[0].Escalated)
}

// TestAutoRouteExternalBelowThresholdWithholds: flag ON but the web-shaped
// judgment is not confident enough ⇒ still withheld (no auto-route).
func TestAutoRouteExternalBelowThresholdWithholds(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.40, WebShaped: true}}
	lister, resolver, doer := scopeMixSetup(oneCommandHit, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "vague-ish", Explain: true})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live", "low web-shaped confidence stays withheld")
}

// TestAutoRouteExternalNotWebShapedWithholds: flag ON but the classifier did not
// judge the query web-shaped ⇒ withheld (the project corpus answered).
func TestAutoRouteExternalNotWebShapedWithholds(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: false}}
	lister, resolver, doer := scopeMixSetup(oneCommandHit, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", Explain: true})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live")
	require.Contains(t, resp.GetCorporaSearched(), "cli-health.commands")
}

// TestFallbackEscalationOnEmptyProject: flag ON, not web-shaped, but the project
// corpus returns NO hits ⇒ escalate to the external provider.
func TestFallbackEscalationOnEmptyProject(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: false}}
	lister, resolver, doer := scopeMixSetup(noHits, oneWebHit)
	rec := &recordingRecorder{}
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf,
		Recorder: rec, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "obscure thing", Explain: true})
	require.NoError(t, err)
	require.Contains(t, resp.GetCorporaSearched(), "web-search.live", "empty project corpus escalates to external")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "escalated to 1 external")
	require.Len(t, rec.samples, 1)
	require.True(t, rec.samples[0].Escalated, "telemetry records the escalation")
	require.False(t, rec.samples[0].AutoRoutedExternal)
}

// TestFallbackEscalationFlagOffStaysEmpty: flag OFF, empty project corpus ⇒ NO
// escalation (the result is honestly empty).
func TestFallbackEscalationFlagOffStaysEmpty(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: false}}
	lister, resolver, doer := scopeMixSetup(noHits, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf,
		// flag off
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "obscure thing"})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live", "no escalation when the flag is off")
}

// TestNoEscalationWhenProjectHasHits: flag ON, project corpus answered ⇒ no
// escalation (escalation only fires on an empty/weak project result).
func TestNoEscalationWhenProjectHasHits(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: false}}
	lister, resolver, doer := scopeMixSetup(oneCommandHit, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario"})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live", "a non-empty project corpus does not escalate")
}

// TestEscalationDegradesGracefullyOnGovernorReject: flag ON, empty project, and
// the external provider returns a rate-limited (non-2xx) response ⇒ escalation
// still happens, the group is degraded, and the query never fails.
func TestEscalationDegradesGracefullyOnGovernorReject(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.95, WebShaped: false}}
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health": "http://cli-health.test",
		"web-search": "http://web-search.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search":         {status: 200, body: noHits},
		"http://web-search.test/vrooli.web_search.v1.livesearch.LiveSearchService/Search": {status: 429, body: `rate-limited, try later`},
	}}
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf, AutoRouteExternal: true,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "obscure thing"})
	require.NoError(t, err, "a rate-limited external provider never fails the query")
	require.Contains(t, resp.GetCorporaSearched(), "web-search.live", "escalation still reached the external provider")
	require.True(t, resp.GetDegraded(), "the rate-limited external leaf degrades the response")
}

// TestExplicitPathUnaffectedByFlag: an explicit --all reaches external whether
// or not the flag is set (the flag only governs the automatic path).
func TestExplicitPathUnaffectedByFlag(t *testing.T) {
	lister, resolver, doer := scopeMixSetup(noHits, oneWebHit)
	r := routing.NewRouter(routing.Deps{
		Lister: lister, Resolver: resolver, Doer: doer,
		// no classifier needed for explicit; flag off
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.Contains(t, resp.GetCorporaSearched(), "web-search.live")
}
