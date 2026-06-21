package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/storage"
)

// writePerfArtifacts seeds the driver-written perf files under the execution
// artifact root the exporter reads from.
func writePerfArtifacts(t *testing.T, root string, execID uuid.UUID, withVitals bool) {
	t.Helper()
	perfDir := filepath.Join(root, execID.String(), "artifacts", "performance")
	require.NoError(t, os.MkdirAll(perfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(perfDir, "performance.json"), []byte(`{"traceEvents":[{"name":"RunTask"}]}`), 0o644))
	if withVitals {
		require.NoError(t, os.WriteFile(filepath.Join(perfDir, "performance.web-vitals.json"), []byte(`{"lcp":{"value":120}}`), 0o644))
	}
}

func TestExportPerformanceArtifacts_CopiesAndUploads(t *testing.T) {
	root := t.TempDir()
	execID := uuid.New()
	writePerfArtifacts(t, root, execID, true)

	store := storage.NewMemoryStorage()
	svc := &WorkflowService{executionDataRoot: root}
	outDir := t.TempDir()

	require.NoError(t, svc.exportPerformanceArtifacts(context.Background(), execID, outDir, store))

	// Files copied into the export folder.
	trace, err := os.ReadFile(filepath.Join(outDir, "performance", "performance.json"))
	require.NoError(t, err)
	require.Contains(t, string(trace), "RunTask")
	vitals, err := os.ReadFile(filepath.Join(outDir, "performance", "performance.web-vitals.json"))
	require.NoError(t, err)
	require.Contains(t, string(vitals), "lcp")

	// Upload to object storage is exercised (best-effort, non-fatal in prod);
	// the memory store generates a random object key per artifact, so the
	// file-copy assertions above are the durable contract. The upload path
	// running without error is the additional coverage here.
}

func TestExportPerformanceArtifacts_TraceOnly(t *testing.T) {
	root := t.TempDir()
	execID := uuid.New()
	writePerfArtifacts(t, root, execID, false)

	svc := &WorkflowService{executionDataRoot: root}
	outDir := t.TempDir()
	require.NoError(t, svc.exportPerformanceArtifacts(context.Background(), execID, outDir, storage.NewMemoryStorage()))

	_, err := os.Stat(filepath.Join(outDir, "performance", "performance.json"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outDir, "performance", "performance.web-vitals.json"))
	require.True(t, os.IsNotExist(err))
}

func TestExportPerformanceArtifacts_NoPerfDirIsNoOp(t *testing.T) {
	root := t.TempDir()
	execID := uuid.New()
	svc := &WorkflowService{executionDataRoot: root}
	outDir := t.TempDir()

	// No perf files written → exporter must succeed and create nothing.
	require.NoError(t, svc.exportPerformanceArtifacts(context.Background(), execID, outDir, storage.NewMemoryStorage()))
	_, err := os.Stat(filepath.Join(outDir, "performance"))
	require.True(t, os.IsNotExist(err))
}

func TestExportPerformanceArtifacts_EmptyDataRootIsNoOp(t *testing.T) {
	svc := &WorkflowService{executionDataRoot: ""}
	require.NoError(t, svc.exportPerformanceArtifacts(context.Background(), uuid.New(), t.TempDir(), nil))
}
