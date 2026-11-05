package usageforecast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/usagelabel"

	"github.com/vrooli/api-core/eventbus"
)

const (
	// NotificationEventType uses notification-hub's generic
	// `**.notification.v1` contract; the hub needs no change to deliver it.
	NotificationEventType = "agent_manager.usage.notification.v1"
	// Notification categories must match .vrooli/notifications.json.
	NotificationCategory         = "usage-will-exhaust"
	NotificationCategoryRecovery = "usage-back-on-track"
	NotificationCategoryAtRisk   = "usage-at-risk"
	NotificationCategoryDigest   = "usage-digest"
	// Usage mode categories; the lifecycle (usagemodes.Notifier) publishes them.
	NotificationCategoryModeEnded       = "usage-mode-ended"
	NotificationCategoryModeUnreachable = "usage-mode-unreachable"
	NotificationCategoryModeStarted     = "usage-mode-started"
	NotificationCategoryModeEvidence    = "usage-mode-evidence"
	// ResendShift is how much earlier the projected exhaustion must move
	// before a window is notified again (SPEC §4 Alert).
	ResendShift = 12 * time.Hour
	// DefaultAlertInterval is how often windows are evaluated. The read is a
	// bounded aggregate, so a short interval is cheap.
	DefaultAlertInterval = 5 * time.Minute
	// DigestHour is the local hour the daily digest is due; one sent later
	// than DigestCatchUp after it is skipped rather than sent at an odd hour.
	DigestHour    = 8
	DigestCatchUp = 4 * time.Hour
	digestLines   = 3

	// Alert state shares usage_forecast_alerts: a will-exhaust row is keyed by
	// the window key, the others by a reserved prefix no window key starts with.
	atRiskKeyPrefix = "at-risk:"
	digestKeyPrefix = "digest:"
	// A warning is withdrawn only after the window has been on track or idle
	// for SafeFor, so a window oscillating at the boundary does not flap:
	// "safe:<window key>" holds when it was first seen safe, and
	// "withdrawn:<window key>" keeps the warned projection for the rest of the
	// window period, so only a projection ResendShift earlier warns again.
	safeKeyPrefix      = "safe:"
	withdrawnKeyPrefix = "withdrawn:"
	SafeFor            = 30 * time.Minute
	// AlertRetention is how long at-risk, digest, safe and withdrawn rows live.
	AlertRetention = 30 * 24 * time.Hour
)

var reservedKeyPrefixes = []string{atRiskKeyPrefix, digestKeyPrefix, safeKeyPrefix, withdrawnKeyPrefix}

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
	DeleteAlert(ctx context.Context, windowKey string) error
	// ExhaustAlerts lists the windows whose will-exhaust warning is standing.
	ExhaustAlerts(ctx context.Context) ([]AlertRecord, error)
	// PruneAlerts deletes at-risk, digest, safe and withdrawn rows notified
	// before cutoff.
	PruneAlerts(ctx context.Context, cutoff time.Time) error
}

// Publisher publishes onto Vrooli Events; eventbus.Client satisfies it.
type Publisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

// EndedMode is a usage mode that ended and has not been announced yet.
type EndedMode struct {
	ID string
	// Line is the sentence that says which mode ended and why.
	Line string
}

// ModeLifecycle joins usage modes to the usage notifications. usagemodes.Notifier
// satisfies it; the alerter calls it at fixed points of every evaluation so a
// mode's end and a window's recovery arrive as one notification.
type ModeLifecycle interface {
	// Observe records which modes are active, after any early end.
	Observe(ctx context.Context, now time.Time)
	// WillExhaust offers to start a mode for a window whose run-out warning
	// has just been sent.
	WillExhaust(ctx context.Context, f Forecast, projected, now time.Time) error
	// EndedFor lists the unannounced ended modes that covered the provider.
	EndedFor(provider string) []EndedMode
	// Announced marks ended modes as told, so they are not announced again.
	Announced(ids []string)
	// Flush publishes everything else that is due: remaining ended notices,
	// ending-soon and reminder decisions, and the reach warning.
	Flush(ctx context.Context, forecasts []Forecast, now time.Time) []error
	// DigestNote is one line about the active modes for the daily digest, or "".
	DigestNote(now time.Time) string
}

