package executionwriter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScreenshotDecodeBudgetWaitsForWeightedCapacity(t *testing.T) {
	budget := newScreenshotDecodeBudget(10)
	require.NoError(t, budget.acquire(context.Background(), 7))

	started := make(chan struct{})
	acquired := make(chan error, 1)
	go func() {
		close(started)
		acquired <- budget.acquire(context.Background(), 4)
	}()
	<-started
	select {
	case err := <-acquired:
		t.Fatalf("acquire(4) passed while 7 of 10 bytes were reserved: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	budget.release(7)
	select {
	case err := <-acquired:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("waiting screenshot decode was not admitted after capacity was released")
	}
	budget.release(4)

	// A canceled waiter must not reserve capacity after its caller has left.
	require.NoError(t, budget.acquire(context.Background(), 10))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, budget.acquire(ctx, 1), context.DeadlineExceeded)
	budget.release(10)
	require.NoError(t, budget.acquire(context.Background(), 10))
	budget.release(10)
}

func TestScreenshotDecodeWeightEnforcesRasterBudget(t *testing.T) {
	maxPixels := maxActiveScreenshotDecodeBytes / screenshotDecodeBytesPerPixel

	weight, err := screenshotDecodeWeight(int(maxPixels), 1)
	require.NoError(t, err)
	require.Equal(t, maxPixels*screenshotDecodeBytesPerPixel, weight)

	_, err = screenshotDecodeWeight(int(maxPixels+1), 1)
	require.ErrorContains(t, err, "exceeds active screenshot decode budget")

	_, err = screenshotDecodeWeight(int(^uint(0)>>1), int(^uint(0)>>1))
	require.ErrorContains(t, err, "exceeds active screenshot decode budget")

	for _, dimensions := range [][2]int{{0, 1}, {1, 0}, {-1, 1}, {1, -1}} {
		_, err = screenshotDecodeWeight(dimensions[0], dimensions[1])
		require.ErrorContains(t, err, "dimensions must be positive")
	}
}
