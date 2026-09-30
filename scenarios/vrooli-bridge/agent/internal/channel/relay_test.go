package channel

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bridgeexec "vrooli-bridge/agent/internal/exec"
	"vrooli-bridge/agent/internal/health"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

type relayCollector struct {
	mu        sync.Mutex
	responses []*sharedv1.RelayResponse
	seen      chan *sharedv1.RelayResponse
}

func newRelayCollector() *relayCollector {
	return &relayCollector{seen: make(chan *sharedv1.RelayResponse, 16)}
}

func (c *relayCollector) ReportRelayResponse(_ context.Context, response *sharedv1.RelayResponse) error {
	c.mu.Lock()
	c.responses = append(c.responses, response)
	c.mu.Unlock()
	c.seen <- response
	return nil
}

type blockingRelayCommand struct {
	started chan struct{}
	done    chan struct{}
	once    sync.Once
}

func (c *blockingRelayCommand) Run(ctx context.Context, _ []string, _ string, _ func(string)) (int, error) {
	c.once.Do(func() { close(c.started) })
	<-ctx.Done()
	close(c.done)
	return 143, ctx.Err()
}

var _ bridgeexec.CommandRunner = (*blockingRelayCommand)(nil)

func waitRelayKind(t *testing.T, seen <-chan *sharedv1.RelayResponse, kind sharedv1.RelayResponseKind) *sharedv1.RelayResponse {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case response := <-seen:
			if response.GetKind() == kind {
				return response
			}
		case <-deadline:
			t.Fatalf("timed out waiting for relay response kind %s", kind)
		}
	}
}

// deadlineCheckingCollector refuses a report whose context has already
// expired, the way the real Presence RPC does.
type deadlineCheckingCollector struct{ *relayCollector }

func (c deadlineCheckingCollector) ReportRelayResponse(ctx context.Context, response *sharedv1.RelayResponse) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.relayCollector.ReportRelayResponse(ctx, response)
}

type slowRelayCommand struct{ delay time.Duration }

func (c slowRelayCommand) Run(ctx context.Context, _ []string, _ string, _ func(string)) (int, error) {
	select {
	case <-time.After(c.delay):
		return 0, nil
	case <-ctx.Done():
		return 143, ctx.Err()
	}
}

// TestRelayCommandOutlivingTheReportBoundStillCompletes pins the relay hang:
// one report context shared by the whole relay expired while a command longer
// than its bound was still running, so the terminal report was refused and
// Bridge held the caller until the relay timeout. A 7-second remote install
// therefore surfaced as a 90-second "deadline exceeded".
func TestRelayCommandOutlivingTheReportBoundStillCompletes(t *testing.T) {
	previous := relayReportTimeout
	relayReportTimeout = 50 * time.Millisecond
	t.Cleanup(func() { relayReportTimeout = previous })

	client, priv := signedClient(t)
	client.logger = log.New(io.Discard, "", 0)
	collector := newRelayCollector()
	client.relayReporter = deadlineCheckingCollector{collector}
	client.commandRunner = slowRelayCommand{delay: 200 * time.Millisecond}
	client.baseCtx = context.Background()

	request := &channelv1.ServerFrame{FrameId: "relay-slow", Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "relay-slow", Scenario: "vrooli", Command: "resource install", MaxResponseBytes: 128,
	}}}
	client.handleServerFrame(signFrame(t, priv, request))
	waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_ACCEPTED)
	waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_COMPLETED)
}

var _ bridgeexec.CommandRunner = slowRelayCommand{}

type refreshCountingSampler struct {
	health.Sampler
	refreshes atomic.Int32
}

func (s *refreshCountingSampler) RefreshCapabilities() { s.refreshes.Add(1) }

// TestRelayedInstallRefreshesTheCapabilityInventory pins the "unconfirmed"
// install: the node re-probed its agents every 10 minutes, so an install
// through Bridge was never reported inside Web Console's confirmation window.
func TestRelayedInstallRefreshesTheCapabilityInventory(t *testing.T) {
	client, priv := signedClient(t)
	client.logger = log.New(io.Discard, "", 0)
	collector := newRelayCollector()
	sampler := &refreshCountingSampler{}
	client.sampler = sampler
	client.relayReporter = collector
	client.commandRunner = slowRelayCommand{}
	client.baseCtx = context.Background()

	for i, command := range []string{"scenario status", "resource install"} {
		request := &channelv1.ServerFrame{FrameId: "relay-" + command, Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
			CorrelationId: "relay-refresh-" + command, Scenario: "vrooli", Command: command, MaxResponseBytes: 128,
		}}}
		client.handleServerFrame(signFrame(t, priv, request))
		waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_COMPLETED)
		if got := sampler.refreshes.Load(); got != int32(i) {
			t.Fatalf("after %q refreshes = %d, want %d", command, got, i)
		}
	}
}

// [REQ:BRG-P1-004] A signed relay request starts a cancellable command and a
// signed relay cancellation terminates that exact execution.
func TestRelayCancellationStopsInFlightWork(t *testing.T) {
	client, priv := signedClient(t)
	client.logger = log.New(io.Discard, "", 0)
	collector := newRelayCollector()
	command := &blockingRelayCommand{started: make(chan struct{}), done: make(chan struct{})}
	client.relayReporter = collector
	client.commandRunner = command
	client.baseCtx = context.Background()

	request := &channelv1.ServerFrame{FrameId: "relay-frame", Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "relay-1", Scenario: "demo", Command: "scenario test", MaxResponseBytes: 128,
	}}}
	client.handleServerFrame(signFrame(t, priv, request))
	waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_ACCEPTED)
	select {
	case <-command.started:
	case <-time.After(2 * time.Second):
		t.Fatal("relay command did not start")
	}

	cancel := &channelv1.ServerFrame{FrameId: "cancel-frame", Payload: &channelv1.ServerFrame_RelayCancel{RelayCancel: &channelv1.RelayCancel{
		CorrelationId: "relay-1", Reason: "caller cancelled",
	}}}
	client.handleServerFrame(signFrame(t, priv, cancel))
	terminated := waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_TERMINATED)
	require.Equal(t, "caller cancelled", terminated.GetReason())
	select {
	case <-command.done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not reach the command runner")
	}
	require.Eventually(t, func() bool {
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.runningRelays) == 0
	}, time.Second, time.Millisecond)
}

func TestRelayUntrustedFramesNeverReachPayloadHandler(t *testing.T) {
	client, _ := signedClient(t)
	collector := newRelayCollector()
	client.relayReporter = collector
	client.commandRunner = &blockingRelayCommand{started: make(chan struct{}), done: make(chan struct{})}

	unsigned := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "unsigned", Scenario: "demo", Command: "scenario test;bad",
	}}}
	client.handleServerFrame(unsigned.String())

	attacker, _ := testCPKeys(t)
	wrongKey := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "wrong-key", Scenario: "demo", Command: "scenario test;bad",
	}}}
	client.handleServerFrame(signFrame(t, attacker, wrongKey))

	require.Equal(t, uint64(2), client.rejectedFrames.Load())
	client.mu.Lock()
	defer client.mu.Unlock()
	require.Empty(t, client.runningRelays, "untrusted frames must be rejected before relay payload handling")
	select {
	case <-collector.seen:
		t.Fatal("untrusted relay frame reached the response handler")
	default:
	}
}