// Action is one recommended change and its weekly saving in window points.
type Action struct {
	Text string
	// Saving is this action's own saving over the window's horizon at the
	// target's current rate; Recoverable is the part still to be had before the
	// window resets.
	Saving, Recoverable float64
}

// ActionSource names the best actions for a window, largest saving first.
// Usage attribution implements it; an alert without actions is still sent.
type ActionSource interface {
	TopActions(ctx context.Context, w *Window, n int, now time.Time) ([]Action, error)
}

// Alerter publishes the usage notifications: will-exhaust, its withdrawal
// (back on track), at-risk, and the daily digest.
type Alerter struct {
	Readings  pricing.QuotaReadingSource
	Store     AlertStore
	Publisher Publisher
	// Activity, when set, lets a stale window with no work since be idle.
	Activity ActivitySource
	// Actions, when set, adds the #1 action to the alert bodies.
	Actions ActionSource
	// UsageURL resolves the Agent Manager Usage page link; empty omits it.
	UsageURL func(context.Context) string
	Now      func() time.Time
	// Location decides when the digest is due; local time when nil.
	Location *time.Location
	Interval time.Duration
	// OnError receives evaluation failures; nil drops them.
	OnError func(error)
	// OnForecasts, when set, sees every evaluation's current forecasts. Usage
	// modes use it to end a window-reset mode once its window has rolled.
	OnForecasts func(ctx context.Context, forecasts []Forecast, now time.Time)
	// Modes, when set, adds the usage mode decisions and notices.
	Modes ModeLifecycle
}

