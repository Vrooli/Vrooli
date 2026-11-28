package routing

import (
	"testing"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// A provider that answers with a typed degraded leg is not "other": the typed
// reason decides, and the free-form note is only the fallback.
func TestClassifyGroupDegradeReasonPrefersTheProvidersTypedReason(t *testing.T) {
	group := func(note string, reasons ...string) *routingv1.ProviderResultGroup {
		g := &routingv1.ProviderResultGroup{Degraded: true, Note: note}
		for _, reason := range reasons {
			g.Degradations = append(g.Degradations, &routingv1.ProviderDegradation{Reason: reason})
		}
		return g
	}
	for _, test := range []struct {
		name  string
		group *routingv1.ProviderResultGroup
		want  string
	}{
		{"semantic leg down", group("semantic retrieval is unavailable; lexical results remain available", "CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE"), "provider_leg_unavailable"},
		{"embedding down", group("", "CONVERSATION_SEARCH_DEGRADATION_REASON_EMBEDDING_UNAVAILABLE"), "provider_leg_unavailable"},
		{"layout mismatch", group("", "CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_LAYOUT_MISMATCH"), "provider_leg_unavailable"},
		{"provider deadline", group("exceeded its request budget", "CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE"), "timeout"},
		{"first recognised reason wins", group("", "CONVERSATION_SEARCH_DEGRADATION_REASON_CANDIDATE_LIMIT", "CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE"), "timeout"},
		{"unrecognised typed reason falls back to the note", group("provider returned HTTP 503", "SOMETHING_NEW"), "http_error"},
		{"no typed reason keeps note keywords", group("request failed: dial tcp: connection refused"), "unreachable"},
		{"no typed reason and no note stays other", group(""), "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, classifyGroupDegradeReason(test.group))
		})
	}
}
