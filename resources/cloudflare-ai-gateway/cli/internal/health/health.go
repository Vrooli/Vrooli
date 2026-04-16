package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"resource-cloudflare-ai-gateway/cli/internal/auth"
	"resource-cloudflare-ai-gateway/cli/internal/config"
)

// HTTPClient is the narrow HTTP client contract used for safe connectivity
// probes.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Result summarizes a safe Cloudflare AI Gateway connectivity probe.
type Result struct {
	Status        string
	Message       string
	Endpoint      string
	HTTPStatus    int
	Authenticated bool
}

// Probe performs a provider-safe GET request against the configured
// Cloudflare endpoint. It does not mutate remote state.
func Probe(ctx context.Context, client HTTPClient, endpointRoot string, creds auth.Credentials) (Result, error) {
	if client == nil {
		client = http.DefaultClient
	}

	endpoint := strings.TrimSpace(endpointRoot)
	authenticated := creds.Valid()
	if authenticated {
		if derived := config.GatewayAPIBaseURL(endpointRoot, creds.AccountID); derived != "" {
			endpoint = derived
		}
	}
	if endpoint == "" {
		return Result{}, fmt.Errorf("health probe endpoint is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	if header := creds.AuthorizationHeader(); header != "" {
		req.Header.Set("Authorization", header)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			Status:        "unreachable",
			Message:       err.Error(),
			Endpoint:      endpoint,
			Authenticated: authenticated,
		}, err
	}
	defer resp.Body.Close()

	result := Result{
		Endpoint:      endpoint,
		HTTPStatus:    resp.StatusCode,
		Authenticated: authenticated,
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		result.Status = "reachable"
		if authenticated {
			result.Message = "Cloudflare AI Gateway API responded"
		} else {
			result.Message = "Cloudflare API endpoint reachable without credentials"
		}
	default:
		result.Status = "degraded"
		result.Message = fmt.Sprintf("unexpected Cloudflare status: %d", resp.StatusCode)
	}

	return result, nil
}
