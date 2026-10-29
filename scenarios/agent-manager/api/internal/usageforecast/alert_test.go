package usageforecast

import (
	"context"
	"errors"
	"strings"
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

func sqliteAlertStore(t *testing.T) *SQLiteAlertStore {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared")
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
	alerter := &Alerter{Readings: readings, Store: store, Publisher: publisher, Now: func() time.Time { return now }, UsageURL: func(context.Context) string { return "https://am.example/usage" }}

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
	if payload["category"] != NotificationCategory || payload["context_url"] != "https://am.example/usage" || payload["urgency"] != "high" {
		t.Fatalf("payload=%v", payload)
	}
	if title := payload["title"].(string); title != "Codex weekly quota runs out early" {
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
	alerter := &Alerter{Readings: &staticReadings{spans: spans(oct28Shape())}, Store: sqliteAlertStore(t), Publisher: publisher, Now: func() time.Time { return now }}
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
	healthy := &Alerter{Readings: &staticReadings{spans: spans(codexWindow(reset, start.Add(48*time.Hour), ramp(start.Add(time.Hour), start.Add(48*time.Hour), 1, 10)))}, Store: sqliteAlertStore(t), Publisher: &recordingPublisher{}, Now: func() time.Time { return start.Add(48 * time.Hour) }}
	if sent, err := healthy.Evaluate(ctx); sent != 0 || err != nil {
		t.Fatalf("on_track windows are not notified: sent=%d err=%v", sent, err)
	}
}
