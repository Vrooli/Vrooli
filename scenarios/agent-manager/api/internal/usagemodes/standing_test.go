package usagemodes

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/usageforecast"

	"github.com/vrooli/api-core/operatordecision"
	"google.golang.org/protobuf/types/known/structpb"
)

const operatorSubject = "operator@example.com"

func untilOffDecisionCount(rec *recorder) int {
	count := 0
	for _, d := range rec.decisions() {
		if corr, _ := d.Payload["correlation"].(map[string]any); corr["kind"] == KindUntilOf {
			count++
		}
	}
	return count
}

// An until-off mode answered "keep it on" is a standing mode: the daily
// question stops, the mode stays on, and it is still named in the digest.
func TestKeepOnIsAStandingChoiceThatStopsTheDailyQuestion(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	off, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "off", StartedBy: operatorSubject})
	if err != nil {
		t.Fatal(err)
	}
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	n.Flush(ctx, nil, c.t)
	if untilOffDecisionCount(rec) != 1 {
		t.Fatalf("the first daily question was not raised: %d", untilOffDecisionCount(rec))
	}
	d := rec.decisions()[0]
	if d.Payload["recommended"] != AnswerKeepOn {
		t.Fatalf("recommended = %v, want keep_on", d.Payload["recommended"])
	}
	var keepLabel string
	for _, o := range d.Payload["options"].([]any) {
		if opt := o.(map[string]any); opt["key"] == AnswerKeepOn {
			keepLabel, _ = opt["label"].(string)
		}
	}
	if !strings.Contains(keepLabel, "stop asking") {
		t.Fatalf("the keep option must say it stops the question: %q", keepLabel)
	}
	if _, err := n.Apply(ctx, answer(d, AnswerKeepOn, operatorSubject, "answered", false)); err != nil {
		t.Fatal(err)
	}
	for day := 2; day <= 6; day++ {
		c.t = off.Mode.StartedAt.Add(time.Duration(day)*24*time.Hour + time.Hour)
		n.Flush(ctx, nil, c.t)
	}
	if got := untilOffDecisionCount(rec); got != 1 {
		t.Fatalf("a mode kept on was asked about again: %d until-off decisions", got)
	}
	if len(n.Modes.List()) != 1 {
		t.Fatal("keeping the mode on ended it")
	}
	if note := n.DigestNote(c.t); !strings.Contains(note, "kept on by you") {
		t.Fatalf("a standing mode must stay visible in the digest: %q", note)
	}
}

func TestKeepOnNeedsAVerifiedHumanAndTurnOffStillWorks(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	off, _ := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "off", StartedBy: operatorSubject})
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	n.Flush(ctx, nil, c.t)
	d := rec.decisions()[0]
	if _, err := n.Apply(ctx, answer(d, AnswerKeepOn, "default", "answered", true)); err != nil {
		t.Fatal(err)
	}
	if len(n.load().Standing) != 0 {
		t.Fatal("a default answer recorded a standing mode")
	}
	c.t = c.t.Add(24 * time.Hour)
	n.Flush(ctx, nil, c.t)
	if untilOffDecisionCount(rec) != 2 {
		t.Fatalf("an unanswered until-off ask must be raised again the next day: %d", untilOffDecisionCount(rec))
	}
	if _, err := n.Apply(ctx, answer(rec.decisions()[len(rec.decisions())-1], AnswerTurnOf, operatorSubject, "answered", false)); err != nil {
		t.Fatal(err)
	}
	if len(n.Modes.List()) != 0 {
		t.Fatal("turn off no longer ends the mode")
	}
}

func TestStandingMarkIsForgottenWhenTheModeEnds(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	off, _ := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "off", StartedBy: operatorSubject})
	c.t = off.Mode.StartedAt.Add(25 * time.Hour)
	n.Flush(ctx, nil, c.t)
	_, _ = n.Apply(ctx, answer(rec.decisions()[0], AnswerKeepOn, operatorSubject, "answered", false))
	if _, _, err := n.Modes.End([]string{off.Mode.ID}, false); err != nil {
		t.Fatal(err)
	}
	n.Observe(ctx, c.t)
	if len(n.load().Standing) != 0 {
		t.Fatalf("standing marks outlived the mode: %+v", n.load().Standing)
	}
}

func autoSaveOffer(t *testing.T, n *Notifier, rec *recorder) operatordecision.DecisionResolution {
	t.Helper()
	f := exhaustForecast(t0.Add(48 * time.Hour))
	if err := n.WillExhaust(context.Background(), f, *f.Projected, t0); err != nil {
		t.Fatal(err)
	}
	return answer(rec.decisions()[0], AnswerAutoSave, operatorSubject, "answered", false)
}

