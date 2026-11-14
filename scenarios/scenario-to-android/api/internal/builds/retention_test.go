package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestRetainedBuildSurvivesWorkspaceRemovalAndDetectsTampering(t *testing.T) {
	workspace, data := t.TempDir(), t.TempDir()
	apk, aab := []byte("signed APK bytes"), []byte("AAB bytes")
	digest := sha256.Sum256(append(append(append([]byte{}, apk...), 0), aab...))
	artifact := deliveryramp.Artifact{Checksum: "sha256:" + hex.EncodeToString(digest[:]), SizeBytes: int64(len(apk) + len(aab)), Metadata: map[string]string{"apk_path": filepath.Join(workspace, "output.apk"), "aab_path": filepath.Join(workspace, "output.aab"), "package_name": "com.example.daily"}}
	require.NoError(t, os.WriteFile(artifact.Metadata["apk_path"], apk, 0600))
	require.NoError(t, os.WriteFile(artifact.Metadata["aab_path"], aab, 0600))
	retained, err := RetainArtifact(context.Background(), data, artifact)
	require.NoError(t, err)
	require.NotEqual(t, artifact.Metadata["apk_path"], retained.LocalPath)
	require.Equal(t, filepath.Join(workspace, "output.apk"), artifact.Metadata["apk_path"], "retention must not mutate caller metadata")
	require.NoError(t, os.RemoveAll(workspace))
	replay, err := RetainArtifact(context.Background(), data, artifact)
	require.NoError(t, err)
	require.Equal(t, retained.LocalPath, replay.LocalPath)
	actual, err := os.ReadFile(retained.LocalPath)
	require.NoError(t, err)
	require.Equal(t, apk, actual)
	require.NoError(t, os.WriteFile(retained.LocalPath, []byte("corrupt APK"), 0600))
	_, err = RetainArtifact(context.Background(), data, artifact)
	require.ErrorContains(t, err, "mismatch")
}

func TestRetentionRejectsWrongDigestAndCancelledWorkWithoutPublishing(t *testing.T) {
	source, data := t.TempDir(), t.TempDir()
	apk, aab := filepath.Join(source, "app.apk"), filepath.Join(source, "app.aab")
	require.NoError(t, os.WriteFile(apk, []byte("apk"), 0600))
	require.NoError(t, os.WriteFile(aab, []byte("aab"), 0600))
	artifact := deliveryramp.Artifact{Checksum: "sha256:0000000000000000000000000000000000000000000000000000000000000000", SizeBytes: 6, Metadata: map[string]string{"apk_path": apk, "aab_path": aab}}
	_, err := RetainArtifact(context.Background(), data, artifact)
	require.ErrorContains(t, err, "mismatch")
	files, err := os.ReadDir(filepath.Join(data, "builds", "artifacts"))
	require.NoError(t, err)
	require.Empty(t, files)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RetainArtifact(ctx, data, artifact)
	require.ErrorIs(t, err, context.Canceled)
}
