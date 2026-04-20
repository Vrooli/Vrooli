package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// CloudHealthClient checks LPBS cloud deployment health for the orchestrator.
// The deploy pipeline uses this to fail-fast if the LPBS runtime is unavailable
// before driving a release through build + publish.
type CloudHealthClient interface {
	CheckLPBSHealth(ctx context.Context) (*CloudHealthResult, error)
}

// CloudHealthResult is the outcome of a cloud-health probe.
type CloudHealthResult struct {
	Healthy bool   `json:"healthy"`
	Details string `json:"details,omitempty"`
}

// HTTPCloudHealthClient calls scenario-to-cloud's health endpoint.
type HTTPCloudHealthClient struct {
	httpClient *http.Client
	baseURL    string
	log        func(string, map[string]interface{})
}

// NewHTTPCloudHealthClient constructs the default cloud health client.
// Reads SCENARIO_TO_CLOUD_URL for testing, otherwise uses service discovery.
func NewHTTPCloudHealthClient(log func(string, map[string]interface{})) (*HTTPCloudHealthClient, error) {
	var baseURL string
	if env := strings.TrimSpace(os.Getenv("SCENARIO_TO_CLOUD_URL")); env != "" {
		baseURL = strings.TrimRight(env, "/")
	} else {
		url, err := discovery.ResolveScenarioURLDefault(context.Background(), "scenario-to-cloud")
		if err != nil {
			return nil, fmt.Errorf("resolve scenario-to-cloud URL: %w", err)
		}
		baseURL = url
	}
	return &HTTPCloudHealthClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		log:        log,
	}, nil
}

// CheckLPBSHealth returns whether the LPBS cloud deployment is healthy.
// We treat HTTP 200 as healthy. Any non-200 response or transport failure
// is a hard signal that the LPBS runtime isn't ready for a new release.
func (c *HTTPCloudHealthClient) CheckLPBSHealth(ctx context.Context) (*CloudHealthResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/deployments/landing-page-business-suite/health", nil)
	if err != nil {
		return nil, fmt.Errorf("build health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &CloudHealthResult{Healthy: false, Details: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		var parsed map[string]interface{}
		_ = json.Unmarshal(body, &parsed)
		return &CloudHealthResult{Healthy: true}, nil
	}
	return &CloudHealthResult{
		Healthy: false,
		Details: fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
	}, nil
}
