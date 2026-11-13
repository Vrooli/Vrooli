package routing

import (
	"context"
	"errors"
	"reflect"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
	internalrouting "search-hub/internal/routing"
)

type fleetSemanticIndexStore struct {
	request aisearch.FleetSemanticQueryRequest
	result  aisearch.FleetSemanticQueryResult
	err     error
	calls   int
}

func (s *fleetSemanticIndexStore) QueryActive(_ context.Context, request aisearch.FleetSemanticQueryRequest) (aisearch.FleetSemanticQueryResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

func (*fleetSemanticIndexStore) UpsertActive(context.Context, aisearch.FleetSemanticUpsertRequest) (aisearch.FleetSemanticOperationResult, error) {
	return aisearch.FleetSemanticOperationResult{}, errors.New("unexpected upsert")
}

func (*fleetSemanticIndexStore) DeleteActive(context.Context, aisearch.FleetSemanticDeleteRequest) (aisearch.FleetSemanticOperationResult, error) {
	return aisearch.FleetSemanticOperationResult{}, errors.New("unexpected delete")
}

type fleetSemanticIndexResolver struct{}

func (fleetSemanticIndexResolver) ResolveProviderSemanticContext(context.Context) (internalrouting.VerifiedProviderSemanticContext, error) {
	return internalrouting.VerifiedProviderSemanticContext{}, errors.New("test resolver must not run during composition")
}

func TestFleetSemanticDescriptionIndexForwardsIdentityAndRanksUniqueProviders(t *testing.T) {
	store := &fleetSemanticIndexStore{result: aisearch.FleetSemanticQueryResult{Results: []aisearch.SearchResult{
		{ID: "point-b-0", Score: 0.93, Payload: map[string]any{"provider_id": "provider-b"}},
		{ID: "point-a-0", Score: 0.82, Payload: map[string]any{"provider_id": "provider-a"}},
		{ID: "point-a-1", Score: 0.88, Payload: map[string]any{"provider_id": "provider-a"}},
		{ID: "foreign-point", Score: 1, Payload: map[string]any{"provider_id": "provider-not-in-live-registry"}},
	}}}
	index, err := newFleetSemanticDescriptionIndex(store)
	if err != nil {
		t.Fatal(err)
	}
	request := internalrouting.ProviderDescriptionQuery{
		OperationID: "op-1", TenantID: "tenant-search-hub", ArmID: "arm-1", RecipeID: "sha256:recipe",
		Text: "semantic search", Limit: 2,
		Attribution: internalrouting.ProviderDescriptionAttribution{
			CallerID: "caller-1", RightsEvidenceURI: "receipt://rights", RightsEvidenceSHA256: testDigest('a'),
			QualityEvidenceURI: "evidence://quality", QualityEvidenceSHA256: testDigest('b'),
		},
	}
	profiles := []internalrouting.ProviderProfile{{ProviderID: "provider-a"}, {ProviderID: "provider-b"}, {ProviderID: "provider-c"}}
	shortlist, result := index.Shortlist(context.Background(), request, profiles)
	if !result.Available || result.Returned != 2 || len(shortlist) != 2 {
		t.Fatalf("unexpected shortlist result: %#v, %#v", shortlist, result)
	}
	if shortlist[0].ProviderID != "provider-b" || shortlist[1].ProviderID != "provider-a" {
		t.Fatalf("shortlist did not rank unique providers by their best segment: %#v", shortlist)
	}
	if result.Scores["provider-a"] != 0.88 || result.Scores["provider-b"] != 0.93 || !reflect.DeepEqual(result.Omitted, []string{"provider-c"}) {
		t.Fatalf("unexpected scores/omissions: %#v", result)
	}
	want := aisearch.FleetSemanticQueryRequest{
		OperationID: request.OperationID, TenantID: request.TenantID, ArmID: request.ArmID, RecipeID: request.RecipeID,
		Text: request.Text, Limit: request.Limit,
		Attribution: aisearch.FleetSemanticAttribution{
			CallerID: request.Attribution.CallerID, RightsEvidenceURI: request.Attribution.RightsEvidenceURI,
			RightsEvidenceSHA256: request.Attribution.RightsEvidenceSHA256, QualityEvidenceURI: request.Attribution.QualityEvidenceURI,
			QualityEvidenceSHA256: request.Attribution.QualityEvidenceSHA256,
		},
	}
	if !reflect.DeepEqual(store.request, want) || store.calls != 1 {
		t.Fatalf("store request was not forwarded exactly: got %#v want %#v calls=%d", store.request, want, store.calls)
	}
}

func TestFleetSemanticDescriptionIndexRejectsMissingIdentityBeforeStore(t *testing.T) {
	store := &fleetSemanticIndexStore{}
	index, err := newFleetSemanticDescriptionIndex(store)
	if err != nil {
		t.Fatal(err)
	}
	shortlist, result := index.Shortlist(context.Background(), internalrouting.ProviderDescriptionQuery{Text: "query", Limit: 1}, []internalrouting.ProviderProfile{{ProviderID: "provider-a"}})
	if store.calls != 0 || result.Available || result.Reason != "routing_identity_unavailable" || len(shortlist) != 0 {
		t.Fatalf("incomplete identity reached semantic store or did not fail closed: calls=%d result=%#v shortlist=%#v", store.calls, result, shortlist)
	}
}

func TestFleetSemanticDescriptionIndexRequiresStore(t *testing.T) {
	index, err := newFleetSemanticDescriptionIndex(nil)
	if err == nil || index != nil {
		t.Fatalf("missing fleet store should remain unavailable: index=%#v err=%v", index, err)
	}
}

func TestFleetSemanticRoutingCompositionRequiresBothTrustedDependencies(t *testing.T) {
	store := &fleetSemanticIndexStore{}
	resolver := fleetSemanticIndexResolver{}
	for _, input := range []struct {
		name     string
		store    aisearch.FleetSemanticStore
		resolver internalrouting.ProviderSemanticContextResolver
	}{
		{name: "no store", resolver: resolver},
		{name: "no resolver", store: store},
		{name: "neither"},
	} {
		t.Run(input.name, func(t *testing.T) {
			index, identity := fleetSemanticRoutingDependencies(input.store, input.resolver)
			if index != nil || identity != nil {
				t.Fatalf("semantic route should stay closed: index=%T identity=%T", index, identity)
			}
		})
	}
	index, identity := fleetSemanticRoutingDependencies(store, resolver)
	if _, ok := index.(*fleetSemanticDescriptionIndex); !ok || identity != resolver {
		t.Fatalf("complete trusted dependencies should compose the fleet path: index=%T identity=%T", index, identity)
	}
}

func testDigest(char byte) string {
	value := make([]byte, 64)
	for i := range value {
		value[i] = char
	}
	return string(value)
}
