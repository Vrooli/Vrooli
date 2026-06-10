package main

// Requirements coverage for MOD-P0-003 (dual search-hub provider registration).
//
// The scenario's half of the registration contract is: (a) the committed
// .vrooli/search.json SSOT declares both providers with the correct scopes,
// (b) the descriptors it yields conform to search-hub's registry proto
// contract, and (c) main() initiates self-registration before the HTTP
// listener opens. The hub's half (persisting the descriptors and routing by
// scope) is covered by search-hub's own registry/store and routing tests.

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	aisearch "github.com/vrooli/aisearch-go"
	"github.com/vrooli/api-core/retry"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// committedSearchJSON is the scenario's SSOT, relative to the api/ test
// working directory. The tests below load the REAL committed file so a drift
// in scope/routing metadata fails here, not at the first live registration.
const committedSearchJSON = "../.vrooli/search.json"

// loadCommittedDescriptors maps the committed search.json through the SAME
// bridge production registration uses (searchregister.Descriptors), returning
// the descriptors keyed by provider id.
func loadCommittedDescriptors(t *testing.T) map[string]*registryv1.ProviderDescriptor {
	t.Helper()
	file, err := aisearch.LoadSearchFile(committedSearchJSON)
	require.NoError(t, err, "committed .vrooli/search.json must parse")
	descriptors, err := searchregister.Descriptors(file)
	require.NoError(t, err, "every provider block must map onto the registry descriptor contract")
	out := make(map[string]*registryv1.ProviderDescriptor, len(descriptors))
	for _, d := range descriptors {
		out[d.GetProviderId()] = d
	}
	return out
}

// TestSearchJSONLiveDescriptorIsExternalScope: web-search.live must declare
// SCOPE_EXTERNAL. That scope IS the explicit-only routing constraint: the
// search-hub router (partitionByScope) withholds SCOPE_EXTERNAL providers from
// every automatic/default route, so they are reachable only via an explicit
// --type web / --all selector or operator-enabled fallback escalation. A
// regression to SCOPE_PROJECT (or an unset scope, which defaults to
// project-routable) would silently put the rate-limited live-web corpus on the
// default federated path.
func TestSearchJSONLiveDescriptorIsExternalScope(t *testing.T) {
	d, ok := loadCommittedDescriptors(t)["web-search.live"]
	require.True(t, ok, "search.json must declare web-search.live")
	require.Equal(t, registryv1.Scope_SCOPE_EXTERNAL, d.GetScope(),
		"web-search.live must be SCOPE_EXTERNAL so the router keeps it explicit-only")
	require.Equal(t, "web", d.GetType(),
		"the explicit selector for the live provider is --type web; the descriptor type must match")
}

// TestSearchJSONLearningsDescriptorIsProjectScope: web-search.learnings must
// declare SCOPE_PROJECT, which is precisely what makes it part of the default
// routing set — the router's partitionByScope keeps every non-external
// provider in the automatic candidate set.
func TestSearchJSONLearningsDescriptorIsProjectScope(t *testing.T) {
	d, ok := loadCommittedDescriptors(t)["web-search.learnings"]
	require.True(t, ok, "search.json must declare web-search.learnings")
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, d.GetScope(),
		"web-search.learnings must be SCOPE_PROJECT so default federated routing includes it")
}

// TestSearchJSONDescriptorsConformToRegistryContract: both committed provider
// blocks must map losslessly onto search-hub's registry ProviderDescriptor
// proto. The mapping is strict protojson — an unknown field, a misspelled enum
// value, or a malformed endpoint/result_mapping sub-object fails the load —
// so this is the operative schema-conformance gate for the descriptor halves
// of .vrooli/search.json. It then asserts the contract fields registration
// and fan-out depend on are populated for both providers.
func TestSearchJSONDescriptorsConformToRegistryContract(t *testing.T) {
	descriptors := loadCommittedDescriptors(t)
	require.Len(t, descriptors, 2, "exactly the two web-search providers are declared")

	for _, id := range []string{"web-search.learnings", "web-search.live"} {
		d, ok := descriptors[id]
		require.True(t, ok, "missing provider %s", id)

		require.Equal(t, "web-search", d.GetProviderGroup(), "%s: provider_group", id)
		require.NotEqual(t, registryv1.Bucket_BUCKET_UNSPECIFIED, d.GetBucket(), "%s: bucket required", id)
		require.NotEmpty(t, d.GetDescription(), "%s: description drives routing/classification", id)

		ep := d.GetEndpoint().GetHttpJson()
		require.NotNil(t, ep, "%s: http_json endpoint required for fan-out", id)
		require.Equal(t, "web-search", ep.GetScenarioId(), "%s: endpoint must resolve back to this scenario", id)
		require.Equal(t, registryv1.HttpMethod_HTTP_METHOD_POST, ep.GetMethod(), "%s: method", id)
		require.Contains(t, ep.GetBodyTemplate(), "{{query}}", "%s: body template must carry the query", id)

		rm := d.GetResultMapping()
		require.NotNil(t, rm, "%s: result_mapping required by the ResultMapping contract", id)
		require.NotEmpty(t, rm.GetResultsPath(), "%s: results_path", id)
		require.NotEmpty(t, rm.GetIdField(), "%s: id_field", id)
		require.NotEmpty(t, rm.GetTitleField(), "%s: title_field", id)
		require.NotEmpty(t, rm.GetScoreField(), "%s: score_field", id)
	}
}

