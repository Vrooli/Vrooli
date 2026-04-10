package deployments

import (
	"testing"
)

func TestExtractDeployResult(t *testing.T) {
	t.Run("extracts artifacts from deploy stage details", func(t *testing.T) {
		// Simulate what JSON deserialization produces: map[string]interface{} for Details
		status := &PipelineStatus{
			Stages: map[string]*PipelineStageResult{
				"deploy": {
					Details: map[string]interface{}{
						"artifacts": []interface{}{
							map[string]interface{}{
								"artifact_id": float64(42),
								"platform":    "windows",
							},
							map[string]interface{}{
								"artifact_id": float64(43),
								"platform":    "linux",
							},
						},
						"update_url": "https://example.com/updates",
					},
				},
			},
		}

		result, err := ExtractDeployResult(status)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Artifacts) != 2 {
			t.Fatalf("expected 2 artifacts, got %d", len(result.Artifacts))
		}
		if result.Artifacts[0].ArtifactID != 42 {
			t.Errorf("expected artifact_id 42, got %d", result.Artifacts[0].ArtifactID)
		}
		if result.Artifacts[0].Platform != "windows" {
			t.Errorf("expected platform windows, got %s", result.Artifacts[0].Platform)
		}
		if result.Artifacts[1].ArtifactID != 43 {
			t.Errorf("expected artifact_id 43, got %d", result.Artifacts[1].ArtifactID)
		}
		if result.UpdateURL != "https://example.com/updates" {
			t.Errorf("expected update_url, got %s", result.UpdateURL)
		}
	})

	t.Run("returns error when deploy stage missing", func(t *testing.T) {
		status := &PipelineStatus{
			Stages: map[string]*PipelineStageResult{},
		}
		_, err := ExtractDeployResult(status)
		if err == nil {
			t.Fatal("expected error for missing deploy stage")
		}
	})

	t.Run("returns error when details nil", func(t *testing.T) {
		status := &PipelineStatus{
			Stages: map[string]*PipelineStageResult{
				"deploy": {Details: nil},
			},
		}
		_, err := ExtractDeployResult(status)
		if err == nil {
			t.Fatal("expected error for nil details")
		}
	})

	t.Run("returns error for nil status", func(t *testing.T) {
		_, err := ExtractDeployResult(nil)
		if err == nil {
			t.Fatal("expected error for nil status")
		}
	})
}

func TestExtractProvenance(t *testing.T) {
	t.Run("extracts provenance from status", func(t *testing.T) {
		status := &PipelineStatus{
			Provenance: &PipelineBuildProvenance{
				Version:       "1.2.3",
				GitCommitHash: "abc123def456",
			},
		}

		prov, err := ExtractProvenance(status)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prov.Version != "1.2.3" {
			t.Errorf("expected version 1.2.3, got %s", prov.Version)
		}
		if prov.GitCommitHash != "abc123def456" {
			t.Errorf("expected git hash abc123def456, got %s", prov.GitCommitHash)
		}
	})

	t.Run("returns error when provenance nil", func(t *testing.T) {
		status := &PipelineStatus{}
		_, err := ExtractProvenance(status)
		if err == nil {
			t.Fatal("expected error for nil provenance")
		}
	})

	t.Run("returns error for nil status", func(t *testing.T) {
		_, err := ExtractProvenance(nil)
		if err == nil {
			t.Fatal("expected error for nil status")
		}
	})
}
