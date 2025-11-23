package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	browserlessEvents "github.com/vrooli/browser-automation-studio/browserless/events"
)

// WSHubSink bridges the new event envelopes to the existing WebSocket hub.
type WSHubSink struct {
	emitter *browserlessEvents.Emitter
	limits  contracts.EventBufferLimits
	log     *logrus.Logger

	mu     sync.Mutex
	queues map[uuid.UUID]*executionQueue
}

// NewWSHubSink constructs a WSHubSink. Limits are carried for interface
// completeness; backpressure is handled upstream.
func NewWSHubSink(hub browserlessEvents.UpdateBroadcaster, log *logrus.Logger, limits contracts.EventBufferLimits) *WSHubSink {
	if limits.Validate() != nil {
		limits = contracts.DefaultEventBufferLimits
	}
	return &WSHubSink{
		emitter: browserlessEvents.NewEmitter(hub, log),
		limits:  limits,
		log:     log,
		queues:  make(map[uuid.UUID]*executionQueue),
	}
}

// Publish adapts the envelope to the legacy event shape. Payload is wrapped
// into the event payload map under "data" to avoid altering consumers.
func (s *WSHubSink) Publish(_ context.Context, event contracts.EventEnvelope) error {
	if s == nil || s.emitter == nil {
		return nil
	}

	queue := s.ensureQueue(event.ExecutionID)
	queue.enqueue(event)
	return nil
}

// Limits returns configured limits (best-effort parity with EventSink).
func (s *WSHubSink) Limits() contracts.EventBufferLimits {
	return s.limits
}

func (s *WSHubSink) ensureQueue(executionID uuid.UUID) *executionQueue {
	s.mu.Lock()
	defer s.mu.Unlock()

	if queue, ok := s.queues[executionID]; ok {
		return queue
	}

	queue := newExecutionQueue(executionID, s.emitter, s.limits, s.log)
	s.queues[executionID] = queue
	go queue.run()
	return queue
}

type executionQueue struct {
	executionID uuid.UUID
	emitter     *browserlessEvents.Emitter
	limits      contracts.EventBufferLimits
	log         *logrus.Logger

	mu          sync.Mutex
	cond        *sync.Cond
	events      []contracts.EventEnvelope
	perAttempt  map[string]int
	inflight    int
	dropCounter contracts.DropCounters
}

func newExecutionQueue(executionID uuid.UUID, emitter *browserlessEvents.Emitter, limits contracts.EventBufferLimits, log *logrus.Logger) *executionQueue {
	q := &executionQueue{
		executionID: executionID,
		emitter:     emitter,
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
		for len(q.events) == 0 {
			q.cond.Wait()
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
	payload := map[string]any{
		"data":      event.Payload,
		"sequence":  event.Sequence,
		"timestamp": event.Timestamp.Format(time.RFC3339Nano),
	}
	if event.Drops.Dropped > 0 {
		payload["drops"] = event.Drops
	}

	stepIndex := -1
	if event.StepIndex != nil {
		stepIndex = *event.StepIndex
	}

	ev := browserlessEvents.NewEvent(
		adaptEventKind(event.Kind),
		event.ExecutionID,
		event.WorkflowID,
		browserlessEvents.WithStep(stepIndex, stepNodeID(event), ""), // Step type not critical for bridge.
		browserlessEvents.WithStatus("event"),
		browserlessEvents.WithMessage(string(event.Kind)),
		browserlessEvents.WithPayload(payload),
	)
	q.emitter.Emit(ev)
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

func adaptEventKind(kind contracts.EventKind) browserlessEvents.EventType {
	switch kind {
	case contracts.EventKindExecutionStarted:
		return browserlessEvents.EventExecutionStarted
	case contracts.EventKindExecutionProgress:
		return browserlessEvents.EventExecutionProgress
	case contracts.EventKindExecutionCompleted:
		return browserlessEvents.EventExecutionCompleted
	case contracts.EventKindExecutionFailed:
		return browserlessEvents.EventExecutionFailed
	case contracts.EventKindExecutionCancelled:
		return browserlessEvents.EventExecutionCancelled
	case contracts.EventKindStepStarted:
		return browserlessEvents.EventStepStarted
	case contracts.EventKindStepCompleted:
		return browserlessEvents.EventStepCompleted
	case contracts.EventKindStepFailed:
		return browserlessEvents.EventStepFailed
	case contracts.EventKindStepScreenshot:
		return browserlessEvents.EventStepScreenshot
	case contracts.EventKindStepHeartbeat:
		return browserlessEvents.EventStepHeartbeat
	default:
		return browserlessEvents.EventStepTelemetry
	}
}

func stepNodeID(event contracts.EventEnvelope) string {
	if event.StepIndex != nil {
		return fmt.Sprintf("step-%d", *event.StepIndex)
	}
	return ""
}
