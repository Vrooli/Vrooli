package stt

import (
	"context"
	"time"
)

const defaultStreamIdleTimeout = 30 * time.Second

// StreamIdleTimer is the injectable timer contract used by Connect stream
// idle detection.
type StreamIdleTimer interface {
	C() <-chan time.Time
	Reset(time.Duration)
	Stop()
}

type wallStreamIdleTimer struct{ timer *time.Timer }

func newWallStreamIdleTimer(timeout time.Duration) StreamIdleTimer {
	return &wallStreamIdleTimer{timer: time.NewTimer(timeout)}
}

func (t *wallStreamIdleTimer) C() <-chan time.Time { return t.timer.C }
func (t *wallStreamIdleTimer) Stop()               { t.timer.Stop() }
func (t *wallStreamIdleTimer) Reset(d time.Duration) {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(d)
}

// watchStreamIdle tracks audio arrival independently of Receive. Receive is
// synchronous and cannot be canceled directly, so timeout must cancel the
// pipeline and let the handler return; Connect then tears down the transport.
func watchStreamIdle(ctx context.Context, timer StreamIdleTimer, timeout time.Duration, activity <-chan struct{}, inputDone <-chan struct{}, onExpire func()) <-chan struct{} {
	expired := make(chan struct{}, 1)
	go func() {
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-inputDone:
				return
			case <-activity:
				timer.Reset(timeout)
			case <-timer.C():
				onExpire()
				expired <- struct{}{}
				return
			}
		}
	}()
	return expired
}
