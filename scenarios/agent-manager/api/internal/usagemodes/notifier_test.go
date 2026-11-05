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
	"github.com/vrooli/cli-core/usagemode"
)

type recorder struct{ events []eventbus.DomainEvent }

func (r *recorder) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	r.events = append(r.events, e)
	return nil
}

func (r *recorder) decisions() (out []eventbus.DomainEvent) {
	for _, e := range r.events {
		if e.EventType == DecisionEventType {
			out = append(out, e)
		}
	}
	return out
}

// notices are the plain notifications other than "mode started".
func (r *recorder) notices() (out []eventbus.DomainEvent) {
	for _, e := range r.events {
		if e.EventType == usageforecast.NotificationEventType && e.Payload["category"] != usageforecast.NotificationCategoryModeStarted {
			out = append(out, e)
		}
	}
	return out
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func exhaustForecast(reset time.Time) usageforecast.Forecast {
	projected := t0.Add(2 * time.Hour)
	return usageforecast.Forecast{
		Window:    &usageforecast.Window{Provider: "openai", Pool: "primary", Window: "w", WindowMinutes: 10080, ResetAt: reset},
		Status:    usageforecast.StatusWillExhaust,
		Projected: &projected,
	}
}

func newNotifier(t *testing.T) (*Notifier, *recorder, *clock) {
	t.Helper()
	reset := t0.Add(48 * time.Hour)
	svc := newService(t, span("openai", 10080, reset, t0.Add(-time.Hour), t0, 40))
	c := &clock{t: t0}
	svc.Now = c.now
	rec := &recorder{}
	return &Notifier{Modes: svc, Publisher: rec, Now: c.now}, rec, c
}

func answer(d eventbus.DomainEvent, key, by string, state string, byDefault bool) operatordecision.DecisionResolution {
	corr := map[string]string{}
	for k, v := range d.Payload["correlation"].(map[string]any) {
		corr[k] = v.(string)
	}
	return operatordecision.DecisionResolution{State: state, Answer: key, AnsweredBy: by, ByDefault: byDefault, SourceEventType: DecisionEventType, Correlation: corr}
}

func TestRunsOutOfferIsReversibleWithNoDefaultAndOnePerWindow(t *testing.T) {
	n, rec, _ := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	for i := 0; i < 3; i++ {
		if err := n.WillExhaust(context.Background(), f, *f.Projected, t0); err != nil {
			t.Fatal(err)
		}
	}
	got := rec.decisions()
	if len(got) != 1 {
		t.Fatalf("decisions = %d, want one per window", len(got))
	}
	p := got[0].Payload
	if p["reversible"] != true || p["default_answer"] != nil || p["recommended"] != AnswerStart {
		t.Fatalf("payload = %+v", p)
	}
	keys := []string{}
	for _, o := range p["options"].([]any) {
		keys = append(keys, o.(map[string]any)["key"].(string))
	}
	if len(keys) != 3 || keys[0] != AnswerStart || keys[1] != AnswerNotNow || keys[2] != AnswerRemind {
		t.Fatalf("options = %v", keys)
	}
}

func TestStartNeedsVerifiedHumanAndIsIdempotent(t *testing.T) {
	for name, res := range map[string]struct {
		by, state string
		def       bool
	}{"no identity": {"", "answered", false}, "default": {"default", "answered", true}, "expired": {"", "expired", false}} {
		t.Run(name, func(t *testing.T) {
			n, rec, _ := newNotifier(t)
			f := exhaustForecast(t0.Add(48 * time.Hour))
			_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
			if _, err := n.Apply(context.Background(), answer(rec.decisions()[0], AnswerStart, res.by, res.state, res.def)); err != nil {
				t.Fatal(err)
			}
			if modes := n.Modes.List(); len(modes) != 0 {
				t.Fatalf("a mode started without a verified human: %+v", modes)
			}
		})
	}
	n, rec, _ := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
	res := answer(rec.decisions()[0], AnswerStart, "operator@example.com", "answered", false)
	for i := 0; i < 2; i++ {
		if handled, err := n.Apply(context.Background(), res); !handled || err != nil {
			t.Fatalf("apply %d: %v %v", i, handled, err)
		}
	}
	modes := n.Modes.List()
	if len(modes) != 1 || modes[0].StartedBy != "operator@example.com" || modes[0].Until.Kind != usagemode.UntilWindowReset {
		t.Fatalf("modes = %+v", modes)
	}
	n.Observe(context.Background(), t0)
	if errs := n.Flush(context.Background(), nil, t0); len(errs) != 0 || len(rec.notices()) != 0 {
		t.Fatalf("a tap-started mode must not announce itself: %v %d", errs, len(rec.notices()))
	}
}

func TestNotNowIsNotAskedAgainUntilTheWindowResets(t *testing.T) {
	n, rec, c := newNotifier(t)
	reset := t0.Add(48 * time.Hour)
	f := exhaustForecast(reset)
	_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
	_, _ = n.Apply(context.Background(), answer(rec.decisions()[0], AnswerNotNow, "operator@example.com", "answered", false))
	c.t = t0.Add(6 * time.Hour)
	_ = n.WillExhaust(context.Background(), f, *f.Projected, c.t)
	n.Flush(context.Background(), []usageforecast.Forecast{f}, c.t)
	if len(rec.decisions()) != 1 {
		t.Fatalf("re-prompted inside the same window: %d decisions", len(rec.decisions()))
	}
	c.t = reset.Add(time.Hour)
	next := exhaustForecast(reset.Add(168 * time.Hour))
	_ = n.WillExhaust(context.Background(), next, c.t.Add(time.Hour), c.t)
	if len(rec.decisions()) != 2 {
		t.Fatalf("a new window should be offered again: %d", len(rec.decisions()))
	}
}

func TestRemindAsksAgainAfterThirtyMinutes(t *testing.T) {
	n, rec, c := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
	_, _ = n.Apply(context.Background(), answer(rec.decisions()[0], AnswerRemind, "operator@example.com", "answered", false))
	c.t = t0.Add(20 * time.Minute)
	n.Flush(context.Background(), []usageforecast.Forecast{f}, c.t)
	if len(rec.decisions()) != 1 {
		t.Fatal("reminded too early")
	}
	c.t = t0.Add(31 * time.Minute)
	n.Flush(context.Background(), []usageforecast.Forecast{f}, c.t)
	if len(rec.decisions()) != 2 {
		t.Fatalf("decisions = %d, want the reminder", len(rec.decisions()))
	}
}

func TestEndedModeMergesWithBackOnTrackAndIsNotAnnouncedTwice(t *testing.T) {
	n, rec, c := newNotifier(t)
	res, err := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "operator@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	n.Observe(context.Background(), c.t)
	c.t = t0.Add(2 * time.Hour)
	n.Observe(context.Background(), c.t)
	ended := n.EndedFor("openai")
	if len(ended) != 1 || ended[0].ID != res.Mode.ID {
		t.Fatalf("ended = %+v", ended)
	}
	n.Announced([]string{res.Mode.ID})
	n.Flush(context.Background(), nil, c.t)
	if len(rec.notices()) != 0 {
		t.Fatalf("merged mode end was also sent alone: %d", len(rec.notices()))
	}
	// Unmerged, it is sent once.
	res2, _ := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"})
	c.t = c.t.Add(time.Minute)
	n.Observe(context.Background(), c.t)
	c.t = c.t.Add(2 * time.Hour)
	n.Observe(context.Background(), c.t)
	n.Flush(context.Background(), nil, c.t)
	n.Flush(context.Background(), nil, c.t)
	if got := rec.notices(); len(got) != 1 || got[0].Payload["category"] != usageforecast.NotificationCategoryModeEnded || got[0].Payload["dedupe_key"] != "mode-ended|"+res2.Mode.ID {
		t.Fatalf("notices = %+v", got)
	}
}

func TestEndingSoonExtendsOnlyForVerifiedHumanAndUntilOffReminderTurnsOff(t *testing.T) {
	n, rec, c := newNotifier(t)
	res, _ := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"})
	c.t = t0.Add(50 * time.Minute)
	n.Flush(context.Background(), nil, c.t)
	n.Flush(context.Background(), nil, c.t)
	d := rec.decisions()
	if len(d) != 1 {
		t.Fatalf("ending decisions = %d", len(d))
	}
	if _, err := n.Apply(context.Background(), answer(d[0], AnswerExtend, "", "answered", false)); err != nil {
		t.Fatal(err)
	}
	if got := n.Modes.List()[0]; !got.ExpiresAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("unverified extend moved the end to %v", got.ExpiresAt)
	}
	n.mu.Lock()
	n.load().Asks[d[0].Payload["correlation"].(map[string]any)["decision_id"].(string)].State = operatordecision.PendingStatePending
	n.mu.Unlock()
	_, _ = n.Apply(context.Background(), answer(d[0], AnswerExtend, "operator@example.com", "answered", false))
	if got := n.Modes.List()[0]; got.ID != res.Mode.ID || !got.ExpiresAt.Equal(t0.Add(3*time.Hour)) || got.StartedBy != "o@example.com" {
		t.Fatalf("extended mode = %+v", got)
	}

	off, _ := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "off", StartedBy: "o@example.com"})
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	n.Flush(context.Background(), nil, c.t)
	var reminder eventbus.DomainEvent
	for _, e := range rec.decisions() {
		if e.Payload["correlation"].(map[string]any)["kind"] == KindUntilOf {
			reminder = e
		}
	}
	if reminder.EventType == "" {
		t.Fatal("no until-off reminder after 24 hours")
	}
	_, _ = n.Apply(context.Background(), answer(reminder, AnswerTurnOf, "operator@example.com", "answered", false))
	for _, m := range n.Modes.listAt(c.t) {
		if m.ID == off.Mode.ID {
			t.Fatal("turn off did not end the mode")
		}
	}
}

