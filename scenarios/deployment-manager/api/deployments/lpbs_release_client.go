package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// LPBSReleaseClient talks to landing-page-business-suite for release-time
// control-plane checks: upload readiness and post-publish verification.
// The same interface is wrapped by tests to inject fakes.
type LPBSReleaseClient interface {
	// CheckDeployReadiness calls POST /api/v1/deploy-readiness for the app.
	// A 200 response indicates LPBS is ready to accept a new release.
	CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error)

	// Verify calls GET /api/v1/updates/{app_key}/verify and reports whether
	// the expected version is live on the channel.
	Verify(ctx context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error)
}

// LPBSReadinessRequest is the body for the deploy-readiness check.
type LPBSReadinessRequest struct {
	AppKey        string `json:"app_key"`
	RemoteProfile string `json:"remote_profile,omitempty"`
	Channel       string `json:"channel,omitempty"`
	ProfileTag    string `json:"profile_tag,omitempty"`
}

// LPBSReadinessResult is the decoded deploy-readiness response.
type LPBSReadinessResult struct {
	Ready bool            `json:"ready"`
	Gates []ReadinessGate `json:"gates,omitempty"`
	Error string          `json:"error,omitempty"`
}

// ReadinessGate reports one gate's status within the readiness response.
type ReadinessGate struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// LPBSVerifyRequest is the input for a post-release verify call.
type LPBSVerifyRequest struct {
	AppKey          string
	Channel         string
	Platform        string
	ExpectedVersion string
	Deep            bool
}

// LPBSVerifyResult mirrors LPBS /verify response.
type LPBSVerifyResult struct {
	AppKey          string `json:"app_key"`
	Channel         string `json:"channel"`
	Platform        string `json:"platform"`
	ExpectedVersion string `json:"expected_version"`
	ObservedVersion string `json:"observed_version,omitempty"`
	SHA512Match     bool   `json:"sha512_match"`
	Match           bool   `json:"match"`
	Error           string `json:"error,omitempty"`
}

// HTTPLPBSReleaseClient is the default LPBS release client over HTTP.
type HTTPLPBSReleaseClient struct {
	httpClient    *http.Client
	baseURL       string
	serviceSecret string
	log           func(string, map[string]interface{})
}

// LPBSClientConfig options for constructing the LPBS client. If BaseURL is
// empty, LPBS_BASE_URL env var is used. If ServiceSecret is empty,
// LPBS_SERVICE_SECRET env var is used.
type LPBSClientConfig struct {
	BaseURL       string
	ServiceSecret string
	Log           func(string, map[string]interface{})
}

// NewHTTPLPBSReleaseClient creates a new HTTP LPBS release client.
// Logs a warning if LPBS_SERVICE_SECRET is missing so misconfig is visible.
func NewHTTPLPBSReleaseClient(cfg LPBSClientConfig) (*HTTPLPBSReleaseClient, error) {
	base := cfg.BaseURL
	if strings.TrimSpace(base) == "" {
		base = strings.TrimSpace(os.Getenv("LPBS_BASE_URL"))
	}
	if base == "" {
		return nil, fmt.Errorf("LPBS base URL not configured (set LPBS_BASE_URL or pass BaseURL)")
	}
	secret := cfg.ServiceSecret
	if strings.TrimSpace(secret) == "" {
		secret = strings.TrimSpace(os.Getenv("LPBS_SERVICE_SECRET"))
	}
	log := cfg.Log
	if log == nil {
		log = func(string, map[string]interface{}) {}
	}
	if secret == "" {
		log("warn", map[string]interface{}{
			"msg": "LPBS_SERVICE_SECRET is not set; deploy-readiness checks will fail until the secret is configured",
		})
	}
	return &HTTPLPBSReleaseClient{
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		baseURL:       strings.TrimRight(base, "/"),
		serviceSecret: secret,
		log:           log,
	}, nil
}

// CheckDeployReadiness calls POST /api/v1/deploy-readiness with service auth.
func (c *HTTPLPBSReleaseClient) CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/deploy-readiness", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceSecret != "" {
		// LPBS requireAdminOrService accepts a Bearer service token; same
		// secret pattern S2D uses for inter-scenario calls.
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceSecret)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result LPBSReadinessResult
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &result)
	}
	if resp.StatusCode == http.StatusOK {
		result.Ready = true
		return &result, nil
	}
	if result.Error == "" {
		result.Error = fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))
	}
	result.Ready = false
	return &result, nil
}

// Verify calls GET /api/v1/updates/{app_key}/verify with query params.
func (c *HTTPLPBSReleaseClient) Verify(ctx context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error) {
	if req.AppKey == "" {
		return nil, fmt.Errorf("app_key is required")
	}
	q := url.Values{}
	if req.Channel != "" {
		q.Set("channel", req.Channel)
	}
	if req.Platform != "" {
		q.Set("platform", req.Platform)
	}
	if req.ExpectedVersion != "" {
		q.Set("expected_version", req.ExpectedVersion)
	}
	if req.Deep {
		q.Set("deep", "true")
	}

	endpoint := fmt.Sprintf("%s/api/v1/updates/%s/verify?%s",
		c.baseURL, url.PathEscape(req.AppKey), q.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &LPBSVerifyResult{
			AppKey:          req.AppKey,
			Channel:         req.Channel,
			Platform:        req.Platform,
			ExpectedVersion: req.ExpectedVersion,
			Match:           false,
			Error:           fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}
	var result LPBSVerifyResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &result, nil
}
