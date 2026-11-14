package sessions

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlowClaimIsDurableAndSingleAdmission(t *testing.T) {
	c, repo, _, auth, lease := desktopFixture(t)
	digest := strings.Repeat("a", 64)
	first, fresh, err := c.ClaimFlow(context.Background(), lease, "run", digest, 1)
	require.NoError(t, err)
	require.True(t, fresh)
	// A new repository object reads the same durable state after caller loss.
	reopened, err := NewSQLiteDesktopRepository(context.Background(), repo.db, repo.destination)
	require.NoError(t, err)
	c.repo = reopened
	next, fresh, err := c.ClaimFlow(context.Background(), lease, "run", digest, 1)
	require.NoError(t, err)
	require.False(t, fresh)
	require.Equal(t, first, next)
	_, fresh, err = c.ClaimFlow(context.Background(), lease, "run", strings.Repeat("b", 64), 1)
	require.Error(t, err)
	require.False(t, fresh)
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, fresh, err := c.ClaimFlow(context.Background(), lease, "concurrent", digest, 1)
			results <- fresh
			failures <- err
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	count := 0
	for fresh := range results {
		if fresh {
			count++
		}
	}
	for err := range failures {
		if err != nil {
			require.Contains(t, err.Error(), "SQLITE_BUSY")
		}
	}
	require.Equal(t, 1, count)
	_, fresh, err = c.ClaimFlow(context.Background(), lease, "concurrent", digest, 1)
	require.NoError(t, err)
	require.False(t, fresh)
	auth.denied = true
	_, fresh, err = c.ClaimFlow(context.Background(), lease, "run", digest, 1)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.False(t, fresh)
}

func TestFlowFinishRequiresReceiptsAndCannotRewriteTerminalState(t *testing.T) {
	c, repo, _, _, lease := desktopFixture(t)
	ctx := context.Background()
	digest := strings.Repeat("a", 64)
	_, _, err := c.ClaimFlow(ctx, lease, "run", digest, 1)
	require.NoError(t, err)
	_, err = c.FinishFlow(ctx, lease, "run", digest, "passed")
	require.Error(t, err)
	command := desktopCommand(lease)
	command.ID = "run:0"
	receipt, err := c.Act(ctx, command)
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Outcome)
	finished, err := c.FinishFlow(ctx, lease, "run", digest, "passed")
	require.NoError(t, err)
	require.EqualValues(t, 1, finished.Confirmed)
	reopened, err := NewSQLiteDesktopRepository(ctx, repo.db, repo.destination)
	require.NoError(t, err)
	c.repo = reopened
	again, fresh, err := c.ClaimFlow(ctx, lease, "run", digest, 1)
	require.NoError(t, err)
	require.False(t, fresh)
	require.Equal(t, finished, again)
	_, err = c.FinishFlow(ctx, lease, "run", digest, "incomplete")
	require.Error(t, err)
	_, _, err = c.ClaimFlow(ctx, lease, "partial", digest, 2)
	require.NoError(t, err)
	partial, err := c.FinishFlow(ctx, lease, "partial", digest, "incomplete")
	require.NoError(t, err)
	require.Zero(t, partial.Confirmed)
	command.ID = "partial:0"
	_, err = c.Act(ctx, command)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	_, err = c.FinishFlow(ctx, lease, "partial", digest, "passed")
	require.Error(t, err)
}
