package provider_lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/scheduletest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

var storeTestStart = time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

func storeTestKey(key string) idempotencyKey {
	return idempotencyKey{key: key, operation: "StartProvider", target: "whisper-stt"}
}

func storeTestMessage(providerID string) proto.Message {
	return &plv1.StartProviderResponse{ProviderId: providerID, Message: "started " + providerID}
}

// entryCount reports the retained entry count under the store's lock.
func (s *IdempotencyStore) entryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// hasKey reports whether k is retained, under the store's lock.
func (s *IdempotencyStore) hasKey(k idempotencyKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.entries[k]
	return ok
}

func TestIdempotencyStore_SequentialHitReturnsClone(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	var calls atomic.Int64
	fn := func(context.Context) (proto.Message, error) {
		calls.Add(1)
		return storeTestMessage("whisper-stt"), nil
	}

	first, outcome, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, outcome)
	require.Equal(t, "started whisper-stt", first.(*plv1.StartProviderResponse).GetMessage())

	second, outcome, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	require.Equal(t, outcomeHit, outcome)
	require.Equal(t, first, second, "hit must return an equivalent message")
	require.Equal(t, int64(1), calls.Load(), "the action must run exactly once")
}

func TestIdempotencyStore_CachedMessageIsNotAliased(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	calls := 0
	produced := &plv1.StartProviderResponse{ProviderId: "whisper-stt", Message: "started whisper-stt"}
	fn := func(context.Context) (proto.Message, error) {
		calls++
		return produced, nil
	}

	first, _, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	first.(*plv1.StartProviderResponse).Message = "mutated by caller"
	produced.Message = "mutated by producer"

	second, _, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	require.Equal(t, "started whisper-stt", second.(*plv1.StartProviderResponse).GetMessage(),
		"mutating a returned message must not corrupt the cached result")
	require.Equal(t, 1, calls)
}

func TestIdempotencyStore_DistinctKeysExecuteSeparately(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	var calls atomic.Int64
	fn := func(context.Context) (proto.Message, error) {
		calls.Add(1)
		return storeTestMessage("whisper-stt"), nil
	}

	_, firstOutcome, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	_, secondOutcome, err := store.Do(context.Background(), storeTestKey("B"), fn)
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, firstOutcome)
	require.Equal(t, outcomeMiss, secondOutcome)
	require.Equal(t, int64(2), calls.Load())
}

func TestIdempotencyStore_OperationAndTargetScopeTheKey(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	var calls atomic.Int64
	fn := func(context.Context) (proto.Message, error) {
		calls.Add(1)
		return storeTestMessage("whisper-stt"), nil
	}

	base := idempotencyKey{key: "shared", operation: "StartProvider", target: "whisper-stt"}
	otherOp := idempotencyKey{key: "shared", operation: "StopProvider", target: "whisper-stt"}
	otherTarget := idempotencyKey{key: "shared", operation: "StartProvider", target: "kokoro-tts"}
	dryRun := idempotencyKey{key: "shared", operation: "StartProvider", target: "whisper-stt", dryRun: true}

	for _, key := range []idempotencyKey{base, otherOp, otherTarget, dryRun} {
		_, outcome, err := store.Do(context.Background(), key, fn)
		require.NoError(t, err)
		require.Equal(t, outcomeMiss, outcome)
	}
	require.Equal(t, int64(4), calls.Load(), "every dimension of the key must namespace separately")
}

