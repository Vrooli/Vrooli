package validation

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"unit-health/internal/readiness"
)

type staticScenarioResolver struct{ url string }

func (r staticScenarioResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type fakeReadinessProvider struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	response *scenariovalidationv1.DescribeProviderResponse
}

func (p fakeReadinessProvider) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(p.response), nil
}

func TestSDAReadinessResolverConsumesProviderContract(t *testing.T) {
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(fakeReadinessProvider{
		response: &scenariovalidationv1.DescribeProviderResponse{
			Provider:    scenarioDependencyAnalyzer,
			Contract:    validationContract,
			SpecVersion: "2.0.0",
			Build:       &scenariovalidationv1.ProviderBuild{BinaryModifiedAt: timestamppb.Now()},
		},
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	report, err := (SDAReadinessResolver{Resolver: staticScenarioResolver{url: server.URL}}).Check(context.Background(), "scenario", "demo", "/tmp/demo")
	if err != nil {
		t.Fatalf("readiness check: %v", err)
	}
	if report.Status != readiness.Ready || report.Source != scenarioDependencyAnalyzer {
		t.Fatalf("report = %+v, want ready SDA report", report)
	}
	if got := report.Requirements[0].Version; got != "2.0.0" {
		t.Fatalf("provider version = %q, want 2.0.0", got)
	}
}

func TestSDAReadinessResolverFailsClosedOnWrongContract(t *testing.T) {
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(fakeReadinessProvider{
		response: &scenariovalidationv1.DescribeProviderResponse{Provider: scenarioDependencyAnalyzer, Contract: "wrong/v1"},
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	_ = path

	report, err := (SDAReadinessResolver{Resolver: staticScenarioResolver{url: server.URL}}).Check(context.Background(), "scenario", "demo", "")
	if err != nil {
		t.Fatalf("readiness check: %v", err)
	}
	if report.Status != readiness.Unavailable {
		t.Fatalf("status = %q, want unavailable", report.Status)
	}
}
