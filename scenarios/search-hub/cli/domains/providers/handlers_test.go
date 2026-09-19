package providers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestListReportRendersIncubatingProvidersAndOmitsProduction(t *testing.T) {
	report := (&handlers{}).listReport(nil, &registryv1.ListProvidersResponse{
		Providers: []*registryv1.ProviderDescriptor{
			{ProviderId: "production.one", Lifecycle: registryv1.Lifecycle_LIFECYCLE_PRODUCTION},
			{ProviderId: "experimental.one", Lifecycle: registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL},
		},
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

func TestFormatProviderIncludesLifecycle(t *testing.T) {
	got := formatProvider(&registryv1.ProviderDescriptor{
		ProviderId: "experimental.one",
		Lifecycle:  registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL,
	})
	require.Contains(t, got, "lifecycle=experimental")
}
