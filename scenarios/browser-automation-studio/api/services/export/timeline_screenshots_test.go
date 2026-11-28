package export

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestTimelineLoaderScreenshotsUsesCanonicalTimelineAndLegacyFallback(t *testing.T) {
	t.Run("canonical timeline wins", func(t *testing.T) {
		dir := t.TempDir()
		resultPath := filepath.Join(dir, "result.json")
		require.NoError(t, os.WriteFile(resultPath, []byte(`{"entries":[{"step_index":1,"node_id":"00000000-0000-0000-0000-000000000001","telemetry":{"screenshot":{"artifact_id":"legacy-shot"}}}]}`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "timeline.proto.json"), []byte(`{"entries":[{"step_index":2,"node_id":"00000000-0000-0000-0000-000000000002","action":{"metadata":{"label":"Current"}},"telemetry":{"screenshot":{"artifact_id":"current-shot","content_type":"image/png"}}}]}`), 0o600))

		executionID := uuid.New()
		loader := NewTimelineLoader(timelineLoaderTestRepository{execution: &database.ExecutionIndex{
			ID: executionID, WorkflowID: uuid.New(), Status: database.ExecutionStatusCompleted, ResultPath: resultPath,
		}})
		got, err := loader.LoadExecutionScreenshots(context.Background(), executionID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "current-shot", got[0].GetScreenshot().GetArtifactId())
		require.Equal(t, int32(2), got[0].GetStepIndex())
		require.Equal(t, "Current", got[0].GetStepLabel())
	})

	t.Run("legacy result timeline remains readable", func(t *testing.T) {
		dir := t.TempDir()
		resultPath := filepath.Join(dir, "result.json")
		require.NoError(t, os.WriteFile(resultPath, []byte(`{"entries":[{"step_index":3,"node_id":"00000000-0000-0000-0000-000000000003","telemetry":{"screenshot":{"artifact_id":"legacy-shot"}}}]}`), 0o600))
		executionID := uuid.New()
		loader := NewTimelineLoader(timelineLoaderTestRepository{execution: &database.ExecutionIndex{
			ID: executionID, WorkflowID: uuid.New(), Status: database.ExecutionStatusCompleted, ResultPath: resultPath,
		}})
		got, err := loader.LoadExecutionScreenshots(context.Background(), executionID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "legacy-shot", got[0].GetScreenshot().GetArtifactId())
	})
}