func TestAskedTimedStartIsAnAskOnlyAndStartsOnlyForAVerifiedHuman(t *testing.T) {
	n, rec, _ := newNotifier(t)
	id, err := n.AskToStart(context.Background(), "codex", 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := n.AskToStart(context.Background(), "codex", 30*time.Minute)
	if again != id || len(rec.decisions()) != 1 || len(n.Modes.List()) != 0 {
		t.Fatalf("asking must be idempotent and start nothing: %s %s %d", id, again, len(rec.decisions()))
	}
	p := rec.decisions()[0].Payload
	if p["subject"] != "Usage-mode validation: start a 30-minute Save mode on Codex?" || p["default_answer"] != nil || p["reversible"] != true {
		t.Fatalf("payload = %+v", p)
	}
	if _, err := n.AskToStart(context.Background(), "codex", time.Second); !errors.Is(err, ErrInvalid) {
		t.Fatalf("short length err = %v", err)
	}
	_, _ = n.Apply(context.Background(), answer(rec.decisions()[0], AnswerStart, "operator@example.com", "answered", false))
	modes := n.Modes.List()
	if len(modes) != 1 || modes[0].StartedBy != "operator@example.com" || !modes[0].ExpiresAt.Equal(t0.Add(30*time.Minute)) {
		t.Fatalf("modes = %+v", modes)
	}
	if state, _, _ := n.AskState(id); state != operatordecision.PendingStateApproved {
		t.Fatalf("state = %s", state)
	}
}

type evidenceStub struct{ reports []EvidenceReport }

func (e evidenceStub) Ready(context.Context, time.Time) ([]EvidenceReport, error) {
	return e.reports, nil
}

func TestEvidenceReadyNoticeIsOffByDefaultAndSentOncePerMode(t *testing.T) {
	n, rec, _ := newNotifier(t)
	n.Evidence = evidenceStub{reports: []EvidenceReport{{ModeID: "um-1", Label: "Save usage", Line: "Codex was cheaper per outcome."}}}
	count := func() int {
		c := 0
		for _, e := range rec.notices() {
			if e.Payload["category"] == usageforecast.NotificationCategoryModeEvidence {
				c++
			}
		}
		return c
	}
	n.Flush(context.Background(), nil, t0)
	if count() != 0 {
		t.Fatalf("evidence notice sent while off")
	}
	n.EvidenceReady = true
	n.Flush(context.Background(), nil, t0)
	n.Flush(context.Background(), nil, t0.Add(time.Hour))
	if count() != 1 {
		t.Fatalf("evidence notices = %d, want exactly 1", count())
	}
}

type qualityStub struct {
	finding QualityFinding
	ok      bool
	calls   int
}

func (q *qualityStub) QualityDrop(context.Context, usagemode.Mode, time.Time) (QualityFinding, bool, error) {
	q.calls++
	return q.finding, q.ok, nil
}

// The quality-drop notice needs a source (off otherwise), waits an hour into
// the mode, is checked at most hourly, is sent once per mode and says what was
// counted and how sure the count is, with no message text.
func TestQualityDropNoticeIsOnlyForAProvenRiseOncePerMode(t *testing.T) {
	n, rec, c := newNotifier(t)
	count := func() (found []eventbus.DomainEvent) {
		for _, e := range rec.notices() {
			if e.Payload["category"] == NotificationCategoryQualityDrop {
				found = append(found, e)
			}
		}
		return found
	}
	res, err := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "off", StartedBy: "operator@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	n.Observe(context.Background(), c.t)
	n.Flush(context.Background(), nil, c.t.Add(2*time.Hour))
	if len(count()) != 0 {
		t.Fatal("a quality notice was sent with no quality source")
	}
	stub := &qualityStub{finding: QualityFinding{Rate: 0.31, Baseline: 0.14, DifferenceLow: 0.02, DifferenceHigh: 0.31, Messages: 42, BaselineMessages: 60, Model: "haiku", ClassifierPrecision: 0.8}}
	n.Quality = stub
	n.Flush(context.Background(), nil, c.t.Add(30*time.Minute))
	if stub.calls != 0 {
		t.Fatal("the quality source was asked before the mode was an hour old")
	}
	n.Flush(context.Background(), nil, c.t.Add(2*time.Hour))
	if stub.calls != 1 || len(count()) != 0 {
		t.Fatalf("no proven rise yet: calls %d, notices %d", stub.calls, len(count()))
	}
	n.Flush(context.Background(), nil, c.t.Add(2*time.Hour+10*time.Minute))
	if stub.calls != 1 {
		t.Fatal("the quality source was asked twice within the hour")
	}
	stub.ok = true
	n.Flush(context.Background(), nil, c.t.Add(4*time.Hour))
	n.Flush(context.Background(), nil, c.t.Add(6*time.Hour))
	sent := count()
	if len(sent) != 1 || sent[0].Payload["dedupe_key"] != "mode-quality-drop|"+res.Mode.ID {
		t.Fatalf("notices = %+v, want exactly one for the mode", sent)
	}
	body, _ := sent[0].Payload["body"].(string)
	for _, want := range []string{"14%", "31%", "haiku", "42 messages", "80%", "can be wrong"} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q lacks %q", body, want)
		}
	}
}

