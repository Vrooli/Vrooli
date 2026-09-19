package components

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	localdb "react-component-library/internal/database"
)

func TestBuildManifestInputRestoredEvictedVersionsHaveUniqueFiles(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	))
	repo := NewSQLiteRepository(db, scheduletest.New(time.Now()))
	idx := NewIndexer(repo, "../../../library", nil)
	for _, path := range []string{
		"components/StatusBadge/component.json",
		"components/Tabs/component.json",
		"primitives/Icon/component.json",
	} {
		in, _, err := idx.buildManifestInput(path)
		require.NoError(t, err, path)
		seenVersions := map[string]bool{}
		for _, version := range in.Versions {
			require.False(t, seenVersions[version.Version], "%s duplicated version %s", path, version.Version)
			seenVersions[version.Version] = true
			seenFiles := map[string]bool{}
			for _, file := range version.Files {
				require.False(t, seenFiles[file.Path], "%s duplicated %s/%s", path, version.Version, file.Path)
				seenFiles[file.Path] = true
			}
		}
	}
}
