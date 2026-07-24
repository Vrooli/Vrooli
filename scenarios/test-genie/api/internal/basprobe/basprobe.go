// Package basprobe resolves the Browser Automation Studio (BAS) workflow
// endpoint and reports whether the engine is reachable. It is the single
// BAS-endpoint resolution path shared by the runnability gate (which uses
// ProbeBAS to skip/degrade BAS-dependent phases when BAS is down instead of
// failing them hard).
package basprobe

import (
	"context"
	"strings"
	"time"

	"test-genie/internal/basprobe/execution"

	"github.com/vrooli/api-core/discovery"
)

// BASScenarioName is the scenario slug for Browser Automation Studio, the engine
// that drives browser workflows and artifact capture.
const BASScenarioName = "browser-automation-studio"

// ResolveBASBaseURL resolves the BAS API base URL (".../api/v1") via scenario
// discovery. Browser phases and the runnability resource probe share this so
// there is exactly one BAS-endpoint resolution path.
func ResolveBASBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURL(ctx, BASScenarioName, "API_PORT")
	if err != nil {
		return "", err
	}
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if !strings.HasSuffix(url, "/api/v1") {
		url += "/api/v1"
	}
	return url, nil
}

// ProbeBAS reports whether the BAS workflow engine is reachable and healthy. It
// resolves the endpoint and issues a bounded health check, reusing the shared
// BAS client (no parallel probe). Used by the runnability gate to skip/degrade
// BAS-dependent phases when BAS is down instead of failing them hard.
func ProbeBAS(ctx context.Context) bool {
	baseURL, err := ResolveBASBaseURL(ctx)
	if err != nil {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client := execution.NewClient(baseURL)
	return client.Health(probeCtx) == nil
}
