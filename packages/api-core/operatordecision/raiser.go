package operatordecision

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/schedule"
)

// Clock preserves the Raiser API while reusing schedule's canonical minimal
// time-observation contract. A nil Clock defaults to the system wall clock.
type Clock = schedule.NowSource

// DefaultMaxOutstanding caps how many distinct un-decided asks a single subject
// may have outstanding at once, bounding the operator's notification exposure to
// a buggy or hostile caller. It matches
// MaxOutstandingAuthorizationRequestsPerSubject.
const DefaultMaxOutstanding = 3

// DefaultWindow bounds how long a raised ask stays outstanding before it ages out
// of the anti-flood count and may be re-raised. It matches
// pendingAuthorizationWindow.
const DefaultWindow = 24 * time.Hour

// ErrFlood is returned when a subject already has MaxOutstanding distinct
// un-decided asks. The caller refuses WITHOUT raising another operator
// notification. It generalizes ErrAuthorizationFlood.
var ErrFlood = errors.New("too many outstanding operator decisions for this subject")

// ErrPendingFactsChanged refuses to publish replacement terms under an
// outstanding correlation id whose first publication has not succeeded.
var ErrPendingFactsChanged = errors.New("pending operator decision facts changed within the same generation")

// Raiser records (idempotently) a pending ask and raises the operator decision
// for a scenario that keeps its pending asks in a PendingStore. It is the
// higher-level path on top of BuildEvent: it dedupes a repeated raise, enforces a
// per-subject flood cap, and publishes best-effort.
//
// A scenario with its OWN durable record (a run in needs_review) does NOT need a
// Raiser — it uses BuildEvent + Publisher directly. The Raiser is optional.
type Raiser struct {
	// Source is the scenario emitting the event (the `<source>` in the event type).
	Source string
	// Kind is the decision kind (the `<kind>` in the event type).
	Kind string
	// EventType, when set, overrides the composed Source+Kind event type with an
	// explicit string — the PARITY escape hatch (see BuildEventTyped) for a
	// producer whose established event type does not match `<Source>.<Kind>...`
	// (agent-manager's underscore source token). Leave empty for new scenarios.
	EventType string
	// Store is the durable pending-ask store. Without it, Raise degrades to a
	// no-op raised=false (the caller's prior hard-refuse), like agent-manager's
	// WithPendingAuthorizations-absent path.
	Store PendingStore
	// Publisher raises the operator notification. A nil Publisher keeps the ask
	// durable but un-notified (best-effort, no regression).
	Publisher Publisher
	// Clock supplies a deterministic now for the raise and expiry; defaults to the
	// system clock. The Occurred time is Now truncated to Window, so a re-raise
	// inside the window maps to the same EventID (matching the authorization path).
	Clock Clock
	// MaxOutstanding is the per-subject flood cap; <=0 uses DefaultMaxOutstanding.
	MaxOutstanding int
	// Window is the outstanding lifetime; <=0 uses DefaultWindow.
	Window time.Duration
}

func (r Raiser) clock() Clock {
	if r.Clock != nil {
		return r.Clock
	}
	return schedule.System()
}

func (r Raiser) maxOutstanding() int {
	if r.MaxOutstanding > 0 {
		return r.MaxOutstanding
	}
	return DefaultMaxOutstanding
}

func (r Raiser) window() time.Duration {
	if r.Window > 0 {
		return r.Window
	}
	return DefaultWindow
}

// Raise records a pending ask for subject and raises the operator decision. It
// returns raised=true when the ask is outstanding (new or already pending), so
// the caller answers "pending". It returns ErrFlood when the subject is already
// at its cap, so the caller refuses WITHOUT notifying. It never grants authority;
// only the resolution does that.
//
// The ask's ID (the correlation key) and Subject make the raise idempotent and
// floodable; the caller fills ask.Extra with whatever the resolver needs on
// approval. RaisedAt/ExpiresAt/State are owned here, never by the caller.
//
// facts carry the operator-facing question. Raise does NOT validate facts (the
// caller may defer validation to the hub); validate with facts.Validate() first
// when you want the error at raise time. The publish is best-effort: a failure is
// returned so the caller can log it, but the durable record already stands.
func (r Raiser) Raise(ctx context.Context, ask PendingAsk, facts Facts) (bool, error) {
	if r.Store == nil {
		// Without a store the path degrades to the prior hard refuse.
		return false, nil
	}
	subject := strings.TrimSpace(ask.Subject)
	id := strings.TrimSpace(ask.ID)
	if subject == "" || id == "" {
		return false, nil
	}
	now := r.clock().Now()

	existing, err := r.Store.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if existing != nil && existing.Outstanding(now) {
		// Idempotent: the same ask is already outstanding. Retry publication only
		// while this generation has no successful publication receipt.
		if !existing.Published {
			if err := r.publish(ctx, existing, facts); err != nil {
				return true, err
			}
		}
		return true, nil
	}

	// A genuinely new (or re-raisable) ask. Enforce the per-subject flood cap
	// before persisting or notifying.
	outstanding, err := r.Store.CountOutstandingForSubject(ctx, subject, now)
	if err != nil {
		return false, err
	}
	if outstanding >= r.maxOutstanding() {
		return false, ErrFlood
	}

	ask.ID = id
	ask.Subject = subject
	ask.State = PendingStatePending
	ask.RaisedAt = now.UTC()
	ask.ExpiresAt = now.Add(r.window()).UTC()
	event := r.event(&ask, facts)
	ask.EventID = EventID(event)
	if err := r.Store.Put(ctx, &ask); err != nil {
		return false, err
	}

	if err := r.publish(ctx, &ask, facts); err != nil {
		return true, err
	}
	return true, nil
}

func (r Raiser) publish(ctx context.Context, ask *PendingAsk, facts Facts) error {
	if r.Publisher == nil || ask == nil {
		return nil
	}
	event := r.event(ask, facts)
	if ask.EventID != "" && EventID(event) != ask.EventID {
		return ErrPendingFactsChanged
	}
	if err := r.Publisher.PublishDomainEvent(ctx, event); err != nil {
		return err
	}
	ask.Published = true
	return r.Store.Put(ctx, ask)
}

func (r Raiser) event(ask *PendingAsk, facts Facts) eventbus.DomainEvent {
	// Keep occurrence attached to the stored generation so every retry publishes
	// the same event identity, even after the clock advances.
	occurred := ask.RaisedAt.UTC().Truncate(r.window())
	eventType := r.EventType
	if eventType == "" {
		eventType = DecisionRequestedEventType(r.Source, r.Kind)
	}
	return BuildEventTyped(r.Source, eventType, occurred, facts)
}
