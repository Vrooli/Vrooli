package sources

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/internal/deployability"
	platformv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/platform_verdict"
	platformconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/platform_verdict/platform_verdict_v1connect"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
)

const platformVerdictScenario = "scenario-dependency-analyzer"
const platformVerdictVerb = "vrooli.scenario_dependency_analyzer.v1.platform_verdict.PlatformVerdictService/ListPlatformVerdicts"

type PlatformVerdictReader struct {
	Resolver *discovery.Resolver
	HTTP     *http.Client
}

func (r PlatformVerdictReader) ListPlatformVerdicts(ctx context.Context) ([]portability.DerivedScenarioPlatformVerdict, error) {
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	response, err := r.Read(readCtx, "")
	if err != nil {
		return nil, err
	}
	result := make([]portability.DerivedScenarioPlatformVerdict, 0)
	for _, scenario := range response.GetScenarios() {
		for _, platform := range scenario.GetPlatforms() {
			result = append(result, portability.DerivedScenarioPlatformVerdict{
				Scenario:           scenario.GetScenario(),
				HostOS:             portabilityHostOS(platform.GetHostOs()),
				Status:             platform.GetStatus(),
				Reason:             platform.GetReason(),
				BlockingDependency: platform.GetBlockingDependency(),
			})
		}
	}
	return result, nil
}

func (r PlatformVerdictReader) ListPlatformFleet(ctx context.Context) (portability.DerivedPlatformFleet, error) {
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	response, err := r.Read(readCtx, "")
	if err != nil {
		return portability.DerivedPlatformFleet{}, err
	}
	result := portability.DerivedPlatformFleet{}
	for _, scenario := range response.GetScenarios() {
		for _, platform := range scenario.GetPlatforms() {
			result.Scenarios = append(result.Scenarios, portability.DerivedScenarioPlatformVerdict{Scenario: scenario.GetScenario(), HostOS: portabilityHostOS(platform.GetHostOs()), Status: platform.GetStatus(), Reason: platform.GetReason(), BlockingDependency: platform.GetBlockingDependency()})
		}
	}
	for _, block := range response.GetDockerBlocked() {
		result.DockerBlocked = append(result.DockerBlocked, portability.DerivedDockerBlock{Scenario: block.GetScenario(), HostOS: portabilityHostOS(block.GetHostOs()), Dependency: block.GetDependency(), Reason: block.GetReason()})
	}
	for _, upgrade := range response.GetTierUpgrades() {
		result.TierUpgrades = append(result.TierUpgrades, portability.DerivedTierUpgrade{Scenario: upgrade.GetScenario(), HostOS: portabilityHostOS(upgrade.GetHostOs()), CurrentTier: deployability.DeliveryTier(upgrade.GetCurrentTier()), NextTier: deployability.DeliveryTier(upgrade.GetNextTier()), Change: upgrade.GetChange(), BlockingDependency: upgrade.GetBlockingDependency()})
	}
	return result, nil
}

func portabilityHostOS(value string) deployability.HostOS {
	return deployability.HostOS(value)
}

func (r PlatformVerdictReader) Read(ctx context.Context, scenario string) (*platformv1.ListPlatformVerdictsResponse, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, platformVerdictScenario)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", platformVerdictScenario, err)
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := platformconnect.NewPlatformVerdictServiceClient(httpClient, base)
	response, err := client.ListPlatformVerdicts(ctx, connect.NewRequest(&platformv1.ListPlatformVerdictsRequest{Scenario: scenario}))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", platformVerdictVerb, err)
	}
	if response == nil || response.Msg == nil || !response.Msg.GetAvailable() {
		return nil, fmt.Errorf("read %s: source returned unavailable: %s", platformVerdictVerb, responseReason(response))
	}
	return response.Msg, nil
}

func ReadPlatformVerdicts(ctx context.Context, reader PlatformVerdictReader, timeout time.Duration, scenario string) TypedResult[*platformv1.ListPlatformVerdictsResponse] {
	return ReadTyped(ctx, "scenario-dependency-analyzer/platform-verdicts", func(ctx context.Context) (*platformv1.ListPlatformVerdictsResponse, error) {
		return reader.Read(ctx, scenario)
	}, timeout)
}

func responseReason(response *connect.Response[platformv1.ListPlatformVerdictsResponse]) string {
	if response == nil || response.Msg == nil {
		return "no response"
	}
	return response.Msg.GetReason()
}
