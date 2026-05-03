package mocks

import (
	"context"
	"sync/atomic"

	"smoke-tier1/internal/store"
)

// FakePinger satisfies store.Pinger for tests that don't want a real
// database. PingErr controls the return value of PingContext — nil means
// healthy. Calls counts every PingContext invocation, useful for
// asserting the health check actually ran. Read with Calls.Load();
// the atomic type keeps go test -race quiet when handlers fan out.
type FakePinger struct {
	PingErr error
	Calls   atomic.Int64
}

// PingContext returns p.PingErr and atomically bumps p.Calls.
func (p *FakePinger) PingContext(ctx context.Context) error {
	p.Calls.Add(1)
	return p.PingErr
}

// Compile-time guarantee that *FakePinger satisfies store.Pinger.
var _ store.Pinger = (*FakePinger)(nil)
