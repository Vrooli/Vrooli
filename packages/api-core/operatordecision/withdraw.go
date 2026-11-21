package operatordecision

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/eventbus"
)

// DecisionWithdrawnSuffix is the event-type suffix a producer uses to close its
// own open decision. notification-hub subscribes to `**.decision_withdrawn.v1`,
// so `<source>.<kind>.decision_withdrawn.v1` closes the decision raised as
// `<source>.<kind>.decision_requested.v1` with no hub change for a new scenario.
const DecisionWithdrawnSuffix = ".decision_withdrawn.v1"

// MaxWithdrawalReason bounds the operator-facing withdrawal reason.
const MaxWithdrawalReason = 300

// DecisionWithdrawnEventType composes `<source>.<kind>.decision_withdrawn.v1`.
func DecisionWithdrawnEventType(source, kind string) string {
	return strings.TrimSpace(source) + "." + strings.TrimSpace(kind) + DecisionWithdrawnSuffix
}

// RequestEventTypeFor maps a withdrawal event type to the request event type it
// closes (`a.b.decision_withdrawn.v1` → `a.b.decision_requested.v1`). It returns
// "" for a type without the withdrawal suffix.
func RequestEventTypeFor(withdrawnEventType string) string {
	eventType := strings.TrimSpace(withdrawnEventType)
	if !strings.HasSuffix(eventType, DecisionWithdrawnSuffix) {
		return ""
	}
	return strings.TrimSuffix(eventType, DecisionWithdrawnSuffix) + DecisionRequestedSuffix
}

// Withdrawal closes a decision its producer raised, because the situation it
// asked about no longer needs the operator: the failure recovered, the request
// was decided elsewhere, or the thing it was about is gone. It is never an
// answer. The hub closes only a decision that is still open; an answered,
// defaulted, expired or dismissed decision keeps its outcome.
//
// Identify the decision by SubjectKey (preferred; closes whichever revision is
// open for that subject) or by RequesterRef (the EventID of the request event
// that opened it, for a decision raised without a subject key).
type Withdrawal struct {
	SubjectKey   string
	RequesterRef string
	// Reason is shown to the operator on the closed decision, in plain words:
	// "Backups to the Elements drive succeeded again at 14:02." Required.
	Reason string
	// AssertedAt is when the producer observed that the decision is no longer
	// needed. The hub ignores a later-delivered request asserted before it.
	// BuildWithdrawalEvent fills it with the occurred time when it is zero.
	AssertedAt time.Time
}

// Validate checks the withdrawal against the hub contract.
func (w Withdrawal) Validate() error {
	subject := strings.TrimSpace(w.SubjectKey)
	ref := strings.TrimSpace(w.RequesterRef)
	reason := strings.TrimSpace(w.Reason)
	switch {
	case subject == "" && ref == "":
		return fmt.Errorf("a withdrawal needs a subject_key or a requester_ref")
	case subject != "" && !SubjectKeyRule.MatchString(subject):
		return fmt.Errorf("subject_key %q must be 1-128 letters, digits, '.', '_', ':', '/', '@' or '-'", subject)
	case len(ref) > 128:
		return fmt.Errorf("requester_ref exceeds 128 bytes")
	case reason == "" || !utf8.ValidString(reason) || len(reason) > MaxWithdrawalReason:
		return fmt.Errorf("a withdrawal reason of 1-%d bytes is required", MaxWithdrawalReason)
	}
	return nil
}

// Payload is the decision_withdrawn event payload. source_event_type names the
// request type being closed, matching the field Agent Manager's typed
// withdrawals already carry.
func (w Withdrawal) Payload(requestEventType string) map[string]any {
	payload := map[string]any{
		"source_event_type": strings.TrimSpace(requestEventType),
		"reason":            strings.TrimSpace(w.Reason),
	}
	if subject := strings.TrimSpace(w.SubjectKey); subject != "" {
		payload["subject_key"] = subject
	}
	if ref := strings.TrimSpace(w.RequesterRef); ref != "" {
		payload["requester_ref"] = ref
	}
	if !w.AssertedAt.IsZero() {
		payload["asserted_at"] = w.AssertedAt.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

// BuildWithdrawalEvent assembles `<source>.<kind>.decision_withdrawn.v1` closing
// the decision raised as `<source>.<kind>.decision_requested.v1`.
func BuildWithdrawalEvent(source, kind string, occurred time.Time, w Withdrawal) eventbus.DomainEvent {
	return BuildWithdrawalEventTyped(source, DecisionRequestedEventType(source, kind), occurred, w)
}

// BuildWithdrawalEventTyped is BuildWithdrawalEvent for a producer whose request
// event type is not `<source>.<kind>...` (the BuildEventTyped parity case). The
// withdrawal type is derived from requestEventType by swapping the suffix.
func BuildWithdrawalEventTyped(source, requestEventType string, occurred time.Time, w Withdrawal) eventbus.DomainEvent {
	requestEventType = strings.TrimSpace(requestEventType)
	if w.AssertedAt.IsZero() {
		w.AssertedAt = occurred
	}
	return eventbus.DomainEvent{
		Source:    source,
		EventType: strings.TrimSuffix(requestEventType, DecisionRequestedSuffix) + DecisionWithdrawnSuffix,
		Occurred:  occurred.UTC(),
		Payload:   w.Payload(requestEventType),
	}
}

// Withdraw closes the Raiser's own outstanding ask id: it records the ask as
// withdrawn in the PendingStore and publishes the withdrawal so the hub closes
// the operator's open decision with reason. subjectKey is the key the ask was
// raised with ("" closes by the stored request EventID instead). It is a no-op
// (false, nil) for an unknown or already-decided ask. The local record is
// written first, so a failed publish is returned for the caller to retry with
// PublishWithdrawal; the hub closes only a still-open decision, so a repeat is
// harmless.
func (r Raiser) Withdraw(ctx context.Context, id, subjectKey, reason string) (bool, error) {
	if r.Store == nil {
		return false, nil
	}
	ask, err := r.Store.Get(ctx, strings.TrimSpace(id))
	if err != nil || ask == nil {
		return false, err
	}
	if ask.State != PendingStatePending {
		return false, nil
	}
	w := Withdrawal{SubjectKey: subjectKey, Reason: reason}
	if strings.TrimSpace(subjectKey) == "" {
		w.RequesterRef = ask.EventID
	}
	if err := w.Validate(); err != nil {
		return false, err
	}
	now := r.clock().Now().UTC()
	ask.State = PendingStateWithdrawn
	ask.ResolvedAt = &now
	ask.ResolvedBy = ""
	if err := r.Store.Put(ctx, ask); err != nil {
		return false, err
	}
	return true, r.PublishWithdrawal(ctx, w)
}

// PublishWithdrawal publishes a withdrawal for this Raiser's request type
// without touching the PendingStore, for a producer that keeps its own record
// or retries a failed Withdraw publish.
func (r Raiser) PublishWithdrawal(ctx context.Context, w Withdrawal) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if r.Publisher == nil {
		return nil
	}
	return r.Publisher.PublishDomainEvent(ctx, BuildWithdrawalEventTyped(r.Source, r.requestEventType(), r.clock().Now(), w))
}

func (r Raiser) requestEventType() string {
	if r.EventType != "" {
		return r.EventType
	}
	return DecisionRequestedEventType(r.Source, r.Kind)
}
