//go:build !audio_lab

package bootstrap

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	trustfloor "audio-tools/internal/qualification"
	"audio-tools/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func TestProductExperimentProjectionsDoNotInitializeLabArtifacts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	db, _, err := OpenDB(context.Background(), Env{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	subsystem := buildExperimentSubsystem(ctx, Env{EnableLabAPI: false}, db, &mocks.FakeLogger{}, nil, nil, nil, nil)
	require.Nil(t, subsystem.labRuntime, "product build must not construct lab service or worker dependencies")
	require.NotNil(t, subsystem.corpusCounter)
	require.NotNil(t, subsystem.experimentCounter)
	require.NotNil(t, subsystem.qualificationEvidence)

	now := time.Now().UTC()
	counts, err := subsystem.experimentCounter.CountCreatedInRange(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, counts.Experiments)
	require.Zero(t, counts.Runs)
	clipCount, err := subsystem.corpusCounter.CountClipsCreatedInWindow(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, clipCount)
	evidence, err := subsystem.qualificationEvidence.ListQualificationEvidence(ctx, trustfloor.QualificationEvidenceFilter{})
	require.NoError(t, err)
	require.Empty(t, evidence)

	var artifactDirs []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == "corpus-blobs" || entry.Name() == "experiment-blobs") {
			artifactDirs = append(artifactDirs, path)
		}
		return nil
	}))
	require.Empty(t, artifactDirs, "product boot must not create lab artifact stores")
}
