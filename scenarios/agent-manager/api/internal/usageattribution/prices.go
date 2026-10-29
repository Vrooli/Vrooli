package usageattribution

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/pricing"
)

// ModelPricer is the price book lookup (pricing.Service).
type ModelPricer interface {
	GetModelPricing(ctx context.Context, model, runnerType string) (*pricing.ModelPricing, error)
}

// priceTTL bounds how long a looked-up price is reused across requests; it
// matches the model-resolution memo, and the price book refreshes on a
// schedule, not per request.
const priceTTL = 10 * time.Minute

// PriceBook memoizes model prices across requests. A lookup of an unpriced
// model walks the pricing fallbacks (seconds each on a large database), so a
// request looks up the models it needs concurrently and later requests reuse
// them; a price older than priceTTL still serves while it is refreshed in the
// background. A nil PriceBook prices every model by its tier prior.
type PriceBook struct {
	pricer  ModelPricer
	now     func() time.Time
	mu      sync.Mutex
	entries map[string]priceEntry
}

type priceEntry struct {
	price Price
	at    time.Time
}

// NewPriceBook memoizes pricer; a nil pricer prices by tier priors.
func NewPriceBook(pricer ModelPricer) *PriceBook {
	return &PriceBook{pricer: pricer, now: time.Now, entries: map[string]priceEntry{}}
}

// prices returns the request's lookup, after fetching the models the turns
// and outcomes name concurrently.
func (b *PriceBook) prices(ctx context.Context, turns []Turn, outcomes []Outcome) Prices {
	if b == nil {
		return func(_, model string) Price { return TierPrice(model) }
	}
	wanted := map[[2]string]bool{}
	for _, turn := range turns {
		wanted[[2]string{turn.Runner, turn.Model}] = true
	}
	for _, outcome := range outcomes {
		wanted[[2]string{outcome.Runner, outcome.Model}] = true
	}
	var wg sync.WaitGroup
	for pair := range wanted {
		if _, known, fresh := b.cached(pair[0], pair[1]); fresh {
			continue
		} else if known {
			// A stale price serves this request; it is refreshed for the next.
			go b.lookup(context.WithoutCancel(ctx), pair[0], pair[1])
			continue
		}
		wg.Add(1)
		go func(runner, model string) {
			defer wg.Done()
			b.lookup(ctx, runner, model)
		}(pair[0], pair[1])
	}
	wg.Wait()
	return func(runner, model string) Price {
		if price, known, _ := b.cached(runner, model); known {
			return price
		}
		return b.lookup(ctx, runner, model)
	}
}

// cached reports the memoized price, whether one exists, and whether it is
// younger than priceTTL.
func (b *PriceBook) cached(runner, model string) (Price, bool, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.entries[runner+"\x00"+model]
	return entry.price, ok, ok && b.now().Sub(entry.at) < priceTTL
}

func (b *PriceBook) lookup(ctx context.Context, runner, model string) Price {
	var price Price
	if b.pricer != nil && model != "" && model != "unknown" {
		if book, err := b.pricer.GetModelPricing(ctx, model, runner); err == nil && book != nil && book.InputTokenPrice != nil {
			price = Price{Input: *book.InputTokenPrice, Known: true}
			if book.OutputTokenPrice != nil {
				price.Output = *book.OutputTokenPrice
			}
			if book.CacheReadPrice != nil {
				price.CacheRead = *book.CacheReadPrice
			}
			if book.CacheCreationPrice != nil {
				price.CacheCreation = *book.CacheCreationPrice
			}
		}
	}
	if !price.Known {
		price = TierPrice(model)
	}
	b.mu.Lock()
	b.entries[runner+"\x00"+model] = priceEntry{price: price, at: b.now()}
	b.mu.Unlock()
	return price
}
