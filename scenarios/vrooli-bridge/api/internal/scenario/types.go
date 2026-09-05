// Package scenario owns the Bridge target-aware unary scenario proxy. It is
// transport-neutral: handlers translate HTTP/Connect at the edge, while this
// package enforces node ownership, catalog admission, correlation, and bounds.
package scenario

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const (
	DefaultMaxResponseBytes uint64 = 8 << 20
	MaxResponseBytes        uint64 = 8 << 20
)

type Request struct {
	CorrelationID    string
	Actor            string
	NodeID           string
	Scenario         string
	Service          string
	Method           string
	HTTPMethod       string
	HTTPPath         string
	Body             []byte
	TimeoutSeconds   int64
	MaxResponseBytes uint64
}

type Response struct {
	CorrelationID string
	Body          []byte
	Error         string
	TimedOut      bool
	Truncated     bool
}

type TargetNode struct {
	ID      string
	Scopes  []string
	Revoked bool
}

type NodeReader interface {
	GetTarget(context.Context, string) (TargetNode, error)
}

type Presence interface {
	IsOnline(string) bool
	Dispatchable(string) bool
}

type Pusher interface {
	Push(context.Context, string, Request) (int, error)
}

type Admission func(Request, TargetNode) error

type waiter struct {
	nodeID string
	ch     chan Response
}

type Broker struct {
	mu      sync.Mutex
	waiters map[string]waiter
}

func NewBroker() *Broker { return &Broker{waiters: make(map[string]waiter)} }

func (b *Broker) Register(id, nodeID string) (<-chan Response, func(), error) {
	if id == "" || nodeID == "" {
		return nil, nil, errors.New("scenario correlation and node are required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.waiters[id]; exists {
		return nil, nil, fmt.Errorf("scenario correlation %q is already in use", id)
	}
	ch := make(chan Response, 1)
	b.waiters[id] = waiter{nodeID: nodeID, ch: ch}
	return ch, func() {
		b.mu.Lock()
		delete(b.waiters, id)
		b.mu.Unlock()
	}, nil
}

func (b *Broker) Deliver(nodeID string, response Response) error {
	if nodeID == "" || response.CorrelationID == "" {
		return errors.New("scenario response correlation and node are required")
	}
	b.mu.Lock()
	waiter, ok := b.waiters[response.CorrelationID]
	if !ok || waiter.nodeID != nodeID {
		b.mu.Unlock()
		return errors.New("scenario response does not belong to node")
	}
	response.Body = append([]byte(nil), response.Body...)
	select {
	case waiter.ch <- response:
		b.mu.Unlock()
		return nil
	default:
		b.mu.Unlock()
		return errors.New("scenario response broker is full")
	}
}
