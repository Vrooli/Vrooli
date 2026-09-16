package smoketest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

const (
	isolatedSmokeVariant             = "desktop-smoke"
	isolatedSmokeStartTimeoutSeconds = 300
)

// resolveSmokeTarget performs an exact-instance lookup for proxy smoke runs.
// LookupScenarioPort intentionally receives the explicit variant and never
// uses a compatibility detector whose fallback may return live.
func resolveSmokeTarget(ctx context.Context, status *Status) (JourneyTarget, error) {
	if status == nil || status.ScenarioName == "" {
		return JourneyTarget{}, fmt.Errorf("isolated_instance_unavailable")
	}
	if status.DeploymentMode == "bundled" {
		return resolveBundledTarget(ctx, status.SmokeTestID)
	}
	target := status.ScenarioName + "@" + isolatedSmokeVariant
	// An exact running instance is already lifecycle-owned. Accept its
	// non-overlapping ports directly; the desktop journey performs the target
	// readiness check before interacting with the application. This also avoids
	// turning a healthy reused instance into a second StartScenario request.
	if resolved, err := resolveIsolatedTarget(ctx, status.ScenarioName, target); err == nil {
		return resolved, nil
	}
	if existing, ok := resolveHealthyIsolatedTarget(ctx, status.ScenarioName, target); ok {
		return existing, nil
	}
	if err := startIsolatedSmokeInstance(ctx, target); err != nil {
		return JourneyTarget{}, fmt.Errorf("isolated_instance_unavailable: %w", err)
	}
	return resolveIsolatedTarget(ctx, status.ScenarioName, target)
}

func resolveHealthyIsolatedTarget(ctx context.Context, scenarioName, target string) (JourneyTarget, bool) {
	resolved, err := resolveIsolatedTarget(ctx, scenarioName, target)
	if err != nil {
		return JourneyTarget{}, false
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved.APIURL+"/health", nil)
	if err != nil {
		return JourneyTarget{}, false
	}
	response, err := newControlPlaneHTTPClient().Do(request)
	if err != nil {
		return JourneyTarget{}, false
	}
	defer response.Body.Close()
	return resolved, response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

func resolveIsolatedTarget(ctx context.Context, scenarioName, target string) (JourneyTarget, error) {
	ui := cliutil.LookupScenarioPort(ctx, target, "UI_PORT", cliutil.PortCachePolicy{})
	api := cliutil.LookupScenarioPort(ctx, target, "API_PORT", cliutil.PortCachePolicy{})
	if !ui.Resolved() || !api.Resolved() {
		return JourneyTarget{}, fmt.Errorf("isolated_instance_unavailable: %s", target)
	}
	liveUI := cliutil.LookupScenarioPort(ctx, scenarioName, "UI_PORT", cliutil.PortCachePolicy{})
	liveAPI := cliutil.LookupScenarioPort(ctx, scenarioName, "API_PORT", cliutil.PortCachePolicy{})
	if (liveUI.Resolved() && liveUI.Port == ui.Port) || (liveAPI.Resolved() && liveAPI.Port == api.Port) {
		return JourneyTarget{}, fmt.Errorf("live_instance_target_refused: %s", target)
	}
	return JourneyTarget{
		RendererURL: "http://127.0.0.1:" + ui.Port,
		APIURL:      "http://127.0.0.1:" + api.Port,
		Instance:    target,
		Source:      "isolated_instance",
	}, nil
}

func startIsolatedSmokeInstance(ctx context.Context, target string) error {
	baseURL := strings.TrimSpace(os.Getenv("VROOLI_CONTROL_PLANE_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("VROOLI_API_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8092"
	}
	// The lifecycle request owns its timeout. A shorter HTTP-client timeout
	// makes healthy but setup-heavy variants look unavailable before the
	// control-plane operation can return its typed lifecycle response.
	client := cliv1connect.NewScenarioControlPlaneServiceClient(newControlPlaneHTTPClient(), baseURL)
	_, err := client.StartScenario(ctx, connect.NewRequest(&cliv1.StartScenarioRequest{
		Name:           target,
		TimeoutSeconds: isolatedSmokeStartTimeoutSeconds,
		BestEffort:     false,
		DemandManaged:  true,
	}))
	return err
}

func newControlPlaneHTTPClient() *http.Client {
	return &http.Client{}
}

type bundledTargetFile struct {
	RendererURL string                             `json:"renderer_url"`
	APIURL      string                             `json:"api_url"`
	Isolation   *deliveryramp.IsolationObservation `json:"isolation,omitempty"`
}

func bundledTargetPath(smokeTestID string) string {
	return filepath.Join(os.TempDir(), "scenario-to-desktop", "bundled-targets", smokeTestID+".json")
}

func resolveBundledTarget(ctx context.Context, smokeTestID string) (JourneyTarget, error) {
	if strings.TrimSpace(smokeTestID) == "" {
		return JourneyTarget{}, fmt.Errorf("bundled_private_target_unavailable")
	}
	path := bundledTargetPath(smokeTestID)
	for {
		raw, err := os.ReadFile(path)
		if err == nil {
			var target bundledTargetFile
			if json.Unmarshal(raw, &target) == nil && target.RendererURL != "" && target.APIURL != "" {
				return JourneyTarget{RendererURL: target.RendererURL, APIURL: target.APIURL, Instance: "bundled-runtime", Source: "bundled_private", Isolation: target.Isolation}, nil
			}
		}
		select {
		case <-ctx.Done():
			return JourneyTarget{}, fmt.Errorf("bundled_private_target_unavailable: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
