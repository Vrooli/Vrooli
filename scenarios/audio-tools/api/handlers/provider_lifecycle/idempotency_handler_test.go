package provider_lifecycle_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/handlers/provider_lifecycle"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"

	"github.com/vrooli/api-core/scheduletest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// gatedController blocks Start until release is closed or the caller's
// context is canceled, so tests can force concurrent callers to overlap.
type gatedController struct {
	*capmocks.FakeController
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func newGatedController() *gatedController {
	return &gatedController{
		FakeController: &capmocks.FakeController{},
		entered:        make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (c *gatedController) Start(ctx context.Context, slug string) error {
	recorded := c.FakeController.Start(ctx, slug)
	c.once.Do(func() { close(c.entered) })
	select {
	case <-c.release:
		return recorded
	case <-ctx.Done():
		return ctx.Err()
	}
}

func idempotentStartReq(providerID, key string) *connect.Request[plv1.StartProviderRequest] {
	req := connect.NewRequest(&plv1.StartProviderRequest{ProviderId: providerID})
	if key != "" {
		req.Header().Set("Idempotency-Key", key)
	}
	return req
}

func hasLogEntry(entries []string, needle string) bool {
	for _, entry := range entries {
		if strings.Contains(entry, needle) {
			return true
		}
	}
	return false
}

func TestStartProvider_SameIdempotencyKeyExecutesOnce(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, logger := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	first, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "key-1"))
	require.NoError(t, err)
	second, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "key-1"))
	require.NoError(t, err)

	require.Equal(t, []string{"whisper"}, ctrl.StartCalls, "controller must run once per key")
	require.Equal(t, first.Msg.GetProviderId(), second.Msg.GetProviderId())
	require.Equal(t, first.Msg.GetMessage(), second.Msg.GetMessage())
	require.False(t, second.Msg.GetDryRun())

	entries := logger.Entries()
	require.True(t, hasLogEntry(entries, "outcome=miss"), "leader logs a miss; got %v", entries)
	require.True(t, hasLogEntry(entries, "outcome=hit"), "replay logs a hit; got %v", entries)
}

func TestStartProvider_DifferentKeysExecuteTwice(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "key-1"))
	require.NoError(t, err)
	_, err = h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "key-2"))
	require.NoError(t, err)

	require.Len(t, ctrl.StartCalls, 2)
}

func TestStartProvider_SameKeyDifferentOperationDoesNotCollide(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "shared"))
	require.NoError(t, err)

	stopReq := connect.NewRequest(&plv1.StopProviderRequest{ProviderId: "whisper-stt"})
	stopReq.Header().Set("Idempotency-Key", "shared")
	_, err = h.StopProvider(context.Background(), stopReq)
	require.NoError(t, err)

	require.Len(t, ctrl.StartCalls, 1)
	require.Len(t, ctrl.StopCalls, 1, "a different operation must not reuse the Start result")
}

func TestStartProvider_SameKeyDifferentTargetDoesNotCollide(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "shared"))
	require.NoError(t, err)
	_, err = h.StartProvider(context.Background(), idempotentStartReq("kokoro-tts", "shared"))
	require.NoError(t, err)

	require.Equal(t, []string{"whisper", "kokoro"}, ctrl.StartCalls)
}

func TestStartProvider_ConcurrentSameKeyExecutesOnce(t *testing.T) {
	ctrl := newGatedController()
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	const callers = 5
	type result struct {
		resp *connect.Response[plv1.StartProviderResponse]
		err  error
	}
	results := make(chan result, callers)

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "concurrent"))
			results <- result{resp: resp, err: err}
		}()
	}

	<-ctrl.entered
	// Give the followers time to attach to the in-flight execution.
	time.Sleep(100 * time.Millisecond)
	close(ctrl.release)
	wg.Wait()
	close(results)

	for res := range results {
		require.NoError(t, res.err)
		require.Equal(t, "whisper-stt", res.resp.Msg.GetProviderId())
		require.Equal(t, "whisper-stt started", res.resp.Msg.GetMessage())
	}
	require.Len(t, ctrl.FakeController.StartCalls, 1, "concurrent duplicates must reach the controller once")
}

func TestStartProvider_LeaderErrorIsRetryableWithSameKey(t *testing.T) {
	ctrl := &capmocks.FakeController{StartErr: errors.New("boom")}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "retry-me"))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeInternal, connErr.Code())
	require.Len(t, ctrl.StartCalls, 1)

	ctrl.StartErr = nil
	resp, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "retry-me"))
	require.NoError(t, err, "errors must not be cached")
	require.Equal(t, "whisper-stt started", resp.Msg.GetMessage())
	require.Len(t, ctrl.StartCalls, 2, "a retry with the same key must execute again")
}

