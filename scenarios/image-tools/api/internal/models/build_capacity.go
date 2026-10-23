package models

import (
	"context"

	"github.com/vrooli/vrooli/packages/capacity"
)

// BuildCapacityBroker is the shared native capacity admission seam. Export
// work requires an exact, no-reclaim RAM claim and retains the shared lease
// until the exporter exits.
type BuildCapacityBroker interface {
	AcquireNoReclaim(context.Context, string, int64, int64) (capacity.Lease, capacity.Verdict, error)
	Close() error
}

// BuildCapacityBrokerFactory keeps the shared ledger connection scoped to one
// durable build attempt while allowing tests to provide a deterministic broker.
type BuildCapacityBrokerFactory func(context.Context) (BuildCapacityBroker, error)
