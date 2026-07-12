package routing

import (
	"fmt"
	"sync"
	"time"
)

// providerBreakers keeps only short-lived availability state. It deliberately
// never caches provider results or corpus content: Search Hub remains a thin
// router and providers stay authoritative for every returned hit.
type providerBreakers struct {
	mu     sync.Mutex
	config rerankBreakerConfig
	byID   map[string]*rerankBreaker
}

func newProviderBreakers(cfg rerankBreakerConfig) *providerBreakers {
	if cfg.FailureThreshold <= 0 || cfg.Cooldown <= 0 {
		return nil
	}
	return &providerBreakers{config: cfg, byID: make(map[string]*rerankBreaker)}
}

func (p *providerBreakers) breaker(id string) *rerankBreaker {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if b := p.byID[id]; b != nil {
		return b
	}
	b := newRerankBreaker(p.config)
	p.byID[id] = b
	return b
}

func (p *providerBreakers) allow(id string, now time.Time) (bool, string) {
	b := p.breaker(id)
	if b == nil {
		return true, ""
	}
	ok, note := b.allow(now)
	if ok {
		return true, ""
	}
	return false, fmt.Sprintf("provider %s circuit unavailable: %s", id, note)
}

func (p *providerBreakers) record(id string, degraded bool, now time.Time) {
	b := p.breaker(id)
	if b == nil {
		return
	}
	if degraded {
		b.recordFailure(now)
		return
	}
	b.recordSuccess()
}

func (p *providerBreakers) status(id string, now time.Time) (bool, string) {
	b := p.breaker(id)
	if b == nil {
		return false, ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return false, ""
	}
	if now.Sub(b.openedAt) >= b.cooldown {
		return true, "provider circuit recovery probe is due"
	}
	return true, fmt.Sprintf("provider circuit open after %d consecutive failure(s); retry after %s", b.failures, b.openedAt.Add(b.cooldown).Format(time.RFC3339))
}
