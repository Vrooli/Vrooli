package runmanager

import "sync"

// subscriberBuffer is the per-follower channel depth. A run emits on the order
// of tens-to-low-hundreds of events (phase boundaries, significant log lines,
// occasional heartbeats), so an actively-draining follower never fills this;
// only a follower that has stopped reading (already detached/dead) can, and a
// non-blocking publish simply skips it without stalling the run.
const subscriberBuffer = 1024

// broadcaster fans canonical run events out to N live followers while retaining
// the full ordered history so a late or re-attaching follower replays
// everything that happened before it subscribed. Subscribe atomically snapshots
// history and registers the live channel, so no event is lost or duplicated
// across that boundary.
type broadcaster struct {
	mu      sync.Mutex
	history []Event
	subs    map[*subscriber]struct{}
	closed  bool
}

type subscriber struct {
	ch chan Event
}

func newBroadcaster() *broadcaster {
	return &broadcaster{subs: make(map[*subscriber]struct{})}
}

// publish appends to the durable history and delivers to every live follower.
// Delivery is non-blocking: a follower whose buffer is full (i.e. it stopped
// draining) is skipped rather than stalling suite execution. The history
// retains the complete record regardless.
func (b *broadcaster) publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.history = append(b.history, ev)
	for s := range b.subs {
		select {
		case s.ch <- ev:
		default:
		}
	}
}

// subscribe returns the replay history (events published before now) plus a
// live channel for subsequent events. The live channel closes when the run
// terminates (close) or the returned cancel removes the follower.
func (b *broadcaster) subscribe() (replay []Event, ch <-chan Event, cancel func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	replay = append([]Event(nil), b.history...)
	s := &subscriber{ch: make(chan Event, subscriberBuffer)}

	if b.closed {
		// Run already terminal: the replay holds the full history including the
		// run_completed boundary, so hand back an already-closed live channel.
		close(s.ch)
		return replay, s.ch, func() {}
	}

	b.subs[s] = struct{}{}
	cancel = func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[s]; ok {
			delete(b.subs, s)
			close(s.ch)
		}
	}
	return replay, s.ch, cancel
}

// close marks the broadcaster terminal and closes every live follower channel.
func (b *broadcaster) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for s := range b.subs {
		close(s.ch)
		delete(b.subs, s)
	}
}