func TestIdempotencyStore_ConcurrentCallersShareOneExecution(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	var calls atomic.Int64
	entered := make(chan struct{})
	release := make(chan struct{})
	var enterOnce sync.Once
	fn := func(context.Context) (proto.Message, error) {
		calls.Add(1)
		enterOnce.Do(func() { close(entered) })
		<-release
		return storeTestMessage("whisper-stt"), nil
	}

	const followers = 4
	type result struct {
		msg     proto.Message
		outcome idempotencyOutcome
		err     error
	}
	results := make(chan result, followers+1)
	call := func() {
		msg, outcome, err := store.Do(context.Background(), storeTestKey("A"), fn)
		results <- result{msg: msg, outcome: outcome, err: err}
	}

	go call()
	<-entered // leader is now inside fn
	for i := 0; i < followers; i++ {
		go call()
	}
	// Give the followers time to attach to the in-flight entry before the
	// leader completes; they must join rather than execute.
	time.Sleep(100 * time.Millisecond)
	close(release)

	outcomes := make(map[idempotencyOutcome]int)
	for i := 0; i < followers+1; i++ {
		res := <-results
		require.NoError(t, res.err)
		require.Equal(t, "started whisper-stt", res.msg.(*plv1.StartProviderResponse).GetMessage())
		outcomes[res.outcome]++
	}
	require.Equal(t, int64(1), calls.Load(), "single-flight must run the action once")
	require.Equal(t, 1, outcomes[outcomeMiss], "exactly one leader")
	require.Equal(t, followers, outcomes[outcomeJoin], "every concurrent caller joins")
}

func TestIdempotencyStore_ErrorIsNotCachedAndEntryIsReleased(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	failure := errors.New("controller exploded")

	_, outcome, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
		return nil, failure
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, outcomeMiss, outcome)
	require.False(t, store.hasKey(storeTestKey("A")), "a failed leader must release the key")

	calls := 0
	msg, outcome, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
		calls++
		return storeTestMessage("whisper-stt"), nil
	})
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, outcome)
	require.NotNil(t, msg)
	require.Equal(t, 1, calls, "retrying the same key after an error must execute again")
}

func TestIdempotencyStore_LeaderCancellationReleasesEntryAndFailsJoiners(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	entered := make(chan struct{})

	leaderErr := make(chan error, 1)
	go func() {
		_, _, err := store.Do(leaderCtx, storeTestKey("A"), func(ctx context.Context) (proto.Message, error) {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		})
		leaderErr <- err
	}()
	<-entered

	joinerErr := make(chan error, 1)
	go func() {
		_, _, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
			t.Error("joiner must not execute the action")
			return nil, nil
		})
		joinerErr <- err
	}()
	// Let the joiner attach before the leader is canceled.
	time.Sleep(100 * time.Millisecond)
	cancelLeader()

	require.ErrorIs(t, <-leaderErr, context.Canceled)
	require.ErrorIs(t, <-joinerErr, context.Canceled, "joiners must observe the leader's cancellation")
	require.False(t, store.hasKey(storeTestKey("A")), "cancellation must release the in-flight entry")

	retryCalls := 0
	msg, outcome, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
		retryCalls++
		return storeTestMessage("whisper-stt"), nil
	})
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, outcome)
	require.NotNil(t, msg)
	require.Equal(t, 1, retryCalls, "a later retry must re-execute")
}

func TestIdempotencyStore_LeaderCancellationAfterSuccessDoesNotCache(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	ctx, cancel := context.WithCancel(context.Background())

	_, _, err := store.Do(ctx, storeTestKey("A"), func(context.Context) (proto.Message, error) {
		// The action "succeeds" but the caller has already gone away; the
		// result must not be published as a cache hit.
		cancel()
		return storeTestMessage("whisper-stt"), nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, store.hasKey(storeTestKey("A")))
}

func TestIdempotencyStore_CanceledJoinerLeavesLeaderAlone(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{})
	entered := make(chan struct{})
	release := make(chan struct{})

	leaderDone := make(chan error, 1)
	go func() {
		_, _, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
			close(entered)
			<-release
			return storeTestMessage("whisper-stt"), nil
		})
		leaderDone <- err
	}()
	<-entered

	joinerCtx, cancelJoiner := context.WithCancel(context.Background())
	joinerDone := make(chan error, 1)
	go func() {
		_, _, err := store.Do(joinerCtx, storeTestKey("A"), func(context.Context) (proto.Message, error) {
			t.Error("joiner must not execute the action")
			return nil, nil
		})
		joinerDone <- err
	}()
	time.Sleep(100 * time.Millisecond)
	cancelJoiner()
	require.ErrorIs(t, <-joinerDone, context.Canceled)

	close(release)
	require.NoError(t, <-leaderDone, "a follower giving up must not cancel the leader")

	msg, outcome, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
		t.Error("completed leader result must be replayed")
		return nil, nil
	})
	require.NoError(t, err)
	require.Equal(t, outcomeHit, outcome)
	require.NotNil(t, msg)
}

