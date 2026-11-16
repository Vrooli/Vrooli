package usagemodes

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/usageforecast"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const withdrawnEventType = "agent_manager.usage_mode.decision_withdrawn.v1"

func (r *recorder) withdrawals() (out []eventbus.DomainEvent) {
	for _, e := range r.events {
		if e.EventType == withdrawnEventType {
			out = append(out, e)
		}
	}
	return out
}

func subjectOf(e eventbus.DomainEvent) string {
	s, _ := e.Payload["subject_key"].(string)
	return s
}

func decisionID(e eventbus.DomainEvent) string {
	return e.Payload["correlation"].(map[string]any)["decision_id"].(string)
}

// startUntilOff starts an until-off mode and runs the daily reminder so one
// until-off decision is open.
func startUntilOff(t *testing.T, n *Notifier, rec *recorder, c *clock) (modeID string, ask eventbus.DomainEvent) {
	t.Helper()
	ctx := context.Background()
	off, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "off", StartedBy: "o@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	if errs := n.Flush(ctx, nil, c.t); len(errs) != 0 {
		t.Fatal(errs)
	}
	if len(rec.decisions()) != 1 {
		t.Fatalf("decisions = %d, want the until-off reminder", len(rec.decisions()))
	}
	return off.Mode.ID, rec.decisions()[0]
}

func TestUntilOffDecisionIsKeyedPerModeAndClosesWhenTheModeIsTurnedOffElsewhere(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	modeID, ask := startUntilOff(t, n, rec, c)
	if got, want := subjectOf(ask), "usage-mode:until-off:"+modeID; got != want {
		t.Fatalf("subject key = %q, want %q", got, want)
	}

	// Still on: nothing to withdraw, and the next day's reminder keeps the same
	// subject so the hub updates the one open decision instead of adding a second.
	n.Flush(ctx, nil, c.t)
	if len(rec.withdrawals()) != 0 {
		t.Fatalf("withdrew a decision whose mode is still on: %d", len(rec.withdrawals()))
	}
	c.t = c.t.Add(24 * time.Hour)
	n.Flush(ctx, nil, c.t)
	if d := rec.decisions(); len(d) != 2 || subjectOf(d[1]) != subjectOf(d[0]) {
		t.Fatalf("daily reminder must reuse the subject key: %d decisions", len(d))
	}
	if len(rec.withdrawals()) != 0 {
		t.Fatal("the superseded day's decision is replaced by the hub, not withdrawn")
	}
	if got := n.load().Asks[decisionID(ask)].State; got != operatordecision.PendingStateWithdrawn {
		t.Fatalf("superseded local ask state = %q, want withdrawn so only one stays pending", got)
	}

	// The operator turns the mode off from the Usage page or the CLI.
	if _, _, err := n.Modes.End([]string{modeID}, false); err != nil {
		t.Fatal(err)
	}
	n.Observe(ctx, c.t)
	if errs := n.Flush(ctx, nil, c.t); len(errs) != 0 {
		t.Fatal(errs)
	}
	w := rec.withdrawals()
	if len(w) != 1 || subjectOf(w[0]) != "usage-mode:until-off:"+modeID || w[0].Payload["source_event_type"] != DecisionEventType {
		t.Fatalf("withdrawals = %+v", w)
	}
	if reason, _ := w[0].Payload["reason"].(string); !strings.Contains(reason, "off") {
		t.Fatalf("reason = %q", reason)
	}
	// Idempotent: a later pass publishes nothing more.
	n.Flush(ctx, nil, c.t)
	if len(rec.withdrawals()) != 1 {
		t.Fatalf("withdrawal repeated: %d", len(rec.withdrawals()))
	}
}

