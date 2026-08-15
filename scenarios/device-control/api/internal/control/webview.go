package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"device-control/strategy"
	"github.com/google/uuid"
)

// AttachWebView is the lease-scoped device verb used by Android WebView
// automation. The strategy owns socket discovery and forwarding; the control
// service owns identity, authorization, and auditability.
func (s *Service) AttachWebView(ctx context.Context, deviceID, actor, leaseToken, packageName string) (WebViewAttachment, error) {
	if strings.TrimSpace(leaseToken) == "" {
		return WebViewAttachment{}, fmt.Errorf("webview attach requires a held device lease")
	}
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return WebViewAttachment{}, err
	}
	transport := "usb"
	if record, found := s.devices.Get(deviceID); found && record.Transport != "" {
		transport = record.Transport
	}
	strat, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return WebViewAttachment{}, fmt.Errorf("device %q is unavailable for WebView attach", deviceID)
	}
	attacher, ok := strat.(strategy.WebViewAttacher)
	if !ok {
		return WebViewAttachment{}, fmt.Errorf("strategy %q does not support WebView attach", strat.ID())
	}
	endpoint, err := attacher.AttachWebView(ctx, packageName)
	if err != nil {
		// A device may have been locked or backgrounded between the native
		// launch verb and this attach request. Re-launch inside the same
		// lease-scoped device-control operation, then retry the attach before
		// returning a typed failure to BAS.
		if actuator, canActuate := strat.(interface {
			Actuate(context.Context, strategy.Actuation) error
		}); canActuate {
			if launchErr := actuator.Actuate(ctx, strategy.Actuation{Action: "launch", Package: packageName}); launchErr == nil {
				endpoint, err = attacher.AttachWebView(ctx, packageName)
			}
		}
	}
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	s.recordWebViewAudit(ctx, actor, deviceID, session.ID, outcome)
	if err != nil {
		return WebViewAttachment{}, err
	}
	return endpoint, nil
}

func (s *Service) recordWebViewAudit(ctx context.Context, actor, deviceID, leaseID, outcome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, LeaseID: leaseID, Verb: "webview-attach", Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true}
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.LeaseID, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0)
	}
	s.audits = append(s.audits, record)
}
