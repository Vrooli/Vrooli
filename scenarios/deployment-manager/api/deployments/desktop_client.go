// Package deployments provides deployment orchestration for bundled desktop apps.
package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"deployment-manager/shared"
)

// DesktopPackagerClient handles communication with scenario-to-desktop.
type DesktopPackagerClient struct {
	httpClient *http.Client
	baseURL    string
	log        func(string, map[string]interface{})
}

// NewDesktopPackagerClient creates a client for the scenario-to-desktop API.
func NewDesktopPackagerClient(log func(string, map[string]interface{})) (*DesktopPackagerClient, error) {
	baseURL, err := shared.GetConfigResolver().ResolveDesktopPackagerURL()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scenario-to-desktop URL: %w", err)
	}

	return &DesktopPackagerClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for builds
		},
		baseURL: baseURL,
		log:     log,
	}, nil
}

// QuickGenerateRequest is the request for quick desktop generation.
type QuickGenerateRequest struct {
	ScenarioName       string   `json:"scenario_name"`
	TemplateType       string   `json:"template_type,omitempty"`
	DeploymentMode     string   `json:"deployment_mode"`
	BundleManifestPath string   `json:"bundle_manifest_path,omitempty"`
	Platforms          []string `json:"platforms,omitempty"`
}

// QuickGenerateResponse is the response from quick desktop generation.
type QuickGenerateResponse struct {
	BuildID      string                 `json:"build_id"`
	Status       string                 `json:"status"`
	ScenarioName string                 `json:"scenario_name"`
	DesktopPath  string                 `json:"desktop_path"`
	StatusURL    string                 `json:"status_url"`
	Metadata     map[string]interface{} `json:"detected_metadata,omitempty"`
}

// BuildStatusResponse is the response from build status polling.
type BuildStatusResponse struct {
	BuildID         string                     `json:"build_id"`
	ScenarioName    string                     `json:"scenario_name"`
	Status          string                     `json:"status"` // building, ready, failed
	OutputPath      string                     `json:"output_path"`
	Platforms       []string                   `json:"platforms"`
	PlatformResults map[string]*PlatformResult `json:"platform_results,omitempty"`
	Artifacts       map[string]string          `json:"artifacts,omitempty"`
	BuildLog        []string                   `json:"build_log,omitempty"`
	ErrorLog        []string                   `json:"error_log,omitempty"`
	Error           string                     `json:"error,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	CompletedAt     *time.Time                 `json:"completed_at,omitempty"`
}

// PlatformResult represents the build result for a single platform.
type PlatformResult struct {
	Platform     string `json:"platform"`
	Status       string `json:"status"` // pending, building, success, failed
	ArtifactPath string `json:"artifact_path,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ScenarioBuildRequest is the request for building desktop installers.
type ScenarioBuildRequest struct {
	Platforms []string `json:"platforms,omitempty"`
	Clean     bool     `json:"clean,omitempty"`
}

// ScenarioBuildResponse is the response from starting a scenario build.
type ScenarioBuildResponse struct {
	BuildID     string   `json:"build_id"`
	Status      string   `json:"status"`
	Scenario    string   `json:"scenario"`
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	StatusURL   string   `json:"status_url"`
}

// QuickGenerate triggers desktop wrapper generation for a scenario.
func (c *DesktopPackagerClient) QuickGenerate(ctx context.Context, req *QuickGenerateRequest) (*QuickGenerateResponse, error) {
	c.log("info", map[string]interface{}{
		"msg":             "calling scenario-to-desktop quick generate",
		"scenario":        req.ScenarioName,
		"deployment_mode": req.DeploymentMode,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/desktop/generate/quick", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scenario-to-desktop returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result QuickGenerateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.log("info", map[string]interface{}{
		"msg":          "quick generate initiated",
		"build_id":     result.BuildID,
		"desktop_path": result.DesktopPath,
	})

	return &result, nil
}

// GetBuildStatus polls for the status of a build.
func (c *DesktopPackagerClient) GetBuildStatus(ctx context.Context, buildID string) (*BuildStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/desktop/status/"+buildID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scenario-to-desktop returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result BuildStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// WaitForBuild polls until the build completes or times out.
func (c *DesktopPackagerClient) WaitForBuild(ctx context.Context, buildID string, pollInterval time.Duration) (*BuildStatusResponse, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetBuildStatus(ctx, buildID)
			if err != nil {
				c.log("warn", map[string]interface{}{
					"msg":      "build status poll failed",
					"build_id": buildID,
					"error":    err.Error(),
				})
				continue
			}

			switch status.Status {
			case "ready", "completed", "success":
				return status, nil
			case "failed":
				return status, fmt.Errorf("build failed: %s", status.Error)
			}
			// Still building, continue polling
		}
	}
}

