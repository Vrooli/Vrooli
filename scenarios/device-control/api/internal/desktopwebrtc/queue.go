package desktopwebrtc

import (
	"context"
	"errors"
	"sync"
)

var ErrQueueClosed = errors.New("desktop transport queue is closed")

// BoundedQueue is the transport backpressure policy shared by media and
// control lanes. Video may drop an old frame; control messages are rejected
// rather than allowing a slow viewer to wedge revoke/close.
type BoundedQueue[T any] struct {
	mu     sync.Mutex
	items  []T
	limit  int
	closed bool
	wake   chan struct{}
}

func NewBoundedQueue[T any](limit int) *BoundedQueue[T] {
	if limit < 1 {
		limit = 1
	}
	return &BoundedQueue[T]{limit: limit, wake: make(chan struct{}, 1)}
}

func (q *BoundedQueue[T]) Push(value T, droppable bool) (dropped bool, err error) {
	if q == nil {
		return false, ErrQueueClosed
	}
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return false, ErrQueueClosed
	}
	if len(q.items) >= q.limit {
		if !droppable {
			q.mu.Unlock()
			return false, errors.New("desktop transport queue full")
		}
		q.items = q.items[1:]
		dropped = true
	}
	q.items = append(q.items, value)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return dropped, nil
}

func (q *BoundedQueue[T]) Pop() (T, bool) {
	if q == nil {
		var zero T
		return zero, false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	value := q.items[0]
	q.items = q.items[1:]
	return value, true
}

func (q *BoundedQueue[T]) PopWait(ctx context.Context) (T, bool) {
	if q == nil {
		var zero T
		return zero, false
	}
	for {
		if value, ok := q.Pop(); ok {
			return value, true
		}
		q.mu.Lock()
		closed := q.closed
		q.mu.Unlock()
		if closed {
			var zero T
			return zero, false
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, false
		case <-q.wake:
		}
	}
}

func (q *BoundedQueue[T]) Close() {
	if q != nil {
		q.mu.Lock()
		q.closed = true
		q.items = nil
		q.mu.Unlock()
		select {
		case q.wake <- struct{}{}:
		default:
		}
	}
}
func (q *BoundedQueue[T]) Len() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
