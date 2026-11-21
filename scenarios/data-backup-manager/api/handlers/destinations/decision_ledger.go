package destinations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// Every operator decision DBM raises is about one ongoing situation (a blocked
// backup destination, a drive waiting for a repair). Each carries a subject key
// naming that situation, so Notification Hub keeps one open decision per
// situation, updates it in place when the facts change, and closes it when DBM
// withdraws it. This ledger is the one place that raises, re-asserts and
// withdraws; the two decision flows only say what the situation is.
const (
	// decisionReassertEvery is how often DBM re-sends a decision it still needs.
	decisionReassertEvery = 2 * time.Hour
	// decisionReassertWithin is the hub's safety net: when DBM stops re-sending
	// (it died, or lost the record) the hub closes the decision by itself.
	decisionReassertWithin = 3 * decisionReassertEvery

	backupSubjectPrefix = "backup-failure"
	volumeSubjectPrefix = "volume-repair"

	// Keys of the ask bookkeeping DBM keeps beside each pending ask.
	extraSubjectKey   = "subject_key"
	extraAssertedAt   = "asserted_at"
	extraLegacyRef    = "legacy_requester_ref"
	extraLateNotified = "late_answer_notified"

	withdrawnReplacedReason = "Replaced by an updated copy of this question."
)

// AskStore is the durable ask ledger: the Raiser's store plus a listing, which
// lets a periodic reconcile find asks whose situation has since cleared.
type AskStore interface {
	operatordecision.PendingStore
	ListAsks(ctx context.Context) ([]*operatordecision.PendingAsk, error)
}

// subjectKeyFor builds the stable key for one situation, falling back to a
// digest when the identifier has characters the hub does not accept.
func subjectKeyFor(prefix, id string) string {
	key := prefix + ":" + strings.TrimSpace(id)
	if operatordecision.SubjectKeyRule.MatchString(key) {
		return key
	}
	sum := sha256.Sum256([]byte(id))
	return prefix + ":" + hex.EncodeToString(sum[:16])
}

// backupSubjectKey names the backup-failure situation of one destination: one
// per destination, however many incidents, generations or causes it goes through.
func backupSubjectKey(destinationID string) string {
	return subjectKeyFor(backupSubjectPrefix, destinationID)
}

// volumeSubjectKey names the repair situation of one recovery journal.
func volumeSubjectKey(journalID string) string {
	return subjectKeyFor(volumeSubjectPrefix, journalID)
}

type decisionLedger struct {
	asks      AskStore
	raiser    operatordecision.Raiser
	publisher operatordecision.Publisher
	clock     func() time.Time
	logger    *log.Logger
}

// now is the owning flow's clock; both flows always wire it.
func (l decisionLedger) now() time.Time { return l.clock().UTC() }

func parseAssertedAt(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return parsed
}

// raise publishes the decision for a situation, idempotently. Within the
// re-assertion interval a repeat is a no-op; after it, the decision is re-sent
// with a fresh assertion time so the hub keeps it open and can close it if DBM
// ever stops. A decision published before subject keys existed is adopted: its
// keyless copy at the hub is withdrawn once the keyed one is delivered.
func (l decisionLedger) raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts, key string) error {
	now := l.now()
	facts.SubjectKey, facts.ReassertWithin = key, decisionReassertWithin

	existing, err := l.asks.Get(ctx, ask.ID)
	if err != nil {
		return err
	}
	asserted := now
	legacy := ""
	if existing != nil && existing.State == operatordecision.PendingStatePending {
		if existing.Extra == nil {
			existing.Extra = map[string]string{}
		}
		legacy = existing.Extra[extraLegacyRef]
		if existing.Extra[extraSubjectKey] == "" {
			if legacy == "" {
				legacy = existing.EventID
			}
			existing.Extra[extraSubjectKey] = key
			if legacy != "" {
				existing.Extra[extraLegacyRef] = legacy
			}
			if err := l.asks.Put(ctx, existing); err != nil {
				return fmt.Errorf("record decision subject: %w", err)
			}
		}
		if at := parseAssertedAt(existing.Extra[extraAssertedAt]); existing.Outstanding(now) && !at.IsZero() && now.Sub(at) < decisionReassertEvery {
			asserted = at
		}
	}
	facts.AssertedAt = asserted

	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.Extra[extraSubjectKey] = key
	ask.Extra[extraAssertedAt] = asserted.Format(time.RFC3339Nano)
	if legacy != "" {
		ask.Extra[extraLegacyRef] = legacy
	}
	_, raiseErr := l.raiser.Raise(ctx, ask, facts)

	current, getErr := l.asks.Get(ctx, ask.ID)
	if getErr != nil || current == nil || current.State != operatordecision.PendingStatePending {
		return raiseErr
	}
	stamp := asserted.Format(time.RFC3339Nano)
	if current.Extra == nil {
		current.Extra = map[string]string{}
	}
	if current.Extra[extraSubjectKey] != key || current.Extra[extraAssertedAt] != stamp {
		current.Extra[extraSubjectKey], current.Extra[extraAssertedAt] = key, stamp
		if err := l.asks.Put(ctx, current); err != nil && raiseErr == nil {
			raiseErr = fmt.Errorf("record decision assertion: %w", err)
		}
	}
	if raiseErr == nil {
		if err := l.finishLegacy(ctx, current); err != nil {
			l.logger.Printf("decision adoption: %v", err)
		}
	}
	return raiseErr
}