// Answering "do it automatically" starts the mode and records the rule; the
// next window is handled with a notice and no question.
func TestAutoSaveAnswerStartsNowAndThenReplacesTheQuestionWithANotice(t *testing.T) {
	n, rec, _ := newNotifier(t)
	ctx := context.Background()
	res := autoSaveOffer(t, n, rec)
	if _, err := n.Apply(ctx, res); err != nil {
		t.Fatal(err)
	}
	modes := n.Modes.List()
	if len(modes) != 1 || modes[0].StartedBy != operatorSubject {
		t.Fatalf("the answer must start one mode for the human who answered: %+v", modes)
	}
	rules := n.AutoSaveRules()
	if len(rules) != 1 || rules[0].Runner != "codex" || rules[0].By != operatorSubject {
		t.Fatalf("rules = %+v", rules)
	}
	if note := n.DigestNote(t0); !strings.Contains(note, "Auto-save rule on for Codex") {
		t.Fatalf("the rule must be visible in the digest: %q", note)
	}

	// The window resets, the mode ends, and the next window is projected to run out.
	_, _, _ = n.Modes.End([]string{modes[0].ID}, false)
	later := t0.Add(72 * time.Hour)
	next := exhaustForecast(later.Add(7 * 24 * time.Hour))
	projected := later.Add(2 * time.Hour)
	next.Projected = &projected
	before := len(rec.decisions())
	for i := 0; i < 3; i++ {
		if err := n.WillExhaust(ctx, next, projected, later); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.decisions()) != before {
		t.Fatalf("a rule asked the operator instead of acting: %d new decisions", len(rec.decisions())-before)
	}
	auto := n.Modes.List()
	if len(auto) != 1 || auto[0].StartedBy != AutoSaveStarter {
		t.Fatalf("the rule did not start exactly one mode: %+v", auto)
	}
	var notice map[string]any
	for _, e := range rec.events {
		if e.EventType == usageforecast.NotificationEventType && strings.HasPrefix(e.Payload["dedupe_key"].(string), "mode-auto-save|") {
			notice = e.Payload
		}
	}
	if notice == nil {
		t.Fatal("the rule started a mode without telling the operator")
	}
	body, _ := notice["body"].(string)
	for _, want := range []string{"auto-save rule", "agent-manager usage mode auto-save off --runner codex", "End now"} {
		if !strings.Contains(body, want) {
			t.Errorf("notice %q lacks %q", body, want)
		}
	}
	// Ended early by the operator inside the same window: the rule does not start it again.
	_, _, _ = n.Modes.End([]string{auto[0].ID}, false)
	if err := n.WillExhaust(ctx, next, projected, later.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(n.Modes.List()) != 0 {
		t.Fatal("the rule overrode the operator ending the mode in the same window")
	}
}

func TestAutoSaveNeedsAVerifiedHumanAndTurnsOffCleanly(t *testing.T) {
	n, rec, _ := newNotifier(t)
	ctx := context.Background()
	res := autoSaveOffer(t, n, rec)
	res.AnsweredBy = ""
	if _, err := n.Apply(ctx, res); err != nil {
		t.Fatal(err)
	}
	if len(n.AutoSaveRules()) != 0 || len(n.Modes.List()) != 0 {
		t.Fatal("an answer without a verified human created a rule or a mode")
	}

	if err := n.SetAutoSave("codex", true, operatorSubject); err != nil {
		t.Fatal(err)
	}
	if err := n.SetAutoSave("gemini", true, operatorSubject); err == nil {
		t.Fatal("a rule for an unknown runner was accepted")
	}
	if err := n.SetAutoSave("codex", false, operatorSubject); err != nil {
		t.Fatal(err)
	}
	if len(n.AutoSaveRules()) != 0 {
		t.Fatal("the rule did not turn off")
	}
	// With the rule off the question comes back as a decision.
	f := exhaustForecast(t0.Add(72 * time.Hour))
	projected := t0.Add(3 * time.Hour)
	f.Projected = &projected
	n2, rec2, _ := newNotifier(t)
	_ = n2.WillExhaust(ctx, f, projected, t0)
	if len(rec2.decisions()) != 1 || len(n2.Modes.List()) != 0 {
		t.Fatal("without a rule a projected run-out must be asked, not started")
	}
}

func TestActiveModeDecisionEnablesFutureRuleWithoutStartingAnotherMode(t *testing.T) {
	n, rec, c := newNotifier(t)
	ctx := context.Background()
	active, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "weekly-reset", StartedBy: operatorSubject})
	if err != nil {
		t.Fatal(err)
	}
	f := exhaustForecast(active.Mode.ExpiresAt.UTC())
	if errs := n.Flush(ctx, []usageforecast.Forecast{f}, c.t); len(errs) != 0 {
		t.Fatalf("forecast flush errors: %v", errs)
	}
	decisions := rec.decisions()
	if len(decisions) != 1 {
		t.Fatalf("active low-power mode should raise one future-rule decision, got %d", len(decisions))
	}
	d := decisions[0]
	payload := d.Payload
	if _, err := structpb.NewStruct(payload); err != nil {
		t.Fatalf("decision payload is not publishable as a Hub event: %v", err)
	}
	correlation := payload["correlation"].(map[string]any)
	if correlation["kind"] != KindAutoSaveRule {
		t.Fatalf("decision kind = %v, want future-rule-only", correlation["kind"])
	}
	data := payload["template_data"].(map[string]any)
	if data["meta"].(map[string]any)["notice"] == "" || !strings.Contains(data["meta"].(map[string]any)["notice"].(string), "does not start or change") {
		t.Fatalf("page does not explain future-only effect: %+v", data["meta"])
	}
	options := data["options"].([]any)
	if len(options) != 2 || options[0].(map[string]any)["key"] != AnswerAutoSave || options[1].(map[string]any)["key"] != AnswerNotNow || options[0].(map[string]any)["projection"] != nil {
		t.Fatalf("future-rule options must not imply a current projection or start: %+v", options)
	}
	window := data["window"].(map[string]any)
	if window["source"].(map[string]any)["status"] != "real" || len(window["readings"].([]any)) != 1 {
		t.Fatalf("real quota source and reading missing: %+v", window)
	}
	if _, err := n.Apply(ctx, answer(d, AnswerAutoSave, operatorSubject, "answered", false)); err != nil {
		t.Fatal(err)
	}
	if got := n.Modes.List(); len(got) != 1 || got[0].ID != active.Mode.ID {
		t.Fatalf("future-only answer changed active modes: %+v", got)
	}
	rules := n.AutoSaveRules()
	if len(rules) != 1 || rules[0].Runner != "codex" || rules[0].By != operatorSubject || rules[0].Starts != 0 {
		t.Fatalf("future rule was not recorded without an immediate start: %+v", rules)
	}

	if _, _, err := n.Modes.End([]string{active.Mode.ID}, false); err != nil {
		t.Fatal(err)
	}
	later := active.Mode.ExpiresAt.Add(time.Minute)
	next := exhaustForecast(later.Add(7 * 24 * time.Hour))
	projected := later.Add(time.Hour)
	next.Projected = &projected
	if err := n.WillExhaust(ctx, next, projected, later); err != nil {
		t.Fatal(err)
	}
	autoModes := n.Modes.List()
	if len(autoModes) != 1 || autoModes[0].StartedBy != AutoSaveStarter {
		t.Fatalf("future rule did not start low power on a later projected run-out: %+v", autoModes)
	}
}

