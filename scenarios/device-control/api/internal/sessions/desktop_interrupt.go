package sessions

import "context"

type desktopNativeCall struct {
	lease  DesktopLease
	cancel context.CancelFunc
}

// Cancellation interrupts bounded native work so Stop can reach the durable
// lease fence. It never substitutes for that fence or confirms an input effect.
func (c *DesktopController) nativeContext(ctx context.Context, lease DesktopLease) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(ctx, lease.ExpiresAt)
	call := &desktopNativeCall{lease: lease, cancel: cancel}
	c.nativeMu.Lock()
	if c.nativeCalls == nil {
		c.nativeCalls = make(map[*desktopNativeCall]struct{})
	}
	c.nativeCalls[call] = struct{}{}
	for stopping := range c.nativeStops {
		if sameDesktopLease(*stopping, lease) {
			cancel()
		}
	}
	c.nativeMu.Unlock()
	return ctx, func() {
		cancel()
		c.nativeMu.Lock()
		delete(c.nativeCalls, call)
		c.nativeMu.Unlock()
	}
}

// Call only after authenticating Stop. Exact lease matching prevents a stale
// holder from interrupting its successor. Concurrent Stops retain their own fence.
func (c *DesktopController) interruptNative(lease DesktopLease) func() {
	c.nativeMu.Lock()
	if c.nativeStops == nil {
		c.nativeStops = make(map[*DesktopLease]struct{})
	}
	c.nativeStops[&lease] = struct{}{}
	for call := range c.nativeCalls {
		if sameDesktopLease(call.lease, lease) {
			call.cancel()
		}
	}
	c.nativeMu.Unlock()
	return func() {
		c.nativeMu.Lock()
		delete(c.nativeStops, &lease)
		c.nativeMu.Unlock()
	}
}
