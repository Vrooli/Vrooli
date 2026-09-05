package adapters

import (
	"context"
	"sync"

	"switchboard/internal/channels"
)

// Transport is a deterministic transport fixture used by the built-in
// adapters. Production transports can replace it without changing core code.
type Transport struct {
	Name    string
	mu      sync.Mutex
	Sent    []channels.Outbound
	Receive func(channels.Envelope) error
}

func NewTransport(id string) *Transport { return &Transport{Name: id} }
func (a *Transport) ID() string         { return a.Name }
func (a *Transport) Connect(_ context.Context, receive func(channels.Envelope) error) error {
	a.mu.Lock()
	a.Receive = receive
	a.mu.Unlock()
	return nil
}

func (a *Transport) Send(_ context.Context, out channels.Outbound) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Sent = append(a.Sent, out)
	return nil
}

func (a *Transport) Probe(context.Context) channels.ProbeResult {
	return channels.ProbeResult{Available: true}
}
