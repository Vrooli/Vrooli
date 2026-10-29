package hub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// App delivery lets a scenario's own installed app show that scenario's
// notifications on one device instead of Chimesail. The hub stays the only
// router: preferences, quiet hours and dedupe run first, and then, per
// physical device, exactly one app is picked. Contract:
// docs/concepts/NOTIFICATIONS.md §"App delivery".
//
// A device is a device_group: the Chimesail subscription and the scenario-app
// subscriptions the operator confirmed are on the same phone or computer. Web
// Push carries no device identity and iOS home-screen apps share no storage,
// so the pairing is confirmed by the operator once per app per device.

// AppConfirmationWindow is how recently a scenario app must have been opened
// for the hub to keep delivering to it. Web Push has no read receipt, so a
// deleted or muted app is otherwise invisible until the push service retires
// its endpoint.
const AppConfirmationWindow = 30 * 24 * time.Hour

const appEligibilityTTL = 5 * time.Minute

var (
	// ErrAppIneligible means the scenario has no permanent public address
	// that matches the subscription's origin.
	ErrAppIneligible = errors.New("app delivery is not available for this scenario")
	// ErrAppNotRegistered means no app subscription matches the request.
	ErrAppNotRegistered = errors.New("this app is not registered for notifications")
)

// AppEligibility returns a scenario's permanent public URL, or an error that
// explains in plain words why its app cannot receive notifications. Production
// asks tunnel-manager for a CORE route; tests inject a stub.
type AppEligibility func(ctx context.Context, scenario string) (publicURL string, err error)

// AppSubscription is one scenario app's push subscription on one device.
type AppSubscription struct {
	ID, Scenario, Endpoint, Origin, DeviceGroup, UserAgent, Permission, ConfirmedAt string
	Healthy                                                                         bool
	UnhealthyReason                                                                 string
	Standalone                                                                      bool
	DeviceLabel                                                                     string
}

// DeviceChoice is a device the calling app can pair with.
type DeviceChoice struct {
	Group, Label, LastUsed string
	BestMatch              bool
	Apps                   []string
}

// AppState is what a scenario's settings panel shows.
type AppState struct {
	Scenario, DisplayName, VAPIDPublicKey string
	Eligible                              bool
	IneligibleReason, PublicURL           string
	Subscription                          *AppSubscription
	Devices                               []DeviceChoice
}

// AppRegistration is a scenario app asking to receive notifications.
type AppRegistration struct {
	Scenario, Endpoint, P256DH, Auth, Origin, UserAgent, Permission string
	PairWithGroup                                                   string
	Standalone                                                      bool
}

// DeviceAppEntry is one app on a device in Notification Hub's settings.
type DeviceAppEntry struct {
	SubscriptionID, Scenario, DisplayName, Status, ConfirmedAt string
	Healthy                                                    bool
}

// DeviceApps is one device and its apps.
type DeviceApps struct {
	Group, Label, LastUsed string
	Apps                   []DeviceAppEntry
}

type eligibilityEntry struct {
	url string
	err error
	at  time.Time
}

type appEligibilityCache struct {
	mu      sync.Mutex
	entries map[string]eligibilityEntry
}

// SetAppEligibility installs the permanent-address check. Without one, app
// delivery is off: every device keeps Chimesail.
func (s *Service) SetAppEligibility(check AppEligibility) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appEligibility = check
	s.appEligibilityCache = &appEligibilityCache{entries: map[string]eligibilityEntry{}}
}

func (s *Service) appPublicURL(ctx context.Context, scenario string, cached bool) (string, error) {
	s.mu.RLock()
	check, cache := s.appEligibility, s.appEligibilityCache
	s.mu.RUnlock()
	if check == nil {
		return "", fmt.Errorf("%w: app delivery is not configured on this hub", ErrAppIneligible)
	}
	now := s.clock.Now()
	if cached && cache != nil {
		cache.mu.Lock()
		entry, ok := cache.entries[scenario]
		cache.mu.Unlock()
		if ok && now.Sub(entry.at) < appEligibilityTTL {
			return entry.url, entry.err
		}
	}
	publicURL, err := check(ctx, scenario)
	if err == nil && strings.TrimSpace(publicURL) == "" {
		err = fmt.Errorf("%w: %s has no public address", ErrAppIneligible, scenario)
	}
	if err != nil && !errors.Is(err, ErrAppIneligible) {
		err = fmt.Errorf("%w: %v", ErrAppIneligible, err)
	}
	publicURL = normalizeOrigin(publicURL)
	if cache != nil {
		cache.mu.Lock()
		cache.entries[scenario] = eligibilityEntry{url: publicURL, err: err, at: now}
		cache.mu.Unlock()
	}
	return publicURL, err
}

func normalizeOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// appNavigateURL is the absolute URL a tap opens inside the app. A context
// URL on the app's own origin (or a relative path) is kept; anything else
// opens the app's start page, because a cross-origin URL would leave the app.
func appNavigateURL(origin, contextURL string) string {
	base := normalizeOrigin(origin)
	if base == "" {
		return ""
	}
	contextURL = strings.TrimSpace(contextURL)
	if strings.HasPrefix(contextURL, "/") && !strings.HasPrefix(contextURL, "//") {
		return base + contextURL
	}
	if u, err := url.Parse(contextURL); err == nil && u.Host != "" && normalizeOrigin(contextURL) == base {
		return base + u.RequestURI()
	}
	return base + "/"
}

const appSubscriptionColumns = `id, app_scenario, endpoint, origin, device_group, user_agent, permission, confirmed_at`

func scanAppSubscription(row rowScanner) (AppSubscription, error) {
	var sub AppSubscription
	err := row.Scan(&sub.ID, &sub.Scenario, &sub.Endpoint, &sub.Origin, &sub.DeviceGroup, &sub.UserAgent, &sub.Permission, &sub.ConfirmedAt)
	return sub, err
}

// appHealth decides whether the hub delivers to an app. Every "no" means the
// device uses Chimesail for this scenario.
func (s *Service) appHealth(sub AppSubscription, now time.Time) (bool, string) {
	switch strings.TrimSpace(sub.Permission) {
	case "granted":
	case "":
		return false, "This app has not been allowed to show notifications; using Chimesail"
	default:
		// iPhone reports a mute in Settings as "default", not "denied".
		return false, "Notifications are turned off for this app in the device settings; using Chimesail"
	}
	confirmed, err := time.Parse(time.RFC3339Nano, sub.ConfirmedAt)
	if err != nil {
		return false, "This app has not checked in yet; using Chimesail"
	}
	if now.Sub(confirmed) > AppConfirmationWindow {
		return false, "Not opened for 30 days; using Chimesail until it is opened again"
	}
	return true, ""
}

