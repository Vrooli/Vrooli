// Responsibility: the named fallback price source for models the price book
// cannot price. Consumers that need a relative price (usage attribution under
// a subscription) read it here instead of carrying their own constants.
package pricing

import "strings"

// TokenPrices is a model's price per token, by token category.
type TokenPrices struct {
	Input, Output, CacheRead, CacheCreation float64
}

// Cache ratios providers commonly apply: cached reads cost a tenth of input
// (input divided by cacheReadInputDivisor), cache writes 1.25 times input.
const (
	cacheReadInputDivisor   = 10
	cacheCreationInputRatio = 1.25
)

// FallbackTokenPrices prices unpriced models: API ratios
// cached:input:creation:output of 1:20:25:100 at a mid-tier (terra-like) level.
func FallbackTokenPrices() TokenPrices {
	return TokenPrices{CacheRead: 5e-8, Input: 1e-6, CacheCreation: 1.25e-6, Output: 5e-6}
}

// tierPriors scale FallbackTokenPrices by model tier when the price book has
// no price for a subscription model (the runner resources resolve them to
// unpriced subscription names). Codex tiers are the per-model fit on this host
// (cached tokens per 1%: sol ≈50M, terra ≈100M, luna ≈520M); Claude tiers
// follow Anthropic's list-price ratios. Only the ratios matter: consumers
// calibrate the scale per window. Every ratio here is a prior until a window
// fits it; the attribution output labels it so.
var tierPriors = []struct {
	marker   string
	relative float64
}{{"sol", 2}, {"terra", 1}, {"luna", 0.2}, {"opus", 1}, {"sonnet", 0.5}, {"haiku", 0.05}}

// TierTokenPrices is the prior price of an unpriced model, and false when the
// model names no known tier.
func TierTokenPrices(model string) (TokenPrices, bool) {
	model = strings.ToLower(model)
	for _, tier := range tierPriors {
		if strings.Contains(model, tier.marker) {
			p := FallbackTokenPrices()
			p.CacheRead, p.Input, p.CacheCreation, p.Output = p.CacheRead*tier.relative, p.Input*tier.relative, p.CacheCreation*tier.relative, p.Output*tier.relative
			return p, true
		}
	}
	return TokenPrices{}, false
}

// cheaperTier maps a dearer tier to the next cheaper tier of the same family
// that a session could move to (opus to sonnet, sol to terra, terra to luna).
var cheaperTier = map[string]string{"opus": "sonnet", "sol": "terra", "terra": "luna"}

// CheaperTier names the next cheaper tier marker a model could move to and
// the prior price ratio of that tier to the model. The ratio is a prior from
// tierPriors, never a fitted value. It returns false for models with no
// cheaper tier.
func CheaperTier(model string) (marker string, ratio float64, ok bool) {
	model = strings.ToLower(model)
	for _, tier := range tierPriors {
		if !strings.Contains(model, tier.marker) {
			continue
		}
		next, found := cheaperTier[tier.marker]
		if !found {
			return "", 0, false
		}
		for _, other := range tierPriors {
			if other.marker == next {
				return next, other.relative / tier.relative, true
			}
		}
	}
	return "", 0, false
}

// WithDefaultCacheRatios fills the cache prices a price book often omits.
func (p TokenPrices) WithDefaultCacheRatios() TokenPrices {
	if p.CacheRead <= 0 {
		p.CacheRead = p.Input / cacheReadInputDivisor
	}
	if p.CacheCreation <= 0 {
		p.CacheCreation = p.Input * cacheCreationInputRatio
	}
	return p
}
