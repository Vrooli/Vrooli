package deletiongrant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// Withdrawer closes asks the operator no longer needs to answer. The api-core
// Raiser is one: it records the ask as withdrawn, then publishes the withdrawal.
type Withdrawer interface {
	Withdraw(ctx context.Context, id, subjectKey, reason string) (bool, error)
	PublishWithdrawal(ctx context.Context, w operatordecision.Withdrawal) error
}

// PathState says whether a requested path is still there.
type PathState int

// Path states. Unknown is a path whose absence cannot be told from an unmounted
// drive or an unreadable parent: such a path keeps its ask open.
const (
	PathPresent PathState = iota
	PathGone
	PathUnknown
)

// StatPath is the production PathState: a path is gone only when its parent
// directory is readable and does not hold it, so an unplugged drive never
// reads as "everything is deleted".
func StatPath(path string) PathState {
	if _, err := os.Lstat(path); err == nil {
		return PathPresent
	} else if !errors.Is(err, os.ErrNotExist) {
		return PathUnknown
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		return PathUnknown
	}
	return PathGone
}

// Reconciler keeps the operator's open deletion decisions honest: it closes an
// ask nobody answered within its window, and an ask whose paths are all gone,
// and it sends again a withdrawal that could not be published.
type Reconciler struct {
	Store     *Store
	Withdrawn Withdrawer
	Now       func() time.Time
	PathState func(path string) PathState
	// Gate, when set, admits a pass only while background work is allowed (a
	// lifecycle fence); a refused pass is skipped and runs on the next tick.
	Gate func(fn func() error) error
}

// Result counts what one pass did.
type Result struct {
	Lapsed, Gone, Republished int
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
	for _, ask := range pending {
		reason := r.clearedReason(&ask, now)
		if reason == "" {
			continue
		}
		if err := r.withdraw(ctx, ask, reason); err != nil {
			errs = append(errs, fmt.Errorf("withdraw %s: %w", ask.ID, err))
			continue
		}
		if !ask.Outstanding(now) {
			res.Lapsed++
		} else {
			res.Gone++
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

// clearedReason says, in the operator's words, why a pending ask no longer
// needs an answer; empty while it does.
func (r Reconciler) clearedReason(ask *operatordecision.PendingAsk, now time.Time) string {
	if !ask.Outstanding(now) {
		window := ask.ExpiresAt.Sub(ask.RaisedAt).Round(time.Hour)
		return fmt.Sprintf("Nobody answered within %s, so this request has lapsed. The agent can ask again.", window)
	}
	paths := askPaths(ask)
	if len(paths) == 0 {
		return ""
	}
	state := r.PathState
	if state == nil {
		state = StatPath
	}
	for _, p := range paths {
		if state(p) != PathGone {
			return ""
		}
	}
	if len(paths) == 1 {
		return "The path this request asked to delete is already gone."
	}
	return fmt.Sprintf("All %d paths this request asked to delete are already gone.", len(paths))
}

// withdraw closes the ask at the Hub. When the withdrawal cannot be published
// the ask stays recorded as withdrawn and the failure is marked, so the next
// pass sends it again.
func (r Reconciler) withdraw(ctx context.Context, ask operatordecision.PendingAsk, reason string) error {
	done, err := r.Withdrawn.Withdraw(ctx, ask.ID, ask.Extra[extraSubjectKey], reason)
	if err == nil || !done {
		return err
	}
	stored, getErr := r.Store.Get(ctx, ask.ID)
	if getErr != nil || stored == nil {
		return errors.Join(err, getErr)
	}
	if stored.Extra == nil {
		stored.Extra = map[string]string{}
	}
	stored.Extra[extraWithdrawalUnpublished], stored.Extra[extraWithdrawalReason] = "1", reason
	return errors.Join(err, r.Store.Put(ctx, stored))
}

// publish sends the withdrawal of an ask already recorded as withdrawn.
func (r Reconciler) publish(ctx context.Context, ask operatordecision.PendingAsk) error {
	w := operatordecision.Withdrawal{Reason: ask.Extra[extraWithdrawalReason]}
	if subject := ask.Extra[extraSubjectKey]; subject != "" {
		w.SubjectKey = subject
	} else {
		w.RequesterRef = ask.EventID
	}
	if strings.TrimSpace(w.Reason) == "" {
		w.Reason = "This request no longer needs an answer."
	}
	if err := r.Withdrawn.PublishWithdrawal(ctx, w); err != nil {
		return err
	}
	delete(ask.Extra, extraWithdrawalUnpublished)
	delete(ask.Extra, extraWithdrawalReason)
	return r.Store.Put(ctx, &ask)
}

func (r Reconciler) gated(fn func() error) error {
	if r.Gate == nil {
		return fn()
	}
	return r.Gate(fn)
}

// Run reconciles once at start and then every interval until ctx ends. A failed
// pass is reported through report and retried on the next one.
func (r Reconciler) Run(ctx context.Context, interval time.Duration, report func(Result, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		var res Result
		err := r.gated(func() (err error) { res, err = r.Reconcile(ctx); return err })
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

// askPaths are the paths an ask requested.
func askPaths(ask *operatordecision.PendingAsk) []string {
	var paths []string
	if err := json.Unmarshal([]byte(ask.Extra[extraPaths]), &paths); err != nil {
		return nil
	}
	return paths
}
