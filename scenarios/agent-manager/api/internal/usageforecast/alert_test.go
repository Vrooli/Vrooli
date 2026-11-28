package usageforecast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/pricing"

	"github.com/jmoiron/sqlx"
	"github.com/vrooli/api-core/eventbus"
	_ "modernc.org/sqlite"
)

type staticReadings struct{ spans []pricing.QuotaReadingSpan }

func (s *staticReadings) QuotaReadingSpans(_ context.Context, _ string, since time.Time) ([]pricing.QuotaReadingSpan, error) {
	var out []pricing.QuotaReadingSpan
	for _, span := range s.spans {
		if !span.LastAt.Before(since) {
			out = append(out, span)
		}
	}
	return out, nil
}

type recordingPublisher struct {
	events []eventbus.DomainEvent
	fail   error
}

func (p *recordingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	if p.fail != nil {
		return p.fail
	}
	p.events = append(p.events, event)
	return nil
}

var storeSeq atomic.Int64

func sqliteAlertStore(t *testing.T) *SQLiteAlertStore {
	t.Helper()
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), storeSeq.Add(1)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewSQLiteAlertStore(db)
}

func TestWillExhaustNotificationIsDedupedPerWindowAndResentOnTwelveHourShift(t *testing.T) {
	ctx := context.Background()
	readings := &staticReadings{spans: spans(oct28Shape())}
	publisher := &recordingPublisher{}
	store := sqliteAlertStore(t)
	now := at("2026-10-29T04:40:00Z")
	alerter := &Alerter{Readings: readings, Store: store, Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC, UsageURL: func(context.Context) string { return "https://am.example/usage" }}

	evaluate := func(want int) {
		t.Helper()
		sent, err := alerter.Evaluate(ctx)
		if err != nil || sent != want {
			t.Fatalf("sent=%d err=%v want %d", sent, err, want)
		}
	}
	evaluate(1)
	event := publisher.events[0]
	if event.EventType != NotificationEventType || !strings.HasSuffix(event.EventType, ".notification.v1") || event.Source != "agent-manager" {
		t.Fatalf("event=%+v", event)
	}
	payload := event.Payload
	if payload["category"] != NotificationCategory || payload["context_url"] != "https://am.example/usage?window=openai/primary/10080m" || payload["urgency"] != "high" {
		t.Fatalf("payload=%v", payload)
	}
	if title := payload["title"].(string); title != "Codex · weekly runs out early" {
		t.Fatalf("title=%q", title)
	}
	if body := payload["body"].(string); !strings.Contains(body, "33% used") || !strings.Contains(body, "Resets Wed Nov 4 06:57 UTC") || !strings.Contains(body, "at most 0.46%/h") {
		t.Fatalf("body=%q", body)
	}

	// The same window and projection is not notified twice, including after a
	// restart (the state is durable).
	evaluate(0)
	alerter.Store = NewSQLiteAlertStore(store.db)
	evaluate(0)

	// Burn speeds up: the projection moves earlier, but by less than 12h.
	readings.spans = append(readings.spans, spans(codexWindow(time.Unix(1793775438, 0).UTC(), at("2026-10-29T05:30:00Z"), ramp(at("2026-10-29T04:50:00Z"), at("2026-10-29T05:30:00Z"), 34, 36)))...)
	now = at("2026-10-29T05:35:00Z")
	evaluate(0)

	// A sustained surge moves it earlier by more than 12h: notify again with a
	// distinct dedupe key.
	readings.spans = append(readings.spans, spans(codexWindow(time.Unix(1793775438, 0).UTC(), at("2026-10-29T12:00:00Z"), ramp(at("2026-10-29T05:40:00Z"), at("2026-10-29T12:00:00Z"), 37, 70)))...)
	now = at("2026-10-29T12:05:00Z")
	evaluate(1)
	if len(publisher.events) != 2 || publisher.events[1].Payload["dedupe_key"] == publisher.events[0].Payload["dedupe_key"] {
		t.Fatalf("re-send must carry a new dedupe key: %v", publisher.events)
	}
	if !strings.HasPrefix(publisher.events[0].Payload["dedupe_key"].(string), "openai|primary|10080m|2026-11-04T06:57Z|") {
		t.Fatalf("dedupe key must name the window: %v", publisher.events[0].Payload["dedupe_key"])
	}
}

