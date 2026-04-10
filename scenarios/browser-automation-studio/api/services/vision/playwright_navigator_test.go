package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
	ws "github.com/vrooli/browser-automation-studio/websocket"
)

// mockHTTPDoer implements HTTPDoer for testing.
type mockHTTPDoer struct {
	response    *http.Response
	err         error
	lastRequest *http.Request
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	return m.response, m.err
}

// mockCreditService implements credits.CreditService for testing.
type mockCreditService struct {
	chargeErr   error
	chargeCount int
}

func (m *mockCreditService) CanCharge(_ context.Context, _ string, _ credits.OperationType) (bool, int, error) {
	return true, 100, nil
}

func (m *mockCreditService) Charge(_ context.Context, _ credits.ChargeRequest) (*credits.ChargeResult, error) {
	m.chargeCount++
	return &credits.ChargeResult{}, m.chargeErr
}

func (m *mockCreditService) ChargeIfAllowed(_ context.Context, _ credits.ChargeRequest) (*credits.ChargeResult, error) {
	return &credits.ChargeResult{}, nil
}

func (m *mockCreditService) GetUsage(_ context.Context, _ string) (*credits.UsageSummary, error) {
	return &credits.UsageSummary{}, nil
}

func (m *mockCreditService) GetOperationCost(_ credits.OperationType) int {
	return 1
}

func (m *mockCreditService) LogFailedOperation(_ context.Context, _ credits.ChargeRequest, _ error) error {
	return nil
}

func (m *mockCreditService) GetUsageHistory(_ context.Context, _ string, _, _ int) ([]credits.UsageSummary, bool, error) {
	return nil, false, nil
}

func (m *mockCreditService) GetOperationLog(_ context.Context, _, _, _ string, _, _ int) (*credits.OperationLogPage, error) {
	return &credits.OperationLogPage{}, nil
}

func (m *mockCreditService) CanPerformAIOperation(_ context.Context, _ string, _ credits.OperationType, _ bool) (bool, string, string, int, error) {
	return true, "", "", 100, nil
}

// mockWSHub implements wsHub.HubInterface for testing.
type mockWSHub struct {
	broadcastCount int
	lastEnvelope   any
}

func (m *mockWSHub) ServeWS(_ *websocket.Conn, _ *uuid.UUID)                      {}
func (m *mockWSHub) BroadcastRecordingEntry(_ string, _ *ws.UnifiedTimelineEntry) ws.BroadcastResult {
	return ws.BroadcastResult{}
}
func (m *mockWSHub) BroadcastRecordingFrame(_ string, _ *ws.RecordingFrame) {}
func (m *mockWSHub) BroadcastBinaryFrame(_ string, _ []byte)                {}
func (m *mockWSHub) HasRecordingSubscribers(_ string) bool                  { return false }
func (m *mockWSHub) BroadcastPerfStats(_ string, _ any)                     {}
func (m *mockWSHub) BroadcastPageEvent(_ string, _ any)                     {}
func (m *mockWSHub) BroadcastPageSwitch(_, _ string)                        {}
func (m *mockWSHub) HasExecutionFrameSubscribers(_ string) bool             { return false }
func (m *mockWSHub) BroadcastExecutionFrame(_ string, _ *ws.ExecutionFrame) {}
func (m *mockWSHub) BroadcastExportProgress(_ *ws.ExportProgress)           {}
func (m *mockWSHub) GetClientCount() int                                    { return 0 }
func (m *mockWSHub) Run()                                                   {}
func (m *mockWSHub) CloseExecution(_ uuid.UUID)                             {}

func (m *mockWSHub) BroadcastEnvelope(envelope any) {
	m.broadcastCount++
	m.lastEnvelope = envelope
}

func TestNewPlaywrightVisionNavigator(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("creates with defaults", func(t *testing.T) {
		nav := NewPlaywrightVisionNavigator(log)

		if nav == nil {
			t.Fatal("NewPlaywrightVisionNavigator returned nil")
		}
		if nav.log != log {
			t.Error("logger not set correctly")
		}
		if nav.activeNavigations == nil {
			t.Error("activeNavigations map not initialized")
		}
	})

	t.Run("applies options", func(t *testing.T) {
		httpClient := &mockHTTPDoer{}
		wsHub := &mockWSHub{}
		creditSvc := &mockCreditService{}

		nav := NewPlaywrightVisionNavigator(log,
			WithPlaywrightHTTPClient(httpClient),
			WithPlaywrightHub(wsHub),
			WithPlaywrightCreditService(creditSvc),
		)

		// Verify options were applied by checking they're not nil
		// We can't compare interface values directly, so we verify non-nil
		if nav.httpClient == nil {
			t.Error("HTTP client option not applied")
		}
		if nav.wsHub == nil {
			t.Error("WebSocket hub option not applied")
		}
		if nav.creditService == nil {
			t.Error("credit service option not applied")
		}
	})
}