// chimesailGroups maps each Chimesail endpoint to its device group.
func (s *Service) chimesailGroups(ctx context.Context, recipient string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT endpoint, CASE WHEN device_group<>'' THEN device_group ELSE id END FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=''`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var endpoint, group string
		if err := rows.Scan(&endpoint, &group); err != nil {
			return nil, err
		}
		out[endpoint] = group
	}
	return out, rows.Err()
}

// applyAppDelivery replaces, per device, the Chimesail web push target with
// the source scenario's own app when that app is paired and healthy and the
// scenario still has its permanent address. It never adds a second copy on a
// device: the replaced Chimesail targets become the app target's fallback.
func (s *Service) applyAppDelivery(ctx context.Context, n Notification, targets []channelTarget) ([]channelTarget, error) {
	source := strings.TrimSpace(n.SourceScenario)
	if source == "" {
		return targets, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+appSubscriptionColumns+` FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=? AND device_group<>'' ORDER BY created_at, id`, n.RequestedBy, source)
	if err != nil {
		return nil, err
	}
	var apps []AppSubscription
	now := s.clock.Now()
	for rows.Next() {
		sub, scanErr := scanAppSubscription(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		if healthy, _ := s.appHealth(sub, now); healthy {
			apps = append(apps, sub)
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return targets, nil
	}
	publicURL, eligErr := s.appPublicURL(ctx, source, true)
	if eligErr != nil {
		s.log.Printf("app delivery for %s skipped; using Chimesail: %v", source, eligErr)
		return targets, nil
	}
	groups, err := s.chimesailGroups(ctx, n.RequestedBy)
	if err != nil {
		return nil, err
	}
	appByGroup := map[string]AppSubscription{}
	for _, app := range apps {
		if normalizeOrigin(app.Origin) != publicURL {
			continue
		}
		if _, taken := appByGroup[app.DeviceGroup]; !taken {
			appByGroup[app.DeviceGroup] = app
		}
	}
	replaced := map[string][]channelTarget{}
	out := make([]channelTarget, 0, len(targets)+len(appByGroup))
	for _, target := range targets {
		if target.Channel == "web_push" && (target.MachineID == "" || target.MachineID == "local") {
			if group, ok := groups[target.Address]; ok {
				if _, paired := appByGroup[group]; paired {
					replaced[group] = append(replaced[group], target)
					continue
				}
			}
		}
		out = append(out, target)
	}
	for _, app := range apps {
		chosen, ok := appByGroup[app.DeviceGroup]
		if !ok || chosen.ID != app.ID {
			continue
		}
		labels := []string{"public", "private", "sensitive", "critical"}
		if fallback := replaced[app.DeviceGroup]; len(fallback) > 0 {
			// The app inherits the device's approved sensitivity labels.
			labels = fallback[0].ApprovedLabels
		}
		out = append(out, channelTarget{Channel: "web_push", Address: app.Endpoint, MachineID: "local", ApprovedLabels: labels, App: source, Fallback: replaced[app.DeviceGroup]})
	}
	return out, nil
}

// RegisterApp stores a scenario app's subscription and pairs it with one
// device. The origin must be the scenario's permanent public address, so a
// page cannot register as a different scenario's app.
func (s *Service) RegisterApp(ctx context.Context, recipient string, reg AppRegistration) (AppState, error) {
	reg.Scenario = strings.TrimSpace(reg.Scenario)
	if recipient == "" || reg.Scenario == "" || reg.Endpoint == "" || reg.P256DH == "" || reg.Auth == "" || reg.Origin == "" {
		return AppState{}, fmt.Errorf("%w: scenario and a complete push subscription are required", ErrInvalidArgument)
	}
	if !reg.Standalone && strings.TrimSpace(reg.PairWithGroup) == "" {
		return AppState{}, fmt.Errorf("%w: choose the device this app is on", ErrInvalidArgument)
	}
	publicURL, err := s.appPublicURL(ctx, reg.Scenario, false)
	if err != nil {
		return AppState{}, err
	}
	if normalizeOrigin(reg.Origin) != publicURL {
		return AppState{}, fmt.Errorf("%w: this app was opened from %s, but %s's permanent address is %s", ErrAppIneligible, normalizeOrigin(reg.Origin), reg.Scenario, publicURL)
	}
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return AppState{}, err
	}
	group := uuid.NewString()
	if !reg.Standalone {
		group = strings.TrimSpace(reg.PairWithGroup)
		res, err := s.db.ExecContext(ctx, `UPDATE push_subscriptions SET device_group=? WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario='' AND (id=? OR device_group=?)`, group, recipient, group, group)
		if err != nil {
			return AppState{}, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return AppState{}, fmt.Errorf("%w: that device no longer has Chimesail; choose another", ErrInvalidArgument)
		}
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	// One app per scenario per device: a reinstall replaces the old endpoint.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=? AND device_group=? AND endpoint<>?`, recipient, reg.Scenario, group, reg.Endpoint); err != nil {
		return AppState{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO push_subscriptions (id, recipient_id, device_id, endpoint, p256dh, auth, origin, app_scenario, device_group, user_agent, permission, confirmed_at, created_at, updated_at) VALUES (?, (SELECT id FROM recipients WHERE subject=?), '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(recipient_id, endpoint) DO UPDATE SET device_id='', p256dh=excluded.p256dh, auth=excluded.auth, origin=excluded.origin, app_scenario=excluded.app_scenario, device_group=excluded.device_group, user_agent=excluded.user_agent, permission=excluded.permission, confirmed_at=excluded.confirmed_at, updated_at=excluded.updated_at`,
		uuid.NewString(), recipient, reg.Endpoint, reg.P256DH, reg.Auth, normalizeOrigin(reg.Origin), reg.Scenario, group, strings.TrimSpace(reg.UserAgent), strings.TrimSpace(reg.Permission), now, now, now); err != nil {
		return AppState{}, err
	}
	return s.GetAppState(ctx, recipient, reg.Scenario, reg.Endpoint, reg.UserAgent)
}

// ConfirmApp records that the app was opened and its current permission.
func (s *Service) ConfirmApp(ctx context.Context, recipient, scenario, endpoint, permission, userAgent string) (bool, error) {
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `UPDATE push_subscriptions SET permission=?, confirmed_at=?, user_agent=CASE WHEN ?<>'' THEN ? ELSE user_agent END, updated_at=? WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=? AND endpoint=?`,
		strings.TrimSpace(permission), now, strings.TrimSpace(userAgent), strings.TrimSpace(userAgent), now, recipient, strings.TrimSpace(scenario), endpoint)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, ErrAppNotRegistered
	}
	sub, err := s.appSubscriptionByEndpoint(ctx, recipient, scenario, endpoint)
	if err != nil {
		return false, err
	}
	healthy, _ := s.appHealth(sub, s.clock.Now())
	return healthy, nil
}

