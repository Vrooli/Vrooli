package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/config"
)

type availabilityProvider struct {
	available atomic.Bool
	checks    int32
}

func (p *availabilityProvider) IsAvailable(context.Context) (bool, string) {
	atomic.AddInt32(&p.checks, 1)
	if p.available.Load() {
		return true, "ok"
	}
	return false, "unavailable"
}

func TestCommandWorkspaceSandboxEnsurer_AvailableSkipsLifecycleStart(t *testing.T) {
	provider := &availabilityProvider{}
	provider.available.Store(true)
	ensurer := NewCommandWorkspaceSandboxEnsurer(provider, config.DefaultLevers().Sandbox)
	ensurer.command = []string{"false"}

	if err := ensurer.EnsureAvailable(context.Background()); err != nil {
		t.Fatalf("EnsureAvailable returned error: %v", err)
	}
}

func TestCommandWorkspaceSandboxEnsurer_ConcurrentCallsCoalesceLifecycleStart(t *testing.T) {
	provider := &availabilityProvider{}
	levers := config.DefaultLevers().Sandbox
	levers.EnsurePollInterval = 50 * time.Millisecond
	ensurer := NewCommandWorkspaceSandboxEnsurer(provider, levers)
	ensurer.command = []string{"bash", "-c", "sleep 0.1"}

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ensurer.EnsureAvailable(context.Background())
		}()
	}

	time.AfterFunc(150*time.Millisecond, func() {
		provider.available.Store(true)
	})
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("EnsureAvailable returned error: %v", err)
		}
	}
}
