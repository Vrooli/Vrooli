package mocks

import (
	"context"
)

// FakePinger satisfies handlers.Pinger (and any other PingContext-style
// interface) for tests that don't want a real database. The Err field
// determines what PingContext returns; nil means healthy.
type FakePinger struct {
	Err error

	// Calls increments on every PingContext invocation. Useful for
	// tests asserting the ping was issued (e.g., health endpoint).
	Calls int
}

// NewFakePinger returns a FakePinger that reports healthy.
func NewFakePinger() *FakePinger {
	return &FakePinger{}
}

// PingContext returns p.Err and bumps p.Calls.
func (p *FakePinger) PingContext(ctx context.Context) error {
	p.Calls++
	return p.Err
}
