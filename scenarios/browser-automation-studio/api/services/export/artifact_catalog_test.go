package export

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

func TestArtifactCatalogProjectsSortedMetadataAndProtectsHAR(t *testing.T) {
	root := t.TempDir()
	executionID := uuid.New()
	for kind, files := range map[string]map[string]string{
		"videos": {"z.mp4": "video bytes", "a.mp4": "video bytes"},
		"traces": {"trace.zip": "trace bytes", "z-opaque": "plain artifact bytes"},
		"har":    {"capture.har": `{"log":{}}`},
	} {
		dir := filepath.Join(root, executionID.String(), "artifacts", kind)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		for name, contents := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600))
		}
	}
	catalog := NewArtifactCatalog(root)

	videos, err := catalog.List(context.Background(), executionID, "videos")
	require.NoError(t, err)
	require.Len(t, videos, 2)
	require.Equal(t, "a.mp4", videos[0].ArtifactID)
	require.Equal(t, "/api/v1/recordings/assets/"+executionID.String()+"/artifacts/videos/a.mp4", videos[0].StorageURL)
	require.Equal(t, int64(len("video bytes")), videos[0].SizeBytes)
	require.NotEmpty(t, videos[0].Payload["sha256"])
	require.NotContains(t, videos[0].Payload, "path")

	traces, err := catalog.List(context.Background(), executionID, "traces")
	require.NoError(t, err)
	require.Len(t, traces, 2)
	require.Equal(t, "/api/v1/recordings/assets/"+executionID.String()+"/artifacts/traces/trace.zip", traces[0].StorageURL)
	require.Equal(t, "text/plain; charset=utf-8", traces[1].ContentType, "unknown extensions use the storage owner's bounded sniff")

	har, err := catalog.List(context.Background(), executionID, "har")
	require.NoError(t, err)
	require.Len(t, har, 1)
	require.Empty(t, har[0].StorageURL)
	require.Equal(t, basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY.String(), har[0].Payload["access_policy"])
	require.NotEmpty(t, har[0].Payload["sha256"])

	_, err = catalog.List(context.Background(), executionID, "unknown")
	require.Error(t, err)
}

func TestArtifactCatalogInspectsExportability(t *testing.T) {
	root := t.TempDir()
	resultDir := filepath.Join(t.TempDir(), "run")
	resultPath := filepath.Join(resultDir, "result.json")
	executionID := uuid.New()
	catalog := NewArtifactCatalog(root)

	got, err := catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.Equal(t, ExecutionExportability{}, got)

	require.NoError(t, os.MkdirAll(filepath.Join(resultDir, "screenshots", "nested"), 0o755))
	got, err = catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.False(t, got.HasScreenshots, "nested directories do not count as screenshot files")
	require.NoError(t, os.WriteFile(filepath.Join(resultDir, "screenshots", "one.png"), []byte("png"), 0o600))
	got, err = catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.True(t, got.HasScreenshots)
	require.False(t, got.IsExportable, "screenshot presence alone preserves the existing exportability contract")

	require.NoError(t, os.WriteFile(filepath.Join(resultDir, "timeline.proto.json"), []byte("timeline"), 0o600))
	got, err = catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.True(t, got.HasTimeline)
	require.True(t, got.IsExportable)

	require.NoError(t, os.Remove(filepath.Join(resultDir, "timeline.proto.json")))
	videoDir := filepath.Join(root, executionID.String(), "artifacts", "videos")
	require.NoError(t, os.MkdirAll(filepath.Join(videoDir, "directory-only"), 0o755))
	got, err = catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.False(t, got.HasRecordedVideo, "directories do not count as recorded video")
	require.NoError(t, os.WriteFile(filepath.Join(videoDir, "run.webm"), []byte("video"), 0o600))
	got, err = catalog.InspectExportability(context.Background(), resultPath, executionID)
	require.NoError(t, err)
	require.True(t, got.HasRecordedVideo)
	require.True(t, got.IsExportable)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = catalog.InspectExportability(canceled, resultPath, executionID)
	require.ErrorIs(t, err, context.Canceled)
}
