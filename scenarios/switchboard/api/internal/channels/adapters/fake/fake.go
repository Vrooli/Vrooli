package fake

import (
	"context"
	"sync"

	"switchboard/internal/channels"
)

type Adapter struct {
	Name    string
	Sent    []channels.Outbound
	Inbound []channels.Envelope
	mu      sync.Mutex
}

func New(id string) *Adapter  { return &Adapter{Name: id} }
func (a *Adapter) ID() string { return a.Name }
func (a *Adapter) Connect(_ context.Context, receive func(channels.Envelope) error) error {
	a.mu.Lock()
	scripted := append([]channels.Envelope(nil), a.Inbound...)
	a.mu.Unlock()
	for _, e := range scripted {
		if err := receive(e); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) Send(_ context.Context, out channels.Outbound) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Sent = append(a.Sent, out)
	return nil
}

func (a *Adapter) Probe(context.Context) channels.ProbeResult {
	return channels.ProbeResult{Available: true}
}
func (a *Adapter) SentCount() int { a.mu.Lock(); defer a.mu.Unlock(); return len(a.Sent) }
