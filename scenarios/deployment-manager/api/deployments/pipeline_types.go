package deployments

import (
	"encoding/json"
	"fmt"
)

// Local mirrors of scenario-to-desktop pipeline types.
// Decoupled via JSON round-trip — do NOT import scenario-to-desktop directly.

// PipelineDeployResult mirrors the deploy stage's DeployResult.
type PipelineDeployResult struct {
	Artifacts []PipelineDeployArtifact `json:"artifacts,omitempty"`
	UpdateURL string                   `json:"update_url,omitempty"`
}

// PipelineDeployArtifact mirrors a single uploaded artifact result.
type PipelineDeployArtifact struct {
	ArtifactID int64  `json:"artifact_id"`
	Platform   string `json:"platform"`
}

// PipelineBuildProvenance mirrors the git/version state from the pipeline.
type PipelineBuildProvenance struct {
	Version       string `json:"version"`
	GitCommitHash string `json:"git_commit_hash"`
}

// PipelineStageResult mirrors the per-stage result from the pipeline status.
type PipelineStageResult struct {
	Details interface{} `json:"details,omitempty"`
}

// PipelineStatus mirrors the top-level pipeline status fields we need.
type PipelineStatus struct {
	CurrentState string                          `json:"current_state"`
	Stages       map[string]*PipelineStageResult `json:"stages"`
	Provenance   *PipelineBuildProvenance        `json:"provenance,omitempty"`
}

// ExtractDeployResult extracts DeployResult from pipeline status via JSON round-trip.
// The Details field is an interface{} that arrives as a map[string]interface{} from JSON
// deserialization, so we marshal it back to JSON and unmarshal into our typed struct.
func ExtractDeployResult(status *PipelineStatus) (*PipelineDeployResult, error) {
	if status == nil {
		return nil, fmt.Errorf("pipeline status is nil")
	}

	stage, ok := status.Stages["deploy"]
	if !ok || stage == nil {
		return nil, fmt.Errorf("deploy stage not found in pipeline status")
	}

	if stage.Details == nil {
		return nil, fmt.Errorf("deploy stage has no details")
	}

	data, err := json.Marshal(stage.Details)
	if err != nil {
		return nil, fmt.Errorf("marshal deploy details: %w", err)
	}

	var result PipelineDeployResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal deploy result: %w", err)
	}

	return &result, nil
}

// ExtractProvenance extracts BuildProvenance from pipeline status.
func ExtractProvenance(status *PipelineStatus) (*PipelineBuildProvenance, error) {
	if status == nil {
		return nil, fmt.Errorf("pipeline status is nil")
	}
	if status.Provenance == nil {
		return nil, fmt.Errorf("pipeline status has no provenance")
	}
	return status.Provenance, nil
}