// DisableApp removes the calling app's subscription; the device goes back to
// Chimesail for this scenario.
func (s *Service) DisableApp(ctx context.Context, recipient, scenario, endpoint string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=? AND endpoint=?`, recipient, strings.TrimSpace(scenario), endpoint)
	return err
}

// RemoveDeviceApp removes one scenario app by id. Chimesail subscriptions are
// not removable here; they belong to Chimesail's own enable flow.
func (s *Service) RemoveDeviceApp(ctx context.Context, recipient, subscriptionID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND id=? AND app_scenario<>''`, recipient, subscriptionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAppNotRegistered
	}
	return nil
}

func (s *Service) appSubscriptionByEndpoint(ctx context.Context, recipient, scenario, endpoint string) (AppSubscription, error) {
	sub, err := scanAppSubscription(s.db.QueryRowContext(ctx, `SELECT `+appSubscriptionColumns+` FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND app_scenario=? AND endpoint=?`, recipient, strings.TrimSpace(scenario), endpoint))
	if errors.Is(err, sql.ErrNoRows) {
		return AppSubscription{}, ErrAppNotRegistered
	}
	return sub, err
}

type deviceSub struct {
	ID, App, Endpoint, Group, UserAgent, Permission, ConfirmedAt, UpdatedAt string
}

