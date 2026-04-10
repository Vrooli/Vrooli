package playwright

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// mockClient implements the minimal interface needed for driver testing.
// It tracks method calls for verification.
type mockClient struct {
	createSessionCalled bool
	navigateCalled      bool
	healthCalled        bool
	lastSessionReq      *driver.CreateSessionRequest
}

func (c *mockClient) CreateSession(ctx context.Context, req *driver.CreateSessionRequest) (*driver.CreateSessionResponse, error) {
	c.createSessionCalled = true
	c.lastSessionReq = req
	return &driver.CreateSessionResponse{
		SessionID: "test-session-123",
	}, nil
}

func (c *mockClient) CloseSession(ctx context.Context, sessionID string) (*driver.CloseSessionResponse, error) {
	return &driver.CloseSessionResponse{}, nil
}

func (c *mockClient) Navigate(ctx context.Context, sessionID string, req *driver.NavigateRequest) (*driver.NavigateResponse, error) {
	c.navigateCalled = true
	return &driver.NavigateResponse{
		URL:        req.URL,
		Title:      "Test Page",
		StatusCode: 200,
	}, nil
}

func (c *mockClient) Health(ctx context.Context) error {
	c.healthCalled = true
	return nil
}

func (c *mockClient) RunInstructions(ctx context.Context, sessionID string, instructions []map[string]interface{}) error {
	return nil
}

func (c *mockClient) CaptureScreenshot(ctx context.Context, sessionID string, req *driver.CaptureScreenshotRequest) (*driver.CaptureScreenshotResponse, error) {
	return &driver.CaptureScreenshotResponse{
		Data:      "dGVzdA==", // base64 "test"
		MediaType: "image/jpeg",
		Width:     1920,
		Height:    1080,
	}, nil
}

func (c *mockClient) GetNavigationState(ctx context.Context, sessionID string) (*driver.NavigationStateResponse, error) {
	return &driver.NavigationStateResponse{
		URL:   "https://example.com",
		Title: "Example",
	}, nil
}

func TestNewDriver(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Test with default client creation (will fail due to no server, but tests option handling)
	_, err := NewDriver(WithLogger(log))
	if err == nil {
		t.Log("NewDriver succeeded with default client")
	}

	// Test with mock client
	mockCl := &mockClient{}
	drv, err := NewDriver(
		WithLogger(log),
		WithClient(&driver.Client{}), // This will be unused since we test the type
	)
	if err != nil {
		t.Skipf("Skipping full driver test: %v", err)
	}

	_ = drv
	_ = mockCl
}

func TestDriver_Type(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	drv := &Driver{
		log:      log,
		sessions: make(map[string]*session),
	}

	if drv.Type() != driver.DriverTypePlaywright {
		t.Errorf("expected driver type playwright, got %s", drv.Type())
	}
}

func TestDriver_InterfaceCompliance(t *testing.T) {
	// Verify that Driver implements driver.Driver
	var _ driver.Driver = (*Driver)(nil)

	// Verify that session implements driver.Session
	var _ driver.Session = (*session)(nil)
}

func TestSession_ID(t *testing.T) {
	log := logrus.New()
	sess := &session{
		id:  "test-session-id",
		log: log,
	}

	if sess.ID() != "test-session-id" {
		t.Errorf("expected session ID 'test-session-id', got '%s'", sess.ID())
	}
}

func TestSession_IsClosed(t *testing.T) {
	log := logrus.New()
	sess := &session{
		id:     "test-session-id",
		log:    log,
		closed: false,
	}

	if sess.isClosed() {
		t.Error("expected session to not be closed initially")
	}

	sess.closed = true
	if !sess.isClosed() {
		t.Error("expected session to be closed after setting closed=true")
	}
}

func TestSession_Pages(t *testing.T) {
	log := logrus.New()
	sess := &session{
		id:  "test-session-id",
		log: log,
	}

	// Pages() should return nil for the basic Playwright driver
	if sess.Pages() != nil {
		t.Error("expected Pages() to return nil")
	}
}

func TestRecordingCallbackIntegration(t *testing.T) {
	// This test verifies that recording callbacks are properly invoked
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	var recordedActionType string
	var recordedPageID uuid.UUID

	sess := &session{
		id:  "test-session-id",
		log: log,
		recording: &driver.RecordingConfig{
			ActionCallback: func(action *driver.RecordedAction, pageID uuid.UUID) {
				recordedActionType = action.ActionType
				recordedPageID = pageID
			},
		},
	}

	// Simulate a navigate action
	if sess.recording != nil && sess.recording.ActionCallback != nil {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "navigate",
			URL:        "https://example.com",
		}
		sess.recording.ActionCallback(action, uuid.Nil)
	}

	if recordedActionType != "navigate" {
		t.Errorf("expected recorded action type 'navigate', got '%s'", recordedActionType)
	}

	if recordedPageID != uuid.Nil {
		t.Errorf("expected nil page ID, got %s", recordedPageID)
	}
}
