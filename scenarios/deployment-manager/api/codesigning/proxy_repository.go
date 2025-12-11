// Package codesigning provides code signing configuration, validation,
// and generation for desktop bundle deployments.
package codesigning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ScenarioLookup provides the ability to get scenario name from profile ID.
type ScenarioLookup interface {
	// GetScenarioAndTier retrieves the scenario name for a profile.
	GetScenarioAndTier(ctx context.Context, idOrName string) (scenario string, tierCount int, err error)
}

// ProxyRepository implements Repository by proxying calls to scenario-to-desktop's signing API.
// This allows deployment-manager to delegate signing configuration storage to the
// scenario-to-desktop service where signing execution actually happens.
type ProxyRepository struct {
	scenarioLookup ScenarioLookup
	baseURL        string
	httpClient     *http.Client
}

// ProxyRepositoryOption configures a ProxyRepository.
type ProxyRepositoryOption func(*ProxyRepository)

// WithHTTPClient sets a custom HTTP client for the proxy.
func WithHTTPClient(client *http.Client) ProxyRepositoryOption {
	return func(r *ProxyRepository) {
		r.httpClient = client
	}
}

// WithBaseURL sets the base URL for scenario-to-desktop API.
func WithBaseURL(url string) ProxyRepositoryOption {
	return func(r *ProxyRepository) {
		r.baseURL = url
	}
}

// NewProxyRepository creates a new proxy repository.
// The scenarioLookup is used to translate profile IDs to scenario names.
func NewProxyRepository(scenarioLookup ScenarioLookup, opts ...ProxyRepositoryOption) *ProxyRepository {
	r := &ProxyRepository{
		scenarioLookup: scenarioLookup,
		baseURL:        getDefaultSigningAPIURL(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// getDefaultSigningAPIURL returns the default URL for the scenario-to-desktop signing API.
func getDefaultSigningAPIURL() string {
	// Check for explicit environment variable first
	if url := os.Getenv("SCENARIO_TO_DESKTOP_URL"); url != "" {
		return url
	}

	// Default to localhost with standard port
	port := os.Getenv("SCENARIO_TO_DESKTOP_PORT")
	if port == "" {
		port = "7800" // Default scenario-to-desktop API port
	}

	return fmt.Sprintf("http://localhost:%s", port)
}

// getScenarioName looks up the scenario name for a profile ID.
func (r *ProxyRepository) getScenarioName(ctx context.Context, profileID string) (string, error) {
	scenario, _, err := r.scenarioLookup.GetScenarioAndTier(ctx, profileID)
	if err != nil {
		return "", fmt.Errorf("failed to look up scenario for profile %s: %w", profileID, err)
	}
	if scenario == "" {
		return "", fmt.Errorf("profile %s has no scenario configured", profileID)
	}
	return scenario, nil
}

// signingConfigResponse is the response format from scenario-to-desktop's signing API.
type signingConfigResponse struct {
	Scenario   string         `json:"scenario"`
	Config     *SigningConfig `json:"config"`
	ConfigPath string         `json:"config_path"`
}

// Get retrieves the signing config for a profile by calling scenario-to-desktop's API.
func (r *ProxyRepository) Get(ctx context.Context, profileID string) (*SigningConfig, error) {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		// Translate scenario lookup errors to profile not found
		return nil, ErrProfileNotFound
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s", r.baseURL, scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		// No signing config exists for this scenario
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result signingConfigResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result.Config, nil
}

// Save stores the signing config for a profile by calling scenario-to-desktop's API.
func (r *ProxyRepository) Save(ctx context.Context, profileID string, config *SigningConfig) error {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("serialize config: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s", r.baseURL, scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(configJSON))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete removes the signing config for a profile.
func (r *ProxyRepository) Delete(ctx context.Context, profileID string) error {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s", r.baseURL, scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetForPlatform retrieves config for a specific platform only.
func (r *ProxyRepository) GetForPlatform(ctx context.Context, profileID string, platform string) (interface{}, error) {
	config, err := r.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	switch platform {
	case PlatformWindows:
		return config.Windows, nil
	case PlatformMacOS:
		return config.MacOS, nil
	case PlatformLinux:
		return config.Linux, nil
	default:
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
}

// SaveForPlatform updates only a specific platform's config.
func (r *ProxyRepository) SaveForPlatform(ctx context.Context, profileID string, platform string, platformConfig interface{}) error {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	configJSON, err := json.Marshal(platformConfig)
	if err != nil {
		return fmt.Errorf("serialize config: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s/%s", r.baseURL, scenario, platform)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(configJSON))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteForPlatform removes the signing config for a specific platform.
func (r *ProxyRepository) DeleteForPlatform(ctx context.Context, profileID string, platform string) error {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s/%s", r.baseURL, scenario, platform)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// --- Validation proxying ---

// Validate calls scenario-to-desktop's validation endpoint.
func (r *ProxyRepository) Validate(ctx context.Context, profileID string) (*ValidationResult, error) {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return nil, ErrProfileNotFound
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s/validate", r.baseURL, scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ValidationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

// CheckReady calls scenario-to-desktop's readiness endpoint.
func (r *ProxyRepository) CheckReady(ctx context.Context, profileID string) (bool, map[string]interface{}, error) {
	scenario, err := r.getScenarioName(ctx, profileID)
	if err != nil {
		return false, nil, ErrProfileNotFound
	}

	url := fmt.Sprintf("%s/api/v1/signing/%s/ready", r.baseURL, scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, nil, fmt.Errorf("parse response: %w", err)
	}

	ready, _ := result["ready"].(bool)
	return ready, result, nil
}

// DetectPrerequisites calls scenario-to-desktop's prerequisites endpoint.
func (r *ProxyRepository) DetectPrerequisites(ctx context.Context) ([]ToolDetectionResult, error) {
	url := fmt.Sprintf("%s/api/v1/signing/prerequisites", r.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tools []ToolDetectionResult `json:"tools"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result.Tools, nil
}

// DiscoverCertificates calls scenario-to-desktop's certificate discovery endpoint.
func (r *ProxyRepository) DiscoverCertificates(ctx context.Context, platform string) ([]DiscoveredCertificate, error) {
	url := fmt.Sprintf("%s/api/v1/signing/discover/%s", r.baseURL, platform)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call signing API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Certificates []DiscoveredCertificate `json:"certificates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result.Certificates, nil
}