func (s *Service) deviceSubs(ctx context.Context, recipient string) ([]deviceSub, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, app_scenario, endpoint, CASE WHEN device_group<>'' THEN device_group ELSE id END, user_agent, permission, confirmed_at, updated_at FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) ORDER BY updated_at DESC, id`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []deviceSub
	for rows.Next() {
		var d deviceSub
		if err := rows.Scan(&d.ID, &d.App, &d.Endpoint, &d.Group, &d.UserAgent, &d.Permission, &d.ConfirmedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetAppState returns what the calling scenario's settings panel shows.
func (s *Service) GetAppState(ctx context.Context, recipient, scenario, endpoint, userAgent string) (AppState, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return AppState{}, fmt.Errorf("%w: the calling scenario is required", ErrInvalidArgument)
	}
	state := AppState{Scenario: scenario, DisplayName: s.resolveDisplayName(ctx, scenario), VAPIDPublicKey: s.WebPushPublicKey()}
	if state.DisplayName == "" {
		state.DisplayName = scenario
	}
	if publicURL, err := s.appPublicURL(ctx, scenario, false); err != nil {
		state.IneligibleReason = strings.TrimPrefix(strings.TrimPrefix(err.Error(), ErrAppIneligible.Error()), ": ")
	} else {
		state.Eligible, state.PublicURL = true, publicURL
	}
	subs, err := s.deviceSubs(ctx, recipient)
	if err != nil {
		return AppState{}, err
	}
	labels := deviceGroupLabels(subs)
	if strings.TrimSpace(endpoint) != "" {
		if sub, err := s.appSubscriptionByEndpoint(ctx, recipient, scenario, endpoint); err == nil {
			sub.Healthy, sub.UnhealthyReason = s.appHealth(sub, s.clock.Now())
			if sub.Healthy && !state.Eligible {
				sub.Healthy, sub.UnhealthyReason = false, "This app's address is not permanent right now; using Chimesail"
			}
			sub.DeviceLabel = labels[sub.DeviceGroup]
			sub.Standalone = true
			for _, d := range subs {
				if d.Group == sub.DeviceGroup && d.App == "" {
					sub.Standalone = false
					break
				}
			}
			state.Subscription = &sub
		} else if !errors.Is(err, ErrAppNotRegistered) {
			return AppState{}, err
		}
	}
	state.Devices = deviceChoices(subs, labels, endpoint, userAgent, func(app string) string {
		if name := s.resolveDisplayName(ctx, app); name != "" {
			return name
		}
		return app
	})
	return state, nil
}

// ListDeviceApps returns every device and its apps for Notification Hub's
// own settings.
func (s *Service) ListDeviceApps(ctx context.Context, recipient string) ([]DeviceApps, error) {
	subs, err := s.deviceSubs(ctx, recipient)
	if err != nil {
		return nil, err
	}
	labels := deviceGroupLabels(subs)
	byGroup := map[string]*DeviceApps{}
	var order []string
	now := s.clock.Now()
	for _, d := range subs {
		dev, ok := byGroup[d.Group]
		if !ok {
			dev = &DeviceApps{Group: d.Group, Label: labels[d.Group], LastUsed: d.UpdatedAt}
			byGroup[d.Group] = dev
			order = append(order, d.Group)
		}
		entry := DeviceAppEntry{SubscriptionID: d.ID, Scenario: d.App, ConfirmedAt: d.ConfirmedAt}
		if d.App == "" {
			entry.DisplayName, entry.Healthy, entry.Status = "Chimesail", true, "Receives every app not listed here"
		} else {
			entry.DisplayName = s.resolveDisplayName(ctx, d.App)
			if entry.DisplayName == "" {
				entry.DisplayName = d.App
			}
			healthy, reason := s.appHealth(AppSubscription{Permission: d.Permission, ConfirmedAt: d.ConfirmedAt}, now)
			entry.Healthy = healthy
			entry.Status = "Shows " + entry.DisplayName + " notifications here instead of Chimesail"
			if !healthy {
				entry.Status = reason
			}
		}
		dev.Apps = append(dev.Apps, entry)
	}
	out := make([]DeviceApps, 0, len(order))
	for _, group := range order {
		dev := byGroup[group]
		sort.SliceStable(dev.Apps, func(i, j int) bool { return dev.Apps[i].Scenario == "" && dev.Apps[j].Scenario != "" })
		out = append(out, *dev)
	}
	return out, nil
}

// deviceGroupLabels names each device group from the best user agent in it.
func deviceGroupLabels(subs []deviceSub) map[string]string {
	labels := map[string]string{}
	for _, d := range subs {
		label := deviceLabel(d.UserAgent, d.Endpoint)
		if current, ok := labels[d.Group]; !ok || (d.UserAgent != "" && isGenericLabel(current)) {
			labels[d.Group] = label
		}
	}
	return labels
}

// deviceChoices lists the devices that have Chimesail, newest first, and
// marks the likeliest match for the calling browser.
func deviceChoices(subs []deviceSub, labels map[string]string, endpoint, userAgent string, displayName func(string) string) []DeviceChoice {
	byGroup := map[string]*DeviceChoice{}
	var order []string
	callerHost := pushServiceHost(endpoint)
	callerPlatform := platformOf(userAgent)
	score := map[string]int{}
	for _, d := range subs {
		if d.App != "" {
			continue
		}
		if _, ok := byGroup[d.Group]; !ok {
			byGroup[d.Group] = &DeviceChoice{Group: d.Group, Label: labels[d.Group], LastUsed: d.UpdatedAt}
			order = append(order, d.Group)
		}
		points := 0
		if callerHost != "" && pushServiceHost(d.Endpoint) == callerHost {
			points += 2
		}
		if callerPlatform != "" && platformOf(d.UserAgent) == callerPlatform {
			points += 2
		}
		if points > score[d.Group] {
			score[d.Group] = points
		}
	}
	for _, d := range subs {
		if d.App == "" {
			continue
		}
		if choice, ok := byGroup[d.Group]; ok {
			choice.Apps = append(choice.Apps, displayName(d.App))
		}
	}
	best, bestScore := "", 0
	for _, group := range order { // newest first, so ties go to the most recent device
		if score[group] > bestScore {
			best, bestScore = group, score[group]
		}
	}
	out := make([]DeviceChoice, 0, len(order))
	for _, group := range order {
		choice := *byGroup[group]
		choice.BestMatch = group == best
		out = append(out, choice)
	}
	return out
}

func pushServiceHost(endpoint string) string {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

func platformOf(userAgent string) string {
	ua := userAgent
	switch {
	case strings.Contains(ua, "iPhone"):
		return "iPhone"
	case strings.Contains(ua, "iPad"):
		return "iPad"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "Macintosh"):
		return "Mac"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "CrOS"):
		return "Chromebook"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	}
	return ""
}

func browserOf(userAgent string) string {
	ua := userAgent
	switch {
	case strings.Contains(ua, "Edg/"):
		return "Edge"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome"
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	}
	return ""
}

// deviceLabel names a device for a person: "iPhone", "Mac · Chrome". Without
// a user agent (subscriptions stored before it was recorded) the push service
// is the only clue.
func deviceLabel(userAgent, endpoint string) string {
	platform := platformOf(userAgent)
	switch platform {
	case "iPhone", "iPad":
		return platform
	case "Android":
		return "Android phone"
	case "":
	default:
		if browser := browserOf(userAgent); browser != "" {
			return platform + " · " + browser
		}
		return platform
	}
	host := pushServiceHost(endpoint)
	switch {
	case strings.HasSuffix(host, "push.apple.com"):
		return "Apple device"
	case strings.Contains(host, "googleapis.com"):
		return "Chrome or Android device"
	case strings.Contains(host, "mozilla"):
		return "Firefox browser"
	case strings.Contains(host, "windows.com"):
		return "Edge browser"
	}
	return "Browser"
}

func isGenericLabel(label string) bool {
	switch label {
	case "Apple device", "Chrome or Android device", "Firefox browser", "Edge browser", "Browser":
		return true
	}
	return false
}
