package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteWorkflowPublishesChecksummedRedactedSummaryReference(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	writer := NewWriter(root, "run-1")

	artifact, err := writer.WriteWorkflow("scenario:bas/cases/smoke.json", "bas/cases/smoke.json", nil, WorkflowLatest{
		RunID:     "run-1",
		AssetID:   "scenario:bas/cases/smoke.json",
		AssetPath: "bas/cases/smoke.json",
		Status:    "completed",
		Success:   true,
	})
	require.NoError(t, err)
	require.Len(t, artifact.References, 1)
	reference := artifact.References[0]
	require.Equal(t, "scenario:bas/cases/smoke.json:latest", reference.ID)
	require.Equal(t, "workflow-summary", reference.Kind)
	require.Equal(t, "application/json", reference.MediaType)
	require.True(t, reference.Redacted)
	require.NotEmpty(t, reference.Checksum)

	data, err := os.ReadFile(filepath.FromSlash(artifact.Latest))
	require.NoError(t, err)
	hash := sha256.Sum256(data)
	require.Equal(t, "sha256:"+hex.EncodeToString(hash[:]), reference.Checksum)
}
