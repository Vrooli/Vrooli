package control

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalizeReviewRecordingRetainsProducerOwnedReviewCopy(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for the recording producer test")
	}
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	chapterPath := filepath.Join(t.TempDir(), "chapter.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", chapterPath).Run())
	chapter, err := os.ReadFile(chapterPath)
	require.NoError(t, err)
	require.NoError(t, svc.persistArtifact(context.Background(), "chapter-1", chapter, "video"))
	require.NoError(t, svc.persistArtifact(context.Background(), "chapter-2", chapter, "video"))

	result, err := svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"chapter-1", "chapter-2"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Reference.ID)
	require.True(t, result.Reference.RedactionVerified)
	require.True(t, result.Reference.ContentVerified)
	require.Equal(t, "chapter-concat", result.Reference.RecordingMethod)
	require.True(t, filepath.IsAbs(result.Path))
	_, err = os.Stat(result.Path)
	require.NoError(t, err)
	retained, kind, err := svc.Artifact(result.Reference.ID)
	require.NoError(t, err)
	require.Equal(t, "video", kind)
	require.NotEmpty(t, retained)
}

func TestFinalizeReviewRecordingRejectsEmptyChapterSet(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	_, err = svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, nil)
	require.ErrorContains(t, err, "at least one chapter")
}