func TestPlaywrightVisionNavigator_Type(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	if nav.Type() != NavigatorPlaywright {
		t.Errorf("Type() = %v, want %v", nav.Type(), NavigatorPlaywright)
	}
}

func TestPlaywrightVisionNavigator_Description(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	desc := nav.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestPlaywrightVisionNavigator_CreditPolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	policy := nav.CreditPolicy()

	if !policy.RequiresCredits {
		t.Error("RequiresCredits should be true")
	}
	if policy.CreditsPerStep != 2 {
		t.Errorf("CreditsPerStep = %d, want 2", policy.CreditsPerStep)
	}
	if !policy.PerStepCharging {
		t.Error("PerStepCharging should be true")
	}
	if len(policy.BypassConditions) != 2 {
		t.Errorf("BypassConditions length = %d, want 2", len(policy.BypassConditions))
	}
}

func TestPlaywrightVisionNavigator_ClientSourcePolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	policy := nav.ClientSourcePolicy()

	// Should allow all sources
	if !policy.IsAllowed(ClientSourceCLI) {
		t.Error("should allow CLI")
	}
	if !policy.IsAllowed(ClientSourceUI) {
		t.Error("should allow UI")
	}
	if !policy.IsAllowed(ClientSourceAPI) {
		t.Error("should allow API")
	}
}

