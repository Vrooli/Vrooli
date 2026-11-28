package control

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type webViewAuditFixture struct {
	*fakes.Strategy
	attachErr error
}

func (f *webViewAuditFixture) AttachWebView(context.Context, string) (strategy.WebViewEndpoint, error) {
	return strategy.WebViewEndpoint{}, f.attachErr
}

func TestWebViewAttachAuditRetainsResolvedTransportAndSanitizedFailureStage(t *testing.T) {
	db, err := sql.Open("sqlite", "file:webview-audit-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	fixture := &webViewAuditFixture{
		Strategy:  fakes.New("android-adb", strategy.StatusAvailable, strategy.CapWebViewAttach),
		attachErr: &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageSocketTableRead, Err: errors.New("raw ADB output must not be copied into the audit")},
	}
	fixture.ActuateErr = errors.New("relaunch failed")
	svc, err := NewWithDB(strategyregistry.New(fixture), db)
	require.NoError(t, err)

	now := time.Now().UTC()
	capability := strategy.Capability{Name: strategy.CapWebViewAttach, Status: strategy.StatusAvailable}
	svc.devices.Upsert(devicedomain.Record{
		ID: "phone", StrategyID: fixture.ID(), Transport: "wireless", Status: strategy.StatusAvailable,
		Health: strategy.StatusAvailable, ObservedAt: now,
		Transports: []strategy.DeviceTransport{
			{Name: "usb", StrategyID: fixture.ID(), Endpoint: "serial-1", Health: strategy.StatusAvailable, ObservedAt: now, Capabilities: map[string]strategy.Capability{strategy.CapWebViewAttach: capability}},
			{Name: "wireless", StrategyID: fixture.ID(), Endpoint: "192.0.2.2:5555", Health: strategy.StatusAvailable, ObservedAt: now, Capabilities: map[string]strategy.Capability{strategy.CapWebViewAttach: capability}},
		},
	})
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = svc.Release(lease.ID) })

	_, err = svc.AttachWebView(context.Background(), "phone", "operator", lease.LeaseToken, "com.example.app", "usb")
	require.ErrorContains(t, err, "relaunch before WebView attach retry")

	audits := svc.AuditContext(context.Background())
	require.Len(t, audits, 1)
	require.Equal(t, "webview-attach", audits[0].Verb)
	require.Equal(t, "failure", audits[0].Outcome)
	require.Equal(t, "usb", audits[0].Transport)
	require.Equal(t, strategy.WebViewStageSocketTableRead, audits[0].Details["failure_stage"])
	require.Equal(t, "1", audits[0].Details["attach_attempts"])
	require.Equal(t, "true", audits[0].Details["relaunch_failed"])
	require.NotContains(t, audits[0].Details["failure_stage"], "raw ADB output")
}
