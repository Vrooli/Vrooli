package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

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
	Channel                 string            `json:"channel,omitempty"`
	ReleaseVersion          string            `json:"release_version,omitempty"`
	ArtifactDigest          string            `json:"artifact_digest,omitempty"`
	ExpectedArtifactDigests map[string]string `json:"expected_artifact_digests,omitempty"`
	CandidateID             string            `json:"candidate_id,omitempty"`
	DestinationRevisionID   string            `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch      uint64            `json:"authorization_epoch,omitempty"`
	ReadinessReviewKey      string            `json:"readiness_review_key,omitempty"`
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
	ReleaseID               string            `json:"release_id,omitempty"`
	Channel                 string            `json:"channel,omitempty"`
	ArtifactDigest          string            `json:"artifact_digest,omitempty"`
	ExpectedArtifactDigests map[string]string `json:"expected_artifact_digests,omitempty"`
	CandidateID             string            `json:"candidate_id,omitempty"`
	DestinationRevisionID   string            `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch      uint64            `json:"authorization_epoch,omitempty"`
	ReadinessReviewKey      string            `json:"readiness_review_key,omitempty"`
}

// PublishPipelineResponse is the response from triggering a pipeline run.
type PublishPipelineResponse struct {
	PipelineID string `json:"pipeline_id"`
	StatusURL  string `json:"status_url"`
	Message    string `json:"message,omitempty"`
}

// RunPublishPipelineConnect starts the canonical owner pipeline. The legacy
// JSON endpoint is retired; release publication must use the typed service.
func (c *DesktopPackagerClient) RunPublishPipelineConnect(ctx context.Context, req *PublishPipelineRequest) (*PublishPipelineResponse, error) {
	if req == nil || strings.TrimSpace(req.ScenarioName) == "" {
		return nil, fmt.Errorf("publish pipeline request requires a scenario")
	}
	if req.DeployConfig == nil {
		return nil, fmt.Errorf("publish pipeline requires deploy config")
	}
	deploy := req.DeployConfig
	config := &pipelinev1.PipelineConfig{
		ScenarioName:            req.ScenarioName,
		Platforms:               platformEnums(req.Platforms),
		PlatformTargets:         append([]string(nil), req.Platforms...),
		Stages:                  []sharedv1.StageName{sharedv1.StageName_STAGE_NAME_DEPLOY},
		Publish:                 boolPtr(true),
		Sign:                    boolPtr(true),
		ArtifactTrustMode:       stringPtr("production"),
		ExpectedArtifactDigests: copyStringMap(req.ExpectedArtifactDigests),
		ArtifactManifestDigest:  req.ArtifactDigest,
		Version:                 optionalStringPtr(req.ReleaseVersion),
		Deploy: &pipelinev1.DeployConfig{
			TargetName: deploy.TargetName, ScenarioName: deploy.ScenarioName,
			RemoteProfile: deploy.RemoteProfile, AppKey: deploy.AppKey,
			UpdateUrl: deploy.UpdateURL, ReleaseId: deploy.ReleaseID,
			Channel:     deploy.Channel,
			CandidateId: deploy.CandidateID, DestinationRevisionId: deploy.DestinationRevisionID,
			AuthorizationEpoch: deploy.AuthorizationEpoch, ReadinessReviewKey: deploy.ReadinessReviewKey,
		},
	}
	client := pipelineconnect.NewPipelineServiceClient(c.httpClient, c.baseURL)
	response, err := client.Run(ctx, connect.NewRequest(&pipelinev1.PipelineRunRequest{Config: config}))
	if err != nil {
		return nil, fmt.Errorf("run typed publish pipeline: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetPipelineId() == "" {
		return nil, fmt.Errorf("typed publish pipeline returned no operation id")
	}
	return &PublishPipelineResponse{PipelineID: response.Msg.GetPipelineId(), Message: response.Msg.GetMessage()}, nil
}

func (c *DesktopPackagerClient) WaitForPipelineConnect(ctx context.Context, pipelineID string) (*PipelineStatus, error) {
	client := pipelineconnect.NewPipelineServiceClient(c.httpClient, c.baseURL)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			response, err := client.Get(ctx, connect.NewRequest(&pipelinev1.PipelineGetRequest{PipelineId: pipelineID}))
			if err != nil {
				return nil, fmt.Errorf("get typed pipeline status: %w", err)
			}
			status := pipelineStatusFromProto(response.Msg)
			switch status.CurrentState {
			case "completed":
				return status, nil
			case "failed", "cancelled":
				return status, fmt.Errorf("pipeline %s ended with state: %s", pipelineID, status.CurrentState)
			}
		}
	}
}

func platformEnums(platforms []string) []sharedv1.Platform {
	result := make([]sharedv1.Platform, 0, len(platforms))
	for _, platform := range platforms {
		switch strings.ToLower(strings.SplitN(platform, "-", 2)[0]) {
		case "win", "windows":
			result = append(result, sharedv1.Platform_PLATFORM_WIN)
		case "mac", "darwin", "macos":
			result = append(result, sharedv1.Platform_PLATFORM_MAC)
		default:
			result = append(result, sharedv1.Platform_PLATFORM_LINUX)
		}
	}
	return result
}

func pipelineStatusFromProto(value *pipelinev1.PipelineStatus) *PipelineStatus {
	status := &PipelineStatus{CurrentState: value.GetCurrentState(), Stages: make(map[string]*PipelineStageResult)}
	for name, stage := range value.GetStages() {
		converted := &PipelineStageResult{}
		if details := stage.GetDetails(); details != nil && details.GetDeploy() != nil {
			deploy := details.GetDeploy()
			artifacts := make([]interface{}, 0, len(deploy.GetArtifacts()))
			for _, artifact := range deploy.GetArtifacts() {
				platform := artifact.GetTargetId()
				if platform == "" {
					platform = platformName(artifact.GetPlatform())
				}
				artifacts = append(artifacts, map[string]interface{}{
					"artifact_id": artifact.GetArtifactId(), "platform": platform,
					"sha512": artifact.GetSha512(), "destination_object": artifact.GetDestinationObject(),
				})
			}
			converted.Details = map[string]interface{}{"artifacts": artifacts, "update_url": deploy.GetUpdateUrl()}
		}
		status.Stages[name] = converted
	}
	return status
}

func platformName(value sharedv1.Platform) string {
	switch value {
	case sharedv1.Platform_PLATFORM_WIN:
		return "win"
	case sharedv1.Platform_PLATFORM_MAC:
		return "mac"
	default:
		return "linux"
	}
}

func boolPtr(value bool) *bool { return &value }

func stringPtr(value string) *string { return &value }

func optionalStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func copyStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
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