// finishLegacy closes the keyless copy of an ask that was published before
// subject keys existed, once the keyed copy is delivered. A failure leaves the
// reference in place so the next pass retries.
func (l decisionLedger) finishLegacy(ctx context.Context, ask *operatordecision.PendingAsk) error {
	ref := ask.Extra[extraLegacyRef]
	if ref == "" || !ask.Published || ask.State != operatordecision.PendingStatePending {
		return nil
	}
	if err := l.raiser.PublishWithdrawal(ctx, operatordecision.Withdrawal{
		RequesterRef: ref, Reason: withdrawnReplacedReason, AssertedAt: l.now(),
	}); err != nil {
		return fmt.Errorf("withdraw the earlier copy of decision %s: %w", ask.ID, err)
	}
	delete(ask.Extra, extraLegacyRef)
	return l.asks.Put(ctx, ask)
}

// withdraw closes a pending ask at the hub and then locally. The hub is told
// first: if that fails the ask stays pending and the next reconcile retries.
// bySubject withdraws a keyed ask by its subject, which closes whatever is open
// under that key, so a caller that wants a newer decision under the same key
// passes false and only the keyless copies are closed. A keyless ask, and the
// keyless copy of an ask still awaiting adoption, are closed by event id (the
// hub's requester reference).
func (l decisionLedger) withdraw(ctx context.Context, ask *operatordecision.PendingAsk, reason string, bySubject bool) error {
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return nil
	}
	now := l.now()
	if key := ask.Extra[extraSubjectKey]; key != "" && bySubject {
		if err := l.raiser.PublishWithdrawal(ctx, operatordecision.Withdrawal{SubjectKey: key, Reason: reason, AssertedAt: now}); err != nil {
			return fmt.Errorf("withdraw decision %s: %w", ask.ID, err)
		}
	}
	for _, ref := range []string{ask.Extra[extraLegacyRef], keylessRef(ask)} {
		if ref == "" {
			continue
		}
		if err := l.raiser.PublishWithdrawal(ctx, operatordecision.Withdrawal{RequesterRef: ref, Reason: reason, AssertedAt: now}); err != nil {
			return fmt.Errorf("withdraw decision %s: %w", ask.ID, err)
		}
	}
	return l.retire(ctx, ask)
}

func keylessRef(ask *operatordecision.PendingAsk) string {
	if ask.Extra[extraSubjectKey] == "" {
		return ask.EventID
	}
	return ""
}

// retire marks a pending ask withdrawn without telling the hub, for a decision
// the hub already closed or is about to replace under the same subject key.
func (l decisionLedger) retire(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return nil
	}
	now := l.now()
	ask.State, ask.ResolvedAt, ask.ResolvedBy = operatordecision.PendingStateWithdrawn, &now, ""
	delete(ask.Extra, extraLegacyRef)
	return l.asks.Put(ctx, ask)
}

// hubWithdrew records that the hub closed a decision DBM still considered
// pending (its re-assertion safety net, or a replacement). Nothing was
// answered. When the hub replaced it with a newer decision for the same
// situation there is nothing to do; otherwise the ask is marked withdrawn so
// the next evaluation raises the situation again if it still exists.
func (l decisionLedger) hubWithdrew(ctx context.Context, ask *operatordecision.PendingAsk, res operatordecision.DecisionResolution) error {
	if ask == nil || ask.State != operatordecision.PendingStatePending || strings.TrimSpace(res.SupersededByAskID) != "" {
		return nil
	}
	return l.retire(ctx, ask)
}

// lateAnswer tells the operator, once, that an answer arrived for a decision
// that had already been withdrawn, so it was not carried out.
func (l decisionLedger) lateAnswer(ctx context.Context, ask *operatordecision.PendingAsk, notice eventbus.DomainEvent) error {
	if ask == nil || ask.Extra[extraLateNotified] == "true" {
		return nil
	}
	if l.publisher != nil {
		if err := l.publisher.PublishDomainEvent(ctx, notice); err != nil {
			return fmt.Errorf("publish late-answer notice: %w", err)
		}
	}
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.Extra[extraLateNotified] = "true"
	return l.asks.Put(ctx, ask)
}
