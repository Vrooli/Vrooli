package routing

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestQueryTaskProtoJSONFixtureMatchesGeneratedGoProducer(t *testing.T) {
	fixture, err := os.ReadFile("testdata/query-task-response.json")
	if err != nil {
		t.Fatal(err)
	}
	groupIndex, hitIndex := int32(0), int32(0)
	response := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{
			ProviderId: "docs",
			Count:      1,
			Hits: []*routingv1.SearchHit{{
				ProviderId: "docs",
				Id:         "health-card",
				Title:      "HealthCard component",
				Path:       "ui/HealthCard.tsx",
			}},
		}},
		QueryParts: []*routingv1.QueryPartEvidence{
			{
				PartId: "surface",
				Label:  "Find the relevant interface",
				Source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
				Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
				Evidence: []*routingv1.SearchHitReference{{
					ProviderId: "docs", GroupIndex: proto.Int32(groupIndex), HitIndex: proto.Int32(hitIndex),
					HitId: "health-card", Path: "ui/HealthCard.tsx",
					RetrievalSources: []routingv1.SearchHitReference_RetrievalSource{
						routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY,
						routingv1.SearchHitReference_RetrievalSource(93),
					},
				}},
			},
			{
				PartId: "route",
				Label:  "Find routing guidance",
				Source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
				Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
				Evidence: []*routingv1.SearchHitReference{{
					ProviderId: "docs", GroupIndex: proto.Int32(groupIndex), HitIndex: proto.Int32(hitIndex), HitId: "stale-hit",
				}},
			},
			{
				PartId: "owner",
				Label:  "Identify the owning capability",
				Source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
				Status: routingv1.QueryPartEvidence_STATUS_NOT_ROUTED,
				Reason: "ambiguous_provider_evidence",
			},
			{
				PartId: "future-state",
				Label:  "Preserve a future coverage state",
				Source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
				Status: routingv1.QueryPartEvidence_Status(93),
			},
			{
				PartId: "future-source",
				Label:  "Preserve a future request-part source",
				Source: routingv1.QueryPartEvidence_Source(93),
				Status: routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY,
			},
		},
	}
	produced, err := protojson.Marshal(response)
	if err != nil {
		t.Fatalf("marshal generated QueryResponse: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(produced, &got); err != nil {
		t.Fatalf("decode generated JSON: %v", err)
	}
	if err := json.Unmarshal(fixture, &want); err != nil {
		t.Fatalf("decode shared fixture: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shared fixture differs from Go ProtoJSON producer\nproduced: %s\nfixture:  %s", produced, fixture)
	}

	var decoded routingv1.QueryResponse
	if err := protojson.Unmarshal(fixture, &decoded); err != nil {
		t.Fatalf("decode shared ProtoJSON into generated QueryResponse: %v", err)
	}
	if got := decoded.GetQueryParts()[0].GetSource(); got != routingv1.QueryPartEvidence_SOURCE_USER_QUERY {
		t.Errorf("source = %s, want SOURCE_USER_QUERY", got)
	}
	if got := decoded.GetQueryParts()[0].GetEvidence()[0].GetHitId(); got != "health-card" {
		t.Errorf("supported hit identity = %q, want health-card", got)
	}
	if got := decoded.GetQueryParts()[0].GetEvidence()[0].GetRetrievalSources(); len(got) != 2 || got[1] != routingv1.SearchHitReference_RetrievalSource(93) {
		t.Errorf("future retrieval source = %v, want user-query and unknown 93", got)
	}
	if got := decoded.GetQueryParts()[1].GetEvidence()[0].GetHitId(); got != "stale-hit" {
		t.Errorf("stale hit identity = %q, want it preserved for consumer-side unresolved handling", got)
	}
	if got := decoded.GetQueryParts()[2].GetStatus(); got != routingv1.QueryPartEvidence_STATUS_NOT_ROUTED {
		t.Errorf("not-routed status = %s", got)
	}
	if got := decoded.GetQueryParts()[3].GetStatus(); got != routingv1.QueryPartEvidence_Status(93) {
		t.Errorf("future numeric status = %d, want 93", got)
	}
	if got := decoded.GetQueryParts()[4].GetSource(); got != routingv1.QueryPartEvidence_Source(93) {
		t.Errorf("future numeric source = %d, want 93", got)
	}
}
