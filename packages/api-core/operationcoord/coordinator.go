// Package operationcoord provides the ephemeral coordination shared by
// durable operation domains. Persistence remains the source of truth; this
// package only closes the race between registering a waiter and observing a
// terminal event, and fans later events out to live subscribers.
package operationcoord

import "sync"

const subscriberBuffer = 256

// Coordinator is safe for concurrent use. T is the domain event type; the
// coordinator deliberately knows nothing about lifecycle status or storage.
type Coordinator[T any] struct {
	mu       sync.Mutex
	waiters  map[string]map[chan struct{}]struct{}
	subs     map[string]map[chan T]struct{}
	synthSeq map[string]uint64
}

func New[T any]() *Coordinator[T] {
	return &Coordinator[T]{
		waiters:  make(map[string]map[chan struct{}]struct{}),
		subs:     make(map[string]map[chan T]struct{}),
		synthSeq: make(map[string]uint64),
	}
}

// RegisterWaiter must be called before the caller's durable terminal recheck.
// That ordering makes a terminal transition racing the recheck observable.
func (c *Coordinator[T]) RegisterWaiter(id string) (<-chan struct{}, func()) {
	ch := make(chan struct{})
	c.mu.Lock()
	set := c.waiters[id]
	if set == nil {
		set = make(map[chan struct{}]struct{})
		c.waiters[id] = set
	}
	set[ch] = struct{}{}
	c.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			c.mu.Lock()
			if set := c.waiters[id]; set != nil {
				delete(set, ch)
				if len(set) == 0 {
					delete(c.waiters, id)
				}
			}
			c.mu.Unlock()
		})
	}
	return ch, cancel
}

func (c *Coordinator[T]) SignalTerminal(id string) {
	c.mu.Lock()
	set := c.waiters[id]
	delete(c.waiters, id)
	c.mu.Unlock()
	for ch := range set {
		close(ch)
	}
}

func (c *Coordinator[T]) Subscribe(id string) (<-chan T, func()) {
	ch := make(chan T, subscriberBuffer)
	c.mu.Lock()
	set := c.subs[id]
	if set == nil {
		set = make(map[chan T]struct{})
		c.subs[id] = set
	}
	set[ch] = struct{}{}
	c.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			c.mu.Lock()
			if set := c.subs[id]; set != nil {
				delete(set, ch)
				if len(set) == 0 {
					delete(c.subs, id)
				}
			}
			c.mu.Unlock()
		})
	}
	return ch, cancel
}

func (c *Coordinator[T]) Publish(id string, event T) {
	c.mu.Lock()
	set := c.subs[id]
	channels := make([]chan T, 0, len(set))
	for ch := range set {
		channels = append(channels, ch)
	}
	c.mu.Unlock()
	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

// NextSyntheticSeq reserves a sequence in a range that node event streams do
// not use. It is for control-plane-generated failure events only.
func (c *Coordinator[T]) NextSyntheticSeq(id string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	const base = 1 << 62
	if c.synthSeq[id] == 0 {
		c.synthSeq[id] = base
	}
	c.synthSeq[id]++
	return c.synthSeq[id]
}
