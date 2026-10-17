package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
)

type batchFailFiles struct {
	*fakeFiles
	writes  int
	failAt  int
	removed int
}

func (f *batchFailFiles) Write(ctx context.Context, scenario, path string, content []byte) (string, error) {
	f.writes++
	if f.writes == f.failAt {
		return "", errors.New("injected batch write failure")
	}
	return f.fakeFiles.Write(ctx, scenario, path, content)
}

func (f *batchFailFiles) Remove(_ context.Context, scenario, path string) error {
	delete(f.bytes, scenario+"::"+path)
	f.removed++
	return nil
}

func TestBatchApplyRollsBackFilesAndRowsWhenLaterWriteFails(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-a": {ID: "cmp-a", LibraryID: "rcl:A", Slug: "a", LatestVersion: "1.0.0"},
			"cmp-b": {ID: "cmp-b", LibraryID: "rcl:B", Slug: "b", LatestVersion: "1.0.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-a@1.0.0": {ComponentID: "cmp-a", LibraryID: "rcl:A", Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "A.tsx", Content: "export const A = 1;", ContentSHA256: sha("export const A = 1;"), Files: []components.ComponentVersionFile{{Path: "A.tsx", Content: "export const A = 1;", ContentSHA256: sha("export const A = 1;"), IsEntry: true}}},
			"cmp-b@1.0.0": {ComponentID: "cmp-b", LibraryID: "rcl:B", Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "B.tsx", Content: "export const B = 1;", ContentSHA256: sha("export const B = 1;"), Files: []components.ComponentVersionFile{{Path: "B.tsx", Content: "export const B = 1;", ContentSHA256: sha("export const B = 1;"), IsEntry: true}}},
		},
	}
	files := &batchFailFiles{fakeFiles: &fakeFiles{bytes: map[string][]byte{}}, failAt: 2}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	_, err := svc.BatchApply(context.Background(), adoptions.BatchApplyInput{Items: []adoptions.BatchApplyItem{
		{ComponentID: "cmp-a", Scenario: "target", AdoptedPath: "ui/src/A.tsx", Version: "1.0.0"},
		{ComponentID: "cmp-b", Scenario: "target", AdoptedPath: "ui/src/B.tsx", Version: "1.0.0"},
	}})
	require.Error(t, err)
	require.Empty(t, files.bytes)
	rows, listErr := repo.List(context.Background(), adoptions.ListQuery{Scenario: "target", Limit: 10})
	require.NoError(t, listErr)
	require.Empty(t, rows)
	require.Equal(t, 2, files.removed)
}
