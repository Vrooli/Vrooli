package usagemodes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/usageforecast"
	"agent-manager/internal/usagelabel"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/usagemode"
)

// Every usage mode decision is about one situation, so each carries a subject
// key and the hub keeps at most one open decision per key: a changed question
// updates it, and the notifier withdraws it when the situation no longer needs
// the operator (the mode ended or was extended elsewhere, the window recovered
// or reset, another mode already covers the runner).

var subjectUnsafe = regexp.MustCompile(`[^A-Za-z0-9._:/@-]`)

// decisionSubjectKey composes "usage-mode:<kind>:<part>...", within the hub's
// 128-byte key rule.
func decisionSubjectKey(kind string, parts ...string) string {
	key := decisionSubject + ":" + kind
	for _, part := range parts {
		if part = subjectUnsafe.ReplaceAllString(strings.TrimSpace(part), "-"); part != "" {
			key += ":" + part
		}
	}
	if len(key) > operatordecisionMaxKey {
		sum := sha256.Sum256([]byte(key))
		key = key[:operatordecisionMaxKey-17] + "-" + hex.EncodeToString(sum[:8])
	}
	return key
}

const operatordecisionMaxKey = 128

// subjectKeyForParams derives the subject key from a decision's parameters. The
// same function names a decision when it is raised and when a stored ask is
// matched to the situation it was about.
func subjectKeyForParams(p map[string]string) string {
	switch p["kind"] {
	case KindRunsOut:
		return decisionSubjectKey(KindRunsOut, p["scope"])
	case KindAutoSaveRule:
		return decisionSubjectKey(KindAutoSaveRule, p["runner"])
	case KindEnding, KindUntilOf:
		return decisionSubjectKey(p["kind"], p["mode_id"])
	case KindTimed:
		return decisionSubjectKey(KindTimed, p["runner"], p["until"])
	case KindRoleDefault:
		return decisionSubjectKey(KindRoleDefault, p["mode_id"], p["role"])
	}
	return ""
}

// legacyAsk is a pending ask published before decisions carried a subject key.
// The hub cannot update or close it by key, only by the request's event id.
func legacyAsk(ask *operatordecision.PendingAsk) bool {
	return ask != nil && ask.State == operatordecision.PendingStatePending && ask.Extra["subject_key"] == ""
}

const withdrawUnsent = "withdraw_unsent"

// raise publishes one keyed decision. A legacy ask for the same id is closed by
// its event id first, so the operator keeps one open decision, not two.
func (n *Notifier) raise(ctx context.Context, raiser operatordecision.Raiser, id, floodSubject string, facts operatordecision.Facts, extra map[string]string) error {
	if facts.TemplateData != nil {
		encoded, err := json.Marshal(facts.TemplateData)
		if err != nil {
			return fmt.Errorf("encode decision template data: %w", err)
		}
		var normalized map[string]any
		if err := json.Unmarshal(encoded, &normalized); err != nil {
			return fmt.Errorf("normalize decision template data: %w", err)
		}
		facts.TemplateData = normalized
	}
	key := subjectKeyForParams(facts.Correlation)
	facts.SubjectKey = key
	if err := n.retireLegacy(ctx, id, "Replaced by an updated version of this question."); err != nil {
		return err
	}
	stored := map[string]string{"subject_key": key}
	for k, v := range extra {
		stored[k] = v
	}
	raised, err := raiser.Raise(ctx, pendingAskFor(id, floodSubject, facts.Correlation, stored), facts)
	if raised {
		n.supersedeAsks(ctx, key, id)
	}
	return err
}

// retireLegacy closes a pending keyless ask at the hub and forgets it, so the
// caller can raise the same question again under its subject key.
func (n *Notifier) retireLegacy(ctx context.Context, id, reason string) error {
	st := n.load()
	ask := st.Asks[id]
	if !legacyAsk(ask) {
		return nil
	}
	if ask.EventID != "" {
		w := operatordecision.Withdrawal{RequesterRef: ask.EventID, Reason: reason}
		if err := n.raiser().PublishWithdrawal(ctx, w); err != nil {
			return err
		}
	}
	delete(st.Asks, id)
	return nil
}

// supersedeAsks closes this process's older pending asks about the same
// situation. A keyed ask is replaced at the hub by the new one; a keyless one is
// closed by its event id.
func (n *Notifier) supersedeAsks(ctx context.Context, key, exceptID string) {
	if key == "" {
		return
	}
	st := n.load()
	for id, ask := range st.Asks {
		if id == exceptID || ask.State != operatordecision.PendingStatePending || subjectKeyForParams(ask.Extra) != key {
			continue
		}
		reason := "Replaced by an updated version of this question."
		if legacyAsk(ask) {
			n.withdrawAsk(ctx, ask, reason)
			continue
		}
		now := n.now()
		ask.State, ask.ResolvedAt = operatordecision.PendingStateWithdrawn, &now
		ask.Extra["note"] = "superseded by " + exceptID
	}
}

// withdrawAsk closes one pending ask at the hub with a plain reason and records
// it as withdrawn. A publish that fails is remembered and retried on the next
// pass, so the operator never keeps a decision the notifier already dropped.
func (n *Notifier) withdrawAsk(ctx context.Context, ask *operatordecision.PendingAsk, reason string) error {
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	_, err := n.raiser().Withdraw(ctx, ask.ID, ask.Extra["subject_key"], reason)
	ask.Extra["note"] = "withdrawn: " + reason
	if err != nil {
		ask.Extra[withdrawUnsent] = reason
	}
	return err
}

