package exec_test

import (
	"context"
	"sync"
	"testing"
	"time"

	bridgeexec "vrooli-bridge/agent/internal/exec"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

type relayCommand struct {
	mu        sync.Mutex
	called    bool
	cancelled chan struct{}
}

func (f *relayCommand) Run(ctx context.Context, _ []string, _ string, onLog func(string)) (int, error) {
	f.mu.Lock()
	f.called = true
	f.mu.Unlock()
	onLog("abcd")
	<-ctx.Done()
	close(f.cancelled)
	return 143, ctx.Err()
}

func TestExecuteRelayBoundsOutputAndCancelsProducer(t *testing.T) {
	command := &relayCommand{cancelled: make(chan struct{})}
	runner := bridgeexec.NewRunner("vrooli", "", nil, bridgeexec.WithCommandRunner(command))
	result := runner.ExecuteRelay(context.Background(), &channelv1.RelayRequest{Scenario: "demo", Command: "scenario test"}, 3, func([]byte) {})

	require.True(t, result.LimitExceeded)
	require.Equal(t, uint64(3), result.TotalBytes)
	select {
	case <-command.cancelled:
	case <-time.After(time.Second):
		t.Fatal("response limit did not cancel the producing command")
	}
}

func TestExecuteRelayRejectsUnsafeTokenBeforeCommand(t *testing.T) {
	command := &relayCommand{cancelled: make(chan struct{})}
	runner := bridgeexec.NewRunner("vrooli", "", nil, bridgeexec.WithCommandRunner(command))
	result := runner.ExecuteRelay(context.Background(), &channelv1.RelayRequest{Scenario: "demo", Command: "scenario test", Args: []string{"x;bad"}}, 64, func([]byte) {})

	require.NotZero(t, result.ExitCode)
	require.Contains(t, result.Reason, "unsafe token")
	command.mu.Lock()
	defer command.mu.Unlock()
	require.False(t, command.called)
}
