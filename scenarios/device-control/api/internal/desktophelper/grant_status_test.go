package desktophelper

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGrantStatusPublisherRevokesAndClearsOnShutdown(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux private file owner")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	path := filepath.Join(dir, "grants.json")
	var active atomic.Bool
	active.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- MaintainGrantStatus(ctx, path, func(context.Context) (GrantStatus, error) {
			now := nowUTC()
			state := GrantStatus{Active: []string{}, ObservedAt: now, ExpiresAt: now.Add(GrantStatusLifetime)}
			if active.Load() {
				state.Active = []string{"grant"}
			}
			return state, nil
		})
	}()
	require.Eventually(t, func() bool { ok, err := grantActive(path, "grant", nowUTC()); return err == nil && ok }, 2*time.Second, 10*time.Millisecond)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	err = MaintainGrantStatus(context.Background(), path, func(context.Context) (GrantStatus, error) {
		t.Error("competing writer must never read or publish a snapshot")
		return GrantStatus{}, nil
	})
	require.Error(t, err)
	stillActive, err := grantActive(path, "grant", nowUTC())
	require.NoError(t, err)
	require.True(t, stillActive, "rejected writer must not clear the live publisher's grants")
	active.Store(false)
	require.Eventually(t, func() bool { ok, err := grantActive(path, "grant", nowUTC()); return err == nil && !ok }, 2*time.Second, 10*time.Millisecond)
	active.Store(true)
	require.Eventually(t, func() bool { ok, err := grantActive(path, "grant", nowUTC()); return err == nil && ok }, 2*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("publisher did not stop")
	}
	ok, err := grantActive(path, "grant", nowUTC())
	require.NoError(t, err)
	require.False(t, ok, "shutdown must clear authority without waiting for expiry")
}

func TestGrantStatusRejectsStalledOwnerAndPublicationFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux private file owner")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	path := filepath.Join(dir, "grants.json")
	now := nowUTC()
	state := GrantStatus{Active: []string{"grant"}, ObservedAt: now, ExpiresAt: now.Add(GrantStatusLifetime)}
	require.NoError(t, writePrivateJSON(path, state))
	ok, err := grantActive(path, "grant", state.ExpiresAt)
	require.Error(t, err)
	require.False(t, ok, "a dead publisher's last snapshot must expire")
	state.ExpiresAt = now.Add(time.Minute)
	require.NoError(t, writePrivateJSON(path, state))
	_, err = grantActive(path, "grant", now)
	require.Error(t, err, "long-lived snapshots must not defeat revocation bounds")
	failure := errors.New("owner state unavailable")
	err = MaintainGrantStatus(context.Background(), path, func(context.Context) (GrantStatus, error) { return GrantStatus{}, failure })
	require.ErrorIs(t, err, failure)
	ok, err = grantActive(path, "grant", nowUTC())
	require.NoError(t, err)
	require.False(t, ok)
}
