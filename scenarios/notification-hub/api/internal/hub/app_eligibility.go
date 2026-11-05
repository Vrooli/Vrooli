package hub

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	exposure "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"
)

// TunnelAppEligibility allows app delivery only for a scenario that
// tunnel-manager exposes on a CORE (always-on) route. An installed app is
// bound to its origin; a leased route expires and would strand the app, so
// it is refused with a reason the settings panel shows.
func TunnelAppEligibility(tunnelBase func(context.Context) (string, error)) AppEligibility {
	client := &http.Client{Timeout: 10 * time.Second}
	return func(ctx context.Context, scenario string) (string, error) {
		base, err := tunnelBase(ctx)
		if err != nil || strings.TrimSpace(base) == "" {
			return "", fmt.Errorf("%w: tunnel-manager is not reachable, so the app's address cannot be checked", ErrAppIneligible)
		}
		resp, err := exposureconnect.NewExposureServiceClient(client, strings.TrimRight(base, "/")).ListExposures(ctx, connect.NewRequest(&exposure.ListExposuresRequest{}))
		if err != nil {
			return "", fmt.Errorf("%w: tunnel-manager did not answer, so the app's address cannot be checked", ErrAppIneligible)
		}
		return coreExposureURL(resp.Msg.GetExposures(), scenario)
	}
}

func coreExposureURL(exposures []*exposure.Exposure, scenario string) (string, error) {
	for _, item := range exposures {
		if item.GetScenario() != scenario || !item.GetEnabled() {
			continue
		}
		if !strings.EqualFold(item.GetTier(), "core") {
			return "", fmt.Errorf("%w: %s only has a temporary address, which would stop working when it expires. Make it always-on in tunnel-manager first", ErrAppIneligible, scenario)
		}
		return item.GetPublicUrl(), nil
	}
	return "", fmt.Errorf("%w: %s has no public address in tunnel-manager", ErrAppIneligible, scenario)
}
