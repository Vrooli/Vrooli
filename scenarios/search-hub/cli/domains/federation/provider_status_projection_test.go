package federation

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProviderStatusCLIJSONPreservesProtoFieldsAndAddsPresentProjection(t *testing.T) {
	message := &routingv1.ProviderStatusResponse{
		ProviderId: "selected.synthetic",
		ObservedAt: timestamppb.New(time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)),
		Health: &routingv1.ProviderHealth{
			ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved",
			IndexAge: "5m", FreshnessBudget: "1h",
		},
	}

	withoutHeaders, err := providerStatusCLIResponse(message, nil)
	require.NoError(t, err)
	require.NotContains(t, withoutHeaders.AsMap(), "status_projection", "older servers must preserve the established JSON shape")

	headers := http.Header{
		"X-Search-Hub-Selected-Owner-Identity-Match":          {"MATCHED"},
		"X-Search-Hub-Selected-Owner-Status-RPC-Reachability": {"REACHABLE"},
		"X-Search-Hub-Selected-Owner-Endpoint-Reachability":   {"REACHABLE"},
		"X-Search-Hub-Selected-Owner-Freshness":               {"CURRENT"},
		"X-Search-Hub-Selected-Owner-Supported-Fallback":      {"UNKNOWN"},
	}
	withHeaders, err := providerStatusCLIResponse(message, headers)
	require.NoError(t, err)
	fields := withHeaders.AsMap()
	require.Equal(t, "selected.synthetic", fields["provider_id"])
	require.NotContains(t, fields, "summary", "machine output stays the original proto-shaped response")
	projection, ok := fields["status_projection"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "MATCHED", projection["identity_match"])
	require.Equal(t, "REACHABLE", projection["status_rpc_reachability"])
	require.Equal(t, "REACHABLE", projection["endpoint_reachability"])
	require.Equal(t, "CURRENT", projection["freshness"])
	require.Equal(t, "UNKNOWN", projection["supported_fallback"])
	require.True(t, proto.Equal(message, providerStatusProto(withHeaders)), "round-trip should retain existing typed fields")

	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(withHeaders)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Contains(t, decoded, "provider_id")
	require.Contains(t, decoded, "health")
	require.Contains(t, decoded, "status_projection")
}

func TestProviderStatusCLIHumanReportAddsProjectionOnlyWhenAvailable(t *testing.T) {
	message := &routingv1.ProviderStatusResponse{
		ProviderId: "selected.synthetic",
		Health:     &routingv1.ProviderHealth{ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved", IndexAge: "5m", FreshnessBudget: "1h"},
	}
	withoutHeaders, err := providerStatusCLIResponse(message, nil)
	require.NoError(t, err)
	report := (&handlers{}).providerStatusReport(nil, withoutHeaders)
	require.NotContains(t, strings.Join(report.Summary, "\n"), "Selected status projection:")

	withHeaders, err := providerStatusCLIResponse(message, http.Header{"X-Search-Hub-Selected-Owner-Supported-Fallback": {"UNKNOWN"}})
	require.NoError(t, err)
	report = (&handlers{}).providerStatusReport(nil, withHeaders)
	require.Contains(t, strings.Join(report.Summary, "\n"), "Selected status projection: supported-fallback=UNKNOWN.")
}
