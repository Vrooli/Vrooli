package offers

import (
	"context"
	"os"
	"strings"

	"github.com/vrooli/api-core/discovery"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const readinessPrefix = "OFFER_DESK_DEPLOYMENT_MANAGER_READINESS"

func newReadinessReporter(ctx context.Context, criterionID, binding string) (*readinessreporter.Reporter, error) {
	if os.Getenv(readinessPrefix+"_TOKEN") == "" {
		return nil, nil
	}
	return readinessreporter.NewFromEnvResolver(readinessPrefix, resolveDeploymentManagerURL, criterionID, binding, "offer-desk")
}

func resolveDeploymentManagerURL(ctx context.Context) (string, error) {
	if base := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_API_BASE_URL")); base != "" {
		return base, nil
	}
	return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
}

func configuredReadinessScenario() string {
	return os.Getenv(readinessPrefix + "_SCENARIO")
}
