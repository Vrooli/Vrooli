package deletiongrant

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// Resolver is the operatordecision.DomainEffect that records an operator's
// decision on a deletion ask. On "approve" it stamps the grant's expiry (from
// the requested TTL, capped) so the policy snapshot can project it as an allow
// rule; on any other answer it records the refusal. It owns only this
// scenario's deletion-grant resolutions.
type Resolver struct {
	store     *Store
	clock     func() time.Time
	onApprove func()
}

// NewResolver builds a resolver over store.
func NewResolver(store *Store) *Resolver { return &Resolver{store: store} }

// WithClock installs a deterministic clock for tests.
func (r *Resolver) WithClock(clock func() time.Time) *Resolver {
	r.clock = clock
	return r
}

// WithOnApprove installs a callback fired after an approval is recorded, so the
// policy bridge can republish the snapshot immediately and the new allow rule
// takes effect in seconds rather than at the next publish interval. It is not
// called for a denial (nothing new to project).
func (r *Resolver) WithOnApprove(fn func()) *Resolver {
	r.onApprove = fn
	return r
}

func (r *Resolver) now() time.Time {
	if r.clock != nil {
		return r.clock().UTC()
	}
	return time.Now().UTC()
}

// Handles claims only this producer's deletion-grant resolutions, matching the
// canonical <source>.<kind>.decision_requested.v1 the Raiser emits.
func (r *Resolver) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(Source, Kind)
}

// Apply records the decision. It is idempotent and replay-safe: an unknown or
// already-decided ask is a benign no-op (handled=true, nil) so a redelivery
// never re-applies the effect or triggers a retry storm.
func (r *Resolver) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	// The resolution arrives keyed by notification-hub's UUID; our record is
	// keyed by the correlation id we stashed in Facts.Correlation at raise time.
	askID := strings.TrimSpace(resolution.Correlation[correlationAskID])
	if askID == "" {
		return true, nil
	}
	ask, err := r.store.Get(ctx, askID)
	if err != nil {
		return false, err
	}
	if ask == nil || ask.State == stateResolved {
		return true, nil
	}
	now := r.now()
	ask.State = stateResolved
	ask.ResolvedAt = &now
	ask.ResolvedBy = resolution.AnsweredBy
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	approved := strings.EqualFold(strings.TrimSpace(resolution.Answer), "approve")
	if approved {
		ask.Extra[extraAnswer] = "approve"
		ask.Extra[extraGrantExpiresAt] = now.Add(grantTTL(ask.Extra[extraRequestedTTL])).Format(ts)
	} else {
		ask.Extra[extraAnswer] = "deny"
	}
	if err := r.store.Put(ctx, ask); err != nil {
		return false, err
	}
	if approved && r.onApprove != nil {
		// Best-effort: ask the publish loop to project the new grant now.
		r.onApprove()
	}
	return true, nil
}

// grantTTL reads the requested validity, applies the default when absent, and
// caps it so an approval can never mint an unbounded or over-long authority.
func grantTTL(requested string) time.Duration {
	ttl := defaultGrantTTL
	if secs, err := strconv.ParseInt(requested, 10, 64); err == nil && secs > 0 {
		ttl = time.Duration(secs) * time.Second
	}
	if ttl > maxGrantTTL {
		ttl = maxGrantTTL
	}
	return ttl
}