func TestPlaywrightVisionNavigator_Navigate(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("missing API key returns error", func(t *testing.T) {
		nav := NewPlaywrightVisionNavigator(log)

		// Ensure env var is not set
		t.Setenv("OPENROUTER_API_KEY", "")

		_, err := nav.Navigate(t.Context(), NavigationRequest{
			SessionID: "session123",
			Prompt:    "Click button",
			Model:     "gpt-4o",
		})

		if err == nil {
			t.Error("expected error for missing API key")
		}
	})

	t.Run("successful navigation start", func(t *testing.T) {
		mockResp := &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"navigation_id": "nav_test123",
				"status": "navigating",
				"model": "gpt-4o",
				"max_steps": 20
			}`))),
		}

		httpClient := &mockHTTPDoer{response: mockResp}
		nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHTTPClient(httpClient))

		handle, err := nav.Navigate(t.Context(), NavigationRequest{
			SessionID: "session123",
			Prompt:    "Click button",
			Model:     "gpt-4o",
			APIKey:    "test-key",
		})

		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}
		if handle == nil {
			t.Fatal("Navigate() returned nil handle")
		}
		if handle.SessionID() != "session123" {
			t.Errorf("SessionID() = %q, want %q", handle.SessionID(), "session123")
		}
	})

	t.Run("driver error", func(t *testing.T) {
		mockResp := &http.Response{
			StatusCode: 400,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"message": "session not found"
			}`))),
		}

		httpClient := &mockHTTPDoer{response: mockResp}
		nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHTTPClient(httpClient))

		_, err := nav.Navigate(t.Context(), NavigationRequest{
			SessionID: "session123",
			Prompt:    "Click button",
			Model:     "gpt-4o",
			APIKey:    "test-key",
		})

		if err == nil {
			t.Error("expected error for driver error response")
		}
	})

	t.Run("max steps capping", func(t *testing.T) {
		mockResp := &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"navigation_id": "nav_test123",
				"status": "navigating"
			}`))),
		}

		httpClient := &mockHTTPDoer{response: mockResp}
		nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHTTPClient(httpClient))

		_, err := nav.Navigate(t.Context(), NavigationRequest{
			SessionID: "session123",
			Prompt:    "Click button",
			Model:     "gpt-4o",
			APIKey:    "test-key",
			MaxSteps:  200, // Over limit
		})

		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Verify max steps was capped in the request
		var reqBody map[string]interface{}
		if httpClient.lastRequest != nil {
			body, _ := io.ReadAll(httpClient.lastRequest.Body)
			_ = json.Unmarshal(body, &reqBody)
			if maxSteps, ok := reqBody["max_steps"].(float64); ok && maxSteps > 100 {
				t.Errorf("max_steps was not capped: %v", maxSteps)
			}
		}
	})
}

func TestPlaywrightVisionNavigator_HandleStepCallback(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	wsHub := &mockWSHub{}
	creditSvc := &mockCreditService{}
	nav := NewPlaywrightVisionNavigator(log,
		WithPlaywrightHub(wsHub),
		WithPlaywrightCreditService(creditSvc),
	)

	// Create a session to track
	session := &NavigationSession{
		NavigationID:  "nav_test123",
		SessionID:     "session123",
		UserID:        "user123",
		Model:         "gpt-4o",
		StartedAt:     time.Now(),
		Status:        StatusNavigating,
		NavigatorType: NavigatorPlaywright,
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_test123"] = session
	nav.mu.Unlock()

	t.Run("updates session and broadcasts", func(t *testing.T) {
		step := &NavigationStep{
			NavigationID: "nav_test123",
			StepNumber:   1,
			Action:       map[string]interface{}{"type": "click"},
			Reasoning:    "Clicking the button",
			CurrentURL:   "https://example.com",
			TokensUsed: TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}

		err := nav.HandleStepCallback(t.Context(), step)
		if err != nil {
			t.Fatalf("HandleStepCallback() error = %v", err)
		}

		// Verify session updated
		nav.mu.RLock()
		s := nav.activeNavigations["nav_test123"]
		nav.mu.RUnlock()

		if s.StepCount != 1 {
			t.Errorf("StepCount = %d, want 1", s.StepCount)
		}
		if s.TotalTokens != 150 {
			t.Errorf("TotalTokens = %d, want 150", s.TotalTokens)
		}

		// Verify WebSocket broadcast
		if wsHub.broadcastCount != 1 {
			t.Errorf("broadcastCount = %d, want 1", wsHub.broadcastCount)
		}

		// Verify credit charge
		if creditSvc.chargeCount != 1 {
			t.Errorf("chargeCount = %d, want 1", creditSvc.chargeCount)
		}
	})

	t.Run("human intervention event", func(t *testing.T) {
		wsHub.broadcastCount = 0

		step := &NavigationStep{
			NavigationID:  "nav_test123",
			StepNumber:    2,
			Action:        map[string]interface{}{"type": "wait"},
			AwaitingHuman: true,
			HumanIntervention: &HumanInterventionInfo{
				Reason:           "CAPTCHA detected",
				InterventionType: "captcha",
				Trigger:          "ai_requested",
			},
		}

		err := nav.HandleStepCallback(t.Context(), step)
		if err != nil {
			t.Fatalf("HandleStepCallback() error = %v", err)
		}

		// Should broadcast 2 events: step + human intervention
		if wsHub.broadcastCount != 2 {
			t.Errorf("broadcastCount = %d, want 2", wsHub.broadcastCount)
		}

		// Verify session status
		nav.mu.RLock()
		s := nav.activeNavigations["nav_test123"]
		nav.mu.RUnlock()

		if s.Status != StatusAwaitingHuman {
			t.Errorf("Status = %v, want %v", s.Status, StatusAwaitingHuman)
		}
	})
}

func TestPlaywrightVisionNavigator_HandleCompleteCallback(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	wsHub := &mockWSHub{}
	nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHub(wsHub))

	// Create a session
	session := &NavigationSession{
		NavigationID:  "nav_complete",
		SessionID:     "session123",
		Status:        StatusNavigating,
		NavigatorType: NavigatorPlaywright,
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_complete"] = session
	nav.mu.Unlock()

	result := &NavigationResult{
		NavigationID:    "nav_complete",
		Status:          StatusCompleted,
		TotalSteps:      5,
		TotalTokens:     750,
		TotalDurationMs: 15000,
		FinalURL:        "https://example.com/success",
		Summary:         "Task completed successfully",
	}

	err := nav.HandleCompleteCallback(t.Context(), result)
	if err != nil {
		t.Fatalf("HandleCompleteCallback() error = %v", err)
	}

	// Verify session updated
	nav.mu.RLock()
	s := nav.activeNavigations["nav_complete"]
	nav.mu.RUnlock()

	if s.Status != StatusCompleted {
		t.Errorf("Status = %v, want %v", s.Status, StatusCompleted)
	}
	if s.TotalTokens != 750 {
		t.Errorf("TotalTokens = %d, want 750", s.TotalTokens)
	}

	// Verify broadcast
	if wsHub.broadcastCount != 1 {
		t.Errorf("broadcastCount = %d, want 1", wsHub.broadcastCount)
	}
}

func TestPlaywrightVisionNavigator_GetSession(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	// Create a session
	session := &NavigationSession{
		NavigationID: "nav_getsession",
		SessionID:    "session123",
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_getsession"] = session
	nav.mu.Unlock()

	t.Run("existing session", func(t *testing.T) {
		s, exists := nav.GetSession("nav_getsession")
		if !exists {
			t.Error("expected session to exist")
		}
		if s.SessionID != "session123" {
			t.Errorf("SessionID = %q, want %q", s.SessionID, "session123")
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		_, exists := nav.GetSession("nav_notfound")
		if exists {
			t.Error("expected session to not exist")
		}
	})
}

func TestPlaywrightVisionNavigator_AbortNavigation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("navigation not found", func(t *testing.T) {
		nav := NewPlaywrightVisionNavigator(log)

		err := nav.AbortNavigation(t.Context(), "nav_notfound")
		if err == nil {
			t.Error("expected error for non-existent navigation")
		}
	})

	t.Run("successful abort", func(t *testing.T) {
		mockResp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}

		httpClient := &mockHTTPDoer{response: mockResp}
		nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHTTPClient(httpClient))

		// Create session
		session := &NavigationSession{
			NavigationID: "nav_abort",
			SessionID:    "session123",
			Status:       StatusNavigating,
		}
		nav.mu.Lock()
		nav.activeNavigations["nav_abort"] = session
		nav.mu.Unlock()

		err := nav.AbortNavigation(t.Context(), "nav_abort")
		if err != nil {
			t.Fatalf("AbortNavigation() error = %v", err)
		}

		// Verify status updated
		nav.mu.RLock()
		s := nav.activeNavigations["nav_abort"]
		nav.mu.RUnlock()

		if s.Status != StatusAborted {
			t.Errorf("Status = %v, want %v", s.Status, StatusAborted)
		}
	})
}

func TestPlaywrightVisionNavigator_ResumeNavigation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("navigation not found", func(t *testing.T) {
		nav := NewPlaywrightVisionNavigator(log)

		err := nav.ResumeNavigation(t.Context(), "nav_notfound")
		if err == nil {
			t.Error("expected error for non-existent navigation")
		}
	})

	t.Run("not awaiting human", func(t *testing.T) {
		nav := NewPlaywrightVisionNavigator(log)

		session := &NavigationSession{
			NavigationID:  "nav_resume",
			SessionID:     "session123",
			Status:        StatusNavigating,
			AwaitingHuman: false,
		}
		nav.mu.Lock()
		nav.activeNavigations["nav_resume"] = session
		nav.mu.Unlock()

		err := nav.ResumeNavigation(t.Context(), "nav_resume")
		if err == nil {
			t.Error("expected error when not awaiting human")
		}
	})

	t.Run("successful resume", func(t *testing.T) {
		mockResp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}

		wsHub := &mockWSHub{}
		httpClient := &mockHTTPDoer{response: mockResp}
		nav := NewPlaywrightVisionNavigator(log,
			WithPlaywrightHTTPClient(httpClient),
			WithPlaywrightHub(wsHub),
		)

		session := &NavigationSession{
			NavigationID:  "nav_resume_ok",
			SessionID:     "session123",
			Status:        StatusAwaitingHuman,
			AwaitingHuman: true,
			HumanIntervention: &HumanInterventionInfo{
				Reason: "CAPTCHA",
			},
		}
		nav.mu.Lock()
		nav.activeNavigations["nav_resume_ok"] = session
		nav.mu.Unlock()

		err := nav.ResumeNavigation(t.Context(), "nav_resume_ok")
		if err != nil {
			t.Fatalf("ResumeNavigation() error = %v", err)
		}

		// Verify status updated
		nav.mu.RLock()
		s := nav.activeNavigations["nav_resume_ok"]
		nav.mu.RUnlock()

		if s.Status != StatusNavigating {
			t.Errorf("Status = %v, want %v", s.Status, StatusNavigating)
		}
		if s.AwaitingHuman {
			t.Error("AwaitingHuman should be false")
		}
		if s.HumanIntervention != nil {
			t.Error("HumanIntervention should be nil")
		}

		// Verify broadcast
		if wsHub.broadcastCount != 1 {
			t.Errorf("broadcastCount = %d, want 1", wsHub.broadcastCount)
		}
	})
}

func TestPlaywrightNavigationHandle(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	mockResp := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(bytes.NewReader([]byte(`{
			"navigation_id": "nav_handle",
			"status": "navigating"
		}`))),
	}

	httpClient := &mockHTTPDoer{response: mockResp}
	nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHTTPClient(httpClient))

	handle, err := nav.Navigate(t.Context(), NavigationRequest{
		SessionID: "session123",
		Prompt:    "Click",
		Model:     "gpt-4o",
		APIKey:    "test-key",
	})
	if err != nil {
		t.Fatalf("Navigate() error = %v", err)
	}

	t.Run("ID returns navigation ID", func(t *testing.T) {
		id := handle.ID()
		if id == "" {
			t.Error("ID() returned empty string")
		}
	})

	t.Run("SessionID returns session ID", func(t *testing.T) {
		if handle.SessionID() != "session123" {
			t.Errorf("SessionID() = %q, want %q", handle.SessionID(), "session123")
		}
	})

	t.Run("Status returns current status", func(t *testing.T) {
		status := handle.Status()
		if status != StatusNavigating {
			t.Errorf("Status() = %v, want %v", status, StatusNavigating)
		}
	})

	t.Run("Abort calls navigator abort", func(t *testing.T) {
		// Reset for abort test
		mockResp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}
		httpClient.response = mockResp

		err := handle.Abort(t.Context())
		if err != nil {
			t.Errorf("Abort() error = %v", err)
		}
	})
}
