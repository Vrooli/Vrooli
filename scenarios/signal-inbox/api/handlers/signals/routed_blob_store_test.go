package signals

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestRoutedBlobStoreWritesTestModeImagesOnlyToLeasedRoots(t *testing.T) {
	primary := storage.Paths{DataDir: filepath.Join(t.TempDir(), "primary-data")}
	leased := storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}
	roots := filerouting.New(primary)
	require.NoError(t, roots.InstallTestRoots(leased, "test-lease", time.Minute))
	store, err := newRoutedBlobStore(roots)
	require.NoError(t, err)

	ctx := database.WithTestMode(context.Background())
	require.NoError(t, store.Put(ctx, "images/example", strings.NewReader("test image"), "image/png"))

	read, mime, err := store.Get(ctx, "images/example")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, read.Close()) })
	body, err := io.ReadAll(read)
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	require.Equal(t, "test image", string(body))

	_, _, err = store.Get(context.Background(), "images/example")
	require.Error(t, err, "the primary store must not receive a test-mode image")
	require.EqualValues(t, 1, roots.LeaseStats().TestRootWrites)
	require.Zero(t, roots.LeaseStats().PrimaryWritesDuringTestMode)
}
