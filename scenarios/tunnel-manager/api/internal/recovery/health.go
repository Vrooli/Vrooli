package recovery

import (
	"context"
	"net/http"

	"tunnel-manager/internal/httpc"
)

// DefaultReadyURL is the cloudflared readiness endpoint probed before and
// after a restart. Matches the tunnel domain's metrics endpoint host.
const DefaultReadyURL = "http://127.0.0.1:20241/ready"

// httpHealthChecker is the production HealthChecker: it GETs the
// cloudflared /ready endpoint through the outbound HTTP seam and reports
// healthy only on 200. Keeping it in the recovery package (rather than
// importing the tunnel domain) keeps recovery's restart decision
// dependent on one thing — is the tunnel actually serving — without a
// cross-domain compile edge.
type httpHealthChecker struct {
	doer httpc.Doer
	url  string
}

// NewHTTPHealthChecker constructs the production HealthChecker.
func NewHTTPHealthChecker(doer httpc.Doer, url string) HealthChecker {
	if url == "" {
		url = DefaultReadyURL
	}
	return &httpHealthChecker{doer: doer, url: url}
}

func (h *httpHealthChecker) Ready(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return false
	}
	resp, err := h.doer.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
