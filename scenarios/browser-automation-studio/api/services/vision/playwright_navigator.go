package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// PlaywrightVisionNavigator implements VisionNavigator using playwright-driver.
type PlaywrightVisionNavigator struct {
	log           *logrus.Logger
	driverBaseURL string
	wsHub         wsHub.HubInterface
	httpClient    HTTPDoer
	creditService credits.CreditService

	// Track active navigations
	mu                sync.RWMutex
	activeNavigations map[string]*NavigationSession
}

// HTTPDoer is an interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// PlaywrightNavigatorOption configures PlaywrightVisionNavigator.
type PlaywrightNavigatorOption func(*PlaywrightVisionNavigator)

// WithPlaywrightHTTPClient sets a custom HTTP client.
func WithPlaywrightHTTPClient(client HTTPDoer) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) {
		n.httpClient = client
	}
}

// WithPlaywrightHub sets the WebSocket hub.
func WithPlaywrightHub(hub wsHub.HubInterface) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) {
		n.wsHub = hub
	}
}

// WithPlaywrightCreditService sets the credit service.
func WithPlaywrightCreditService(svc credits.CreditService) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) {
		n.creditService = svc
	}
}

// NewPlaywrightVisionNavigator creates a new playwright-based navigator.
func NewPlaywrightVisionNavigator(log *logrus.Logger, opts ...PlaywrightNavigatorOption) *PlaywrightVisionNavigator {
	driverURL := resolveDriverURL()

	n := &PlaywrightVisionNavigator{
		log:               log,
		driverBaseURL:     driverURL,
		activeNavigations: make(map[string]*NavigationSession),
		httpClient:        &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(n)
	}

	return n
}

// resolveDriverURL gets the playwright-driver URL from environment.
func resolveDriverURL() string {
	url := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL"))
	if url == "" {
		url = "http://127.0.0.1:39400"
	}
	return strings.TrimRight(url, "/")
}

// Type returns the navigator type.
func (n *PlaywrightVisionNavigator) Type() NavigatorType {
	return NavigatorPlaywright
}

// Description returns a human-readable description.
func (n *PlaywrightVisionNavigator) Description() string {
	return "AI navigation using vision models via playwright-driver"
}

// IsAvailable checks if playwright-driver is available.
func (n *PlaywrightVisionNavigator) IsAvailable(ctx context.Context) bool {
	// Check if we can reach the driver health endpoint
	healthURL := n.driverBaseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 400
}

// UnavailableReason returns why the navigator is unavailable.
func (n *PlaywrightVisionNavigator) UnavailableReason(ctx context.Context) string {
	if n.IsAvailable(ctx) {
		return ""
	}
	return "playwright-driver not reachable at " + n.driverBaseURL
}

// CreditPolicy returns the credit policy for this navigator.
func (n *PlaywrightVisionNavigator) CreditPolicy() CreditPolicy {
	return CreditPolicy{
		RequiresCredits:  true,
		OperationType:    credits.OpAIVisionNavigate,
		PerStepCharging:  true,
		CreditsPerStep:   2,
		BypassConditions: []BypassCondition{BypassBYOK, BypassResourceOpenrouter},
	}
}

// ClientSourcePolicy returns the client source policy (all sources allowed).
func (n *PlaywrightVisionNavigator) ClientSourcePolicy() ClientSourcePolicy {
	return AllSourcesPolicy()
}

// Navigate starts an AI navigation session.
func (n *PlaywrightVisionNavigator) Navigate(ctx context.Context, req NavigationRequest) (NavigationHandle, error) {
	// Generate navigation ID
	navigationID := "nav_" + uuid.New().String()[:12]

	// Track the navigation session
	session := &NavigationSession{
		NavigationID:  navigationID,
		SessionID:     req.SessionID,
		UserID:        req.UserID,
		Model:         req.Model,
		StartedAt:     time.Now(),
		Status:        StatusNavigating,
		IsBYOK:        req.APIKey != "",
		NavigatorType: NavigatorPlaywright,
	}

	n.mu.Lock()
	n.activeNavigations[navigationID] = session
	n.mu.Unlock()

	// Resolve API key
	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		n.removeNavigation(navigationID)
		return nil, errors.New("API key required: provide api_key in request or set OPENROUTER_API_KEY environment variable")
	}

	// Set defaults
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}
	if maxSteps > 100 {
		maxSteps = 100
	}

	// Forward to playwright-driver
	driverReq := map[string]interface{}{
		"prompt":       req.Prompt,
		"model":        req.Model,
		"api_key":      apiKey,
		"max_steps":    maxSteps,
		"callback_url": req.CallbackURL,
	}

	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate", n.driverBaseURL, req.SessionID)

	body, err := json.Marshal(driverReq)
	if err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("marshal driver request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(body))
	if err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("driver request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		n.removeNavigation(navigationID)
		var driverErr map[string]interface{}
		if json.Unmarshal(respBody, &driverErr) == nil {
			if msg, ok := driverErr["message"].(string); ok {
				return nil, fmt.Errorf("driver error: %s", msg)
			}
		}
		return nil, fmt.Errorf("driver error: status %d", resp.StatusCode)
	}

	// Parse driver response
	var driverResp struct {
		NavigationID string `json:"navigation_id"`
		Status       string `json:"status"`
		Model        string `json:"model"`
		MaxSteps     int    `json:"max_steps"`
	}
	if err := json.Unmarshal(respBody, &driverResp); err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("parse driver response: %w", err)
	}

	// Update navigation ID from driver if different
	if driverResp.NavigationID != "" && driverResp.NavigationID != navigationID {
		n.mu.Lock()
		delete(n.activeNavigations, navigationID)
		navigationID = driverResp.NavigationID
		session.NavigationID = navigationID
		n.activeNavigations[navigationID] = session
		n.mu.Unlock()
	}

	n.log.WithFields(logrus.Fields{
		"navigation_id": navigationID,
		"session_id":    req.SessionID,
		"model":         req.Model,
		"max_steps":     maxSteps,
		"navigator":     "playwright",
	}).Info("vision_navigation: started")

	return &playwrightNavigationHandle{
		navigator: n,
		session:   session,
	}, nil
}