// Start evaluates now and then on every interval until ctx ends. A nil
// alerter, or one without readings, does nothing; without a store and
// publisher it only reports forecasts to OnForecasts.
func (a *Alerter) Start(ctx context.Context) {
	if a == nil || a.Readings == nil || (!a.notifies() && a.OnForecasts == nil) {
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

func (a *Alerter) notifies() bool { return a.Store != nil && a.Publisher != nil }

// Evaluate forecasts the current windows and notifies: each will_exhaust
// window that was not notified (or whose projection moved earlier by more than
// ResendShift); each previously warned window that is on track, idle or reset;
// each at_risk window once per period; and the daily digest once a day. State
// is saved only after a successful publish, so a failed publish is retried on
// the next evaluation. It returns the number sent.
func (a *Alerter) Evaluate(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	spans, err := a.Readings.QuotaReadingSpans(ctx, "", now.Add(-CurrentLookback))
	if err != nil {
		return 0, fmt.Errorf("read quota readings: %w", err)
	}
	forecasts := CurrentWithActivity(ctx, spans, now, a.Activity)
	if a.OnForecasts != nil {
		a.OnForecasts(ctx, forecasts, now)
	}
	if !a.notifies() {
		return 0, nil
	}
	if a.Modes != nil {
		a.Modes.Observe(ctx, now)
	}
	var failures []error
	if err := a.Store.PruneAlerts(ctx, now.Add(-AlertRetention)); err != nil {
		failures = append(failures, fmt.Errorf("prune alert state: %w", err))
	}
	sent := 0
	for _, step := range []func(context.Context, []Forecast, time.Time) (int, []error){a.warn, a.withdraw, a.atRisk, a.digest} {
		n, errs := step(ctx, forecasts, now)
		sent += n
		failures = append(failures, errs...)
	}
	if a.Modes != nil {
		failures = append(failures, a.Modes.Flush(ctx, forecasts, now)...)
	}
	return sent, errors.Join(failures...)
}

// warn notifies will_exhaust windows.
func (a *Alerter) warn(ctx context.Context, forecasts []Forecast, now time.Time) (int, []error) {
	sent := 0
	var failures []error
	for _, forecast := range forecasts {
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
		if last == nil {
			// Withdrawn earlier in this period: warn again only if the
			// projection moved ResendShift earlier than the one withdrawn.
			withdrawn, err := a.Store.LastAlert(ctx, withdrawnKeyPrefix+key)
			if err != nil {
				failures = append(failures, fmt.Errorf("read alert state %s: %w", key, err))
				continue
			}
			if withdrawn != nil && !projected.Before(withdrawn.ProjectedExhaustion.Add(-ResendShift)) {
				continue
			}
		}
		event := a.event(ctx, forecast)
		event.facts["category"], event.facts["urgency"] = NotificationCategory, "high"
		event.facts["title"] = usagelabel.Title(forecast.Window.Provider, forecast.Window.Window, forecast.Window.WindowMinutes) + " runs out early"
		event.facts["body"] = a.withAction(ctx, forecast, now, willExhaustBody(forecast, projected))
		event.facts["dedupe_key"] = key + "|" + projected.UTC().Format("2006-01-02T15Z")
		if err := a.publishAndSave(ctx, event, AlertRecord{WindowKey: key, ProjectedExhaustion: projected, NotifiedAt: now}); err != nil {
			failures = append(failures, fmt.Errorf("will-exhaust notification %s: %w", key, err))
			continue
		}
		if err := a.Store.DeleteAlert(ctx, withdrawnKeyPrefix+key); err != nil {
			failures = append(failures, fmt.Errorf("clear withdrawn state %s: %w", key, err))
		}
		if a.Modes != nil {
			if err := a.Modes.WillExhaust(ctx, forecast, projected, now); err != nil {
				failures = append(failures, fmt.Errorf("usage mode offer %s: %w", key, err))
			}
		}
		sent++
	}
	return sent, failures
}

// withdraw sends "back on track" once for each standing warning whose window
// has reset, or has been on track or idle for SafeFor, and clears that
// warning. A warned window no longer read at all is cleared silently.
func (a *Alerter) withdraw(ctx context.Context, forecasts []Forecast, now time.Time) (int, []error) {
	standing, err := a.Store.ExhaustAlerts(ctx)
	if err != nil {
		return 0, []error{fmt.Errorf("list standing warnings: %w", err)}
	}
	byScope := map[string]Forecast{}
	for _, f := range forecasts {
		byScope[f.Window.Scope()] = f
	}
	sent := 0
	var failures []error
	for _, record := range standing {
		key := record.WindowKey
		f, ok := byScope[scopeOfKey(key)]
		reset := ok && f.Window.Key() != key
		if ok && !reset {
			if f.Status != StatusOnTrack && f.Status != StatusIdle {
				if err := a.Store.DeleteAlert(ctx, safeKeyPrefix+key); err != nil {
					failures = append(failures, fmt.Errorf("clear safe mark %s: %w", key, err))
				}
				continue
			}
			since, err := a.Store.LastAlert(ctx, safeKeyPrefix+key)
			if err != nil {
				failures = append(failures, fmt.Errorf("read alert state %s: %w", key, err))
				continue
			}
			if since == nil {
				if err := a.Store.SaveAlert(ctx, AlertRecord{WindowKey: safeKeyPrefix + key, NotifiedAt: now}); err != nil {
					failures = append(failures, fmt.Errorf("save safe mark %s: %w", key, err))
				}
				continue
			}
			if now.Sub(since.NotifiedAt) < SafeFor {
				continue
			}
		}
		if ok {
			event := a.event(ctx, f)
			label := usagelabel.Title(f.Window.Provider, f.Window.Window, f.Window.WindowMinutes)
			event.facts["category"], event.facts["urgency"] = NotificationCategoryRecovery, "low"
			event.facts["title"] = label + " is back on track"
			event.facts["body"] = backOnTrackBody(f, reset)
			event.facts["dedupe_key"] = "back-on-track|" + key + "|" + record.NotifiedAt.UTC().Format(time.RFC3339)
			// A mode that ended for this provider is announced in this same
			// notification, so one arrives instead of two.
			var ended []EndedMode
			if a.Modes != nil {
				ended = a.Modes.EndedFor(f.Window.Provider)
			}
			if len(ended) > 0 {
				lines := make([]string, len(ended))
				for i, mode := range ended {
					lines[i] = mode.Line
				}
				event.facts["category"] = NotificationCategoryModeEnded
				event.facts["title"] = "Usage mode ended; " + label + " is back on track"
				event.facts["body"] = strings.Join(lines, " ") + " " + backOnTrackBody(f, reset)
				event.facts["dedupe_key"] = "mode-ended|" + key + "|" + record.NotifiedAt.UTC().Format(time.RFC3339)
			}
			if err := a.Publisher.PublishDomainEvent(ctx, event.domain(now)); err != nil {
				failures = append(failures, fmt.Errorf("back-on-track notification %s: %w", key, err))
				continue
			}
			if len(ended) > 0 {
				ids := make([]string, len(ended))
				for i, mode := range ended {
					ids[i] = mode.ID
				}
				a.Modes.Announced(ids)
			}
			sent++
			if !reset {
				// Keep the warned projection for the rest of this period.
				if err := a.Store.SaveAlert(ctx, AlertRecord{WindowKey: withdrawnKeyPrefix + key, ProjectedExhaustion: record.ProjectedExhaustion, NotifiedAt: now}); err != nil {
					failures = append(failures, fmt.Errorf("save withdrawn state %s: %w", key, err))
				}
			}
		}
		for _, stale := range []string{key, safeKeyPrefix + key} {
			if err := a.Store.DeleteAlert(ctx, stale); err != nil {
				failures = append(failures, fmt.Errorf("clear warning %s: %w", stale, err))
			}
		}
	}
	return sent, failures
}

// atRisk notifies at_risk windows once per window period.
func (a *Alerter) atRisk(ctx context.Context, forecasts []Forecast, now time.Time) (int, []error) {
	sent := 0
	var failures []error
	for _, f := range forecasts {
		if f.Status != StatusAtRisk {
			continue
		}
		key := atRiskKeyPrefix + f.Window.Key()
		last, err := a.Store.LastAlert(ctx, key)
		if err != nil {
			failures = append(failures, fmt.Errorf("read alert state %s: %w", key, err))
			continue
		}
		if last != nil {
			continue
		}
		event := a.event(ctx, f)
		event.facts["category"], event.facts["urgency"] = NotificationCategoryAtRisk, "normal"
		event.facts["title"] = usagelabel.Title(f.Window.Provider, f.Window.Window, f.Window.WindowMinutes) + " may run out"
		event.facts["body"] = a.withAction(ctx, f, now, atRiskBody(f))
		event.facts["dedupe_key"] = key
		if err := a.publishAndSave(ctx, event, AlertRecord{WindowKey: key, NotifiedAt: now}); err != nil {
			failures = append(failures, fmt.Errorf("at-risk notification %s: %w", key, err))
			continue
		}
		sent++
	}
	return sent, failures
}

// digest sends the daily summary once, from 08:00 local.
func (a *Alerter) digest(ctx context.Context, forecasts []Forecast, now time.Time) (int, []error) {
	loc := a.Location
	if loc == nil {
		loc = time.Local
	}
	local := now.In(loc)
	due := time.Date(local.Year(), local.Month(), local.Day(), DigestHour, 0, 0, 0, loc)
	if len(forecasts) == 0 || local.Before(due) || local.After(due.Add(DigestCatchUp)) {
		return 0, nil
	}
	key := digestKeyPrefix + local.Format("2006-01-02")
	last, err := a.Store.LastAlert(ctx, key)
	if err != nil {
		return 0, []error{fmt.Errorf("read alert state %s: %w", key, err)}
	}
	if last != nil {
		return 0, nil
	}
	live := digestWindows(forecasts)
	if len(live) == 0 {
		return 0, nil
	}
	lines := make([]string, 0, len(live))
	for _, f := range live {
		line := digestLine(f)
		if top, err := a.topActions(ctx, f, 1, now); err == nil && len(top) > 0 {
			line += " — " + actionText(top[0])
		}
		lines = append(lines, line)
	}
	if a.Modes != nil {
		if note := a.Modes.DigestNote(now); note != "" {
			lines = append(lines, note)
		}
	}
	body := strings.Join(lines, "\n")
	event := notification{facts: map[string]any{
		"category": NotificationCategoryDigest, "urgency": "low", "sensitivity": "private",
		"title": "Usage today", "body": body, "dedupe_key": key,
	}}
	if link := a.link(ctx, nil); link != "" {
		event.facts["context_url"] = link
	}
	if err := a.publishAndSave(ctx, event, AlertRecord{WindowKey: key, NotifiedAt: now}); err != nil {
		return 0, []error{fmt.Errorf("digest notification: %w", err)}
	}
	return 1, nil
}

func (a *Alerter) publishAndSave(ctx context.Context, event notification, record AlertRecord) error {
	if err := a.Publisher.PublishDomainEvent(ctx, event.domain(record.NotifiedAt)); err != nil {
		return err
	}
	if err := a.Store.SaveAlert(ctx, record); err != nil {
		return fmt.Errorf("save alert state: %w", err)
	}
	return nil
}

// notification is the neutral notificationFacts payload.
type notification struct{ facts map[string]any }

func (n notification) domain(now time.Time) eventbus.DomainEvent {
	return eventbus.DomainEvent{Source: "agent-manager", EventType: NotificationEventType, Occurred: now, Payload: n.facts}
}

// event starts the payload every window notification shares: private, linked
// to the Usage page with the window selected, correlated by window key.
func (a *Alerter) event(ctx context.Context, f Forecast) notification {
	facts := map[string]any{
		"sensitivity": "private",
		"correlation": map[string]any{"window_key": f.Window.Key()},
	}
	if link := a.link(ctx, f.Window); link != "" {
		facts["context_url"] = link
	}
	return notification{facts: facts}
}

// link opens the Usage page, with the window selected when there is one.
func (a *Alerter) link(ctx context.Context, w *Window) string {
	if a.UsageURL == nil {
		return ""
	}
	link := a.UsageURL(ctx)
	if link == "" || w == nil {
		return link
	}
	return link + "?window=" + WindowRef(w)
}

// WindowRef is the Usage page's window selector: provider/pool/window.
func WindowRef(w *Window) string {
	return url.PathEscape(w.Provider) + "/" + url.PathEscape(w.Pool) + "/" + url.PathEscape(w.Window)
}

func (a *Alerter) topActions(ctx context.Context, f Forecast, n int, now time.Time) ([]Action, error) {
	if a.Actions == nil {
		return nil, nil
	}
	return a.Actions.TopActions(ctx, f.Window, n, now)
}

// withAction appends the window's #1 action and its saving to body.
func (a *Alerter) withAction(ctx context.Context, f Forecast, now time.Time, body string) string {
	actions, err := a.topActions(ctx, f, 1, now)
	if err != nil || len(actions) == 0 {
		return body
	}
	return fmt.Sprintf("%s Biggest lever: %s.", body, actionText(actions[0]))
}

// actionText names an action with what it could still save before the reset.
// An action with nothing recoverable (its target has stopped) states no figure:
// the full-horizon rate is not what an alert is about.
func actionText(a Action) string {
	if a.Recoverable <= 0 {
		return a.Text
	}
	return fmt.Sprintf("%s (could save ≈ %.0f pts before the reset)", a.Text, a.Recoverable)
}

// scopeOfKey is the provider|pool|window part of a window key.
func scopeOfKey(key string) string {
	if i := strings.LastIndex(key, "|"); i >= 0 {
		return key[:i]
	}
	return key
}

func willExhaustBody(f Forecast, projected time.Time) string {
	w := f.Window
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
	return body.String()
}

func atRiskBody(f Forecast) string {
	body := fmt.Sprintf("%.0f%% used. The projection straddles the reset", f.Used)
	if !f.Window.ResetAt.IsZero() {
		body += " at " + stamp(f.Window.ResetAt)
	}
	return body + "; this window may run out."
}

func backOnTrackBody(f Forecast, reset bool) string {
	switch {
	case reset:
		return "The earlier warning no longer applies: the window reset."
	case f.Status == StatusIdle:
		return "The earlier warning no longer applies: no work is using it, so it lasts at this pace."
	}
	return fmt.Sprintf("The earlier warning no longer applies: usage held at %.0f%% and now lasts until the reset.", f.Used)
}

// digestWindows keeps the live windows, most urgent status first, at most
// digestLines of them. A window with no usable state (unknown) is left out.
func digestWindows(forecasts []Forecast) []Forecast {
	rank := map[string]int{StatusWillExhaust: 0, StatusAtRisk: 1, StatusOnTrack: 2, StatusIdle: 3}
	var live []Forecast
	for _, f := range forecasts {
		if _, ok := rank[f.Status]; ok {
			live = append(live, f)
		}
	}
	sort.SliceStable(live, func(i, j int) bool { return rank[live[i].Status] < rank[live[j].Status] })
	return live[:min(len(live), digestLines)]
}

func digestLine(f Forecast) string {
	title := usagelabel.Title(f.Window.Provider, f.Window.Window, f.Window.WindowMinutes)
	switch {
	case f.Status == StatusWillExhaust && f.Projected != nil:
		return fmt.Sprintf("%s: %.0f%% used, runs out ~%s", title, f.Used, stamp(*f.Projected))
	case f.Status == StatusIdle:
		return fmt.Sprintf("%s: %.0f%% used, idle, lasts at this pace", title, f.Used)
	}
	return fmt.Sprintf("%s: %.0f%% used, %s", title, f.Used, strings.ReplaceAll(f.Status, "_", " "))
}

func stamp(t time.Time) string { return t.UTC().Format("Mon Jan 2 15:04 UTC") }

// Stamp formats a time the way every usage notification does.
func Stamp(t time.Time) string { return stamp(t) }

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

func (s *SQLiteAlertStore) DeleteAlert(ctx context.Context, windowKey string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM usage_forecast_alerts WHERE window_key = ?`, windowKey)
	return err
}

func reservedKeyFilter(operator, joiner string) string {
	parts := make([]string, len(reservedKeyPrefixes))
	for i := range parts {
		parts[i] = "window_key " + operator + " ?"
	}
	return strings.Join(parts, joiner)
}

func reservedKeyArgs() []any {
	args := make([]any, len(reservedKeyPrefixes))
	for i, prefix := range reservedKeyPrefixes {
		args[i] = prefix + "%"
	}
	return args
}

func (s *SQLiteAlertStore) PruneAlerts(ctx context.Context, cutoff time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM usage_forecast_alerts WHERE (`+reservedKeyFilter("LIKE", " OR ")+`) AND notified_at < ?`, append(reservedKeyArgs(), cutoff.UTC().Format(time.RFC3339Nano))...)
	return err
}

func (s *SQLiteAlertStore) ExhaustAlerts(ctx context.Context) ([]AlertRecord, error) {
	rows, err := s.db.QueryxContext(ctx, `SELECT window_key, projected_exhaustion_at, notified_at FROM usage_forecast_alerts WHERE `+reservedKeyFilter("NOT LIKE", " AND "), reservedKeyArgs()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []AlertRecord
	for rows.Next() {
		var key, projected, notified string
		if err := rows.Scan(&key, &projected, &notified); err != nil {
			return nil, err
		}
		record := AlertRecord{WindowKey: key}
		if record.ProjectedExhaustion, err = time.Parse(time.RFC3339Nano, projected); err != nil {
			return nil, err
		}
		if record.NotifiedAt, err = time.Parse(time.RFC3339Nano, notified); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLiteAlertStore) SaveAlert(ctx context.Context, record AlertRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO usage_forecast_alerts (window_key, projected_exhaustion_at, notified_at) VALUES (?, ?, ?)
		ON CONFLICT(window_key) DO UPDATE SET projected_exhaustion_at = excluded.projected_exhaustion_at, notified_at = excluded.notified_at`,
		record.WindowKey, record.ProjectedExhaustion.UTC().Format(time.RFC3339Nano), record.NotifiedAt.UTC().Format(time.RFC3339Nano))
	return err
}
