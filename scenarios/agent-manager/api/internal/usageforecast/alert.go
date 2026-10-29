package usageforecast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/sqlcompat"

	"github.com/vrooli/api-core/eventbus"
)

const (
	// NotificationEventType uses notification-hub's generic
	// `**.notification.v1` contract; the hub needs no change to deliver it.
	NotificationEventType = "agent_manager.usage.notification.v1"
	// NotificationCategory must match .vrooli/notifications.json.
	NotificationCategory = "usage-will-exhaust"
	// ResendShift is how much earlier the projected exhaustion must move
	// before a window is notified again (SPEC §4 Alert).
	ResendShift = 12 * time.Hour
	// DefaultAlertInterval is how often windows are evaluated. The read is a
	// bounded aggregate, so a short interval is cheap.
	DefaultAlertInterval = 5 * time.Minute
)

// AlertRecord is the last notified projection for one window.
type AlertRecord struct {
	WindowKey           string
	ProjectedExhaustion time.Time
	NotifiedAt          time.Time
}

// AlertStore persists dedupe state durably.
type AlertStore interface {
	LastAlert(ctx context.Context, windowKey string) (*AlertRecord, error)
	SaveAlert(ctx context.Context, record AlertRecord) error
}

// Publisher publishes onto Vrooli Events; eventbus.Client satisfies it.
type Publisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

// Alerter publishes one notification when a window becomes will_exhaust.
type Alerter struct {
	Readings  pricing.QuotaReadingSource
	Store     AlertStore
	Publisher Publisher
	// UsageURL resolves the Agent Manager Usage page link; empty omits it.
	UsageURL func(context.Context) string
	Now      func() time.Time
	Interval time.Duration
	// OnError receives evaluation failures; nil drops them.
	OnError func(error)
}