// retryUnsentWithdrawals publishes the withdrawals a previous pass could not.
func (n *Notifier) retryUnsentWithdrawals(ctx context.Context) []error {
	var errs []error
	for _, ask := range n.load().Asks {
		reason := ask.Extra[withdrawUnsent]
		if reason == "" || ask.State != operatordecision.PendingStateWithdrawn {
			continue
		}
		w := operatordecision.Withdrawal{SubjectKey: ask.Extra["subject_key"], Reason: reason}
		if w.SubjectKey == "" {
			w.RequesterRef = ask.EventID
		}
		if err := n.raiser().PublishWithdrawal(ctx, w); err != nil {
			errs = append(errs, fmt.Errorf("usage mode withdrawal %s: %w", ask.ID, err))
			continue
		}
		delete(ask.Extra, withdrawUnsent)
	}
	return errs
}

func modeCoversRunner(m usagemode.Mode, runner string) bool {
	if usagemode.NormalizeRunner(m.Scope.Runner) != runner && m.Scope.Runner != "" {
		return false
	}
	return m.Scope.Role == "" && m.Scope.Profile == ""
}

func coveredBy(modes []usagemode.Mode, runner string) bool {
	for _, m := range modes {
		if modeCoversRunner(m, runner) {
			return true
		}
	}
	return false
}

func labelOr(label, fallback string) string {
	if strings.TrimSpace(label) == "" {
		return fallback
	}
	return label
}

// staleReason says, in plain words, why a pending ask no longer needs the
// operator, or "" while it still does. backOnTrack is true when the ask was a
// run-out offer whose window recovered, so it may be offered again if the
// window turns bad later.
func (n *Notifier) staleReason(ctx context.Context, ask *operatordecision.PendingAsk, modes []usagemode.Mode, forecasts []usageforecast.Forecast, now time.Time) (reason string, backOnTrack bool) {
	p := ask.Extra
	active := map[string]usagemode.Mode{}
	for _, m := range modes {
		active[m.ID] = m
	}
	switch p["kind"] {
	case KindEnding:
		m, ok := active[p["mode_id"]]
		switch {
		case !ok:
			return fmt.Sprintf("%s has already ended.", labelOr(p["label"], "The usage mode")), false
		case p["expires_at"] != "" && (m.ExpiresAt == nil || m.ExpiresAt.UTC().Format(time.RFC3339) != p["expires_at"]):
			return fmt.Sprintf("%s was already extended, so it is not ending then.", labelOr(p["label"], m.Label)), false
		}
	case KindUntilOf:
		if _, ok := active[p["mode_id"]]; !ok {
			return fmt.Sprintf("%s is already off.", labelOr(p["label"], "The usage mode")), false
		}
	case KindRoleDefault:
		// The operator applied the change by hand: the policy file already names
		// the proposed model as the role's default.
		if p["runner"] != "" {
			change := n.Modes.RoleDefaultChange(ctx, p["runner"], p["role"], p["from"], p["to"])
			if len(change.Ops) > 0 && change.Ops[0].From == p["to"] {
				return fmt.Sprintf("%s is already the default for %s.", p["to"], p["role"]), false
			}
		}
	case KindTimed:
		if runner := usagemode.NormalizeRunner(p["runner"]); runner != "" && coveredBy(modes, runner) {
			return fmt.Sprintf("A usage mode is already on for %s.", runnerTitle(runner)), false
		}
	case KindRunsOut:
		if o := n.load().Offers[p["scope"]]; o != nil && !now.Before(o.ResetAt) {
			return "The usage window has reset, so it no longer needs a decision.", false
		}
		if runner := usagemode.NormalizeRunner(p["runner"]); runner != "" && coveredBy(modes, runner) {
			return fmt.Sprintf("A usage mode is already on for %s.", runnerTitle(runner)), false
		}
		for _, f := range forecasts {
			if f.Window != nil && f.Window.Scope() == p["scope"] && f.Status != usageforecast.StatusWillExhaust {
				return usagelabel.Title(f.Window.Provider, f.Window.Window, f.Window.WindowMinutes) + " is back on track, so it is no longer projected to run out.", true
			}
		}
	}
	return "", false
}

// reconcileAsks withdraws every pending usage mode decision whose situation has
// cleared and retries withdrawals that could not be sent. It runs on every
// evaluation, so a decision the operator already settled elsewhere (by ending
// or extending the mode from the Usage page or the CLI) closes by itself.
func (n *Notifier) reconcileAsks(ctx context.Context, forecasts []usageforecast.Forecast, now time.Time) []error {
	st := n.load()
	errs := n.retryUnsentWithdrawals(ctx)
	modes := n.Modes.listAt(now)
	ids := make([]string, 0, len(st.Asks))
	for id, ask := range st.Asks {
		if ask.State == operatordecision.PendingStatePending {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		ask := st.Asks[id]
		reason, backOnTrack := n.staleReason(ctx, ask, modes, forecasts, now)
		if reason == "" {
			continue
		}
		if backOnTrack {
			if o := st.Offers[ask.Extra["scope"]]; o != nil && !o.Declined {
				o.Gen++ // offer again, as a fresh decision, if the window turns bad again
				o.Reopen = true
			}
		}
		if err := n.withdrawAsk(ctx, ask, reason); err != nil {
			errs = append(errs, errors.Join(fmt.Errorf("usage mode withdrawal %s", id), err))
		}
	}
	return errs
}