// TestMainWiresRegistrationBeforeServing is the startup lifecycle-order guard:
// main() must initiate search-hub self-registration (searchregister.Register,
// launched on its background goroutine because search-hub is an OPTIONAL
// dependency that must never delay boot) BEFORE handing control to
// apiserver.Run, which opens the listener. main() is not callable from a test
// (it blocks serving), so the order is asserted on the source: a refactor that
// moves registration after the serve call — or drops it — fails here.
func TestMainWiresRegistrationBeforeServing(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)
	text := string(src)

	registerAt := strings.Index(text, "searchregister.Register(")
	serveAt := strings.Index(text, "apiserver.Run(")
	require.GreaterOrEqual(t, registerAt, 0, "main.go must call searchregister.Register")
	require.GreaterOrEqual(t, serveAt, 0, "main.go must serve via apiserver.Run")
	require.Less(t, registerAt, serveAt,
		"self-registration must be initiated before the HTTP server starts accepting connections")
}

// --- registration wiring against a fake hub ---------------------------------

// fakeRegistryClient implements searchregister.RegistryClient, standing in for
// search-hub's RegistryService: it records every upserted descriptor and the
// control token presented with it, and mints a per-provider token in response.
type fakeRegistryClient struct {
	mu        sync.Mutex
	calls     []*registryv1.ProviderDescriptor
	presented []string
}

func (f *fakeRegistryClient) RegisterProvider(
	_ context.Context,
	req *connect.Request[registryv1.RegisterProviderRequest],
) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := req.Msg.GetDescriptor_()
	f.calls = append(f.calls, d)
	f.presented = append(f.presented, req.Msg.GetControlToken())
	return connect.NewResponse(&registryv1.RegisterProviderResponse{
		Descriptor_:  d,
		Created:      true,
		ControlToken: "tok-" + d.GetProviderId(),
	}), nil
}

// registerConfig wires Register exactly as main.go does (same SSOT file, same
// token-holder callbacks), but with the hub seams faked so the test is hermetic.
func registerConfig(client *fakeRegistryClient, tokens *tokenHolder) searchregister.Config {
	return searchregister.Config{
		ScenarioID:     "web-search",
		SearchFilePath: committedSearchJSON,
		Logger:         log.New(io.Discard, "", 0),
		ResolveBaseURL: func(context.Context) (string, error) { return "http://search-hub.test", nil },
		NewClient:      func(string) searchregister.RegistryClient { return client },
		OnControlToken: func(providerID, token string) { tokens.set(providerID, token) },
		ControlToken:   func(providerID string) string { return tokens.get(providerID) },
		Retry: retry.Config{
			MaxAttempts:    1,
			Sleeper:        func(time.Duration) {},
			Rand:           func() float64 { return 0 },
			JitterFraction: 0,
		},
	}
}

// TestRegisterUpsertsBothProvidersWithFakeHub drives the production
// registration path (committed search.json → searchregister.Register) against
// a fake hub and asserts BOTH providers are upserted with correct scope
// metadata — the request payloads are exactly what search-hub's registry
// persists and later returns to registry queries (hub-side persistence has its
// own round-trip coverage in search-hub's registry store tests).
func TestRegisterUpsertsBothProvidersWithFakeHub(t *testing.T) {
	client := &fakeRegistryClient{}
	results := searchregister.Register(context.Background(), registerConfig(client, newTokenHolder()))

	require.Len(t, results, 2)
	scopes := map[string]registryv1.Scope{}
	for i, res := range results {
		require.NoError(t, res.Err, "registration of %s must succeed", res.ProviderID)
		scopes[client.calls[i].GetProviderId()] = client.calls[i].GetScope()
	}
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, scopes["web-search.learnings"],
		"learnings registers as project scope (default-routable)")
	require.Equal(t, registryv1.Scope_SCOPE_EXTERNAL, scopes["web-search.live"],
		"live registers as external scope (explicit-only)")
}

// TestControlTokenEchoedOnReRegistration covers the tokenHolder wiring in
// searchreg.go: the token minted by the hub on first registration is cached
// per provider and presented again on re-registration as the ownership proof.
func TestControlTokenEchoedOnReRegistration(t *testing.T) {
	client := &fakeRegistryClient{}
	tokens := newTokenHolder()
	cfg := registerConfig(client, tokens)

	first := searchregister.Register(context.Background(), cfg)
	require.Len(t, first, 2)
	for _, res := range first {
		require.NoError(t, res.Err)
	}
	require.Equal(t, []string{"", ""}, client.presented[:2],
		"first boot presents no token (the holder starts empty)")
	require.Equal(t, "tok-web-search.learnings", tokens.get("web-search.learnings"))
	require.Equal(t, "tok-web-search.live", tokens.get("web-search.live"))

	second := searchregister.Register(context.Background(), cfg)
	require.Len(t, second, 2)
	for _, res := range second {
		require.NoError(t, res.Err)
	}
	require.Len(t, client.presented, 4)
	for i := 2; i < 4; i++ {
		wantToken := "tok-" + client.calls[i].GetProviderId()
		require.Equal(t, wantToken, client.presented[i],
			"re-registration echoes the cached control token for %s", client.calls[i].GetProviderId())
	}
}