func TestEndingDecisionClosesWhenTheModeWasExtendedOrEndedElsewhere(t *testing.T) {
	for name, settle := range map[string]func(*Notifier, string){
		"extended": func(n *Notifier, id string) { _, _ = n.Modes.Extend(context.Background(), id, ExtendTwoHours) },
		"ended":    func(n *Notifier, id string) { _, _, _ = n.Modes.End([]string{id}, false) },
	} {
		t.Run(name, func(t *testing.T) {
			n, rec, c := newNotifier(t)
			ctx := context.Background()
			res, _ := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"})
			c.t = t0.Add(50 * time.Minute)
			n.Flush(ctx, nil, c.t)
			d := rec.decisions()
			if len(d) != 1 || subjectOf(d[0]) != "usage-mode:ending:"+res.Mode.ID {
				t.Fatalf("ending decision = %+v", d)
			}
			settle(n, res.Mode.ID)
			n.Observe(ctx, c.t)
			n.Flush(ctx, nil, c.t)
			w := rec.withdrawals()
			if len(w) != 1 || subjectOf(w[0]) != subjectOf(d[0]) {
				t.Fatalf("withdrawals = %+v", w)
			}
			// The settled decision can no longer extend or end anything.
			before := n.Modes.List()
			_, _ = n.Apply(ctx, answer(d[0], AnswerExtend, "operator@example.com", "answered", false))
			if after := n.Modes.List(); len(after) != len(before) || (len(after) == 1 && !after[0].ExpiresAt.Equal(*before[0].ExpiresAt)) {
				t.Fatalf("a withdrawn decision changed a mode: %+v -> %+v", before, after)
			}
		})
	}
}

func TestRunsOutOfferClosesWhenTheWindowRecoversAndIsOfferedAgainIfItTurnsBad(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	reset := t0.Add(48 * time.Hour)
	bad := exhaustForecast(reset)
	if err := n.WillExhaust(ctx, bad, *bad.Projected, t0); err != nil {
		t.Fatal(err)
	}
	first := rec.decisions()[0]
	wantKey := "usage-mode:runs-out:" + bad.Window.Scope()
	if subjectOf(first) != strings.NewReplacer(" ", "-").Replace(wantKey) && !strings.HasPrefix(subjectOf(first), "usage-mode:runs-out:") {
		t.Fatalf("subject key = %q", subjectOf(first))
	}

	good := bad
	good.Status = usageforecast.StatusOnTrack
	n.Flush(ctx, []usageforecast.Forecast{good}, c.t)
	w := rec.withdrawals()
	if len(w) != 1 || subjectOf(w[0]) != subjectOf(first) || !strings.Contains(w[0].Payload["reason"].(string), "back on track") {
		t.Fatalf("withdrawals = %+v", w)
	}

	c.t = t0.Add(time.Hour)
	n.Flush(ctx, []usageforecast.Forecast{bad}, c.t)
	d := rec.decisions()
	if len(d) != 2 || subjectOf(d[1]) != subjectOf(first) || decisionID(d[1]) == decisionID(first) {
		t.Fatalf("a window that turns bad again must be offered again under the same subject: %d decisions", len(d))
	}
}

func TestRunsOutOfferClosesWhenAModeAlreadyCoversTheRunnerOrTheWindowReset(t *testing.T) {
	t.Run("mode started elsewhere", func(t *testing.T) {
		n, rec, c := newNotifier(t)
		ctx := context.Background()
		f := exhaustForecast(t0.Add(48 * time.Hour))
		_ = n.WillExhaust(ctx, f, *f.Projected, t0)
		if _, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"}); err != nil {
			t.Fatal(err)
		}
		n.Flush(ctx, []usageforecast.Forecast{f}, c.t)
		if w := rec.withdrawals(); len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "already on") {
			t.Fatalf("withdrawals = %+v", w)
		}
	})
	t.Run("window reset", func(t *testing.T) {
		n, rec, c := newNotifier(t)
		ctx := context.Background()
		f := exhaustForecast(t0.Add(48 * time.Hour))
		_ = n.WillExhaust(ctx, f, *f.Projected, t0)
		c.t = t0.Add(49 * time.Hour)
		n.Flush(ctx, nil, c.t)
		if w := rec.withdrawals(); len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "reset") {
			t.Fatalf("withdrawals = %+v", w)
		}
	})
}

