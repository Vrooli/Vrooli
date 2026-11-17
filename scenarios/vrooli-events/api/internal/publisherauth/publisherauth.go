// Package publisherauth observes whether each event's claimed source is proven
// by the publisher.
//
// vrooli-events accepts an event from any caller and takes its source scenario
// from the envelope, so a decision request, an operator's answer or a withdrawal
// carries a source nothing has checked. A scenario proves its source with an
// operator-issued service principal named "<source>-event-publisher" (api-core
// eventpublisher). This package is the first step of making that proof required:
// it classifies every accepted event against the proof it presented and counts
// the result, and it never refuses, delays beyond a short bound, or alters an
// event. The counts say which publishers have adopted the proof and whether any
// presents one for a source that is not its own, which is the evidence an
// operator needs before choosing to enforce.
package publisherauth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/eventpublisher"
	apiidentity "github.com/vrooli/api-core/identity"
)

// Class is how an event's claimed source fared against the proof it came with.
type Class string

const (
	// Authenticated: a verified event-publisher principal whose source is the
	// claimed source.
	Authenticated Class = "authenticated"
	// Mismatch: a verified event-publisher principal for a different source than
	// the one claimed. This is a forgery or a mis-labelled publisher.
	Mismatch Class = "mismatch"
	// Unauthenticated: no bearer token. Every publisher is here until the operator
	// issues its credential.
	Unauthenticated Class = "unauthenticated"
	// Invalid: a token that did not verify, or verified but is not an
	// event-publisher principal.
	Invalid Class = "invalid"
	// Unverifiable: a token that could not be checked because the authenticator
	// could not be reached.
	Unverifiable Class = "unverifiable"
)

// Classes lists every class, in the order a report shows them.
var Classes = []Class{Authenticated, Mismatch, Unauthenticated, Invalid, Unverifiable}

// Family groups event types by what acting on a forged one would cost, so a report
// shows the high-consequence ones first. It is how the operator will choose what
// to enforce first, not a policy by itself.
type Family string

const (
	// Answer events carry the operator's answer or question about an ask from
	// Notification Hub; every consumer applies them.
	Answer Family = "answer"
	// Decision events request or withdraw an operator decision.
	Decision Family = "decision"
	// Notification events notify the operator (notices, incidents, heartbeats).
	Notification Family = "notification"
	// Other events carry no operator-facing consequence at the Hub.
	Other Family = "other"
)

// FamilyOf classifies an event type.
func FamilyOf(eventType string) Family {
	switch {
	case strings.HasPrefix(eventType, "notification_hub.ask."):
		return Answer
	case strings.HasSuffix(eventType, ".decision_requested.v1"), strings.HasSuffix(eventType, ".decision_withdrawn.v1"):
		return Decision
	case strings.HasSuffix(eventType, ".notification.v1"), strings.HasPrefix(eventType, "incident."), strings.HasPrefix(eventType, "prompt-manager.heartbeat."):
		return Notification
	}
	return Other
}

// verifyWait bounds how long ingest waits to classify a token. Classification is
// local once the authenticator's keys are cached; the bound only matters when the
// authenticator is unreachable, and then the event is accepted as unverifiable.
const verifyWait = 2 * time.Second

type key struct {
	claimed, authenticated string
	family                 Family
	class                  Class
}

type tally struct {
	count       int64
	first, last time.Time
	sample      string
}

// Observer classifies and counts accepted events. The zero value is not usable;
// build one with New.
type Observer struct {
	verifier authn.TokenVerifier
	db       DB
	now      func() time.Time

	mu      sync.Mutex
	pending map[key]*tally
}

// DB is the database surface the observer persists counts through.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// New builds an observer. A nil verifier classifies every presented token as
// unverifiable.
func New(verifier authn.TokenVerifier, db DB) *Observer {
	return &Observer{verifier: verifier, db: db, now: time.Now, pending: map[key]*tally{}}
}

// Classify says how the claimed source fared against the request's proof. The
// authenticated source is the principal's own, set for authenticated and mismatch.
func (o *Observer) Classify(ctx context.Context, headers http.Header, claimed string) (class Class, authenticated string) {
	if strings.TrimSpace(headers.Get("Authorization")) == "" {
		return Unauthenticated, ""
	}
	if o == nil || o.verifier == nil {
		return Unverifiable, ""
	}
	ctx, cancel := context.WithTimeout(ctx, verifyWait)
	defer cancel()
	source, err := eventpublisher.VerifiedSource(ctx, o.verifier, headers)
	switch {
	case err == nil && source == claimed:
		return Authenticated, source
	case err == nil:
		return Mismatch, source
	case errors.Is(err, context.DeadlineExceeded):
		return Unverifiable, ""
	}
	if failure, ok := apiidentity.FailureFromError(err); ok && failure.Class == apiidentity.FailureUnavailable {
		return Unverifiable, ""
	}
	return Invalid, ""
}

// Observe classifies one accepted event and counts it. It never fails and never
// changes the event.
func (o *Observer) Observe(ctx context.Context, headers http.Header, claimed, eventType, eventID string) Class {
	if o == nil {
		return Unauthenticated
	}
	class, authenticated := o.Classify(ctx, headers, claimed)
	k := key{claimed: claimed, authenticated: authenticated, family: FamilyOf(eventType), class: class}
	now := o.now().UTC()
	o.mu.Lock()
	t := o.pending[k]
	if t == nil {
		t = &tally{first: now}
		o.pending[k] = t
	}
	t.count++
	t.last, t.sample = now, eventID
	o.mu.Unlock()
	return class
}