func TestResolvedEndingAndUntilOffAsksAreNotRaisedAgain(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	_, _ = n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"})
	off, _ := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "off", StartedBy: "o@example.com"})
	c.t = t0.Add(50 * time.Minute)
	n.Flush(ctx, nil, c.t)
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	n.Flush(ctx, nil, c.t)
	first := rec.decisions()
	if len(first) != 2 {
		t.Fatalf("want one ending and one until-off decision, got %d", len(first))
	}
	for _, d := range first {
		if _, err := n.Apply(ctx, answer(d, AnswerLetEnd, "o@example.com", "answered", false)); err != nil {
			t.Fatal(err)
		}
		_, _ = n.Apply(ctx, answer(d, AnswerKeepOn, "o@example.com", "answered", false))
	}
	for i := 0; i < 3; i++ {
		n.Flush(ctx, nil, c.t)
	}
	if got := len(rec.decisions()); got != 2 {
		t.Fatalf("resolved asks were raised again: %d decisions", got)
	}
	c.t = c.t.Add(24 * time.Hour)
	n.Flush(ctx, nil, c.t)
	if got := len(rec.decisions()); got != 3 {
		t.Fatalf("until-off reminds daily, once per day: %d decisions", got)
	}
}

func TestTimedAskCannotFloodDecisions(t *testing.T) {
	n, rec, _ := newNotifier(t)
	ctx := context.Background()
	var floods int
	for minutes := 6; minutes <= 600; minutes += 7 { // an agent varying the length every call
		if _, err := n.AskToStart(ctx, "codex", time.Duration(minutes)*time.Minute); errors.Is(err, operatordecision.ErrFlood) {
			floods++
		}
	}
	if got := len(rec.decisions()); got > MaxOutstandingTimedAsks {
		t.Fatalf("%d decisions reached the operator; the cap is %d per runner", got, MaxOutstandingTimedAsks)
	}
	if floods == 0 {
		t.Fatal("the cap never refused anything")
	}
	a, _ := n.AskToStart(ctx, "claude", 61*time.Minute)
	b, _ := n.AskToStart(ctx, "claude", 62*time.Minute)
	if a == "" || a != b {
		t.Fatalf("nearby lengths must round to one ask: %q %q", a, b)
	}
}

func TestApplyActsOnTheStoredAskNotTheEchoedCorrelation(t *testing.T) {
	n, rec, _ := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
	res := answer(rec.decisions()[0], AnswerStart, "operator@example.com", "answered", false)
	res.Correlation["runner"], res.Correlation["until"], res.Correlation["kind"] = "claude", "off", KindTimed
	if _, err := n.Apply(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	modes := n.Modes.List()
	if len(modes) != 1 || modes[0].Until.Kind != usagemode.UntilWindowReset || usagemode.NormalizeRunner(modes[0].Scope.Runner) != "codex" {
		t.Fatalf("a forged correlation changed what started: %+v", modes)
	}
}

func TestLateAnswerStartsNothing(t *testing.T) {
	n, rec, _ := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(context.Background(), f, *f.Projected, t0)
	res := answer(rec.decisions()[0], AnswerStart, "operator@example.com", "answered", false)
	res.Late = true
	_, _ = n.Apply(context.Background(), res)
	if modes := n.Modes.List(); len(modes) != 0 {
		t.Fatalf("a late answer started a mode: %+v", modes)
	}
}
