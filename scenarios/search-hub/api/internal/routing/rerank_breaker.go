package routing

import (
	"fmt"
	"sync"
	"time"
)

type rerankBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
}

type rerankBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	cooldown         time.Duration
	failures         int
	open             bool
	openedAt         time.Time
	probing          bool
}

func newRerankBreaker(cfg rerankBreakerConfig) *rerankBreaker {
	if cfg.FailureThreshold <= 0 || cfg.Cooldown <= 0 {
		return nil
	}
	return &rerankBreaker{
		failureThreshold: cfg.FailureThreshold,
		cooldown:         cfg.Cooldown,
	}
}

func (b *rerankBreaker) allow(now time.Time) (bool, string) {
	if b == nil {
		return true, ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.open {
		return true, ""
	}
	if now.Sub(b.openedAt) >= b.cooldown {
		if b.probing {
			return false, "reranker circuit half-open (probe in flight) - showing honest by-provider grouping"
		}
		b.probing = true
		return true, ""
	}

	retryAt := b.openedAt.Add(b.cooldown)
	return false, fmt.Sprintf("reranker circuit open after %d consecutive failure(s); retry after %s - showing honest by-provider grouping",
		b.failures, retryAt.Format(time.RFC3339))
}

func (b *rerankBreaker) recordSuccess() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.open = false
	b.openedAt = time.Time{}
	b.probing = false
}

func (b *rerankBreaker) recordFailure(now time.Time) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.open {
		b.failures++
		b.openedAt = now
		b.probing = false
		return
	}
	b.failures++
	if b.failures >= b.failureThreshold {
		b.open = true
		b.openedAt = now
		b.probing = false
	}
}
