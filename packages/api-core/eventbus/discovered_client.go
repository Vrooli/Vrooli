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
// scenario registry. A registry miss is retried on the next operation, and a
// transport failure clears the discovered address so the caller's next retry
// can resolve a replacement. Explicit overrides stay pinned. Discovery remains
// best effort; this client never replays a failed event automatically.
func NewDiscoveredClient(ctx context.Context) Client {
	baseURL := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE"))
	if baseURL != "" || ctx == nil {
		return Client{BaseURL: baseURL}
	}
	return newDiscoveredClient(baseURL, func() string {
		lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		resolved, err := discovery.ResolveScenarioURLDefault(lookupCtx, "vrooli-events")
		if err != nil {
			return ""
		}
		return resolved
	})
}

func newDiscoveredClient(baseURL string, resolve func() string) Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		return Client{BaseURL: baseURL}
	}
	if resolve == nil {
		return Client{}
	}
	return Client{endpoint: &endpointRef{resolve: resolve}}
}
