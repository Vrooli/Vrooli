package graph

import (
	"context"
)

// DefaultMaxConcurrentExtracts protects the host from the full type-checking
// and dependency-loading cost of several different modules being extracted at
// once. Per-path serialization remains independently enforced by PathMutex.
const DefaultMaxConcurrentExtracts = 1

type extractionLimiter struct {
	slots chan struct{}
}

func newExtractionLimiter(maxConcurrent int) *extractionLimiter {
	if maxConcurrent <= 0 {
		return nil
	}
	return &extractionLimiter{slots: make(chan struct{}, maxConcurrent)}
}

func (l *extractionLimiter) acquire(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
