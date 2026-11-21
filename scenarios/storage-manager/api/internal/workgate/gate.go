// Package workgate tracks background work across a lifecycle fence.
package workgate

import (
	"context"
	"sync"
)

type Gate struct {
	mu      sync.Mutex
	closed  bool
	active  int
	changed chan struct{}
}

// Run admits a callback only while the gate is open. A refused cycle is skipped;
// schedulers retain their normal cadence and can run again after Resume.
func (g *Gate) Run(fn func() error) error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	g.active++
	g.mu.Unlock()
	defer g.finish()
	return fn()
}

func (g *Gate) Active() int { g.mu.Lock(); defer g.mu.Unlock(); return g.active }

// RunWhenOpen defers a one-shot task across maintenance instead of dropping it.
func (g *Gate) RunWhenOpen(ctx context.Context, fn func() error) error {
	for {
		g.mu.Lock()
		if err := ctx.Err(); err != nil {
			g.mu.Unlock()
			return err
		}
		if !g.closed {
			g.active++
			g.mu.Unlock()
			defer g.finish()
			return fn()
		}
		if g.changed == nil {
			g.changed = make(chan struct{})
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (g *Gate) finish() {
	g.mu.Lock()
	g.active--
	if g.changed != nil {
		close(g.changed)
		g.changed = nil
	}
	g.mu.Unlock()
}

// Quiesce closes admission before waiting. Cancellation leaves it closed, so a
// failed maintenance attempt cannot claim quiescence or admit new work.
func (g *Gate) Quiesce(ctx context.Context) error {
	g.mu.Lock()
	g.closed = true
	for g.active != 0 {
		if g.changed == nil {
			g.changed = make(chan struct{})
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
		g.mu.Lock()
	}
	g.mu.Unlock()
	return nil
}

func (g *Gate) Resume() {
	g.mu.Lock()
	g.closed = false
	if g.changed != nil {
		close(g.changed)
		g.changed = nil
	}
	g.mu.Unlock()
}