// Start evaluates now and then on every interval until ctx ends. A nil
// alerter, or one without readings, store or publisher, does nothing.
func (a *Alerter) Start(ctx context.Context) {
	if a == nil || a.Readings == nil || a.Store == nil || a.Publisher == nil {
		return
	}
	interval := a.Interval
	if interval <= 0 {
		interval = DefaultAlertInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if _, err := a.Evaluate(ctx); err != nil && a.OnError != nil {
				a.OnError(err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Evaluate forecasts the current windows and notifies each will_exhaust
// window that was not notified, or whose projection moved earlier by more than
// ResendShift. State is saved only after a successful publish, so a failed
// publish is retried on the next evaluation. It returns the number sent.
func (a *Alerter) Evaluate(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	spans, err := a.Readings.QuotaReadingSpans(ctx, "", now.Add(-CurrentLookback))
	if err != nil {
		return 0, fmt.Errorf("read quota readings: %w", err)
	}
	sent := 0
	var failures []error
	for _, forecast := range Current(spans, now) {
		if forecast.Status != StatusWillExhaust {
			continue
		}
		projected := forecast.AsOf
		if forecast.Projected != nil {
			projected = *forecast.Projected
		}
		key := forecast.Window.Key()
		last, err := a.Store.LastAlert(ctx, key)
		if err != nil {
			failures = append(failures, fmt.Errorf("read alert state %s: %w", key, err))
			continue
		}
		if last != nil && !projected.Before(last.ProjectedExhaustion.Add(-ResendShift)) {
			continue
		}
		link := ""
		if a.UsageURL != nil {
			link = a.UsageURL(ctx)
		}
		if err := a.Publisher.PublishDomainEvent(ctx, NotificationEvent(forecast, projected, link, now)); err != nil {
			failures = append(failures, fmt.Errorf("publish will-exhaust notification %s: %w", key, err))
			continue
		}
		if err := a.Store.SaveAlert(ctx, AlertRecord{WindowKey: key, ProjectedExhaustion: projected, NotifiedAt: now}); err != nil {
			failures = append(failures, fmt.Errorf("save alert state %s: %w", key, err))
			continue
		}
		sent++
	}
	return sent, errors.Join(failures...)
}

// NotificationEvent is the neutral notificationFacts payload for a
// will_exhaust window. The dedupe key names the window and the projected hour,
// so a re-send after a 12-hour shift is a distinct notification while a
// redelivery of the same one collapses.
func NotificationEvent(f Forecast, projected time.Time, usageURL string, now time.Time) eventbus.DomainEvent {
	w := f.Window
	label := providerLabel(w.Provider) + " " + windowLabel(w.WindowMinutes, w.Window)
	title := label + " quota runs out early"
	var body strings.Builder
	if w.HasPercent {
		fmt.Fprintf(&body, "%.0f%% used. Projected to run out ~%s", f.Used, stamp(projected))
		if f.BandStart != nil && f.BandEnd != nil {
			fmt.Fprintf(&body, " (band %s – %s)", stamp(*f.BandStart), stamp(*f.BandEnd))
		}
		body.WriteString(".")
	} else {
		body.WriteString("The provider reports this window exhausted.")
	}
	if !w.ResetAt.IsZero() {
		fmt.Fprintf(&body, " Resets %s", stamp(w.ResetAt))
		if lead := w.ResetAt.Sub(projected); lead > 0 {
			fmt.Fprintf(&body, ", %.1f days later", lead.Hours()/24)
		}
		body.WriteString(".")
	}
	if f.Lasting != nil {
		fmt.Fprintf(&body, " To last until the reset, burn at most %.2f%%/h.", math.Max(0, *f.Lasting))
	}
	payload := map[string]any{
		"title":       title,
		"body":        body.String(),
		"category":    NotificationCategory,
		"urgency":     "high",
		"sensitivity": "private",
		"dedupe_key":  w.Key() + "|" + projected.UTC().Format("2006-01-02T15Z"),
		"correlation": map[string]any{"window_key": w.Key()},
	}
	if usageURL != "" {
		payload["context_url"] = usageURL
	}
	return eventbus.DomainEvent{Source: "agent-manager", EventType: NotificationEventType, Occurred: now, Payload: payload}
}

func stamp(t time.Time) string { return t.UTC().Format("Mon Jan 2 15:04 UTC") }

func providerLabel(provider string) string {
	switch provider {
	case "openai":
		return "Codex"
	case "anthropic":
		return "Claude"
	}
	return provider
}

func windowLabel(minutes int64, window string) string {
	switch {
	case minutes == 10080:
		return "weekly"
	case minutes > 0 && minutes%60 == 0:
		return fmt.Sprintf("%dh", minutes/60)
	}
	return window
}

// SQLiteAlertStore keeps alert dedupe state in usage_forecast_alerts.
type SQLiteAlertStore struct{ db sqlcompat.DB }

// NewSQLiteAlertStore wraps Agent Manager's primary database.
func NewSQLiteAlertStore(db sqlcompat.DB) *SQLiteAlertStore { return &SQLiteAlertStore{db: db} }

func (s *SQLiteAlertStore) LastAlert(ctx context.Context, windowKey string) (*AlertRecord, error) {
	var projected, notified string
	err := s.db.QueryRowContext(ctx, `SELECT projected_exhaustion_at, notified_at FROM usage_forecast_alerts WHERE window_key = ?`, windowKey).Scan(&projected, &notified)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record := &AlertRecord{WindowKey: windowKey}
	if record.ProjectedExhaustion, err = time.Parse(time.RFC3339Nano, projected); err != nil {
		return nil, err
	}
	if record.NotifiedAt, err = time.Parse(time.RFC3339Nano, notified); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *SQLiteAlertStore) SaveAlert(ctx context.Context, record AlertRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO usage_forecast_alerts (window_key, projected_exhaustion_at, notified_at) VALUES (?, ?, ?)
		ON CONFLICT(window_key) DO UPDATE SET projected_exhaustion_at = excluded.projected_exhaustion_at, notified_at = excluded.notified_at`,
		record.WindowKey, record.ProjectedExhaustion.UTC().Format(time.RFC3339Nano), record.NotifiedAt.UTC().Format(time.RFC3339Nano))
	return err
}