func TestFutureAutoSaveRuleAnswerRequiresVerifiedHuman(t *testing.T) {
	n, rec, c := newNotifier(t)
	active, err := n.Modes.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "weekly-reset", StartedBy: operatorSubject})
	if err != nil {
		t.Fatal(err)
	}
	f := exhaustForecast(active.Mode.ExpiresAt.UTC())
	if err := n.WillExhaust(context.Background(), f, *f.Projected, c.t); err != nil {
		t.Fatal(err)
	}
	res := answer(rec.decisions()[0], AnswerAutoSave, "", "answered", false)
	if _, err := n.Apply(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	if len(n.AutoSaveRules()) != 0 || len(n.Modes.List()) != 1 {
		t.Fatal("unverified answer enabled a future rule or changed the active mode")
	}
}

func TestAutoSaveRuleRespectsADeclinedWindow(t *testing.T) {
	n, rec, _ := newNotifier(t)
	ctx := context.Background()
	f := exhaustForecast(t0.Add(48 * time.Hour))
	_ = n.WillExhaust(ctx, f, *f.Projected, t0)
	if _, err := n.Apply(ctx, answer(rec.decisions()[0], AnswerNotNow, operatorSubject, "answered", false)); err != nil {
		t.Fatal(err)
	}
	if err := n.SetAutoSave("codex", true, operatorSubject); err != nil {
		t.Fatal(err)
	}
	_ = n.WillExhaust(ctx, f, *f.Projected, t0.Add(time.Minute))
	if len(n.Modes.List()) != 0 {
		t.Fatal("the rule started a mode in a window the operator declined")
	}
}
