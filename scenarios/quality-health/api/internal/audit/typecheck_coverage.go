package audit

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	apiDiscovery "github.com/vrooli/api-core/discovery"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"

	"quality-health/internal/surfaces"
)

// NewUnitHealthPlannerCoverageChecker returns the provider-owned coverage
// query used by TYPECHECK_PLANNER_COVERAGE. Quality Health consumes the
// Unit Health plan response; it does not duplicate adapter matching rules.
func NewUnitHealthPlannerCoverageChecker() func(context.Context, surfaces.Inventory) (map[string]bool, error) {
	return func(ctx context.Context, inv surfaces.Inventory) (map[string]bool, error) {
		resolver := apiDiscovery.NewResolver(apiDiscovery.ResolverConfig{})
		baseURL, err := resolver.ResolveScenarioURLDefault(ctx, "unit-health")
		if err != nil {
			return nil, fmt.Errorf("resolve unit-health: %w", err)
		}
		return typecheckCoverage(ctx, validationconnect.NewValidationServiceClient(http.DefaultClient, baseURL), inv)
	}
}

// typecheckCoverage reports which workspaces of inv the Unit Health plan
// covers with a typecheck command.
func typecheckCoverage(ctx context.Context, client validationconnect.ValidationServiceClient, inv surfaces.Inventory) (map[string]bool, error) {
	// Send the resolved root too: an isolated workspace (template deep
	// validation, routed runs) is not under unit-health's own scenarios/
	// directory, so the slug alone would plan nothing.
	resp, err := client.ValidateScenario(ctx, connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario: inv.Scenario,
		Path:     inv.RootPath,
		UseCache: true,
	}))
	if err != nil {
		return nil, fmt.Errorf("query unit-health plan: %w", err)
	}
	covered := make(map[string]bool)
	for _, command := range resp.Msg.GetPlan().GetCommands() {
		if command.GetTestKind() == "typecheck" {
			covered[command.GetWorkspaceId()] = true
		}
	}
	return covered, nil
}
