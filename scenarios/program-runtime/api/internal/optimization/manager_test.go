package optimization

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockingRightsAdapter struct {
	*fakeAdapter
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

type unavailableRightsAdapter struct{ *fakeAdapter }

func (a *unavailableRightsAdapter) CheckRights(context.Context, string, Stage, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("rights owner is unavailable: %w", ErrOwnerUnavailable)
}

func (a *blockingRightsAdapter) CheckRights(ctx context.Context, id string, stage Stage, request Request, evidence []Evidence) (Evidence, error) {
	if stage == StagePreflightRights {
		a.once.Do(func() { close(a.started) })
		select {
		case <-a.release:
		case <-ctx.Done():
			return Evidence{}, ctx.Err()
		}
	}
	return a.fakeAdapter.CheckRights(ctx, id, stage, request, evidence)
}

func TestManagerDeduplicatesInflightRunAndBoundsActiveQueue(t *testing.T) {
	store := newMemoryStore()
	adapter := &blockingRightsAdapter{
		fakeAdapter: &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"},
		started:     make(chan struct{}),
		release:     make(chan struct{}),
	}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	manager.limit = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop()

	request := sampleRequest()
	_, err = manager.Submit(ctx, request)
	require.NoError(t, err)
	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("first run did not enter its owner stage")
	}

	// Re-submitting the same immutable run is idempotent and must not put a
	// second queue entry behind its in-flight stage.
	_, err = manager.Submit(ctx, request)
	require.NoError(t, err)
	require.Empty(t, manager.queue)

	second := sampleRequest()
	second.RunID = "r-2"
	_, err = manager.Submit(ctx, second)
	require.ErrorIs(t, err, ErrQueueFull)
	_, err = store.Load(ctx, second.RunID)
	require.True(t, errors.Is(err, ErrNotFound), "a rejected run must not be persisted")

	close(adapter.release)
	terminal, done, err := manager.Wait(ctx, request.RunID, 5*time.Second)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, StageComplete, terminal.Stage)

	// The first run's terminal receipt must return its active slot. The same
	// distinct run that was rejected above can now enter and finish.
	_, err = manager.Submit(ctx, second)
	require.NoError(t, err)
	secondTerminal, secondDone, err := manager.Wait(ctx, second.RunID, 5*time.Second)
	require.NoError(t, err)
	require.True(t, secondDone)
	require.Equal(t, StageComplete, secondTerminal.Stage)
}

func TestManagerRefusesUnconfiguredOwnerWithoutTransientRetries(t *testing.T) {
	store := newMemoryStore()
	adapter := &unavailableRightsAdapter{fakeAdapter: &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	manager.limit = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop()

	request := sampleRequest()
	_, err = manager.Submit(ctx, request)
	require.NoError(t, err)
	started := time.Now()
	record, terminal, err := manager.Wait(ctx, request.RunID, time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	require.Less(t, time.Since(started), time.Second)
	require.Equal(t, StageRefused, record.Stage)
	require.Equal(t, 1, record.StageAttempts[StagePreflightRights])
	require.Contains(t, record.Failure, "not configured")

	second := request
	second.RunID = "after-owner-refusal"
	_, err = manager.Submit(ctx, second)
	require.NoError(t, err, "terminal refusal must release its active-run slot")
	secondRecord, terminal, err := manager.Wait(ctx, second.RunID, time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	require.Equal(t, StageRefused, secondRecord.Stage)
}