func TestWillExhaustNotificationRetriesAfterAFailedPublishAndIgnoresHealthyWindows(t *testing.T) {
	ctx := context.Background()
	publisher := &recordingPublisher{fail: errors.New("vrooli-events down")}
	now := at("2026-10-29T04:40:00Z")
	alerter := &Alerter{Readings: &staticReadings{spans: spans(oct28Shape())}, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC}
	if sent, err := alerter.Evaluate(ctx); sent != 0 || err == nil {
		t.Fatalf("a failed publish must report an error and save nothing: sent=%d err=%v", sent, err)
	}
	publisher.fail = nil
	if sent, err := alerter.Evaluate(ctx); sent != 1 || err != nil {
		t.Fatalf("retry: sent=%d err=%v", sent, err)
	}
	if publisher.events[0].Payload["context_url"] != nil {
		t.Fatal("an unresolved Usage URL must omit context_url")
	}

	reset := at("2026-11-04T06:57:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	healthyEvents := &recordingPublisher{}
	healthy := &Alerter{Readings: &staticReadings{spans: spans(codexWindow(reset, start.Add(48*time.Hour), ramp(start.Add(time.Hour), start.Add(48*time.Hour), 1, 10)))}, Store: sqliteAlertStore(t), Publisher: healthyEvents, Location: time.UTC, Now: func() time.Time { return start.Add(48 * time.Hour) }}
	if sent, err := healthy.Evaluate(ctx); sent != 0 || err != nil {
		t.Fatalf("on_track windows are not notified: sent=%d err=%v %v", sent, err, healthyEvents.categories())
	}
}

type stubActivity struct {
	last time.Time
	err  error
}

func (s stubActivity) LastProviderActivity(context.Context, string) (time.Time, error) {
	return s.last, s.err
}

type stubActions []Action

func (s stubActions) TopActions(_ context.Context, _ *Window, n int, _ time.Time) ([]Action, error) {
	if n < len(s) {
		return s[:n], nil
	}
	return s, nil
}

func (r *recordingPublisher) categories() []string {
	var out []string
	for _, e := range r.events {
		out = append(out, e.Payload["category"].(string))
	}
	return out
}

func TestWillExhaustBodyNamesTheTopActionAndItsSaving(t *testing.T) {
	publisher := &recordingPublisher{}
	now := at("2026-10-29T04:40:00Z")
	alerter := &Alerter{
		Readings: &staticReadings{spans: spans(oct28Shape())}, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC,
		Actions: stubActions{{Text: "Brief sub-agents in session 76d970de", Saving: 90, Recoverable: 46.4}, {Text: "second", Saving: 5, Recoverable: 5}},
	}
	if sent, err := alerter.Evaluate(context.Background()); sent != 1 || err != nil {
		t.Fatalf("sent=%d err=%v", sent, err)
	}
	body := publisher.events[0].Payload["body"].(string)
	if !strings.Contains(body, "Biggest lever: Brief sub-agents in session 76d970de (could save ≈ 46 pts before the reset).") || strings.Contains(body, "second") {
		t.Fatalf("body must name only the #1 action and its saving: %q", body)
	}
}

func TestBackOnTrackIsSentOnceAfterANotifiedWindowRecovers(t *testing.T) {
	ctx := context.Background()
	publisher := &recordingPublisher{}
	readings := &staticReadings{spans: spans(oct28Shape())}
	store := sqliteAlertStore(t)
	now := at("2026-10-29T04:40:00Z")
	alerter := &Alerter{Readings: readings, Store: store, Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC, Activity: stubActivity{last: at("2026-10-29T04:30:00Z")}}
	evaluate := func(want int) {
		t.Helper()
		if sent, err := alerter.Evaluate(ctx); err != nil || sent != want {
			t.Fatalf("sent=%d err=%v want %d (%v)", sent, err, want, publisher.categories())
		}
	}
	evaluate(1)
	evaluate(0) // warning stands: nothing more is said

	// Eight hours later with no work since the last reading the window is idle
	// and the earlier warning is withdrawn, once.
	now = at("2026-10-29T12:40:00Z")
	evaluate(0) // first seen idle: the withdrawal waits out SafeFor
	now = at("2026-10-29T13:20:00Z")
	evaluate(1)
	evaluate(0)
	if got := publisher.categories(); len(got) != 2 || got[1] != NotificationCategoryRecovery {
		t.Fatalf("categories=%v", got)
	}
	recovery := publisher.events[1].Payload
	if !strings.Contains(recovery["body"].(string), "no longer applies") || recovery["urgency"] != "low" {
		t.Fatalf("recovery=%v", recovery)
	}
}

func TestBackOnTrackIsSentWhenANotifiedWindowResets(t *testing.T) {
	ctx := context.Background()
	publisher := &recordingPublisher{}
	readings := &staticReadings{spans: spans(oct28Shape())}
	now := at("2026-10-29T04:40:00Z")
	alerter := &Alerter{Readings: readings, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC}
	if sent, _ := alerter.Evaluate(ctx); sent != 1 {
		t.Fatalf("warning sent=%d", sent)
	}
	// An early reset: the next window opens on a later reset time.
	nextReset := at("2026-11-05T05:00:00Z")
	now = at("2026-10-30T05:30:00Z")
	readings.spans = append(readings.spans, spans(codexWindow(nextReset, now, ramp(at("2026-10-29T05:10:00Z"), now, 0, 2)))...)
	if sent, err := alerter.Evaluate(ctx); sent != 1 || err != nil {
		t.Fatalf("sent=%d err=%v %v", sent, err, publisher.categories())
	}
	if last := publisher.events[1].Payload; last["category"] != NotificationCategoryRecovery || !strings.Contains(last["body"].(string), "reset") {
		t.Fatalf("recovery=%v", last)
	}
}

func TestAtRiskIsSentOncePerWindowAndOffByDefault(t *testing.T) {
	ctx := context.Background()
	reset := at("2026-11-04T06:57:00Z")
	now := reset.Add(-5 * 24 * time.Hour)
	week := codexWindow(reset, now, ramp(now.Add(-47*time.Hour), now, 1, 28))
	publisher := &recordingPublisher{}
	alerter := &Alerter{Readings: &staticReadings{spans: spans(week)}, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC}
	for i := 0; i < 2; i++ {
		if _, err := alerter.Evaluate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := publisher.categories(); len(got) != 1 || got[0] != NotificationCategoryAtRisk {
		t.Fatalf("categories=%v, want one at-risk", got)
	}
	// The hub enforces the default: at-risk and digest ship disabled.
	defaults := map[string]bool{NotificationCategory: true, NotificationCategoryRecovery: true, NotificationCategoryAtRisk: false, NotificationCategoryDigest: false}
	for _, c := range declaredCategories(t) {
		if want, ok := defaults[c.Key]; ok && c.DefaultEnabled != want {
			t.Errorf("%s defaultEnabled=%v, want %v", c.Key, c.DefaultEnabled, want)
		}
	}
}

func TestDigestListsLiveWindowsMostUrgentFirstEachWithItsOwnTopAction(t *testing.T) {
	ctx := context.Background()
	publisher := &recordingPublisher{}
	now := at("2026-10-29T08:30:00Z")
	reset := at("2026-11-04T06:57:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	healthy := codexWindow(reset, now, ramp(start.Add(time.Hour), now, 1, 12))
	healthy.pool = "secondary"
	five := windowSpec{provider: "anthropic", pool: "five_hour", minutes: 300, reset: now.Add(2 * time.Hour), last: now, steps: ramp(now.Add(-2*time.Hour), now, 5, 20)}
	alerter := &Alerter{
		Readings: &staticReadings{spans: spans(oct28Shape(), healthy, five)}, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }, Location: time.UTC,
		Actions: stubActions{{Text: "do the thing", Saving: 90, Recoverable: 40}},
	}
	run := func(stamp string) {
		now = at(stamp)
		if _, err := alerter.Evaluate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	digests := func() (n int) {
		for _, c := range publisher.categories() {
			if c == NotificationCategoryDigest {
				n++
			}
		}
		return n
	}
	for _, stamp := range []string{"2026-10-29T07:59:00Z", "2026-10-29T08:30:00Z", "2026-10-29T09:30:00Z", "2026-10-29T23:30:00Z"} {
		run(stamp)
	}
	if digests() != 1 {
		t.Fatalf("day one: digests=%d categories=%v", digests(), publisher.categories())
	}
	run("2026-10-30T08:10:00Z")
	if digests() != 1 {
		t.Fatalf("a day whose readings are all stale has no live window and sends no digest: %v", publisher.categories())
	}
	for _, e := range publisher.events {
		if e.Payload["category"] != NotificationCategoryDigest {
			continue
		}
		lines := strings.Split(e.Payload["body"].(string), "\n")
		if len(lines) > 3 {
			t.Fatalf("digest is capped at 3 lines: %q", lines)
		}
		for _, line := range lines {
			if !strings.Contains(line, "do the thing") {
				t.Errorf("each window carries its own top action: %q", line)
			}
		}
		if !strings.Contains(lines[0], "Codex · weekly") || !strings.Contains(lines[0], "could save ≈ 40 pts before the reset") {
			t.Errorf("most urgent first, with the recoverable saving: %q", lines[0])
		}
		return
	}
}

type declaredCategory struct {
	Key            string `json:"key"`
	DefaultEnabled bool   `json:"defaultEnabled"`
}

func declaredCategories(t *testing.T) []declaredCategory {
	t.Helper()
	raw, err := os.ReadFile("../../../.vrooli/notifications.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		AppDelivery bool               `json:"appDelivery"`
		Categories  []declaredCategory `json:"categories"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if !manifest.AppDelivery {
		t.Error("notifications.json must enable appDelivery")
	}
	for _, key := range []string{NotificationCategory, NotificationCategoryRecovery, NotificationCategoryAtRisk, NotificationCategoryDigest} {
		found := false
		for _, c := range manifest.Categories {
			found = found || c.Key == key
		}
		if !found {
			t.Errorf("category %s is not declared", key)
		}
	}
	return manifest.Categories
}

func TestActionStatesWhatCouldStillBeSavedBeforeTheResetNotTheFullHorizonRate(t *testing.T) {
	if got := actionText(Action{Text: "Compact it", Saving: 90, Recoverable: 31.2}); got != "Compact it (could save ≈ 31 pts before the reset)" {
		t.Errorf("got %q", got)
	}
	got := actionText(Action{Text: "Compact it", Saving: 90})
	if got != "Compact it" || strings.Contains(got, "pts") {
		t.Errorf("a stopped target has nothing recoverable and shows no rate: %q", got)
	}
}

func boundaryWindow(reset time.Time) *Window {
	return &Window{Provider: "openai", Pool: "primary", Window: "10080m", WindowMinutes: 10080, ResetAt: reset, HasPercent: true}
}

func boundaryForecast(w *Window, status string, projected time.Time) Forecast {
	return Forecast{Window: w, Status: status, Used: 60, AsOf: projected, Projected: &projected}
}

func TestAWindowOscillatingAtTheBoundaryWarnsAndWithdrawsOncePerPeriod(t *testing.T) {
	ctx := context.Background()
	reset := at("2026-11-04T06:57:00Z")
	w := boundaryWindow(reset)
	projected := at("2026-10-31T12:00:00Z")
	publisher := &recordingPublisher{}
	alerter := &Alerter{Store: sqliteAlertStore(t), Publisher: publisher, Location: time.UTC}
	tick := func(now time.Time, status string, projection time.Time) {
		t.Helper()
		forecasts := []Forecast{boundaryForecast(w, status, projection)}
		if _, errs := alerter.warn(ctx, forecasts, now); len(errs) > 0 {
			t.Fatal(errs)
		}
		if _, errs := alerter.withdraw(ctx, forecasts, now); len(errs) > 0 {
			t.Fatal(errs)
		}
	}
	count := func(category string) (n int) {
		for _, c := range publisher.categories() {
			if c == category {
				n++
			}
		}
		return n
	}
	start := at("2026-10-29T10:00:00Z")
	// Two hours flipping every tick: never safe for 30 minutes, so one warning
	// and no withdrawal.
	for i := 0; i < 25; i++ {
		status := StatusWillExhaust
		if i%2 == 1 {
			status = StatusOnTrack
		}
		tick(start.Add(time.Duration(i)*5*time.Minute), status, projected)
	}
	if count(NotificationCategory) != 1 || count(NotificationCategoryRecovery) != 0 {
		t.Fatalf("oscillation must not flap: %v", publisher.categories())
	}
	// Steadily safe: withdrawn once, only after SafeFor.
	safeFrom := start.Add(25 * 5 * time.Minute)
	tick(safeFrom, StatusOnTrack, projected)
	tick(safeFrom.Add(SafeFor-time.Minute), StatusOnTrack, projected)
	if count(NotificationCategoryRecovery) != 0 {
		t.Fatalf("withdrawn before %s of safety: %v", SafeFor, publisher.categories())
	}
	tick(safeFrom.Add(SafeFor), StatusOnTrack, projected)
	tick(safeFrom.Add(SafeFor+5*time.Minute), StatusOnTrack, projected)
	if count(NotificationCategoryRecovery) != 1 {
		t.Fatalf("one withdrawal after %s of safety: %v", SafeFor, publisher.categories())
	}
	// Flapping again in the same period, same projection or less than 12h
	// earlier: silent. 12h or more earlier: a fresh warning.
	after := safeFrom.Add(time.Hour)
	for i := 0; i < 6; i++ {
		status := StatusWillExhaust
		if i%2 == 1 {
			status = StatusOnTrack
		}
		tick(after.Add(time.Duration(i)*5*time.Minute), status, projected.Add(-6*time.Hour))
	}
	if count(NotificationCategory) != 1 || count(NotificationCategoryRecovery) != 1 {
		t.Fatalf("re-flapping inside the period must stay silent: %v", publisher.categories())
	}
	tick(after.Add(time.Hour), StatusWillExhaust, projected.Add(-13*time.Hour))
	if count(NotificationCategory) != 2 {
		t.Fatalf("a projection 13h earlier warns again: %v", publisher.categories())
	}
	// Each withdrawal is a distinct notification: its key names the warning it
	// withdraws.
	tick(after.Add(2*time.Hour), StatusOnTrack, projected)
	tick(after.Add(3*time.Hour), StatusOnTrack, projected)
	var keys []string
	for _, e := range publisher.events {
		if e.Payload["category"] == NotificationCategoryRecovery {
			keys = append(keys, e.Payload["dedupe_key"].(string))
		}
	}
	if len(keys) != 2 || keys[0] == keys[1] {
		t.Fatalf("withdrawal keys must differ: %v", keys)
	}
}

func TestAResetWithdrawsImmediatelyAndOldAuxiliaryRowsArePruned(t *testing.T) {
	ctx := context.Background()
	reset := at("2026-11-04T06:57:00Z")
	publisher := &recordingPublisher{}
	store := sqliteAlertStore(t)
	alerter := &Alerter{Store: store, Publisher: publisher, Location: time.UTC}
	now := at("2026-10-29T10:00:00Z")
	if sent, _ := alerter.warn(ctx, []Forecast{boundaryForecast(boundaryWindow(reset), StatusWillExhaust, now.Add(24*time.Hour))}, now); sent != 1 {
		t.Fatal("warning not sent")
	}
	next := boundaryWindow(reset.Add(7 * 24 * time.Hour))
	if sent, errs := alerter.withdraw(ctx, []Forecast{boundaryForecast(next, StatusWillExhaust, now.Add(48*time.Hour))}, now.Add(time.Minute)); sent != 1 || len(errs) > 0 {
		t.Fatalf("a reset withdraws at once: sent=%d %v", sent, errs)
	}
	old := now.Add(-AlertRetention - time.Hour)
	for _, key := range []string{atRiskKeyPrefix + "a", digestKeyPrefix + "2026-09-01", safeKeyPrefix + "a", withdrawnKeyPrefix + "a"} {
		if err := store.SaveAlert(ctx, AlertRecord{WindowKey: key, NotifiedAt: old}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveAlert(ctx, AlertRecord{WindowKey: digestKeyPrefix + "2026-10-28", NotifiedAt: now.Add(-24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.PruneAlerts(ctx, now.Add(-AlertRetention)); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{atRiskKeyPrefix + "a", digestKeyPrefix + "2026-09-01", safeKeyPrefix + "a", withdrawnKeyPrefix + "a"} {
		if got, _ := store.LastAlert(ctx, key); got != nil {
			t.Errorf("%s older than 30 days must be pruned", key)
		}
	}
	if got, _ := store.LastAlert(ctx, digestKeyPrefix+"2026-10-28"); got == nil {
		t.Error("a recent digest row must be kept")
	}
}

// fakeModes is a ModeLifecycle with one ended mode to announce.
type fakeModes struct {
	offers, announced, flushes int
	ended                      []EndedMode
}

func (f *fakeModes) Observe(context.Context, time.Time) {}
func (f *fakeModes) WillExhaust(context.Context, Forecast, time.Time, time.Time) error {
	f.offers++
	return nil
}
func (f *fakeModes) EndedFor(string) []EndedMode { return f.ended }
func (f *fakeModes) Announced(ids []string)      { f.announced += len(ids); f.ended = nil }
func (f *fakeModes) Flush(context.Context, []Forecast, time.Time) []error {
	f.flushes++
	return nil
}
func (f *fakeModes) DigestNote(time.Time) string { return "" }

func TestOfferFollowsTheRunOutWarningAndAnEndedModeMergesIntoBackOnTrack(t *testing.T) {
	ctx := context.Background()
	publisher := &recordingPublisher{}
	readings := &staticReadings{spans: spans(oct28Shape())}
	now := at("2026-10-29T04:40:00Z")
	modes := &fakeModes{}
	alerter := &Alerter{Readings: readings, Store: sqliteAlertStore(t), Publisher: publisher, Modes: modes, Now: func() time.Time { return now }, Location: time.UTC}
	if _, err := alerter.Evaluate(ctx); err != nil || modes.offers != 1 {
		t.Fatalf("offers=%d err=%v: the offer must follow the run-out warning", modes.offers, err)
	}
	modes.ended = []EndedMode{{ID: "um-1", Line: "Lower tier for codex ended: it reached its end time."}}
	nextReset := at("2026-11-05T05:00:00Z")
	now = at("2026-10-30T05:30:00Z")
	readings.spans = append(readings.spans, spans(codexWindow(nextReset, now, ramp(at("2026-10-29T05:10:00Z"), now, 0, 2)))...)
	if _, err := alerter.Evaluate(ctx); err != nil {
		t.Fatal(err)
	}
	last := publisher.events[len(publisher.events)-1].Payload
	if last["category"] != NotificationCategoryModeEnded || !strings.Contains(last["body"].(string), "reached its end time") || !strings.Contains(last["body"].(string), "window reset") {
		t.Fatalf("merged notice = %v", last)
	}
	if modes.announced != 1 || modes.flushes != 2 {
		t.Fatalf("announced=%d flushes=%d", modes.announced, modes.flushes)
	}
	for _, e := range publisher.events {
		if e.Payload["category"] == NotificationCategoryRecovery {
			t.Fatal("a separate back-on-track arrived next to the merged notice")
		}
	}
}