func TestIdempotencyStore_TTLExpiryReExecutes(t *testing.T) {
	clock := scheduletest.New(storeTestStart)
	store := NewIdempotencyStore(IdempotencyOptions{TTL: time.Minute, Now: clock.Now})
	calls := 0
	fn := func(context.Context) (proto.Message, error) {
		calls++
		return storeTestMessage("whisper-stt"), nil
	}

	_, _, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	_, outcome, err := store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	require.Equal(t, outcomeHit, outcome)
	require.Equal(t, 1, calls)

	clock.Advance(time.Minute + time.Second)
	_, outcome, err = store.Do(context.Background(), storeTestKey("A"), fn)
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, outcome)
	require.Equal(t, 2, calls, "an expired entry must re-execute")
}

func TestIdempotencyStore_ExpiredEntriesAreEvicted(t *testing.T) {
	clock := scheduletest.New(storeTestStart)
	store := NewIdempotencyStore(IdempotencyOptions{TTL: time.Minute, MaxEntries: 8, Now: clock.Now})

	_, _, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
		return storeTestMessage("whisper-stt"), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, store.entryCount())

	clock.Advance(2 * time.Minute)
	_, outcome, err := store.Do(context.Background(), storeTestKey("B"), func(context.Context) (proto.Message, error) {
		return storeTestMessage("kokoro-tts"), nil
	})
	require.NoError(t, err)
	require.Equal(t, outcomeMiss, outcome)
	require.Equal(t, 1, store.entryCount(), "expired entries must be swept")
	require.False(t, store.hasKey(storeTestKey("A")))
}

func TestIdempotencyStore_HardCapEvictsOldestCompleted(t *testing.T) {
	clock := scheduletest.New(storeTestStart)
	store := NewIdempotencyStore(IdempotencyOptions{TTL: time.Hour, MaxEntries: 2, Now: clock.Now})
	fn := func(context.Context) (proto.Message, error) {
		return storeTestMessage("whisper-stt"), nil
	}

	for _, key := range []string{"A", "B", "C"} {
		_, _, err := store.Do(context.Background(), storeTestKey(key), fn)
		require.NoError(t, err)
		clock.Advance(time.Second)
	}

	require.Equal(t, 2, store.entryCount(), "the store must stay hard-capped")
	require.False(t, store.hasKey(storeTestKey("A")), "the oldest completed entry is evicted first")
	require.True(t, store.hasKey(storeTestKey("B")))
	require.True(t, store.hasKey(storeTestKey("C")))
}

func TestIdempotencyStore_BypassesWhenCapIsAllInFlight(t *testing.T) {
	store := NewIdempotencyStore(IdempotencyOptions{MaxEntries: 1})
	entered := make(chan struct{})
	release := make(chan struct{})

	leaderDone := make(chan error, 1)
	go func() {
		_, _, err := store.Do(context.Background(), storeTestKey("A"), func(context.Context) (proto.Message, error) {
			close(entered)
			<-release
			return storeTestMessage("whisper-stt"), nil
		})
		leaderDone <- err
	}()
	<-entered

	calls := 0
	msg, outcome, err := store.Do(context.Background(), storeTestKey("B"), func(context.Context) (proto.Message, error) {
		calls++
		return storeTestMessage("kokoro-tts"), nil
	})
	require.NoError(t, err)
	require.Equal(t, outcomeBypass, outcome, "an all-in-flight store must not grow past its cap")
	require.NotNil(t, msg)
	require.Equal(t, 1, calls)
	require.Equal(t, 1, store.entryCount())

	close(release)
	require.NoError(t, <-leaderDone)
}

func TestCloneMessage_NilSafe(t *testing.T) {
	require.Nil(t, cloneMessage(nil))
	src := storeTestMessage("whisper-stt")
	clone := cloneMessage(src)
	require.Equal(t, src, clone)
	require.NotSame(t, src, clone)
}
