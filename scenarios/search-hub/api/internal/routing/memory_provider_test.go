package routing_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"search-hub/internal/routing"
)

func TestMemoryProviderPreservesQueryAndZoomProvenance(t *testing.T) {
	file, err := aisearch.LoadSearchFile("../../../../source-ledger/.vrooli/search.json")
	require.NoError(t, err)
	var provider *registryv1.ProviderDescriptor
	for _, config := range file.Providers {
		if config.ProviderID != "source-ledger.scopes" {
			continue
		}
		provider = &registryv1.ProviderDescriptor{
			ProviderId: config.ProviderID, ProviderGroup: config.ProviderGroup, Type: config.Type,
			Scope: registryv1.Scope_SCOPE_PROJECT, State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
			Endpoint: &registryv1.Endpoint{}, ResultMapping: &registryv1.ResultMapping{},
		}
		require.NoError(t, protojson.Unmarshal(config.Endpoint, provider.Endpoint))
		require.NoError(t, protojson.Unmarshal(config.ResultMapping, provider.ResultMapping))
		break
	}
	require.NotNil(t, provider)

	for _, tc := range []struct {
		name, query, scope, request, response, node, evidenceScope, snippet string
		degraded                                                            bool
	}{
		{
			name: "category terms reach memory recall", query: "open browser capture bug reports", scope: "scenario:browser-automation-studio",
			request:  `{"query":"open browser capture bug reports","limit":2,"scope":"browser-automation-studio"}`,
			response: `{"hits":[{"entryId":"summary-1","nodeId":"summary-1","scope":"browser-automation-studio","facetId":"bug","text":"Local capture is fixed. Remote capture still times out.","score":0.89,"summary":true},{"entryId":"first-report","nodeId":"first-report","scope":"browser-automation-studio","facetId":"bug","text":"Browser capture is broken.","score":0.51}]}`,
			node:     "summary-1", evidenceScope: "browser-automation-studio", snippet: "Local capture is fixed. Remote capture still times out.",
		},
		{
			name: "broad query keeps setup evidence and lexical degradation", query: "how do I set up browser capture",
			request:  `{"query":"how do I set up browser capture","limit":2,"scope":""}`,
			response: `{"hits":[{"entryId":"setup-1","nodeId":"setup-1","scope":"agent-memory","facetId":"procedure","text":"Start the browser capture setup wizard.","score":0.71}],"degradations":[{"reason":"SEMANTIC_UNAVAILABLE","leg":"semantic","detail":"lexical fallback active","retryable":true}]}`,
			node:     "setup-1", evidenceScope: "agent-memory", snippet: "Start the browser capture setup wizard.", degraded: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doer := &recordingBodyDoer{response: tc.response}
			router := routing.NewRouter(routing.Deps{
				Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
				Resolver: staticResolver{urls: map[string]string{"source-ledger": "http://ledger.test"}},
				Doer:     doer, QueryTimeout: time.Second, PerProviderTimeout: time.Second,
			})
			response, err := router.Query(context.Background(), &routingv1.QueryRequest{
				Query: tc.query, Scope: tc.scope, Group: provider.GetProviderGroup(), Limit: 2,
			})
			require.NoError(t, err)
			require.Equal(t, "http://ledger.test"+recallconnect.RecallServiceRecallProcedure, doer.request)
			require.JSONEq(t, tc.request, doer.body)
			require.Len(t, response.GetGroups(), 1)
			group := response.GetGroups()[0]
			require.Equal(t, tc.degraded, group.GetDegraded())
			require.NotEmpty(t, group.GetHits())
			hit := group.GetHits()[0]
			require.Equal(t, tc.node, hit.GetId())
			require.Equal(t, tc.node, hit.GetPath())
			require.Equal(t, tc.snippet, hit.GetSnippet())
			metadata := hit.GetMetadata().GetFields()
			require.Equal(t, tc.node, metadata["node_id"].GetStringValue())
			require.Equal(t, tc.evidenceScope, metadata["scope"].GetStringValue())
			require.Equal(t, strings.HasPrefix(tc.node, "summary-"), metadata["summary"].GetBoolValue())
		})
	}
}
