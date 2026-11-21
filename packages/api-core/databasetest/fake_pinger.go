package databasetest

import (
	"context"
	"sync/atomic"

	apidb "github.com/vrooli/api-core/database"
)

// FakePinger is a thread-safe database ping fake for API tests. PingErr
// controls the result, while Calls records every invocation.
type FakePinger struct {
	PingErr error
	Calls   atomic.Int64
}

func (p *FakePinger) PingContext(context.Context) error {
	p.Calls.Add(1)
	return p.PingErr
}

var _ apidb.Pinger = (*FakePinger)(nil)