func TestTimedAskClosesWhenAModeIsAlreadyOn(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	if _, err := n.AskToStart(ctx, "codex", 30*time.Minute); err != nil {
		t.Fatal(err)
	}
	ask := rec.decisions()[0]
	if subjectOf(ask) != "usage-mode:timed-start:codex:30m0s" {
		t.Fatalf("subject key = %q", subjectOf(ask))
	}
	if _, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"}); err != nil {
		t.Fatal(err)
	}
	n.Flush(ctx, nil, c.t)
	if w := rec.withdrawals(); len(w) != 1 || subjectOf(w[0]) != subjectOf(ask) {
		t.Fatalf("withdrawals = %+v", w)
	}
}

func TestWithdrawnResolutionIsNeverAnAnswer(t *testing.T) {
	n, rec, _ := newNotifier(t)
	ctx := context.Background()
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(ctx, f, *f.Projected, t0)
	res := answer(rec.decisions()[0], AnswerStart, "operator@example.com", operatordecision.StateWithdrawn, false)
	if handled, err := n.Apply(ctx, res); !handled || err != nil {
		t.Fatalf("apply = %v %v", handled, err)
	}
	if modes := n.Modes.List(); len(modes) != 0 {
		t.Fatalf("a withdrawn decision started a mode: %+v", modes)
	}
	if state, _, _ := n.AskState(decisionID(rec.decisions()[0])); state != operatordecision.PendingStateWithdrawn {
		t.Fatalf("ask state = %q, want withdrawn", state)
	}
}

type flakyPublisher struct {
	recorder
	fail bool
}

func (p *flakyPublisher) PublishDomainEvent(ctx context.Context, e eventbus.DomainEvent) error {
	if p.fail && e.EventType == withdrawnEventType {
		return errors.New("hub unavailable")
	}
	return p.recorder.PublishDomainEvent(ctx, e)
}

func TestFailedWithdrawalIsRetriedOnTheNextPass(t *testing.T) {
	n, _, c := newNotifier(t)
	pub := &flakyPublisher{}
	n.Publisher = pub
	ctx := context.Background()
	modeID, _ := startUntilOff(t, n, &pub.recorder, c)
	_, _, _ = n.Modes.End([]string{modeID}, false)
	n.Observe(ctx, c.t)
	pub.fail = true
	if errs := n.Flush(ctx, nil, c.t); len(errs) == 0 {
		t.Fatal("a failed withdrawal must be reported")
	}
	if len(pub.withdrawals()) != 0 {
		t.Fatal("nothing should be recorded as sent")
	}
	pub.fail = false
	if errs := n.Flush(ctx, nil, c.t); len(errs) != 0 {
		t.Fatal(errs)
	}
	if w := pub.withdrawals(); len(w) != 1 || subjectOf(w[0]) != "usage-mode:until-off:"+modeID {
		t.Fatalf("retry withdrawals = %+v", w)
	}
	n.Flush(ctx, nil, c.t)
	if len(pub.withdrawals()) != 1 {
		t.Fatal("withdrawal repeated after success")
	}
}

// legacyize rewrites a pending ask as an older build stored it: published, with
// no subject key.
func legacyize(n *Notifier, id string) {
	ask := n.load().Asks[id]
	delete(ask.Extra, "subject_key")
}

func TestKeylessAskClosedByEventIDWhenClearedAndReplacedWhenStillNeeded(t *testing.T) {
	t.Run("cleared", func(t *testing.T) {
		n, rec, c := newNotifier(t)
		ctx := context.Background()
		modeID, ask := startUntilOff(t, n, rec, c)
		legacyize(n, decisionID(ask))
		_, _, _ = n.Modes.End([]string{modeID}, false)
		n.Observe(ctx, c.t)
		n.Flush(ctx, nil, c.t)
		w := rec.withdrawals()
		if len(w) != 1 || w[0].Payload["requester_ref"] != n.load().Asks[decisionID(ask)].EventID || subjectOf(w[0]) != "" {
			t.Fatalf("a keyless ask must be closed by its request event id: %+v", w)
		}
	})
	t.Run("still needed", func(t *testing.T) {
		n, rec, c := newNotifier(t)
		ctx := context.Background()
		_, ask := startUntilOff(t, n, rec, c)
		legacyize(n, decisionID(ask))
		oldRef := n.load().Asks[decisionID(ask)].EventID
		n.Flush(ctx, nil, c.t)
		w := rec.withdrawals()
		if len(w) != 1 || w[0].Payload["requester_ref"] != oldRef {
			t.Fatalf("the keyless request must be closed by event id: %+v", w)
		}
		d := rec.decisions()
		if len(d) != 2 || subjectOf(d[1]) == "" {
			t.Fatalf("the still-needed decision must be raised again under a subject key: %d", len(d))
		}
		// Idempotent afterwards.
		n.Flush(ctx, nil, c.t)
		if len(rec.withdrawals()) != 1 || len(rec.decisions()) != 2 {
			t.Fatalf("reconcile repeated: %d withdrawals %d decisions", len(rec.withdrawals()), len(rec.decisions()))
		}
	})
}

