package session

import (
	"context"
	"errors"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// AppMonitorTraceArtifact is the bounded, driver-sanitized trace result. It is
// returned only at capture completion and never included in metric receipts.
type AppMonitorTraceArtifact = driver.AppMonitorTraceArtifact

// ReadAppMonitorMetrics asks the driver for its fixed scalar metric envelope.
// The caller cannot provide a selector, script, URL, trace category, or path.
// Agent Access invokes this only from its capability-gated Broker capture.
func (s *Session) ReadAppMonitorMetrics(ctx context.Context) (*driver.AppMonitorMetricsResponse, error) {
	if s == nil || s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetAppMonitorMetrics(ctx, s.id, s.executionID, s.leaseID)
}

func (s *Session) BeginAppMonitorCapture(ctx context.Context, captureID string) error {
	if s == nil || s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.BeginAppMonitorCapture(ctx, s.id, s.executionID, s.leaseID, captureID)
}

func (s *Session) FinishAppMonitorCapture(ctx context.Context, captureID string) (*AppMonitorTraceArtifact, error) {
	if s == nil || s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.FinishAppMonitorCapture(ctx, s.id, s.executionID, s.leaseID, captureID)
}
