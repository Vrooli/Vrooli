package validation

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	apiDiscovery "github.com/vrooli/api-core/discovery"
	"unit-health/internal/readiness"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

const (
	scenarioDependencyAnalyzer = "scenario-dependency-analyzer"
	validationContract         = "scenario-validation/v1"
)

// SDAReadinessResolver consumes the dependency analyzer's provider-owned
// readiness contract. It deliberately does not call target analysis or any
// installation command: Unit Health only needs to know whether the governed
// dependency-readiness authority is live and speaking the expected contract.
type SDAReadinessResolver struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient connect.HTTPClient
}

// Check implements the Service readiness seam. Target arguments are accepted
// to keep the shared resolver contract target-aware, but provider readiness is
// intentionally target-independent and O(1).
func (r SDAReadinessResolver) Check(ctx context.Context, _, _, _ string) (readiness.Report, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = apiDiscovery.NewResolver(apiDiscovery.ResolverConfig{})
	}
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, scenarioDependencyAnalyzer)
	if err != nil {
		return unavailableReadiness(fmt.Sprintf("resolve %s: %v", scenarioDependencyAnalyzer, err)), nil
	}
	provider := scenariovalidationconnect.NewScenarioValidationServiceClient(client, baseURL)
	resp, err := provider.DescribeProvider(ctx, connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		return unavailableReadiness(fmt.Sprintf("query %s readiness: %v", scenarioDependencyAnalyzer, err)), nil
	}
	if resp == nil || resp.Msg == nil {
		return unavailableReadiness("dependency analyzer returned an empty readiness response"), nil
	}
	description := resp.Msg
	if strings.TrimSpace(description.GetProvider()) != scenarioDependencyAnalyzer {
		return unavailableReadiness(fmt.Sprintf("dependency analyzer identified itself as %q", description.GetProvider())), nil
	}
	if strings.TrimSpace(description.GetContract()) != validationContract {
		return unavailableReadiness(fmt.Sprintf("dependency analyzer contract is %q, want %q", description.GetContract(), validationContract)), nil
	}
	return readiness.Report{
		Status: readiness.Ready,
		Source: scenarioDependencyAnalyzer,
		Requirements: []readiness.Requirement{{
			ID:          scenarioDependencyAnalyzer,
			Kind:        "provider",
			Version:     description.GetSpecVersion(),
			Status:      readiness.Ready,
			Source:      scenarioDependencyAnalyzer,
			Remediation: "No action required; dependency readiness is governed by Scenario Dependency Analyzer.",
		}},
	}, nil
}

func unavailableReadiness(reason string) readiness.Report {
	return readiness.Report{
		Status: readiness.Unavailable,
		Source: scenarioDependencyAnalyzer,
		Requirements: []readiness.Requirement{{
			ID:          scenarioDependencyAnalyzer,
			Kind:        "provider",
			Status:      readiness.Unavailable,
			Source:      scenarioDependencyAnalyzer,
			Remediation: fmt.Sprintf("Restore Scenario Dependency Analyzer readiness (%s), then retry validation; Unit Health does not install dependencies.", reason),
		}},
	}
}
