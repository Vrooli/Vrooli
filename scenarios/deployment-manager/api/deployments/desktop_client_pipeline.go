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
)

// SmokeTestRequest is the request for running a smoke test with optional recording.
type SmokeTestRequest struct {
	ScenarioName string                 `json:"scenario_name"`
	ArtifactPath string                 `json:"artifact_path"`
	Platform     string                 `json:"platform"`
	Recording    *ScreenRecordingConfig `json:"recording,omitempty"`
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
	SmokeTestID     string                 `json:"smoke_test_id"`
	Status          string                 `json:"status"`
	ScreenRecording *ScreenRecordingResult `json:"screen_recording,omitempty"`
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

// PublishPipelineRequest is the request body for triggering a deploy-only pipeline run.
type PublishPipelineRequest struct {
	ScenarioName    string               `json:"scenario_name"`
	Platforms       []string             `json:"platforms,omitempty"`
	DeployConfig    *PublishDeployConfig `json:"deploy,omitempty"`
	ResumeFromStage string               `json:"resume_from_stage,omitempty"`
	StopAfterStage  string               `json:"stop_after_stage,omitempty"`
	Publish         bool                 `json:"publish,omitempty"`
	// ReleaseID is the DM-owned release UUID forwarded to S2D so the LPBS
	// commit payload carries it on download_artifacts.release_id.
	ReleaseID string `json:"release_id,omitempty"`
	// Channel is the release channel; S2D maps it to LPBS variant_key on apply.
	Channel string `json:"channel,omitempty"`
}

// PublishDeployConfig mirrors scenario-to-desktop's DeployConfig for LPBS deployment.
type PublishDeployConfig struct {
	TargetName    string `json:"target_name,omitempty"`
	ScenarioName  string `json:"scenario_name,omitempty"`
	RemoteProfile string `json:"remote_profile,omitempty"`
	AppKey        string `json:"app_key"`
	UpdateURL     string `json:"update_url,omitempty"`
	// ReleaseID + Channel ride on the deploy config so S2D's pipeline Config
	// decoder forwards them to lpbs_client.go.
	ReleaseID string `json:"release_id,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// PublishPipelineResponse is the response from triggering a pipeline run.
type PublishPipelineResponse struct {
	PipelineID string `json:"pipeline_id"`
	StatusURL  string `json:"status_url"`
	Message    string `json:"message,omitempty"`
}

// RunPublishPipeline triggers a deploy-stage-only pipeline via scenario-to-desktop.
func (c *DesktopPackagerClient) RunPublishPipeline(ctx context.Context, req *PublishPipelineRequest) (*PublishPipelineResponse, error) {
	c.log("info", map[string]interface{}{
		"msg":      "triggering publish pipeline",
		"scenario": req.ScenarioName,
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
		return nil, fmt.Errorf("pipeline API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result PublishPipelineResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetPipelineStatus polls the status of a pipeline run.
func (c *DesktopPackagerClient) GetPipelineStatus(ctx context.Context, pipelineID string) (*PipelineStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/pipeline/"+pipelineID, nil)
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
		return nil, fmt.Errorf("pipeline status API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var status PipelineStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("decode pipeline status: %w", err)
	}

	return &status, nil
}

// WaitForPipeline polls pipeline status until it completes or fails.
func (c *DesktopPackagerClient) WaitForPipeline(ctx context.Context, pipelineID string) (*PipelineStatus, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetPipelineStatus(ctx, pipelineID)
			if err != nil {
				return nil, err
			}

			switch status.CurrentState {
			case "completed":
				return status, nil
			case "failed", "cancelled":
				return status, fmt.Errorf("pipeline %s ended with state: %s", pipelineID, status.CurrentState)
			}
		}
	}
}

// SetSigningConfig sets the signing configuration for a scenario via scenario-to-desktop.
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
