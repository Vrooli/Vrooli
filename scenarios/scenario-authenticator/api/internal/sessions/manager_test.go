package sessions

import (
	"context"
	"errors"
	"sync"
	"testing"

	"scenario-authenticator/internal/redisstate"
)

func TestRotateRefreshConcurrentPresentationIsSingleUse(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(redisstate.NewMemory(), nil)
	original, err := manager.IssueRefresh(ctx, "user-1", "scenario-authenticator:default")
	if err != nil {
		t.Fatalf("issue refresh: %v", err)
	}

	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, _, rotateErr := manager.RotateRefresh(ctx, original)
			results <- rotateErr
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	replays := 0
	invalid := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrRefreshReuse):
			replays++
		case errors.Is(err, ErrInvalidRefresh):
			invalid++
		default:
			t.Fatalf("unexpected concurrent rotation error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful rotations = %d, want exactly one", successes)
	}
	if replays == 0 || replays+invalid != callers-1 {
		t.Fatalf("replay/invalid rejections = %d/%d, want %d total and at least one replay", replays, invalid, callers-1)
	}
}
