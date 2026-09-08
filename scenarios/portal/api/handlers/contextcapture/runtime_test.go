package contextcapture

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
)

func TestCleanupRunsAtStartupRetriesAndStops(t *testing.T) {
	clk := scheduletest.New(time.Now())
	calls := make(chan int, 3)
	reports := make(chan struct{}, 1)
	count := 0
	stop := startCleanup(reaperFunc(func(ctx context.Context, limit int) error {
		require.Equal(t, 64, limit)
		count++
		calls <- count
		if count == 1 {
			return errors.New("injected cleanup failure")
		}
		<-ctx.Done()
		return ctx.Err()
	}), clk, func() { reports <- struct{}{} })
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = stop(ctx)
	}()
	select {
	case <-reports:
	case <-time.After(time.Second):
		t.Fatal("startup cleanup failure not reported")
	}
	require.Equal(t, 1, <-calls)
	clk.Advance(time.Minute)
	select {
	case n := <-calls:
		require.Equal(t, 2, n)
	case <-time.After(time.Second):
		t.Fatal("scheduled cleanup did not retry")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, stop(ctx))
}
