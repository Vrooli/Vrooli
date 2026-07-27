package evidence

import (
	"context"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"
	"testing"
	"time"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeCaptureStore struct {
	items map[string][]captures.Capture
}

func (s *fakeCaptureStore) List(scenario string) ([]captures.Capture, error) {
	return append([]captures.Capture(nil), s.items[scenario]...), nil
}

func (s *fakeCaptureStore) Add(capture captures.Capture) error {
	s.items[capture.ScenarioName] = append(s.items[capture.ScenarioName], capture)
	return nil
}
func (s *fakeCaptureStore) Delete(scenario, captureID string) error { return nil }
func (s *fakeCaptureStore) DeleteAll(scenario string) ([]captures.Capture, error) {
	deleted := s.items[scenario]
	delete(s.items, scenario)
	return deleted, nil
}

func (s *fakeCaptureStore) Summary(scenario string) (captures.CapturesSummary, error) {
	var total int64
	for _, capture := range s.items[scenario] {
		total += capture.FileSizeBytes
	}
	return captures.CapturesSummary{Count: len(s.items[scenario]), TotalBytes: total}, nil
}

type fakeCaptureService struct{ store captures.Store }

func (s fakeCaptureService) Store() captures.Store              { return s.store }
func (s fakeCaptureService) DeleteCapture(string, string) error { return nil }
func (s fakeCaptureService) CleanAll(string) error              { return nil }

func TestConnectServiceRejectsBridgeTargetWithoutDispatch(t *testing.T) {
	handler := NewConnectService(nil, nil)
	_, err := handler.StartDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRequest{
		ScenarioName: "hello-desktop",
		Target:       &domainv1.EvidenceTarget{Kind: domainv1.EvidenceTarget_KIND_BRIDGE_NODE},
	}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("StartDesktopSession() error code = %v, want unimplemented (err %v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceListsEvidenceCaptureSummary(t *testing.T) {
	store := &fakeCaptureStore{items: map[string][]captures.Capture{
		"hello-desktop": {{ID: "capture-1", ScenarioName: "hello-desktop", FileSizeBytes: 42, CreatedAt: time.Now()}},
	}}
	handler := NewConnectService(nil, fakeCaptureService{store: store})

	response, err := handler.GetEvidenceCapturesSummary(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: "hello-desktop"}))
	if err != nil {
		t.Fatalf("GetEvidenceCapturesSummary() error = %v", err)
	}
	if response.Msg.GetCount() != 1 || response.Msg.GetTotalBytes() != 42 {
		t.Fatalf("summary = (%d, %d), want (1, 42)", response.Msg.GetCount(), response.Msg.GetTotalBytes())
	}
}

func TestSessionToProtoPreservesTypedStateAndMetrics(t *testing.T) {
	splashDuration, readyDuration := int64(800), int64(1500)
	currentCPU, currentRSS, peakRSS := 25.5, 150.0, 192.0
	result := sessionToProto(livedesktop.SessionView{
		ID:           "session-1",
		ScenarioName: "hello-desktop",
		State:        livedesktop.StateRunning,
		NetworkMode:  "slow",
		Metrics: &livedesktop.MetricsView{
			SplashDurationMs: &splashDuration,
			SplashDetected:   true,
			ReadyDurationMs:  &readyDuration,
			ReadyDetected:    true,
			CurrentCPU:       &currentCPU,
			CurrentRSSMB:     &currentRSS,
			PeakRSSMB:        &peakRSS,
			SampleCount:      3,
		},
	}, nil)

	if result.GetState() != domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_RUNNING {
		t.Fatalf("state = %v, want running enum", result.GetState())
	}
	if result.GetNetworkMode() != domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_SLOW {
		t.Fatalf("network mode = %v, want slow enum", result.GetNetworkMode())
	}
	if got := networkModeProto(""); got != domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_NORMAL {
		t.Fatalf("empty network mode = %v, want normal enum", got)
	}
	if result.GetMetrics().GetSampleCount() != 3 || result.GetMetrics().GetCurrentCpuPercent() != currentCPU {
		t.Fatalf("metrics = %#v, want sample count 3 and CPU %v", result.GetMetrics(), currentCPU)
	}
}
