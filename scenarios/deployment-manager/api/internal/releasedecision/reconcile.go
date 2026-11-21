package releasedecision

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// Extra keys the reconciler reads and writes on an ask.
const (
	// ExtraSubjectKey is the Hub subject key the ask was raised with. An ask
	// raised before subject keys has none and is closed by its request event id.
	ExtraSubjectKey = "subject_key"
	// ExtraWithdrawalReason and ExtraWithdrawalUnpublished record a withdrawal
	// whose event could not be published, so a later pass sends it again.
	ExtraWithdrawalReason      = "withdrawal_reason"
	ExtraWithdrawalUnpublished = "withdrawal_unpublished"
)

// ReviewVerdict is what the reconciler needs to know of a readiness review.
type ReviewVerdict int

// Review verdicts. Unknown (not found, or unreadable) keeps the ask open.
const (
	ReviewUnknown ReviewVerdict = iota
	ReviewOpen
	ReviewApproved
	ReviewSuperseded
)

// Withdrawer is the api-core Raiser's closing half: it records an ask as
// withdrawn and publishes the withdrawal, or only publishes one.
type Withdrawer interface {
	Withdraw(ctx context.Context, id, subjectKey, reason string) (bool, error)
	PublishWithdrawal(ctx context.Context, w operatordecision.Withdrawal) error
}

// Reconciler keeps the operator's open release decisions honest. It closes an
// ask nobody answered within its window, an ask whose readiness review was
// approved elsewhere or superseded, and an ask a newer release of the same
// profile, channel and targets replaced; and it sends again a withdrawal that
// could not be published.
type Reconciler struct {
	Store     *SQLStore
	Withdrawn Withdrawer
	// Review reports the state of a readiness review by key.
	Review func(ctx context.Context, reviewKey string) (ReviewVerdict, error)
	Now    func() time.Time
}

// Result counts what one pass did.
type Result struct {
	Lapsed, Cleared, Replaced, Republished int
}

