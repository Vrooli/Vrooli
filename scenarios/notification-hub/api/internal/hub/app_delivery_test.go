package hub_test

import (
	"context"
	"errors"
	"testing"
	"time"

	hub "notification-hub/internal/hub"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

const (
	appOrigin   = "https://web-console.example.test"
	iPhoneAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.5 Mobile/15E148 Safari/604.1"
	macAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
)

type sentPush struct {
	endpoint, title, navigate string
}

type appRecordingSender struct {
	sent []sentPush
	gone map[string]bool
}

func (s *appRecordingSender) Send(_ context.Context, sub hub.PushSubscription, msg hub.PushMessage) (string, error) {
	s.sent = append(s.sent, sentPush{endpoint: sub.Endpoint, title: msg.Title, navigate: msg.Navigate})
	if s.gone[sub.Endpoint] {
		return "", retiredEndpointError{}
	}
	return "provider:" + sub.Endpoint, nil
}

func (s *appRecordingSender) endpoints() []string {
	out := make([]string, 0, len(s.sent))
	for _, p := range s.sent {
		out = append(out, p.endpoint)
	}
	return out
}

// appFixture is one operator with Chimesail on an iPhone and on a Mac, and
// web-console eligible for app delivery at appOrigin.
type appFixture struct {
	service           *hub.Service
	db                *database.RoutedDB
	iphone, mac       string // Chimesail endpoints
	iphoneGroup       string
	eligibilityErrors map[string]error
}

func newAppFixture(t *testing.T) *appFixture {
	t.Helper()
	service, db := testService(t)
	f := &appFixture{service: service, db: db, eligibilityErrors: map[string]error{}}
	f.iphone = registerChimesail(t, service, "alice", "https://web.push.apple.com/"+uuid.NewString(), iPhoneAgent)
	f.mac = registerChimesail(t, service, "alice", "https://fcm.googleapis.com/fcm/send/"+uuid.NewString(), macAgent)
	service.SetAppEligibility(func(_ context.Context, scenario string) (string, error) {
		if err := f.eligibilityErrors[scenario]; err != nil {
			return "", err
		}
		return "https://" + scenario + ".example.test", nil
	})
	state, err := service.GetAppState(context.Background(), "alice", "web-console", "", iPhoneAgent)
	require.NoError(t, err)
	for _, d := range state.Devices {
		if d.Label == "iPhone" {
			f.iphoneGroup = d.Group
		}
	}
	require.NotEmpty(t, f.iphoneGroup, "the iPhone's Chimesail subscription must be offered as a device")
	return f
}

func registerChimesail(t *testing.T, service *hub.Service, recipient, endpoint, agent string) string {
	t.Helper()
	require.NoError(t, service.RegisterPushSubscription(context.Background(), recipient, hub.PushSubscription{
		Endpoint: endpoint, P256DH: "client-key", Auth: "auth-secret", Origin: "https://notification-hub.example.test", UserAgent: agent,
	}))
	return endpoint
}

func (f *appFixture) pairApp(t *testing.T, scenario string) string {
	t.Helper()
	endpoint := "https://web.push.apple.com/" + uuid.NewString()
	_, err := f.service.RegisterApp(context.Background(), "alice", hub.AppRegistration{
		Scenario: scenario, Endpoint: endpoint, P256DH: "k", Auth: "a", Origin: "https://" + scenario + ".example.test",
		UserAgent: iPhoneAgent, Permission: "granted", PairWithGroup: f.iphoneGroup,
	})
	require.NoError(t, err)
	return endpoint
}

func (f *appFixture) send(t *testing.T, source, contextURL string) (string, *appRecordingSender) {
	t.Helper()
	return f.sendWith(t, source, contextURL, &appRecordingSender{gone: map[string]bool{}})
}

func (f *appFixture) sendWith(t *testing.T, source, contextURL string, sender *appRecordingSender) (string, *appRecordingSender) {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := f.db.ExecContext(context.Background(), `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, source_scenario, context_url, created_at, updated_at) VALUES (?, 'alice', 'Agent needs your input', 'Codex · tab 2', 'normal', 'private', ?, '', 300, ?, ?, ?, ?)`, id, uuid.NewString(), source, contextURL, now, now)
	require.NoError(t, err)
	_, err = f.db.ExecContext(context.Background(), `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'test accepted', ?)`, id, now)
	require.NoError(t, err)
	f.service.SetPushSender(sender)
	require.NoError(t, f.service.Process(context.Background(), id))
	return id, sender
}

func TestAppDelivery_PairedAppReplacesChimesailOnItsDeviceOnly(t *testing.T) {
	f := newAppFixture(t)
	app := f.pairApp(t, "web-console")

	_, sender := f.send(t, "web-console", appOrigin+"/sessions/abc?tab=2")

	require.ElementsMatch(t, []string{f.mac, app}, sender.endpoints(),
		"the iPhone gets one copy, in the paired app; the Mac without the app keeps Chimesail")
	for _, p := range sender.sent {
		if p.endpoint == app {
			require.Equal(t, appOrigin+"/sessions/abc?tab=2", p.navigate, "a tap opens the exact in-app screen")
			require.Equal(t, "Agent needs your input", p.title, "the OS already names the app, so the title is unprefixed")
		} else {
			require.Empty(t, p.navigate, "Chimesail keeps the classic payload")
		}
	}
}

func TestAppDelivery_OtherScenariosAndSourcelessNotificationsStayOnChimesail(t *testing.T) {
	f := newAppFixture(t)
	app := f.pairApp(t, "web-console")

	_, other := f.send(t, "agent-manager", "")
	require.ElementsMatch(t, []string{f.iphone, f.mac}, other.endpoints(), "a web-console app never shows another scenario's notifications")
	_, none := f.send(t, "", "")
	require.ElementsMatch(t, []string{f.iphone, f.mac}, none.endpoints())
	require.NotContains(t, none.endpoints(), app)
}

func TestAppDelivery_GoneAppFallsBackToThatDevicesChimesailInTheSamePass(t *testing.T) {
	f := newAppFixture(t)
	app := f.pairApp(t, "web-console")

	id, sender := f.sendWith(t, "web-console", "", &appRecordingSender{gone: map[string]bool{app: true}})

	require.ElementsMatch(t, []string{app, f.mac, f.iphone}, sender.endpoints(),
		"the deleted app is tried once, then the iPhone's Chimesail delivers the same notification")
	notification, _, err := f.service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", notification.State)

	_, next := f.send(t, "web-console", "")
	require.ElementsMatch(t, []string{f.iphone, f.mac}, next.endpoints(), "the retired app subscription is removed")
}

func TestAppDelivery_UnhealthyAppUsesChimesail(t *testing.T) {
	cases := map[string]func(f *appFixture, app string){
		"notifications turned off on the device": func(f *appFixture, app string) {
			_, err := f.service.ConfirmApp(context.Background(), "alice", "web-console", app, "denied", iPhoneAgent)
			require.NoError(t, err)
		},
		"not opened for 30 days": func(f *appFixture, app string) {
			old := time.Now().Add(-31 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
			_, err := f.db.ExecContext(context.Background(), `UPDATE push_subscriptions SET confirmed_at=? WHERE endpoint=?`, old, app)
			require.NoError(t, err)
		},
		"address no longer permanent": func(f *appFixture, _ string) {
			f.eligibilityErrors["web-console"] = errors.New("web-console only has a temporary address")
			f.service.SetAppEligibility(func(_ context.Context, scenario string) (string, error) {
				if err := f.eligibilityErrors[scenario]; err != nil {
					return "", err
				}
				return "https://" + scenario + ".example.test", nil
			})
		},
	}
	for name, degrade := range cases {
		t.Run(name, func(t *testing.T) {
			f := newAppFixture(t)
			app := f.pairApp(t, "web-console")
			degrade(f, app)
			_, sender := f.send(t, "web-console", "")
			require.ElementsMatch(t, []string{f.iphone, f.mac}, sender.endpoints())
		})
	}
}

func TestAppDelivery_ConfirmationRestoresDeliveryAfterAStaleApp(t *testing.T) {
	f := newAppFixture(t)
	app := f.pairApp(t, "web-console")
	old := time.Now().Add(-31 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	_, err := f.db.ExecContext(context.Background(), `UPDATE push_subscriptions SET confirmed_at=? WHERE endpoint=?`, old, app)
	require.NoError(t, err)

	healthy, err := f.service.ConfirmApp(context.Background(), "alice", "web-console", app, "granted", iPhoneAgent)
	require.NoError(t, err)
	require.True(t, healthy)
	_, sender := f.send(t, "web-console", "")
	require.ElementsMatch(t, []string{app, f.mac}, sender.endpoints())

	_, err = f.service.ConfirmApp(context.Background(), "alice", "web-console", "https://web.push.apple.com/unknown", "granted", "")
	require.ErrorIs(t, err, hub.ErrAppNotRegistered, "an app the hub does not know must re-enable instead of silently checking in")
}

func TestAppDelivery_RegistrationIsBoundToTheScenariosPermanentAddress(t *testing.T) {
	f := newAppFixture(t)
	reg := hub.AppRegistration{Scenario: "web-console", Endpoint: "https://web.push.apple.com/x", P256DH: "k", Auth: "a", Permission: "granted", PairWithGroup: f.iphoneGroup}

	reg.Origin = "https://agent-manager.example.test"
	_, err := f.service.RegisterApp(context.Background(), "alice", reg)
	require.ErrorIs(t, err, hub.ErrAppIneligible, "a page on another scenario's address cannot register as web-console's app")

	f.eligibilityErrors["web-console"] = errors.New("web-console only has a temporary address")
	reg.Origin = appOrigin
	_, err = f.service.RegisterApp(context.Background(), "alice", reg)
	require.ErrorIs(t, err, hub.ErrAppIneligible)
	state, err := f.service.GetAppState(context.Background(), "alice", "web-console", "", iPhoneAgent)
	require.NoError(t, err)
	require.False(t, state.Eligible)
	require.Contains(t, state.IneligibleReason, "temporary address")

	reg.PairWithGroup = ""
	delete(f.eligibilityErrors, "web-console")
	_, err = f.service.RegisterApp(context.Background(), "alice", reg)
	require.ErrorIs(t, err, hub.ErrInvalidArgument, "the operator must say which device the app is on")
}

func TestAppDelivery_ChimesailReRegistrationKeepsThePairing(t *testing.T) {
	f := newAppFixture(t)
	app := f.pairApp(t, "web-console")
	registerChimesail(t, f.service, "alice", f.iphone, iPhoneAgent) // pushsubscriptionchange or reload

	_, sender := f.send(t, "web-console", "")
	require.ElementsMatch(t, []string{app, f.mac}, sender.endpoints())
}

func TestAppDelivery_StandaloneAppAddsADeviceWithoutReplacingChimesail(t *testing.T) {
	f := newAppFixture(t)
	endpoint := "https://web.push.apple.com/" + uuid.NewString()
	state, err := f.service.RegisterApp(context.Background(), "alice", hub.AppRegistration{
		Scenario: "web-console", Endpoint: endpoint, P256DH: "k", Auth: "a", Origin: appOrigin, Permission: "granted", Standalone: true, UserAgent: iPhoneAgent,
	})
	require.NoError(t, err)
	require.NotNil(t, state.Subscription)
	require.True(t, state.Subscription.Standalone)

	_, sender := f.send(t, "web-console", "")
	require.ElementsMatch(t, []string{f.iphone, f.mac, endpoint}, sender.endpoints())
}

func TestAppDelivery_StateNamesDevicesAndPreselectsTheLikelyOne(t *testing.T) {
	f := newAppFixture(t)
	state, err := f.service.GetAppState(context.Background(), "alice", "web-console", "https://web.push.apple.com/new", iPhoneAgent)
	require.NoError(t, err)
	require.True(t, state.Eligible)
	require.Equal(t, appOrigin, state.PublicURL)
	require.Nil(t, state.Subscription)
	labels := map[string]bool{}
	for _, d := range state.Devices {
		labels[d.Label] = d.BestMatch
	}
	require.Equal(t, map[string]bool{"iPhone": true, "Mac · Chrome": false}, labels)

	app := f.pairApp(t, "web-console")
	state, err = f.service.GetAppState(context.Background(), "alice", "web-console", app, iPhoneAgent)
	require.NoError(t, err)
	require.NotNil(t, state.Subscription)
	require.True(t, state.Subscription.Healthy)
	require.Equal(t, "iPhone", state.Subscription.DeviceLabel)

	devices, err := f.service.ListDeviceApps(context.Background(), "alice")
	require.NoError(t, err)
	var iphoneApps []string
	for _, d := range devices {
		if d.Group == f.iphoneGroup {
			for _, a := range d.Apps {
				iphoneApps = append(iphoneApps, a.Scenario)
			}
		}
	}
	require.Equal(t, []string{"", "web-console"}, iphoneApps, "Chimesail is listed first, then the paired app")

	for _, d := range devices {
		for _, a := range d.Apps {
			if a.Scenario == "web-console" {
				require.NoError(t, f.service.RemoveDeviceApp(context.Background(), "alice", a.SubscriptionID))
			} else {
				require.ErrorIs(t, f.service.RemoveDeviceApp(context.Background(), "alice", a.SubscriptionID), hub.ErrAppNotRegistered,
					"Chimesail itself is not removable from the device list")
			}
		}
	}
	_, sender := f.send(t, "web-console", "")
	require.ElementsMatch(t, []string{f.iphone, f.mac}, sender.endpoints())
}
