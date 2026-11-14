package goals

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"golang.org/x/sync/singleflight"
)

const (
	// DefaultOfferDeskCacheTTL bounds how stale a release-ladder read may be.
	// Rank changes are operator edits measured in minutes, so a short window
	// costs nothing in freshness while collapsing per-ref reads to one RPC.
	DefaultOfferDeskCacheTTL = 15 * time.Second
	// DefaultOfferDeskNegativeTTL keeps an Offer Desk outage from becoming a
	// retry storm: a failed read is remembered briefly instead of re-issued
	// by every caller.
	DefaultOfferDeskNegativeTTL = 5 * time.Second
	// DefaultOfferDeskTimeout caps a single Offer Desk RPC when the caller
	// supplies no HTTP client of its own.
	DefaultOfferDeskTimeout = 10 * time.Second
	// OfferDeskCacheTTLEnv overrides DefaultOfferDeskCacheTTL (a Go duration).
	OfferDeskCacheTTLEnv = "SWARM_OFFER_DESK_CACHE_TTL"
)

// OfferDeskRankReader resolves the current release rank from Offer Desk at
// read time. It deliberately does not copy rank into Swarm Manager storage;
// the only state it keeps is a short-lived read cache so that callers which
// resolve many goals per tick share one RPC instead of issuing one each.
type OfferDeskRankReader struct {
	client offersconnect.CatalogServiceClient
	ladder offersconnect.ReleaseLadderServiceClient

	ttl         time.Duration
	negativeTTL time.Duration
	now         func() time.Time

	flight   singleflight.Group
	mu       sync.Mutex
	nodes    cachedResult[[]*offersv1.Node]
	enabling cachedResult[[]*offersv1.PrerequisiteNode]
}

type cachedResult[T any] struct {
	value   T
	err     error
	expires time.Time
}

// OfferDeskRankReaderOption tunes a reader at construction.
type OfferDeskRankReaderOption func(*OfferDeskRankReader)

// WithOfferDeskCacheTTL sets the positive cache TTL. Zero or negative disables
// positive caching.
func WithOfferDeskCacheTTL(ttl time.Duration) OfferDeskRankReaderOption {
	return func(r *OfferDeskRankReader) { r.ttl = ttl }
}

// WithOfferDeskNegativeTTL sets how long a failed read is remembered. Zero or
// negative disables negative caching.
func WithOfferDeskNegativeTTL(ttl time.Duration) OfferDeskRankReaderOption {
	return func(r *OfferDeskRankReader) { r.negativeTTL = ttl }
}

// WithOfferDeskClock injects the clock used for cache expiry (tests).
func WithOfferDeskClock(now func() time.Time) OfferDeskRankReaderOption {
	return func(r *OfferDeskRankReader) {
		if now != nil {
			r.now = now
		}
	}
}

func NewOfferDeskRankReader(ctx context.Context, opts ...OfferDeskRankReaderOption) (*OfferDeskRankReader, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "offer-desk")
	if err != nil {
		return nil, err
	}
	return NewOfferDeskRankReaderAt(base, nil, opts...), nil
}

func NewOfferDeskRankReaderAt(base string, client *http.Client, opts ...OfferDeskRankReaderOption) *OfferDeskRankReader {
	if client == nil {
		client = &http.Client{Timeout: DefaultOfferDeskTimeout}
	}
	r := &OfferDeskRankReader{
		client:      offersconnect.NewCatalogServiceClient(client, strings.TrimRight(base, "/")),
		ladder:      offersconnect.NewReleaseLadderServiceClient(client, strings.TrimRight(base, "/")),
		ttl:         offerDeskCacheTTLFromEnv(),
		negativeTTL: DefaultOfferDeskNegativeTTL,
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func offerDeskCacheTTLFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(OfferDeskCacheTTLEnv))
	if raw == "" {
		return DefaultOfferDeskCacheTTL
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return DefaultOfferDeskCacheTTL
	}
	return ttl
}

// DerivedUrgency resolves an enabling deliverable's urgency from the typed
// release graph. It is deliberately read-time data, never goal persistence.
func (r *OfferDeskRankReader) DerivedUrgency(name string) (int, error) {
	if r == nil || r.ladder == nil {
		return 0, fmt.Errorf("offer desk release ladder is unavailable")
	}
	enabling, err := r.enablingDeliverables()
	if err != nil {
		return 0, err
	}
	for _, item := range enabling {
		if item.GetNode().GetName() == name {
			return int(item.GetDerivedUrgency()), nil
		}
	}
	return 0, fmt.Errorf("enabling deliverable %q was not found", name)
}

func (r *OfferDeskRankReader) ReleaseRank(name string) (int, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return 0, err
	}
	for _, node := range nodes {
		if node.GetName() == name {
			return int(node.GetReleaseRank()), nil
		}
	}
	return 0, fmt.Errorf("deliverable %q was not found", name)
}

func (r *OfferDeskRankReader) MaxReleaseRank() (int, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, node := range nodes {
		if int(node.GetReleaseRank()) > max {
			max = int(node.GetReleaseRank())
		}
	}
	return max, nil
}

func (r *OfferDeskRankReader) ValidateDeliverable(name string) (bool, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return false, err
	}
	for _, node := range nodes {
		if node.GetName() == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *OfferDeskRankReader) deliverables() ([]*offersv1.Node, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("offer desk rank reader is unavailable")
	}
	return cachedRead(r, "nodes", &r.nodes, func() ([]*offersv1.Node, error) {
		response, err := r.client.ListNodes(context.Background(), connect.NewRequest(&offersv1.ListNodesRequest{Kind: offersv1.NodeKind_DELIVERABLE}))
		if err != nil {
			return nil, err
		}
		return response.Msg.GetNodes(), nil
	})
}

func (r *OfferDeskRankReader) enablingDeliverables() ([]*offersv1.PrerequisiteNode, error) {
	return cachedRead(r, "enabling", &r.enabling, func() ([]*offersv1.PrerequisiteNode, error) {
		response, err := r.ladder.GetEnablingDeliverables(context.Background(), connect.NewRequest(&offersv1.ReleaseLadderRequest{}))
		if err != nil {
			return nil, err
		}
		return response.Msg.GetEnabling(), nil
	})
}

// cachedRead serves slot from cache while unexpired, otherwise performs fetch
// exactly once for all concurrent callers and stores the outcome (positive or
// negative) with its own TTL.
func cachedRead[T any](r *OfferDeskRankReader, key string, slot *cachedResult[T], fetch func() (T, error)) (T, error) {
	r.mu.Lock()
	if !slot.expires.IsZero() && r.now().Before(slot.expires) {
		value, err := slot.value, slot.err
		r.mu.Unlock()
		return value, err
	}
	r.mu.Unlock()

	result, err, _ := r.flight.Do(key, func() (any, error) {
		// Re-check under the lock: a sibling flight may have refilled the slot
		// between our miss and this leader election.
		r.mu.Lock()
		if !slot.expires.IsZero() && r.now().Before(slot.expires) {
			value, cachedErr := slot.value, slot.err
			r.mu.Unlock()
			return value, cachedErr
		}
		r.mu.Unlock()

		value, fetchErr := fetch()
		ttl := r.ttl
		if fetchErr != nil {
			ttl = r.negativeTTL
		}
		r.mu.Lock()
		slot.value, slot.err = value, fetchErr
		if ttl > 0 {
			slot.expires = r.now().Add(ttl)
		} else {
			slot.expires = time.Time{}
		}
		r.mu.Unlock()
		return value, fetchErr
	})
	value, _ := result.(T)
	return value, err
}
