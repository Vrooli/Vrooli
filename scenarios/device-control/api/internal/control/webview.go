package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"device-control/strategy"

	"github.com/google/uuid"
)

// AttachWebView is the lease-scoped device verb used by Android WebView
// automation. The strategy owns socket discovery and forwarding; the control
// service owns identity, authorization, and auditability.
func (s *Service) AttachWebView(ctx context.Context, deviceID, actor, leaseToken, packageName string, requestedTransport ...string) (WebViewAttachment, error) {
	if strings.TrimSpace(leaseToken) == "" {
		return WebViewAttachment{}, fmt.Errorf("webview attach requires a held device lease")
	}
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return WebViewAttachment{}, err
	}
	transport := ""
	if len(requestedTransport) > 0 {
		transport = strings.TrimSpace(requestedTransport[0])
	}
	if transport == "" {
		if record, found := s.devices.Get(deviceID); found {
			transport = record.Transport
		}
		if transport == "" {
			transport = "usb"
		}
	}
	var routeErr error
	transport, routeErr = s.requireCurrentEffectRoute(deviceID, transport)
	if routeErr != nil {
		return WebViewAttachment{}, routeErr
	}
	strat, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return WebViewAttachment{}, fmt.Errorf("device %q is unavailable for WebView attach", deviceID)
	}
	attacher, ok := strat.(strategy.WebViewAttacher)
	if !ok {
		return WebViewAttachment{}, fmt.Errorf("strategy %q does not support WebView attach", strat.ID())
	}
	startedAt := time.Now()
	attachAttempts := 1
	relaunchFailed := false
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
				attachAttempts++
				endpoint, err = attacher.AttachWebView(ctx, packageName)
			} else {
				relaunchFailed = true
				err = errors.Join(err, fmt.Errorf("relaunch before WebView attach retry: %w", launchErr))
			}
		}
	}
	outcome := "success"
	details := map[string]string{"duration_ms": fmt.Sprint(time.Since(startedAt).Milliseconds()), "attach_attempts": fmt.Sprint(attachAttempts)}
	if err != nil {
		outcome = "failure"
		var attachFailure *strategy.WebViewAttachFailure
		if errors.As(err, &attachFailure) && attachFailure.Stage != "" {
			details["failure_stage"] = attachFailure.Stage
		} else {
			details["failure_stage"] = "unknown"
		}
		if relaunchFailed {
			details["relaunch_failed"] = "true"
		}
	}
	s.recordWebViewAudit(ctx, actor, deviceID, transport, session.ID, outcome, details)
	if err != nil {
		return WebViewAttachment{}, err
	}
	return endpoint, nil
}

func (s *Service) recordWebViewAudit(ctx context.Context, actor, deviceID, transport, leaseID, outcome string, details map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, Transport: transport, LeaseID: leaseID, Verb: "webview-attach", Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true, EvidenceBacked: false, Details: details}
	if s.db != nil {
		encodedDetails, _ := json.Marshal(record.Details)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, transport, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out, details_json, evidence_backed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.Transport, record.LeaseID, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0, string(encodedDetails), 0)
	}
	s.audits = append(s.audits, record)
}
