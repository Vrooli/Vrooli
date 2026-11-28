package campaigns

import (
	"context"
	"os"
	"strings"

	"github.com/vrooli/api-core/discovery"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const readinessPrefix = "CONTENT_DESK_DEPLOYMENT_MANAGER_READINESS"

func newReadinessReporter(ctx context.Context) (*readinessreporter.Reporter, error) {
	if os.Getenv(readinessPrefix+"_TOKEN") == "" {
		return nil, nil
	}
	return readinessreporter.NewFromEnvResolver(readinessPrefix, resolveDeploymentManagerURL, "marketing-assets-available", "content-desk.launch-assets.readiness", "content-desk")
}

func resolveDeploymentManagerURL(ctx context.Context) (string, error) {
	if base := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_API_BASE_URL")); base != "" {
		return base, nil
	}
	return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
}
