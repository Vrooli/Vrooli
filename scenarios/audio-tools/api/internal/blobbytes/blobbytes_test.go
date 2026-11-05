package blobbytes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
)

func TestStoreAdaptsFilesystemBlobStore(t *testing.T) {
	store := New(blobstore.NewFilesystemBlobStore(t.TempDir()))
	ctx := context.Background()

	require.NoError(t, store.Put(ctx, "reports/run.json", []byte(`{"ok":true}`), "application/json"))
	got, err := store.Get(ctx, "reports/run.json")
	require.NoError(t, err)
	require.Equal(t, []byte(`{"ok":true}`), got)
	require.NoError(t, store.Delete(ctx, "reports/run.json"))
	_, err = store.Get(ctx, "reports/run.json")
	require.Error(t, err)
}