// HandleStepCallback processes a step callback from playwright-driver.
func (n *PlaywrightVisionNavigator) HandleStepCallback(ctx context.Context, event *NavigationStep) error {
	n.log.WithFields(logrus.Fields{
		"navigation_id":  event.NavigationID,
		"step_number":    event.StepNumber,
		"action_type":    event.Action["type"],
		"goal_achieved":  event.GoalAchieved,
		"awaiting_human": event.AwaitingHuman,
	}).Debug("vision_navigation_callback: received step")

	// Update navigation session
	n.mu.Lock()
	session := n.activeNavigations[event.NavigationID]
	if session != nil {
		session.StepCount = event.StepNumber
		session.TotalTokens += event.TokensUsed.TotalTokens
		session.AwaitingHuman = event.AwaitingHuman
		session.HumanIntervention = event.HumanIntervention
		if event.AwaitingHuman {
			session.Status = StatusAwaitingHuman
		}
	}
	n.mu.Unlock()

	// Charge credits per step
	if n.creditService != nil && session != nil {
		_, err := n.creditService.Charge(ctx, credits.ChargeRequest{
			UserIdentity: session.UserID,
			Operation:    credits.OpAIVisionNavigate,
			Metadata: credits.ChargeMetadata{
				Model:            session.Model,
				PromptTokens:     event.TokensUsed.PromptTokens,
				CompletionTokens: event.TokensUsed.CompletionTokens,
			},
			IsBYOK: session.IsBYOK,
		})
		if err != nil {
			n.log.WithError(err).Warn("vision_navigation_callback: failed to charge credits")
		}
	}

	// Broadcast via WebSocket
	if n.wsHub != nil && session != nil {
		wsEvent := map[string]interface{}{
			"type":         "ai_navigation_step",
			"navigationId": event.NavigationID,
			"sessionId":    session.SessionID,
			"stepNumber":   event.StepNumber,
			"action":       event.Action,
			"reasoning":    event.Reasoning,
			"currentUrl":   event.CurrentURL,
			"goalAchieved": event.GoalAchieved,
			"tokensUsed":   event.TokensUsed,
			"durationMs":   event.DurationMs,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		if event.Error != "" {
			wsEvent["error"] = event.Error
		}

		n.wsHub.BroadcastEnvelope(wsEvent)

		// If awaiting human intervention, send additional event
		if event.AwaitingHuman && event.HumanIntervention != nil {
			humanEvent := map[string]interface{}{
				"type":             "ai_navigation_awaiting_human",
				"navigationId":     event.NavigationID,
				"sessionId":        session.SessionID,
				"stepNumber":       event.StepNumber,
				"reason":           event.HumanIntervention.Reason,
				"interventionType": event.HumanIntervention.InterventionType,
				"trigger":          event.HumanIntervention.Trigger,
				"timestamp":        time.Now().UTC().Format(time.RFC3339),
			}
			if event.HumanIntervention.Instructions != "" {
				humanEvent["instructions"] = event.HumanIntervention.Instructions
			}

			n.wsHub.BroadcastEnvelope(humanEvent)

			n.log.WithFields(logrus.Fields{
				"navigation_id":     event.NavigationID,
				"intervention_type": event.HumanIntervention.InterventionType,
				"trigger":           event.HumanIntervention.Trigger,
			}).Info("vision_navigation_callback: awaiting human intervention")
		}
	}

	return nil
}

// HandleCompleteCallback processes a completion callback from playwright-driver.
func (n *PlaywrightVisionNavigator) HandleCompleteCallback(ctx context.Context, result *NavigationResult) error {
	n.log.WithFields(logrus.Fields{
		"navigation_id": result.NavigationID,
		"status":        result.Status,
		"total_steps":   result.TotalSteps,
		"total_tokens":  result.TotalTokens,
	}).Info("vision_navigation_callback: navigation completed")

	// Update navigation session
	n.mu.Lock()
	session := n.activeNavigations[result.NavigationID]
	if session != nil {
		session.Status = result.Status
		session.StepCount = result.TotalSteps
		session.TotalTokens = result.TotalTokens
	}
	n.mu.Unlock()

	// Broadcast via WebSocket
	if n.wsHub != nil && session != nil {
		wsEvent := map[string]interface{}{
			"type":            "ai_navigation_complete",
			"navigationId":    result.NavigationID,
			"sessionId":       session.SessionID,
			"status":          result.Status,
			"totalSteps":      result.TotalSteps,
			"totalTokens":     result.TotalTokens,
			"totalDurationMs": result.TotalDurationMs,
			"finalUrl":        result.FinalURL,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		}
		if result.Error != "" {
			wsEvent["error"] = result.Error
		}
		if result.Summary != "" {
			wsEvent["summary"] = result.Summary
		}

		n.wsHub.BroadcastEnvelope(wsEvent)
	}

	// Schedule cleanup after a delay
	go func() {
		time.Sleep(5 * time.Minute)
		n.removeNavigation(result.NavigationID)
	}()

	return nil
}

// GetSession returns a navigation session by ID.
func (n *PlaywrightVisionNavigator) GetSession(navigationID string) (*NavigationSession, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	session, exists := n.activeNavigations[navigationID]
	return session, exists
}

// AbortNavigation sends an abort request to playwright-driver.
func (n *PlaywrightVisionNavigator) AbortNavigation(ctx context.Context, navigationID string) error {
	session, exists := n.GetSession(navigationID)
	if !exists {
		return fmt.Errorf("navigation not found: %s", navigationID)
	}

	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate/abort", n.driverBaseURL, session.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		return fmt.Errorf("create abort request: %w", err)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("abort request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update status locally
	n.mu.Lock()
	if s := n.activeNavigations[navigationID]; s != nil {
		s.Status = StatusAborted
	}
	n.mu.Unlock()

	n.log.WithField("navigation_id", navigationID).Info("vision_navigation: abort requested")
	return nil
}

// ResumeNavigation sends a resume request to playwright-driver.
func (n *PlaywrightVisionNavigator) ResumeNavigation(ctx context.Context, navigationID string) error {
	session, exists := n.GetSession(navigationID)
	if !exists {
		return fmt.Errorf("navigation not found: %s", navigationID)
	}

	if !session.AwaitingHuman {
		return errors.New("navigation is not awaiting human intervention")
	}

	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate/resume", n.driverBaseURL, session.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		return fmt.Errorf("create resume request: %w", err)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resume request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resume failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	// Update status locally
	n.mu.Lock()
	if s := n.activeNavigations[navigationID]; s != nil {
		s.Status = StatusNavigating
		s.AwaitingHuman = false
		s.HumanIntervention = nil
	}
	n.mu.Unlock()

	// Broadcast resume event
	if n.wsHub != nil {
		resumeEvent := map[string]interface{}{
			"type":         "ai_navigation_resumed",
			"navigationId": navigationID,
			"sessionId":    session.SessionID,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		n.wsHub.BroadcastEnvelope(resumeEvent)
	}

	n.log.WithField("navigation_id", navigationID).Info("vision_navigation: resumed after human intervention")
	return nil
}

// removeNavigation removes a navigation session from tracking.
func (n *PlaywrightVisionNavigator) removeNavigation(navigationID string) {
	n.mu.Lock()
	delete(n.activeNavigations, navigationID)
	n.mu.Unlock()
}

// playwrightNavigationHandle implements NavigationHandle for playwright navigations.
type playwrightNavigationHandle struct {
	navigator *PlaywrightVisionNavigator
	session   *NavigationSession
}

func (h *playwrightNavigationHandle) ID() string {
	return h.session.NavigationID
}

func (h *playwrightNavigationHandle) SessionID() string {
	return h.session.SessionID
}

func (h *playwrightNavigationHandle) Status() NavigationStatus {
	session, exists := h.navigator.GetSession(h.session.NavigationID)
	if !exists {
		return StatusCompleted // Session cleaned up
	}
	return session.Status
}

func (h *playwrightNavigationHandle) Wait(ctx context.Context) error {
	// Poll for completion
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			session, exists := h.navigator.GetSession(h.session.NavigationID)
			if !exists {
				return nil // Session cleaned up, assume completed
			}
			switch session.Status {
			case StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected:
				return nil
			}
		}
	}
}

func (h *playwrightNavigationHandle) Abort(ctx context.Context) error {
	return h.navigator.AbortNavigation(ctx, h.session.NavigationID)
}

func (h *playwrightNavigationHandle) Resume(ctx context.Context) error {
	return h.navigator.ResumeNavigation(ctx, h.session.NavigationID)
}
