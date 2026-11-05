package hyperframes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

// InvocationAdmission describes this invocation, including exact-output reuse.
// The immutable render bundle remains the receipt of the original rendering.
type InvocationAdmission struct {
	OwnerID  string `json:"owner_id"`
	RAMBytes int64  `json:"ram_bytes"`
	CPUMilli int64  `json:"cpu_milli"`
	Verdict  string `json:"verdict"`
	Reason   string `json:"reason,omitempty"`
	Release  string `json:"release"`
}

type renderBroker interface {
	Acquire(context.Context, string, int64, int64) (capacity.Lease, capacity.Verdict, error)
	Close() error
}

type renderBrokerFactory func(context.Context) (renderBroker, error)

func newRenderBroker(ctx context.Context) (renderBroker, error) {
	return capacity.NewBroker(ctx, "")
}

var errOwnedCleanup = errors.New("owned render cleanup is unverified")

func withRenderAdmission(ctx context.Context, memoryMiB int, factory renderBrokerFactory, interval time.Duration, work func(context.Context) (Receipt, error)) (result Receipt, resultErr error) {
	a := &InvocationAdmission{RAMBytes: int64(memoryMiB) << 20, CPUMilli: 2000, Verdict: "unavailable", Release: "not_acquired"}
	defer func() { result.Admission = a }()
	if memoryMiB < 1024 || memoryMiB > 16384 || interval <= 0 {
		return result, errors.New("invalid render admission budget")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return result, err
	}
	a.OwnerID = "hyperframes:" + hex.EncodeToString(nonce[:])
	b, err := factory(ctx)
	if err != nil {
		return result, fmt.Errorf("render capacity unavailable: %w", err)
	}
	if b == nil {
		return result, errors.New("render capacity broker is nil")
	}
	defer func() { resultErr = errors.Join(resultErr, b.Close()) }()
	l, v, err := b.Acquire(ctx, a.OwnerID, a.RAMBytes, a.CPUMilli)
	a.Verdict, a.Reason = v.Kind, v.Reason
	if l != nil {
		a.Release = "pending"
		defer func() {
			if errors.Is(resultErr, errOwnedCleanup) {
				a.Release = "retained_until_expiry_cleanup_unverified"
				return
			}
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if e := l.Release(cleanup); e != nil {
				a.Release = "failed"
				resultErr = errors.Join(resultErr, fmt.Errorf("release render capacity: %w", e))
			} else {
				a.Release = "released"
			}
		}()
	}
	if err != nil {
		return result, fmt.Errorf("render capacity acquisition: %w", err)
	}
	if v.Kind != "grant" || l == nil {
		return result, fmt.Errorf("render capacity %s: %s; retry when capacity is available", v.Kind, v.Reason)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	renewable, ok := l.(capacity.RenewableLease)
	if !ok {
		return result, errors.New("render capacity lease cannot renew")
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				done <- nil
				return
			case <-workCtx.Done():
				done <- nil
				return
			case <-tick.C:
				renewCtx, end := context.WithTimeout(workCtx, 5*time.Second)
				err := renewable.Heartbeat(renewCtx)
				end()
				if err != nil {
					// Normal completion cancels any in-flight heartbeat.
					select {
					case <-stop:
						done <- nil
						return
					default:
					}
					cancel()
					done <- fmt.Errorf("render capacity renewal: %w", err)
					return
				}
			}
		}
	}()
	result, resultErr = work(workCtx)
	close(stop)
	cancel()
	resultErr = errors.Join(resultErr, ctx.Err(), <-done)
	return result, resultErr
}
