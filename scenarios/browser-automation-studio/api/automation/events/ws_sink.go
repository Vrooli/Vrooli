package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// WSHubSink bridges contract event envelopes to the websocket hub while
// preserving per-execution ordering and drop policy.
type WSHubSink struct {
	hub    wsHub.HubInterface
	limits contracts.EventBufferLimits
	log    *logrus.Logger

	mu     sync.Mutex
	queues map[uuid.UUID]*executionQueue
	closed map[uuid.UUID]struct{}
}

// NewWSHubSink constructs a WSHubSink. Limits are carried for interface
// completeness; backpressure is handled upstream.
func NewWSHubSink(hub wsHub.HubInterface, log *logrus.Logger, limits contracts.EventBufferLimits) *WSHubSink {
	if limits.Validate() != nil {
		limits = contracts.DefaultEventBufferLimits
	}
	return &WSHubSink{
		hub:    hub,
		limits: limits,
		log:    log,
		queues: make(map[uuid.UUID]*executionQueue),
		closed: make(map[uuid.UUID]struct{}),
	}
}

// Publish pushes the contract envelope into the websocket hub with ordering and
// backpressure handled by the per-execution queue.
func (s *WSHubSink) Publish(_ context.Context, event contracts.EventEnvelope) error {
	if s == nil || s.hub == nil {
		return nil
	}
	if s.isClosed(event.ExecutionID) {
		return nil
	}

	queue := s.ensureQueue(event.ExecutionID)
	if queue == nil {
		return nil
	}
	queue.enqueue(event)
	return nil
}

// Limits returns configured limits (best-effort parity with EventSink).
func (s *WSHubSink) Limits() contracts.EventBufferLimits {
	return s.limits
}

func (s *WSHubSink) ensureQueue(executionID uuid.UUID) *executionQueue {
	if s.isClosed(executionID) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if queue, ok := s.queues[executionID]; ok {
		return queue
	}

	queue := newExecutionQueue(executionID, s.hub, s.limits, s.log)
	s.queues[executionID] = queue
	go queue.run()
	return queue
}

// CloseExecution drains and closes the per-execution queue to avoid goroutine
// leaks after a run finishes.
func (s *WSHubSink) CloseExecution(executionID uuid.UUID) {
	if s == nil {
		return
	}
	s.mu.Lock()
	queue, ok := s.queues[executionID]
	if ok {
		delete(s.queues, executionID)
	}
	s.closed[executionID] = struct{}{}
	s.mu.Unlock()
	if ok && queue != nil {
		queue.close()
	}
	if s.hub != nil {
		s.hub.CloseExecution(executionID)
	}
}

type executionQueue struct {
	executionID uuid.UUID
	hub         wsHub.HubInterface
	limits      contracts.EventBufferLimits
	log         *logrus.Logger

	mu          sync.Mutex
	cond        *sync.Cond
	events      []contracts.EventEnvelope
	perAttempt  map[string]int
	inflight    int
	dropCounter contracts.DropCounters
	closed      bool
}

func newExecutionQueue(executionID uuid.UUID, hub wsHub.HubInterface, limits contracts.EventBufferLimits, log *logrus.Logger) *executionQueue {
	q := &executionQueue{
		executionID: executionID,
		hub:         hub,
		limits:      limits,
		log:         log,
		perAttempt:  make(map[string]int),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *executionQueue) enqueue(event contracts.EventEnvelope) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	// Drop oldest droppable events when execution or attempt buffers are full.
	currentSize := len(q.events) + q.inflight
	if currentSize >= q.limits.PerExecution && isDroppable(event.Kind) {
		if droppedSeq, ok := q.dropOldest(nil, nil); ok && droppedSeq > 0 && q.dropCounter.OldestDropped == 0 {
			q.dropCounter.OldestDropped = droppedSeq
		}
	}
	if event.StepIndex != nil && event.Attempt != nil && isDroppable(event.Kind) {
		key := attemptKey(*event.StepIndex, *event.Attempt)
		if q.perAttempt[key] >= q.limits.PerAttempt {
			if droppedSeq, ok := q.dropOldest(event.StepIndex, event.Attempt); ok && droppedSeq > 0 && q.dropCounter.OldestDropped == 0 {
				q.dropCounter.OldestDropped = droppedSeq
			}
		}
	}

	// Attach drop counters to the event if any were accumulated.
	if q.dropCounter.Dropped > 0 {
		event.Drops = q.dropCounter
	}

	if event.StepIndex != nil && event.Attempt != nil {
		q.perAttempt[attemptKey(*event.StepIndex, *event.Attempt)]++
	}

	q.events = append(q.events, event)
	q.cond.Signal()
}

func (q *executionQueue) run() {
	for {
		q.mu.Lock()
		for len(q.events) == 0 && !q.closed {
			q.cond.Wait()
		}
		if q.closed && len(q.events) == 0 {
			q.mu.Unlock()
			return
		}
		event := q.events[0]
		q.events = q.events[1:]
		q.inflight++
		q.mu.Unlock()

		q.emit(event)

		q.mu.Lock()
		q.inflight--
		if event.StepIndex != nil && event.Attempt != nil {
			key := attemptKey(*event.StepIndex, *event.Attempt)
			if q.perAttempt[key] > 0 {
				q.perAttempt[key]--
			}
		}
		q.mu.Unlock()
	}
}

func (q *executionQueue) emit(event contracts.EventEnvelope) {
	q.hub.BroadcastEnvelope(event)
}

func (q *executionQueue) dropOldest(stepIndex, attempt *int) (uint64, bool) {
	for i, ev := range q.events {
		if !isDroppable(ev.Kind) {
			continue
		}
		if stepIndex != nil && attempt != nil {
			if ev.StepIndex == nil || ev.Attempt == nil || *ev.StepIndex != *stepIndex || *ev.Attempt != *attempt {
				continue
			}
		}
		seq := ev.Sequence
		q.events = append(q.events[:i], q.events[i+1:]...)
		if ev.StepIndex != nil && ev.Attempt != nil {
			key := attemptKey(*ev.StepIndex, *ev.Attempt)
			if q.perAttempt[key] > 0 {
				q.perAttempt[key]--
			}
		}
		q.dropCounter.Dropped++
		return seq, true
	}
	return 0, false
}

func attemptKey(stepIndex, attempt int) string {
	return fmt.Sprintf("%d:%d", stepIndex, attempt)
}

func (q *executionQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.events = nil
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (s *WSHubSink) isClosed(executionID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.closed[executionID]
	return ok
}