// BuildScenario triggers building of desktop installers for a scenario.
func (c *DesktopPackagerClient) BuildScenario(ctx context.Context, scenarioName string, req *ScenarioBuildRequest) (*ScenarioBuildResponse, error) {
	c.log("info", map[string]interface{}{
		"msg":       "calling scenario-to-desktop build",
		"scenario":  scenarioName,
		"platforms": req.Platforms,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/desktop/build/"+scenarioName, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scenario-to-desktop returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ScenarioBuildResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.log("info", map[string]interface{}{
		"msg":      "scenario build initiated",
		"build_id": result.BuildID,
	})

	return &result, nil
}

// IsAvailable checks if scenario-to-desktop is reachable.
func (c *DesktopPackagerClient) IsAvailable(ctx context.Context) bool {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// SigningReadinessResponse is the response from the signing readiness check.
type SigningReadinessResponse struct {
	Ready     bool                      `json:"ready"`
	Scenario  string                    `json:"scenario"`
	Issues    []string                  `json:"issues,omitempty"`
	Platforms map[string]PlatformStatus `json:"platforms,omitempty"`
}

// PlatformStatus represents the signing status for a single platform.
type PlatformStatus struct {
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// CheckSigningReadiness checks if signing is configured and ready for a scenario.
// This is a non-blocking check - returns the status without failing.
func (c *DesktopPackagerClient) CheckSigningReadiness(ctx context.Context, scenarioName string) (*SigningReadinessResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/signing/"+scenarioName+"/ready", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 404 means no signing config exists - that's fine, just not ready
	if resp.StatusCode == http.StatusNotFound {
		return &SigningReadinessResponse{
			Ready:    false,
			Scenario: scenarioName,
			Issues:   []string{"No signing configuration exists for this scenario"},
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SigningReadinessResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// SmokeTestRequest is the request for running a smoke test with optional recording.
type SmokeTestRequest struct {
	ScenarioName string                  `json:"scenario_name"`
	ArtifactPath string                  `json:"artifact_path"`
	Platform     string                  `json:"platform"`
	Recording    *ScreenRecordingConfig  `json:"recording,omitempty"`
}

// ScreenRecordingConfig controls screen capture during smoke tests.
type ScreenRecordingConfig struct {
	Enabled       bool `json:"enabled"`
	DisplayWidth  int  `json:"display_width,omitempty"`
	DisplayHeight int  `json:"display_height,omitempty"`
	FPS           int  `json:"fps,omitempty"`
}

// SmokeTestStatusResponse is the status of a smoke test run.
type SmokeTestStatusResponse struct {
	SmokeTestID     string                  `json:"smoke_test_id"`
	Status          string                  `json:"status"`
	ScreenRecording *ScreenRecordingResult  `json:"screen_recording,omitempty"`
}

// ScreenRecordingResult holds the outcome of a smoke test recording.
type ScreenRecordingResult struct {
	Recorded      bool   `json:"recorded"`
	VideoPath     string `json:"video_path,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
	FileSizeBytes int64  `json:"file_size_bytes,omitempty"`
	Error         string `json:"error,omitempty"`
}

// RunSmokeTest triggers a smoke test with optional recording on scenario-to-desktop.
func (c *DesktopPackagerClient) RunSmokeTest(ctx context.Context, req *SmokeTestRequest) (*SmokeTestStatusResponse, error) {
	c.log("info", map[string]interface{}{
		"msg":      "running smoke test with recording",
		"scenario": req.ScenarioName,
		"platform": req.Platform,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/pipeline/run", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("smoke test API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SmokeTestStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetSmokeTestStatus polls the status of a running smoke test.
func (c *DesktopPackagerClient) GetSmokeTestStatus(ctx context.Context, smokeTestID string) (*SmokeTestStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/smoketest/"+smokeTestID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("smoke test status returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SmokeTestStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// WaitForSmokeTest polls until a smoke test completes or times out.
func (c *DesktopPackagerClient) WaitForSmokeTest(ctx context.Context, smokeTestID string, pollInterval time.Duration) (*SmokeTestStatusResponse, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetSmokeTestStatus(ctx, smokeTestID)
			if err != nil {
				c.log("warn", map[string]interface{}{
					"msg":           "smoke test poll failed",
					"smoke_test_id": smokeTestID,
					"error":         err.Error(),
				})
				continue
			}

			switch status.Status {
			case "passed", "failed":
				return status, nil
			}
		}
	}
}

// DownloadVideo downloads a smoke test video to a local path.
func (c *DesktopPackagerClient) DownloadVideo(ctx context.Context, smokeTestID, destPath string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/smoketest/"+smokeTestID+"/video", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("video download returned %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write video: %w", err)
	}

	return nil
}

// SetSigningConfig sets the signing configuration for a scenario via scenario-to-desktop.
// The config is a map containing platform-specific signing settings.
func (c *DesktopPackagerClient) SetSigningConfig(ctx context.Context, scenarioName string, config map[string]interface{}) error {
	c.log("info", map[string]interface{}{
		"msg":      "applying signing configuration",
		"scenario": scenarioName,
	})

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+"/api/v1/signing/"+scenarioName, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("signing API returned %d: %s", resp.StatusCode, string(respBody))
	}

	c.log("info", map[string]interface{}{
		"msg":      "signing configuration applied successfully",
		"scenario": scenarioName,
	})

	return nil
}
