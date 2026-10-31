package control

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDesktopOwnerUnavailableCanStopWithoutBlockingAPI(t *testing.T) {
	svc, _ := testService(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reported := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.RunDesktopOwner(ctx, filepath.Join(t.TempDir(), "missing.json"), func(err error) { reported <- err })
	}()
	select {
	case err := <-reported:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("missing config was not reported")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("retry delay blocked owner shutdown")
	}
	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err, "optional owner failure must leave ordinary device control usable")
	require.NotEmpty(t, lease.ID)
}
