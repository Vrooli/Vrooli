package eventbus

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// NewDiscoveredClient creates an event client from the explicit environment
// override when present, otherwise resolving vrooli-events through the local
// scenario registry. Discovery is best effort: callers retain the existing
// degraded behavior when the event service is unavailable.
func NewDiscoveredClient(ctx context.Context) Client {
	baseURL := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE"))
	if baseURL != "" || ctx == nil {
		return Client{BaseURL: baseURL}
	}
	endpoint := &endpointRef{}
	endpoint.resolve = func() string {
		lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		resolved, err := discovery.ResolveScenarioURLDefault(lookupCtx, "vrooli-events")
		if err != nil {
			return ""
		}
		return resolved
	}
	return Client{endpoint: endpoint}
}
