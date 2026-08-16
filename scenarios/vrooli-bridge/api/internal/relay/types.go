// Package relay owns the short-lived, typed command relay over an already
// authenticated node channel. It deliberately has no protobuf dependency;
// handlers translate the wire envelope at the boundary.
package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"vrooli-bridge/internal/dispatch"
)

const (
	DefaultMaxResponseBytes uint64 = 1 << 20
	MaxResponseBytes        uint64 = 8 << 20
)

const (
	KindAccepted   = "accepted"
	KindData       = "data"
	KindCompleted  = "completed"
	KindFailed     = "failed"
	KindTerminated = "terminated"
)

const ResponseLimitReason = "relay response exceeds byte limit"

type Request struct {
	CorrelationID    string
	Actor            string
	NodeID           string
	Scenario         string
	Command          string
	Args             []string
	TimeoutSeconds   int64
	MaxResponseBytes uint64
}

type Response struct {
	CorrelationID string
	Kind          string
	Sequence      uint64
	Data          []byte
	Reason        string
	ExitCode      int32
	TotalBytes    uint64
}

// NodeReader and Presence are aliases to the same seams used by durable
// dispatch. Keeping the relay constructor on those seams makes parity explicit
// and lets unit tests exercise it without a registry or channel server.
type (
	NodeReader = dispatch.NodeReader
	Presence   = dispatch.Presence
)

// Pusher is the wire adapter. Push and Cancel must only carry typed, signed
// frames; the relay service never constructs protobufs or shell strings.
type Pusher interface {
	Push(ctx context.Context, nodeID string, request Request) (delivered int, err error)
	Cancel(ctx context.Context, nodeID, correlationID, reason string) (delivered int, err error)
}

var (
	ErrInvalidRequest       = errors.New("relay request is invalid")
	ErrResponseBackpressure = errors.New("relay response broker is full")
	ErrCorrelationConflict  = errors.New("relay correlation id is already in use")
)

type ErrResponseLimit struct {
	Limit uint64
}

func (e ErrResponseLimit) Error() string {
	return fmt.Sprintf("%s: limit=%d", ResponseLimitReason, e.Limit)
}

type waiter struct {
	nodeID string
	ch     chan Response
}

// Broker is the authenticated response rendezvous. It binds a correlation to
// the node that requested it and uses a bounded channel so a noisy node cannot
// consume unbounded control-plane memory.
type Broker struct {
	mu      sync.Mutex
	waiters map[string]waiter
}

func NewBroker() *Broker { return &Broker{waiters: make(map[string]waiter)} }

func (b *Broker) Register(correlationID, nodeID string) (<-chan Response, func(), error) {
	if correlationID == "" || nodeID == "" {
		return nil, nil, ErrInvalidRequest
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.waiters[correlationID]; exists {
		return nil, nil, ErrCorrelationConflict
	}
	ch := make(chan Response, 16)
	b.waiters[correlationID] = waiter{nodeID: nodeID, ch: ch}
	return ch, func() {
		b.mu.Lock()
		delete(b.waiters, correlationID)
		b.mu.Unlock()
	}, nil
}

func (b *Broker) Deliver(_ context.Context, nodeID string, response Response) error {
	if response.CorrelationID == "" || nodeID == "" {
		return ErrInvalidRequest
	}
	b.mu.Lock()
	w, ok := b.waiters[response.CorrelationID]
	if !ok || w.nodeID != nodeID {
		b.mu.Unlock()
		return ErrInvalidRequest
	}
	select {
	case w.ch <- cloneResponse(response):
		b.mu.Unlock()
		return nil
	default:
		b.mu.Unlock()
		return ErrResponseBackpressure
	}
}

func cloneResponse(response Response) Response {
	response.Data = append([]byte(nil), response.Data...)
	return response
}
