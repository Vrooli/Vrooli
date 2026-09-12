package campaigns

import (
	"context"
	"os"

	"github.com/vrooli/api-core/discovery"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const readinessPrefix = "CONTENT_DESK_DEPLOYMENT_MANAGER_READINESS"

func newReadinessReporter(ctx context.Context) (*readinessreporter.Reporter, error) {
	if os.Getenv(readinessPrefix+"_TOKEN") == "" {
		return nil, nil
	}
	baseURL := os.Getenv("DEPLOYMENT_MANAGER_API_BASE_URL")
	if baseURL == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
		if err != nil {
			return nil, err
		}
		baseURL = resolved
	}
	return readinessreporter.NewFromEnv(readinessPrefix, baseURL, "marketing-assets-available", "content-desk.launch-assets.readiness", "content-desk")
}
