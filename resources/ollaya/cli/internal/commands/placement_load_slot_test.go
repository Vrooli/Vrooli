package commands

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// requirePlacement runs the dispatch gate and releases the load slot at once,
// the way a decision that finished would. Tests that care about the slot call
// requireModelPlacement directly.
func requirePlacement(h *Handlers, p policy.Policy, r policy.Resolution) error {
	if h.LoadSlot == nil {
		h.LoadSlot = func(context.Context) (func(), error) { return func() {}, nil }
	}
	release, err := h.requireModelPlacement(context.Background(), p, r)
	release()
	return err
}

// Two decisions that both need a model that is not resident must not both be
// told the same free memory is theirs. The load slot makes admission and the
// load one critical section per daemon, using the real cross-process
// semaphore: the second caller waits, then finds the model resident and asks
// the broker nothing.
func TestConcurrentColdLoadsAskTheBrokerOnceAndTheSecondWaits(t *testing.T) {
	p := shippedModelPolicy(t)
	kev := policy.Resolution{Role: "decide.local.kev", Locality: policy.LocalityLocal, Model: "kev:4b"}
	var (
		mu     sync.Mutex
		loaded bool
		asks   atomic.Int32
	)
	// Both callers share one lock directory, as separate resource-ollaya
	// invocations on one host do.
	dir := t.TempDir()
	newHandlers := func() *Handlers {
		return &Handlers{
			GetEnv:       func(key string) string { return map[string]string{"OLLAYA_LOAD_LOCK_DIR": dir}[key] },
			DaemonDevice: func(context.Context) (string, error) { return "cuda", nil },
			Residents: func(context.Context) ([]upstream.ResidentModel, error) {
				mu.Lock()
				defer mu.Unlock()
				if loaded {
					return []upstream.ResidentModel{{Name: "kev:4b", Device: "cuda:0", SizeVRAM: 7 << 30}}, nil
				}
				return nil, nil
			},
			CapacityAdmit: func(context.Context, string, int64) (bool, string, error) {
				asks.Add(1)
				return true, "", nil
			},
		}
	}
	first, second := newHandlers(), newHandlers()

	releaseFirst, err := first.requireModelPlacement(context.Background(), p, kev)
	if err != nil {
		t.Fatalf("first gate = %v", err)
	}
	if asks.Load() != 1 {
		t.Fatalf("broker asks after the first gate = %d, want 1", asks.Load())
	}

	done := make(chan error, 1)
	var releaseSecond func()
	go func() {
		var gateErr error
		releaseSecond, gateErr = second.requireModelPlacement(context.Background(), p, kev)
		done <- gateErr
	}()
	select {
	case err := <-done:
		t.Fatalf("second gate returned (%v) while the first load still held the slot", err)
	case <-time.After(200 * time.Millisecond):
	}

	// The first decision finishes: its model is resident and the slot frees.
	mu.Lock()
	loaded = true
	mu.Unlock()
	releaseFirst()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second gate = %v, want admitted", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second gate never acquired the slot after the first released it")
	}
	releaseSecond()
	if asks.Load() != 1 {
		t.Fatalf("broker asks = %d, want 1: the second caller found the model resident", asks.Load())
	}
}

func TestLoadSlotWaitIsBoundedByTheCallersContextAndRefusesRetryably(t *testing.T) {
	p := shippedModelPolicy(t)
	kev := policy.Resolution{Role: "decide.local.kev", Locality: policy.LocalityLocal, Model: "kev:4b"}
	h := &Handlers{
		Residents: func(context.Context) ([]upstream.ResidentModel, error) { return nil, nil },
		LoadSlot:  func(ctx context.Context) (func(), error) { <-ctx.Done(); return nil, ctx.Err() },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := h.requireModelPlacement(ctx, p, kev)
	var providerErr *upstream.ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != upstream.CodeProviderOverloaded {
		t.Fatalf("err = %v, want a retryable provider_overloaded refusal", err)
	}
}

func TestAResidentModelNeverTakesTheLoadSlot(t *testing.T) {
	p := shippedModelPolicy(t)
	layaEN := policy.Resolution{Role: "decide.local", Locality: policy.LocalityLocal, Model: "laya:en"}
	h := &Handlers{
		Residents: func(context.Context) ([]upstream.ResidentModel, error) {
			return []upstream.ResidentModel{{Name: "laya:en", Device: "cuda:0", SizeVRAM: 1 << 30}}, nil
		},
		LoadSlot: func(context.Context) (func(), error) {
			t.Fatal("a warm decision must not queue for a load slot")
			return nil, nil
		},
	}
	release, err := h.requireModelPlacement(context.Background(), p, layaEN)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestTheSlotIsReleasedWhenAdmissionRefuses(t *testing.T) {
	p := shippedModelPolicy(t)
	kev := policy.Resolution{Role: "decide.local.kev", Locality: policy.LocalityLocal, Model: "kev:4b"}
	var held atomic.Int32
	h := &Handlers{
		Residents:     func(context.Context) ([]upstream.ResidentModel, error) { return nil, nil },
		DaemonDevice:  func(context.Context) (string, error) { return "cuda", nil },
		CapacityAdmit: func(context.Context, string, int64) (bool, string, error) { return false, "insufficient vram", nil },
		LoadSlot: func(context.Context) (func(), error) {
			held.Add(1)
			return func() { held.Add(-1) }, nil
		},
	}
	if _, err := h.requireModelPlacement(context.Background(), p, kev); err == nil {
		t.Fatal("want a refusal")
	}
	if held.Load() != 0 {
		t.Fatal("a refused admission must not keep the load slot")
	}
}
