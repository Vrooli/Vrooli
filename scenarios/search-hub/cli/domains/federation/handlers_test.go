package federation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/proto"
)

func TestStatusReportNamesStuckProviderInRecoverySummary(t *testing.T) {
	report := (&handlers{}).statusReport(nil, &routingv1.StatusResponse{
		Providers: []*routingv1.ProviderHealth{{
			ProviderId:    "provider.one",
			Reachable:     true,
			RecoveryState: "stuck",
			Stuck:         proto.Bool(true),
		}},
	})
	require.True(t, strings.Contains(strings.Join(report.Summary, "\n"), "provider.one"))
	require.Contains(t, report.Results[0], "recovery: stuck")
}

func TestStatusReportRendersIncubatingProvidersSeparately(t *testing.T) {
	report := (&handlers{}).statusReport(nil, &routingv1.StatusResponse{
		Providers: []*routingv1.ProviderHealth{{
			ProviderId: "production.one",
			Lifecycle:  "production",
		}},
		Incubating: []*registryv1.IncubatingProvider{{
			ProviderId: "experimental.one",
			DeclaredAt: "2026-08-13T20:38:00Z",
			NextAction: "run a recent passing evaluation",
		}},
	})
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	require.Contains(t, joined, "Incubating providers")
	require.Contains(t, joined, "experimental.one")
	require.Contains(t, joined, "run a recent passing evaluation")
	require.NotContains(t, joined, "production.one — declared=")
}

func TestStatusReportKeepsUnregisteredAccountingRowsInAuditView(t *testing.T) {
	report := (&handlers{}).statusReport(nil, &routingv1.StatusResponse{
		Providers: []*routingv1.ProviderHealth{{ProviderId: "registered.one", Reachable: true}},
		AuditProviders: []*routingv1.ProviderHealth{{
			ProviderId:  "stale.accounting.key",
			TimesRouted: 4,
			TotalHits:   1,
		}},
	})
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	require.Contains(t, joined, "Audit-only accounting rows: 1")
	require.Contains(t, joined, "Audit: stale.accounting.key")
	require.NotContains(t, report.Results[0], "stale.accounting.key")
}
