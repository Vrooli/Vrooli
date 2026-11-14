package routing_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

// Feature: --group accepts a scenario name.
//
//	Scenario: a group that is not a provider group scopes results to that scenario's leaves
//	  Given a command corpus holding leaves of two scenarios
//	  When the caller queries with --group naming one scenario
//	  Then only that scenario's leaves are returned and the scope is explained
func TestGroupNamingScenarioKeepsOnlyThatScenariosLeaves(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[
			{"name":"portal surfaces apps","score":0.9},
			{"name":"device-control device capture","score":0.8},
			{"name":"device-control device screen","score":0.7}
		]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "screenshot", Group: "device-control", Types: []string{"command"}, Limit: 5})
	require.NoError(t, err)
	require.Len(t, resp.GetGroups(), 1)
	require.Equal(t, int32(2), resp.GetGroups()[0].GetCount())
	for _, hit := range resp.GetGroups()[0].GetHits() {
		require.True(t, strings.HasPrefix(hit.GetTitle(), "device-control "), hit.GetTitle())
	}
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), `read as scenario scope`)
}

// Scenario: a scenario scope that matches nothing is never a silent empty result.
func TestGroupNamingScenarioWithNoLeavesExplainsTheEmptyResult(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"portal surfaces apps","score":0.9}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "screenshot", Group: "no-such-scenario"})
	require.NoError(t, err)
	joined := strings.Join(resp.GetRoutingExplanation(), "\n")
	require.Contains(t, joined, `no leaf of scenario "no-such-scenario" matched`)
	require.Contains(t, joined, "cli-health", "known provider groups are offered as the alternative reading")
}

// Scenario: without --type, a scenario scope searches the leaf-bearing buckets only.
func TestScenarioScopeSkipsKnowledgeCorpora(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), swarmRecords()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test", "swarm-manager": "http://swarm-manager.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"device-control device capture","score":0.8}]}`},
		"http://swarm-manager.test/api/v1/search/ai":                              {status: 200, body: `{"results":[{"id":"p","score":0.6,"payload":{"record_id":"device-control notes","scenario":"s"}}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", Group: "device-control"})
	require.NoError(t, err)
	require.Equal(t, []string{"cli-health.commands"}, resp.GetCorporaSearched())
}