func (r Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

// Reconcile runs one pass. It continues past a failing ask and returns the
// failures together.
func (r Reconciler) Reconcile(ctx context.Context) (Result, error) {
	var (
		res  Result
		errs []error
	)
	pending, err := r.Store.PendingAsks(ctx)
	if err != nil {
		return res, err
	}
	now := r.now()

	// Asks about one profile, channel and target set are one situation; the
	// newest speaks for it, if the Hub has been told about it.
	groups := map[string][]operatordecision.PendingAsk{}
	for _, ask := range pending {
		key := subjectOf(ask)
		groups[key] = append(groups[key], ask)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		asks := groups[key]
		newest := asks[len(asks)-1]
		current := newest
		if newest.Published {
			for _, old := range asks[:len(asks)-1] {
				if err := r.replace(ctx, old, now); err != nil {
					errs = append(errs, fmt.Errorf("replace %s: %w", old.ID, err))
					continue
				}
				res.Replaced++
			}
		} else {
			// The Hub has not seen the newest ask yet; the older one is still the
			// decision the operator can answer.
			current = asks[0]
			for _, ask := range asks {
				if ask.Published {
					current = ask
				}
			}
		}
		reason, lapsed, err := r.clearedReason(ctx, &current, now)
		if err != nil {
			errs = append(errs, fmt.Errorf("check %s: %w", current.ID, err))
			continue
		}
		if reason == "" {
			continue
		}
		if err := r.withdraw(ctx, current, reason); err != nil {
			errs = append(errs, fmt.Errorf("withdraw %s: %w", current.ID, err))
			continue
		}
		if lapsed {
			res.Lapsed++
		} else {
			res.Cleared++
		}
	}

	unpublished, err := r.Store.UnpublishedWithdrawals(ctx)
	if err != nil {
		return res, errors.Join(append(errs, err)...)
	}
	for _, ask := range unpublished {
		if err := r.publish(ctx, ask); err != nil {
			errs = append(errs, fmt.Errorf("publish withdrawal of %s: %w", ask.ID, err))
			continue
		}
		res.Republished++
	}
	return res, errors.Join(errs...)
}

// subjectOf is the subject an ask belongs to: the key it was raised with, or,
// for an ask raised before subject keys, the key its release identity gives.
func subjectOf(ask operatordecision.PendingAsk) string {
	if key := ask.Extra[ExtraSubjectKey]; key != "" {
		return key
	}
	var targets []string
	if raw := strings.TrimSpace(ask.Extra["targets"]); raw != "" {
		targets = strings.Split(raw, ",")
	}
	return SubjectKey(ask.Extra["profile_id"], ask.Extra["channel"], targets)
}

// clearedReason says, in the operator's words, why a pending ask no longer needs
// an answer; empty while it does. lapsed is true when its window ran out.
func (r Reconciler) clearedReason(ctx context.Context, ask *operatordecision.PendingAsk, now time.Time) (reason string, lapsed bool, err error) {
	if !ask.Outstanding(now) {
		window := ask.ExpiresAt.Sub(ask.RaisedAt).Round(time.Hour)
		return fmt.Sprintf("Nobody answered within %s, so this release request has lapsed. The agent can ask again.", window), true, nil
	}
	if r.Review == nil || ask.Extra["review_key"] == "" {
		return "", false, nil
	}
	verdict, err := r.Review(ctx, ask.Extra["review_key"])
	if err != nil {
		return "", false, err
	}
	switch verdict {
	case ReviewApproved:
		return "This release has already been approved.", false, nil
	case ReviewSuperseded:
		return "A newer readiness review replaced the one this request was about.", false, nil
	}
	return "", false, nil
}

// replace closes the local record of an ask a newer one replaced. An ask raised
// with a subject key was already updated or replaced at the Hub by the newer
// request, so nothing is published: a withdrawal naming the subject would close
// the newer decision. An ask raised before subject keys is a separate decision
// at the Hub and is closed by its request event id.
func (r Reconciler) replace(ctx context.Context, ask operatordecision.PendingAsk, now time.Time) error {
	const reason = "A newer request for this release replaced this one."
	keyed := ask.Extra[ExtraSubjectKey] != ""
	if err := r.markWithdrawn(ctx, &ask, now, reason, !keyed && ask.EventID != ""); err != nil {
		return err
	}
	if keyed || ask.EventID == "" {
		return nil
	}
	return r.publish(ctx, ask)
}

// withdraw closes the ask at the Hub. When the withdrawal cannot be published the
// ask stays recorded as withdrawn and the failure is marked, so the next pass
// sends it again. An ask with neither a subject key nor an event id cannot be
// named to the Hub; it is closed locally only.
func (r Reconciler) withdraw(ctx context.Context, ask operatordecision.PendingAsk, reason string) error {
	if ask.Extra[ExtraSubjectKey] == "" && ask.EventID == "" {
		return r.markWithdrawn(ctx, &ask, r.now(), reason, false)
	}
	done, err := r.Withdrawn.Withdraw(ctx, ask.ID, ask.Extra[ExtraSubjectKey], reason)
	if err == nil || !done {
		return err
	}
	return errors.Join(err, r.markUnpublished(ctx, ask.ID, reason))
}

func (r Reconciler) markWithdrawn(ctx context.Context, ask *operatordecision.PendingAsk, now time.Time, reason string, unpublished bool) error {
	ask.State = operatordecision.PendingStateWithdrawn
	ask.ResolvedAt = &now
	ask.ResolvedBy = ""
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	if unpublished {
		ask.Extra[ExtraWithdrawalUnpublished], ask.Extra[ExtraWithdrawalReason] = "1", reason
	}
	return r.Store.Put(ctx, ask)
}

func (r Reconciler) markUnpublished(ctx context.Context, id, reason string) error {
	stored, err := r.Store.Get(ctx, id)
	if err != nil || stored == nil {
		return err
	}
	if stored.Extra == nil {
		stored.Extra = map[string]string{}
	}
	stored.Extra[ExtraWithdrawalUnpublished], stored.Extra[ExtraWithdrawalReason] = "1", reason
	return r.Store.Put(ctx, stored)
}

// publish sends the withdrawal of an ask already recorded as withdrawn, and
// clears the marker once the Hub has it.
func (r Reconciler) publish(ctx context.Context, ask operatordecision.PendingAsk) error {
	w := operatordecision.Withdrawal{Reason: ask.Extra[ExtraWithdrawalReason]}
	if subject := ask.Extra[ExtraSubjectKey]; subject != "" {
		w.SubjectKey = subject
	} else {
		w.RequesterRef = ask.EventID
	}
	if strings.TrimSpace(w.Reason) == "" {
		w.Reason = "This release request no longer needs an answer."
	}
	if err := r.Withdrawn.PublishWithdrawal(ctx, w); err != nil {
		return err
	}
	delete(ask.Extra, ExtraWithdrawalUnpublished)
	delete(ask.Extra, ExtraWithdrawalReason)
	return r.Store.Put(ctx, &ask)
}

// Run reconciles once at start and then every interval until ctx ends. The
// outcome of each pass goes to report.
func (r Reconciler) Run(ctx context.Context, interval time.Duration, report func(Result, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		res, err := r.Reconcile(ctx)
		if report != nil {
			report(res, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// KeyedRaiser raises through Inner but keeps an ask that was raised before
// subject keys unkeyed while it is still outstanding: its Hub decision has no
// subject, so a keyed re-send would open a second decision beside it.
type KeyedRaiser struct {
	Inner interface {
		Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error)
	}
	Store *SQLStore
	Now   func() time.Time
}

// Raise implements the release handler's raiser seam.
func (k KeyedRaiser) Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error) {
	existing, err := k.Store.Get(ctx, ask.ID)
	if err != nil {
		return false, err
	}
	now := time.Now()
	if k.Now != nil {
		now = k.Now()
	}
	if existing != nil && existing.Outstanding(now) && existing.Extra[ExtraSubjectKey] == "" {
		facts.SubjectKey = ""
	}
	return k.Inner.Raise(ctx, ask, facts)
}
