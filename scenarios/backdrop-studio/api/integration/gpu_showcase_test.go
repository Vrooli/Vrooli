//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"backdrop-studio/integration"

	"github.com/stretchr/testify/require"
)

// TestGPUModelBackedShowcase is an explicit live-host assertion. It is kept
// out of the normal broad integration lane because it consumes an admitted GPU
// job; gpu-showcase opts in after the operator has checked capacity.
func TestGPUModelBackedShowcase(t *testing.T) {
	if os.Getenv("BACKDROP_STUDIO_REQUIRE_GPU") != "1" {
		t.Skip("set BACKDROP_STUDIO_REQUIRE_GPU=1 after GPU capacity admission")
	}
	env, caps := newEnvironment(t)
	require.True(t, caps.GenerationOK, "image-tools must have an installed model-backed generation operation")
	styles, err := env.Styles(context.Background())
	require.NoError(t, err)
	var selected *integration.Style
	for i := range styles {
		if styles[i].ID == "synth-celestial" && styles[i].ModelBacked() {
			selected = &styles[i]
			break
		}
	}
	require.NotNil(t, selected, "the showcase needs the stable local model-backed style synth-celestial")

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	job, err := env.Submit(ctx, integration.SubmitOptions{StyleID: selected.ID, Seed: laneSeed, RequireGPU: true})
	finished := time.Now()
	elapsed := finished.Sub(started)
	require.NoError(t, err)
	require.Greater(t, elapsed, time.Duration(0))
	require.Less(t, elapsed, time.Minute, "a measured-seconds GPU render must finish within 60 seconds")
	require.Equal(t, "completed", job.Status)
	require.NotEmpty(t, job.Candidates)
	candidate := job.Candidates[0]
	var provenance map[string]any
	require.NoError(t, json.Unmarshal([]byte(candidate.ProvenanceJSON), &provenance))
	require.Equal(t, "local-gpu", provenance["generation_tier"], "a CPU fallback must not count as the GPU showcase")
	require.NotEmpty(t, provenance["generation_model"], "the model identity must be recorded")
	require.NotEmpty(t, provenance["image_tools_job_id"], "the exact image-tools admission receipt must be traceable")

	id, ok := provenance["image_tools_job_id"].(string)
	require.True(t, ok, "image-tools job ID must be a string")
	imageTools, err := env.ImageToolsJob(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "JOB_STATE_SUCCEEDED", imageTools.State)
	require.Equal(t, "JOB_LANE_GPU", imageTools.Lane)
	require.NotEmpty(t, imageTools.ResultMeta["gpu_capacity_claim_id"])
	require.NotEmpty(t, imageTools.ResultMeta["gpu_uuid"])
	require.NotEmpty(t, imageTools.ResultMeta["disk_job_reservation_id"])
	diskPeak, parseErr := strconv.ParseInt(imageTools.ResultMeta["disk_job_filesystem_0_peak_bytes"], 10, 64)
	require.NoError(t, parseErr)
	require.Greater(t, diskPeak, int64(0))

	receipt := map[string]any{
		"style_id": selected.ID, "job_id": job.ID, "elapsed_seconds": elapsed.Seconds(),
		"backdrop_started_at":  started.UTC().Format(time.RFC3339Nano),
		"backdrop_finished_at": finished.UTC().Format(time.RFC3339Nano),
		"generation_tier":      provenance["generation_tier"], "model": provenance["generation_model"],
		"image_tools_job_id": id, "gpu_claim_id": imageTools.ResultMeta["gpu_capacity_claim_id"],
		"gpu_uuid": imageTools.ResultMeta["gpu_uuid"], "image_tools_started_at": imageTools.StartedAt,
		"image_tools_finished_at": imageTools.FinishedAt, "disk_job_reservation_id": imageTools.ResultMeta["disk_job_reservation_id"],
		"disk_job_peak_bytes": diskPeak,
	}
	if out := os.Getenv("BACKDROP_STUDIO_GPU_SHOWCASE_OUT"); out != "" {
		require.NoError(t, os.MkdirAll(out, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(out, "backdrop.png"), candidate.ImagePNG, 0o644))
		encoded, marshalErr := json.MarshalIndent(receipt, "", "  ")
		require.NoError(t, marshalErr)
		require.NoError(t, os.WriteFile(filepath.Join(out, "receipt.json"), append(encoded, '\n'), 0o644))
	}
	t.Logf("Backdrop Studio GPU showcase: style=%s model=%v tier=%v elapsed=%.2fs job=%s", selected.ID, provenance["generation_model"], provenance["generation_tier"], elapsed.Seconds(), job.ID)
}
