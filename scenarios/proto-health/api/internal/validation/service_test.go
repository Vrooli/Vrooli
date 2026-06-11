package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"proto-health/internal/protosurface"
)

type fakeLoader struct {
	surface protosurface.Surface
	err     error
}

func (f fakeLoader) LoadScenario(string) (protosurface.Surface, error) {
	return f.surface, f.err
}

type fakeGenSyncChecker struct {
	status GenSyncStatus
	err    error
}

func (f fakeGenSyncChecker) CheckScenario(context.Context, string) (GenSyncStatus, error) {
	return f.status, f.err
}

func TestValidateScenarioCleanSurfacePasses(t *testing.T) {
	svc := New(Deps{Loader: fakeLoader{surface: cleanSurface()}})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
	require.Zero(t, report.Summary.Errors)
}

func TestValidateScenarioFindsGeneratedArtifactDrift(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		GenSyncChecker: fakeGenSyncChecker{status: GenSyncStatus{
			InSync: false,
			Drift:  []string{"packages/proto/gen/go/demo"},
			Detail: "1 generated slice differs after regeneration",
		}},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeGenOutOfSync, SeverityError)
}

func TestValidateScenarioFindsPolicyViolations(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files,
		protosurface.File{
			Path:        "demo/v1/orders/orders.proto",
			Package:     "vrooli.demo.v1.orders",
			Version:     "v1",
			Domain:      "orders",
			Stability:   "experimental",
			Annotations: []protosurface.Annotation{{Name: "layer", Value: "5"}},
		},
		protosurface.File{
			Path:      "demo/v3/billing/billing.proto",
			Package:   "vrooli.demo.v3.wrong",
			Version:   "v3",
			Domain:    "billing",
			Stability: "stable",
		},
	)
	surface.Services = append(surface.Services,
		protosurface.Service{
			FilePath: "demo/v1/orders/orders.proto",
			Package:  "vrooli.demo.v1.orders",
			Name:     "OrdersService",
			FullName: "vrooli.demo.v1.orders.OrdersService",
			Domain:   "orders",
			RPCs: []protosurface.RPC{{
				Name:      "ListOrders",
				Input:     "vrooli.demo.v1.orders.ListOrdersRequest",
				Output:    "vrooli.demo.v1.orders.ListOrdersResponse",
				Transport: protosurface.TransportKindConnect,
			}},
		},
	)
	surface.Messages = append(surface.Messages,
		protosurface.Message{FilePath: "demo/v1/orders/orders.proto", Package: "vrooli.demo.v1.orders", Name: "ListOrdersRequest", FullName: "vrooli.demo.v1.orders.ListOrdersRequest", Domain: "orders"},
		protosurface.Message{FilePath: "demo/v1/orders/orders.proto", Package: "vrooli.demo.v1.orders", Name: "ListOrdersResponse", FullName: "vrooli.demo.v1.orders.ListOrdersResponse", Domain: "orders"},
		protosurface.Message{FilePath: "demo/v3/billing/billing.proto", Package: "vrooli.demo.v3.wrong", Name: "Unused", FullName: "vrooli.demo.v3.wrong.Unused", Domain: "billing"},
	)
	surface.IntraScenarioImports = append(surface.IntraScenarioImports,
		protosurface.Import{FromFile: "demo/v1/orders/orders.proto", ToFile: "demo/v3/billing/billing.proto", FromDomain: "orders", ToDomain: "billing"},
	)

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)

	requireFinding(t, report, CodePackageMismatch, SeverityError)
	requireFinding(t, report, CodeStabilityDishonest, SeverityError)
	requireFinding(t, report, CodeCrossDomainImport, SeverityWarning)
	requireFinding(t, report, CodeUnsupportedAnnotation, SeverityWarning)
	requireFinding(t, report, CodeVersionNaming, SeverityWarning)
}

func TestValidateScenarioFindsImportCycle(t *testing.T) {
	surface := cleanSurface()
	surface.IntraScenarioImports = []protosurface.Import{
		{FromFile: "demo/v1/shared/health.proto", ToFile: "demo/v1/shared/errors.proto", FromDomain: "shared", ToDomain: "shared"},
		{FromFile: "demo/v1/shared/errors.proto", ToFile: "demo/v1/shared/health.proto", FromDomain: "shared", ToDomain: "shared"},
	}
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/shared/errors.proto",
		Package:   "vrooli.demo.v1.shared",
		Version:   "v1",
		Domain:    "shared",
		Stability: "stable",
	})

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeCycle, SeverityError)
}

func TestDescribeScenarioProtosReturnsSurface(t *testing.T) {
	surface := cleanSurface()
	svc := New(Deps{Loader: fakeLoader{surface: surface}})

	got, err := svc.DescribeScenarioProtos(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, surface, got)
}

func cleanSurface() protosurface.Surface {
	return protosurface.Surface{
		Scenario:       "demo",
		TransportWorld: protosurface.TransportWorldConnect,
		Files: []protosurface.File{{
			Path:        "demo/v1/shared/health.proto",
			Package:     "vrooli.demo.v1.shared",
			Version:     "v1",
			Domain:      "shared",
			Stability:   "stable",
			Annotations: []protosurface.Annotation{{Name: "stability", Value: "stable"}},
		}},
		Messages: []protosurface.Message{{
			FilePath: "demo/v1/shared/health.proto",
			Package:  "vrooli.demo.v1.shared",
			Name:     "HealthResponse",
			FullName: "vrooli.demo.v1.shared.HealthResponse",
			Domain:   "shared",
		}},
		AdoptionSignals: []protosurface.AdoptionSignal{
			{Name: "api_go_mod_replace", Present: true, Detail: "api/go.mod references the shared packages/proto module"},
			{Name: "api_generated_go_import", Present: true, Detail: "api code imports this scenario's generated Go proto package"},
		},
	}
}

func requireFinding(t *testing.T, report Report, code string, severity Severity) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code && finding.Severity == severity {
			return
		}
	}
	require.Failf(t, "missing finding", "code=%s severity=%s findings=%+v", code, severity, report.Findings)
}