func TestRoleDefaultProposalIsKeyedAndClosesOnceTheOperatorAppliedIt(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	n.Modes.Tiers = ResourceTiers{Files: policyFiles{"codex": proposalPolicy}}
	change := n.Modes.RoleDefaultChange(ctx, "codex", "code.economy.delivery", "gpt-6.1-terra", "gpt-6-luna")
	if _, err := n.ProposeRoleDefault(ctx, proposal(change)); err != nil {
		t.Fatal(err)
	}
	if got := subjectOf(rec.decisions()[0]); got != "usage-mode:role-default:um-l:code.economy.delivery" {
		t.Fatalf("subject key = %q", got)
	}
	// Not applied yet: the proposal stands.
	n.Flush(ctx, nil, c.t)
	if len(rec.withdrawals()) != 0 {
		t.Fatal("withdrew a proposal that is still open")
	}
	// The operator edits the policy file by hand.
	applied := strings.Replace(proposalPolicy, `"roles":{"code.economy.delivery":{"model":"gpt-6.1-terra"`, `"roles":{"code.economy.delivery":{"model":"gpt-6-luna"`, 1)
	n.Modes.Tiers = ResourceTiers{Files: policyFiles{"codex": applied}}
	n.Flush(ctx, nil, c.t)
	w := rec.withdrawals()
	if len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "already the default") {
		t.Fatalf("withdrawals = %+v", w)
	}
}

// The daily until-off decision is read cold on a phone, often days after the
// mode was started: it says what the mode does, what each answer does and
// what happens without one, and keeps the exact configuration for agents.
func TestUntilOffDecisionExplainsTheModeToAColdReader(t *testing.T) {
	n, rec, c := newNotifier(t)
	modeID, ask := startUntilOff(t, n, rec, c)
	p := ask.Payload
	q, _ := p["question"].(string)
	if !strings.Contains(q, "25 hours") || !strings.Contains(q, "Keep it on, or turn it off?") {
		t.Fatalf("question = %q", q)
	}
	big, _ := p["big_picture"].(string)
	for _, want := range []string{"new Codex work to cheaper models", "Keep it on and stop asking", "Turn it off", "If you do not answer, it stays on"} {
		if !strings.Contains(big, want) {
			t.Errorf("big picture lacks %q: %s", want, big)
		}
	}
	if strings.Contains(big, modeID) {
		t.Error("the mode id belongs in the technical details, not the big picture")
	}
	tech, _ := p["technical_details"].(string)
	if !strings.Contains(tech, "Usage mode "+modeID) || !strings.Contains(tech, "codex: ") || !strings.Contains(tech, "usage mode end "+modeID) {
		t.Fatalf("technical details lack the mode's id, mapping or end command: %s", tech)
	}
	if reason, _ := p["recommendation_reason"].(string); reason == "" {
		t.Fatal("the recommendation carries its reason")
	}
}

func TestOnForSaysDaysAfterTwoDays(t *testing.T) {
	for d, want := range map[time.Duration]string{25 * time.Hour: "25 hours", 97 * time.Hour: "about 4 days", 60 * time.Hour: "about 3 days"} {
		if got := onFor(d); got != want {
			t.Errorf("onFor(%v) = %q, want %q", d, got, want)
		}
	}
}
