package monetization

import "context"

// Billing is the shared decision adapter for paid consumers. It makes the
// distinction between a UX decision and an authority-side charge explicit:
// authorization never reserves or spends credits.
type Billing struct {
	Gate *Gate
}

func NewBilling(gate *Gate) *Billing { return &Billing{Gate: gate} }

func (b *Billing) Feature(ctx context.Context, identity, feature string, minimumRank int32) Decision {
	if b == nil || b.Gate == nil {
		return fallbackDecision(nil, "/settings/subscription")
	}
	return b.Gate.Feature(ctx, identity, feature, minimumRank)
}

func (b *Billing) Limit(ctx context.Context, identity, limitKey string) Decision {
	if b == nil || b.Gate == nil {
		return fallbackDecision(nil, "/settings/subscription")
	}
	return b.Gate.Meter(ctx, identity, limitKey)
}

func (b *Billing) CachedFeature(identity, feature string, minimumRank int32) Decision {
	if b == nil || b.Gate == nil {
		return fallbackDecision(nil, "/settings/subscription")
	}
	return b.Gate.CachedFeature(identity, feature, minimumRank)
}
