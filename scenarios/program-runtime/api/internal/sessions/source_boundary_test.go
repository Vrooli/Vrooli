package sessions

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLiveAdmissionBoundarySerializesDeletionAndRejectsReclaimed(t *testing.T) {
	m := NewManager(Options{})
	session, _, err := m.CreateOneShot(context.Background(), "fixture", "", nil, 0, 0, 1000, 0)
	require.NoError(t, err)
	entered, release, finished, deleted := make(chan struct{}), make(chan struct{}), make(chan error, 1), make(chan error, 1)
	go func() {
		finished <- m.WithLiveSession(context.Background(), session.ID, func(context.Context) error { close(entered); <-release; return nil })
	}()
	<-entered
	go func() { _, err := m.Delete(context.Background(), session.ID, "fixture invalidation"); deleted <- err }()
	select {
	case <-deleted:
		t.Fatal("delete crossed active admission boundary")
	case <-time.After(20 * time.Millisecond):
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, m.WithLiveSession(ctx, session.ID, func(context.Context) error { return nil }), context.Canceled)
	close(release)
	require.NoError(t, <-finished)
	require.NoError(t, <-deleted)
	called := false
	err = m.WithLiveSession(context.Background(), session.ID, func(context.Context) error { called = true; return nil })
	require.Error(t, err)
	require.False(t, called)
}