func TestStartProvider_DryRunAndRealRunDoNotShareResults(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	dryRunFirst := idempotentStartReq("whisper-stt", "shared")
	dryRunFirst.Header().Set("X-Dry-Run", "true")
	dry, err := h.StartProvider(context.Background(), dryRunFirst)
	require.NoError(t, err)
	require.True(t, dry.Msg.GetDryRun())
	require.Empty(t, ctrl.StartCalls)

	real, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "shared"))
	require.NoError(t, err)
	require.False(t, real.Msg.GetDryRun(), "a cached dry run must never satisfy a real run")
	require.Equal(t, "whisper-stt started", real.Msg.GetMessage())
	require.Len(t, ctrl.StartCalls, 1)

	dryRunLast := idempotentStartReq("whisper-stt", "shared")
	dryRunLast.Header().Set("X-Dry-Run", "true")
	dry, err = h.StartProvider(context.Background(), dryRunLast)
	require.NoError(t, err)
	require.True(t, dry.Msg.GetDryRun(), "a cached real run must never satisfy a dry run")
	require.Len(t, ctrl.StartCalls, 1)
}

func TestStartProvider_TTLExpiryReExecutes(t *testing.T) {
	clock := scheduletest.New(canonicalNow)
	store := provider_lifecycle.NewIdempotencyStore(provider_lifecycle.IdempotencyOptions{
		TTL: time.Minute,
		Now: clock.Now,
	})
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	deps.Idempotency = store
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "ttl"))
	require.NoError(t, err)
	_, err = h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "ttl"))
	require.NoError(t, err)
	require.Len(t, ctrl.StartCalls, 1)

	clock.Advance(time.Minute + time.Second)
	_, err = h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "ttl"))
	require.NoError(t, err)
	require.Len(t, ctrl.StartCalls, 2, "an expired cached result must re-execute")
}

func TestStartProvider_CachedResponseIsNotAliased(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	first, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "alias"))
	require.NoError(t, err)
	first.Msg.Message = "mutated by caller"
	first.Msg.ProviderId = "mutated"

	second, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "alias"))
	require.NoError(t, err)
	require.Equal(t, "whisper-stt", second.Msg.GetProviderId())
	require.Equal(t, "whisper-stt started", second.Msg.GetMessage())
	require.Len(t, ctrl.StartCalls, 1)
}

func TestStartProvider_NoKeyExecutesEveryTime(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, logger := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", ""))
	require.NoError(t, err)
	_, err = h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", ""))
	require.NoError(t, err)

	require.Len(t, ctrl.StartCalls, 2, "an empty Idempotency-Key must bypass dedup")
	require.False(t, hasLogEntry(logger.Entries(), "idempotency_key"), "no idempotency log without a key")
}

func TestStartProvider_LeaderCancellationFailsWaitersAndIsRetryable(t *testing.T) {
	ctrl := newGatedController()
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := h.StartProvider(leaderCtx, idempotentStartReq("whisper-stt", "cancel-me"))
		leaderErr <- err
	}()
	<-ctrl.entered

	joinerErr := make(chan error, 1)
	go func() {
		_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "cancel-me"))
		joinerErr <- err
	}()
	// Let the joiner attach before the leader is canceled.
	time.Sleep(100 * time.Millisecond)
	cancelLeader()

	for name, errCh := range map[string]chan error{"leader": leaderErr, "joiner": joinerErr} {
		err := <-errCh
		require.Error(t, err, "%s must observe the cancellation", name)
		var connErr *connect.Error
		require.True(t, errors.As(err, &connErr))
		require.Equal(t, connect.CodeCanceled, connErr.Code(), name)
	}

	close(ctrl.release)
	resp, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "cancel-me"))
	require.NoError(t, err, "the entry must be released after a canceled leader")
	require.Equal(t, "whisper-stt started", resp.Msg.GetMessage())
	require.Len(t, ctrl.FakeController.StartCalls, 2, "the canceled attempt plus the retry")
}

func TestPullModel_SameIdempotencyKeyExecutesOnce(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	pull := func() *connect.Request[plv1.PullModelRequest] {
		req := connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: "phi3"})
		req.Header().Set("Idempotency-Key", "pull-1")
		return req
	}
	first, err := h.PullModel(context.Background(), pull())
	require.NoError(t, err)
	second, err := h.PullModel(context.Background(), pull())
	require.NoError(t, err)

	require.Equal(t, []string{"phi3"}, ctrl.PullCalls)
	require.Equal(t, first.Msg.GetMessage(), second.Msg.GetMessage())
}

// Compile-time reminder that the harness controller seam is the interface
// the handler consumes; keeps the gated wrapper honest.
var _ capabilities.ResourceController = (*gatedController)(nil)

// TestStartProvider_CanceledJoinerMapsToCanceled keeps the transport contract
// honest: a joiner that gives up while the leader is still running must see
// Canceled, not Connect's generic Unknown for a bare context error.
func TestStartProvider_CanceledJoinerMapsToCanceled(t *testing.T) {
	ctrl := newGatedController()
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	leaderDone := make(chan error, 1)
	go func() {
		_, err := h.StartProvider(context.Background(), idempotentStartReq("whisper-stt", "join-cancel"))
		leaderDone <- err
	}()
	<-ctrl.entered

	joinerCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.StartProvider(joinerCtx, idempotentStartReq("whisper-stt", "join-cancel"))
	require.Error(t, err)
	require.Equal(t, connect.CodeCanceled, connect.CodeOf(err), "got: %v", err)

	close(ctrl.release)
	require.NoError(t, <-leaderDone)
}
