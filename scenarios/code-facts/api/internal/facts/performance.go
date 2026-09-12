package facts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FamilyCost records the measured wall-clock cost of a family on a named
// target. Keeping the measurements explicit makes regressions reviewable and
// lets tests assert that an intentionally lowered budget fails loudly.
type FamilyCost struct {
	Family string
	Target string
	ColdMS int64
	WarmMS int64
}

func AssertFamilyCost(cost FamilyCost, coldBudgetMS, warmBudgetMS int64) error {
	if cost.ColdMS > coldBudgetMS {
		return fmt.Errorf("%s cold cost %dms exceeds %dms budget", cost.Family, cost.ColdMS, coldBudgetMS)
	}
	if cost.WarmMS > warmBudgetMS {
		return fmt.Errorf("%s warm cost %dms exceeds %dms budget", cost.Family, cost.WarmMS, warmBudgetMS)
	}
	return nil
}

// AdmissionSnapshot is a constant-time view of the shared expensive-work
// budget. It is intentionally small enough to expose from health without
// walking jobs, queues, or cache rows.
type AdmissionSnapshot struct {
	Capacity       int
	InUse          int
	HighWater      int
	Queued         int
	QueueHighWater int
	Admitted       uint64
	Rejected       uint64
	Cancelled      uint64
	WaitP50MS      int64
	WaitP95MS      int64
	WaitP99MS      int64
}

type admissionWaiter struct {
	weight  int
	granted chan struct{}
	started time.Time
}

// WeightedAdmission is the single process-wide budget for CPU- and
// memory-heavy work. FIFO granting avoids starvation; waiters consume no
// capacity until their complete weight is available, and cancellation removes
// them from the queue immediately.
type WeightedAdmission struct {
	mu          sync.Mutex
	capacity    int
	maxQueued   int
	wait        time.Duration
	inUse       int
	highWater   int
	queueHigh   int
	admitted    uint64
	rejected    uint64
	cancelled   uint64
	waitBuckets [11]uint64
	queue       []*admissionWaiter
}

func NewWeightedAdmission(capacity, maxQueued int, wait time.Duration) *WeightedAdmission {
	if capacity < 1 {
		capacity = 16
	}
	if maxQueued < 1 {
		maxQueued = 64
	}
	if wait <= 0 {
		wait = 2 * time.Second
	}
	return &WeightedAdmission{capacity: capacity, maxQueued: maxQueued, wait: wait}
}

func (a *WeightedAdmission) Acquire(ctx context.Context, workload string, weight int) (func(), error) {
	if a == nil {
		return func() {}, nil
	}
	if weight <= 0 {
		weight = workloadWeight(workload)
	}
	if weight > a.capacity {
		return nil, fmt.Errorf("%s work weight %d exceeds capacity %d", workload, weight, a.capacity)
	}
	a.mu.Lock()
	if len(a.queue) == 0 && a.inUse+weight <= a.capacity {
		a.inUse += weight
		a.admitted++
		a.recordWaitLocked(0)
		if a.inUse > a.highWater {
			a.highWater = a.inUse
		}
		a.mu.Unlock()
		return a.release(weight), nil
	}
	if len(a.queue) >= a.maxQueued {
		a.rejected++
		a.mu.Unlock()
		return nil, fmt.Errorf("%s work queue is full", workload)
	}
	w := &admissionWaiter{weight: weight, granted: make(chan struct{}), started: time.Now()}
	a.queue = append(a.queue, w)
	if len(a.queue) > a.queueHigh {
		a.queueHigh = len(a.queue)
	}
	a.mu.Unlock()

	waitCtx, cancel := context.WithTimeout(ctx, a.wait)
	defer cancel()
	select {
	case <-w.granted:
		return a.release(weight), nil
	case <-waitCtx.Done():
		a.mu.Lock()
		for i, queued := range a.queue {
			if queued == w {
				a.queue = append(a.queue[:i], a.queue[i+1:]...)
				a.cancelled++
				a.grantLocked()
				a.mu.Unlock()
				return nil, fmt.Errorf("%s admission wait: %w", workload, waitCtx.Err())
			}
		}
		// Grant won the race. Capacity belongs to this caller and must be
		// returned even when cancellation became observable simultaneously.
		a.mu.Unlock()
		return a.release(weight), nil
	}
}

func (a *WeightedAdmission) release(weight int) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			a.mu.Lock()
			a.inUse -= weight
			if a.inUse < 0 {
				a.inUse = 0
			}
			a.grantLocked()
			a.mu.Unlock()
		})
	}
}

func (a *WeightedAdmission) grantLocked() {
	for len(a.queue) > 0 && a.inUse+a.queue[0].weight <= a.capacity {
		w := a.queue[0]
		a.queue = a.queue[1:]
		a.inUse += w.weight
		a.admitted++
		a.recordWaitLocked(time.Since(w.started))
		if a.inUse > a.highWater {
			a.highWater = a.inUse
		}
		close(w.granted)
	}
}

func (a *WeightedAdmission) Snapshot() AdmissionSnapshot {
	if a == nil {
		return AdmissionSnapshot{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return AdmissionSnapshot{Capacity: a.capacity, InUse: a.inUse, HighWater: a.highWater, Queued: len(a.queue), QueueHighWater: a.queueHigh, Admitted: a.admitted, Rejected: a.rejected, Cancelled: a.cancelled, WaitP50MS: a.percentileWaitLocked(50), WaitP95MS: a.percentileWaitLocked(95), WaitP99MS: a.percentileWaitLocked(99)}
}

var admissionWaitBounds = [...]time.Duration{0, time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second}

func (a *WeightedAdmission) recordWaitLocked(wait time.Duration) {
	for index, bound := range admissionWaitBounds {
		if wait <= bound {
			a.waitBuckets[index]++
			return
		}
	}
	a.waitBuckets[len(a.waitBuckets)-1]++
}

func (a *WeightedAdmission) percentileWaitLocked(percentile uint64) int64 {
	var total uint64
	for _, count := range a.waitBuckets {
		total += count
	}
	if total == 0 {
		return 0
	}
	target := (total*percentile + 99) / 100
	var seen uint64
	for index, count := range a.waitBuckets {
		seen += count
		if seen >= target {
			return admissionWaitBounds[index].Milliseconds()
		}
	}
	return admissionWaitBounds[len(admissionWaitBounds)-1].Milliseconds()
}

func workloadWeight(workload string) int {
	switch workload {
	case "indexing":
		return 12
	case "graph", "fleet":
		return 8
	case "embedding", "fallback":
		return 4
	default:
		return 1
	}
}

type graphFlightCall struct {
	done    chan struct{}
	result  *GraphResult
	err     error
	waiters int
}

type graphFlightGroup struct {
	mu    sync.Mutex
	calls map[string]*graphFlightCall
}

func newGraphFlightGroup() *graphFlightGroup {
	return &graphFlightGroup{calls: map[string]*graphFlightCall{}}
}

func (g *graphFlightGroup) Do(ctx context.Context, key string, fn func(context.Context) (*GraphResult, error)) (*GraphResult, error) {
	g.mu.Lock()
	if call := g.calls[key]; call != nil {
		call.waiters++
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return call.result, call.err
		}
	}
	call := &graphFlightCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()
	call.result, call.err = fn(ctx)
	g.mu.Lock()
	delete(g.calls, key)
	close(call.done)
	g.mu.Unlock()
	return call.result, call.err
}

func (g *graphFlightGroup) waiterCount(key string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if call := g.calls[key]; call != nil {
		return call.waiters
	}
	return 0
}
