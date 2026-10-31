package vision

import (
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

// ActionRecordCallback is called when an AI navigation action should be recorded.
// This allows the navigator to report actions to the unified recording service.
type ActionRecordCallback func(sessionID string, action *RecordedNavigationAction)

// RecordedNavigationAction contains action details from AI navigation.
type RecordedNavigationAction struct {
	ActionType string
	URL        string
	PageTitle  string
	Selector   string
	Reasoning  string
	StepNumber int
	Timestamp  string
	Source     string // "ai"
}

// PlaywrightVisionNavigator implements VisionNavigator using playwright-driver.
type PlaywrightVisionNavigator struct {
	log           *logrus.Logger
	driverBaseURL string
	wsHub         wsHub.HubInterface
	httpClient    HTTPDoer
	creditService credits.CreditService
	sessions      SessionRouteBroker

	// Recording callback for unified action capture.
	// When set, all AI navigation actions are reported for recording.
	onActionRecord ActionRecordCallback

	// Track active navigations
	mu                sync.RWMutex
	activeNavigations map[string]*NavigationSession
}

// HTTPDoer is an interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type SessionRouteBroker interface {
	RouteSessionRequest(context.Context, string, string, string, []byte) (*http.Response, error)
}

// PlaywrightNavigatorOption configures PlaywrightVisionNavigator.
type PlaywrightNavigatorOption func(*PlaywrightVisionNavigator)

// WithPlaywrightHTTPClient sets a custom HTTP client.
func WithPlaywrightHTTPClient(client HTTPDoer) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) {
		n.httpClient = client
		if broker, ok := client.(SessionRouteBroker); ok {
			n.sessions = broker
		}
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

// WithPlaywrightSessionBroker routes every /session request through the API's
// owned session registry.
func WithPlaywrightSessionBroker(mgr SessionRouteBroker) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) { n.sessions = mgr }
}

func (n *PlaywrightVisionNavigator) SetSessionRouteBroker(mgr SessionRouteBroker) { n.sessions = mgr }

// WithActionRecordCallback sets the callback for recording AI navigation actions.
// This enables unified recording of AI-initiated browser actions.
func WithActionRecordCallback(callback ActionRecordCallback) PlaywrightNavigatorOption {
	return func(n *PlaywrightVisionNavigator) {
		n.onActionRecord = callback
	}
}

// SetActionRecordCallback sets the callback for recording AI navigation actions.
// This is an alternative to WithActionRecordCallback for post-construction configuration.
func (n *PlaywrightVisionNavigator) SetActionRecordCallback(callback ActionRecordCallback) {
	n.onActionRecord = callback
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

func (n *PlaywrightVisionNavigator) routeSession(ctx context.Context, sessionID, method, suffix string, body []byte) (*http.Response, error) {
	if n.sessions == nil {
		return nil, fmt.Errorf("session broker unavailable")
	}
	return n.sessions.RouteSessionRequest(ctx, sessionID, method, suffix, body)
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
		BypassConditions: []BypassCondition{BypassCredentialProvenance, BypassResourceOpenrouter},
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
		NavigationID:         navigationID,
		SessionID:            req.SessionID,
		UserID:               req.UserID,
		Model:                req.Model,
		StartedAt:            time.Now(),
		Status:               StatusNavigating,
		CredentialProvenance: CredentialProvenanceNone,
		NavigatorType:        NavigatorPlaywright,
	}

	n.mu.Lock()
	n.activeNavigations[navigationID] = session
	n.mu.Unlock()

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
		"prompt":         req.Prompt,
		"effect_policy":  req.EffectPolicy,
		"postconditions": req.Postconditions,
		"extraction":     req.Extraction,
		"model":          req.Model,
		"max_steps":      maxSteps,
		"callback_url":   req.CallbackURL,
	}

	body, err := json.Marshal(driverReq)
	if err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("marshal driver request: %w", err)
	}

	resp, err := n.routeSession(ctx, req.SessionID, http.MethodPost, "/ai-navigate", body)
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
	actionType, _ := event.Action["type"].(string)
	var safeAction map[string]interface{}
	safeEvent := *event
	n.log.WithFields(logrus.Fields{
		"navigation_id":  event.NavigationID,
		"step_number":    event.StepNumber,
		"action_type":    actionType,
		"goal_achieved":  event.GoalAchieved,
		"awaiting_human": event.AwaitingHuman,
	}).Debug("vision_navigation_callback: received step")

	// Update navigation session
	n.mu.Lock()
	session := n.activeNavigations[event.NavigationID]
	if session != nil {
		switch session.Status {
		case StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected:
			n.mu.Unlock()
			return nil
		}
		if event.StepNumber <= session.StepCount {
			n.mu.Unlock()
			return nil
		}
		safeAction = redactNavigationAction(event.Action)
		session.StepCount = event.StepNumber
		session.TotalTokens += event.TokensUsed.TotalTokens
		session.AwaitingHuman = event.AwaitingHuman
		if event.HumanIntervention != nil {
			session.HumanIntervention = redactedHumanIntervention(event.HumanIntervention)
		} else {
			session.HumanIntervention = nil
		}
		safeEvent.Action = safeAction
		safeEvent.CurrentURL = redactNavigationURL(event.CurrentURL)
		safeEvent.Reasoning = redactNavigationText(event.Reasoning)
		safeEvent.Error = redactNavigationText(event.Error)
		safeEvent.HumanIntervention = session.HumanIntervention
		session.RecordStep(stepRecordFromEvent(&safeEvent, safeAction))
		if event.AwaitingHuman {
			session.SetStatus(StatusAwaitingHuman)
		}
	}
	n.mu.Unlock()

	// Record action to unified recording service if callback is configured
	if n.onActionRecord != nil && session != nil {
		actionType := ""
		selector := ""
		if t, ok := safeAction["type"].(string); ok {
			actionType = t
		}
		if s, ok := safeAction["selector"].(string); ok {
			selector = s
		}

		recordedAction := &RecordedNavigationAction{
			ActionType: actionType,
			URL:        safeEvent.CurrentURL,
			Selector:   selector,
			Reasoning:  safeEvent.Reasoning,
			StepNumber: event.StepNumber,
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Source:     "ai",
		}

		n.onActionRecord(session.SessionID, recordedAction)
	}

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
			IsBYOK: session.CredentialProvenance == CredentialProvenanceAuthority,
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
			"action":       safeAction,
			"reasoning":    safeEvent.Reasoning,
			"currentUrl":   safeEvent.CurrentURL,
			"goalAchieved": event.GoalAchieved,
			"tokensUsed":   event.TokensUsed,
			"durationMs":   event.DurationMs,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		if event.Error != "" {
			wsEvent["error"] = safeEvent.Error
		}

		n.wsHub.BroadcastEnvelope(wsEvent)

		// If awaiting human intervention, send additional event
		if event.AwaitingHuman && safeEvent.HumanIntervention != nil {
			safeIntervention := safeEvent.HumanIntervention
			humanEvent := map[string]interface{}{
				"type":             "ai_navigation_awaiting_human",
				"navigationId":     event.NavigationID,
				"sessionId":        session.SessionID,
				"stepNumber":       event.StepNumber,
				"reason":           safeIntervention.Reason,
				"interventionType": safeIntervention.InterventionType,
				"trigger":          safeIntervention.Trigger,
				"timestamp":        time.Now().UTC().Format(time.RFC3339),
			}
			if safeIntervention.Instructions != "" {
				humanEvent["instructions"] = safeIntervention.Instructions
			}

			n.wsHub.BroadcastEnvelope(humanEvent)

			n.log.WithFields(logrus.Fields{
				"navigation_id":     event.NavigationID,
				"intervention_type": safeIntervention.InterventionType,
				"trigger":           safeIntervention.Trigger,
			}).Info("vision_navigation_callback: awaiting human intervention")
		}
	}

	return nil
}

// HandleCompleteCallback processes a completion callback from playwright-driver.
func (n *PlaywrightVisionNavigator) HandleCompleteCallback(ctx context.Context, result *NavigationResult) error {
	safeFinalURL := redactNavigationURL(result.FinalURL)
	safeError := redactNavigationText(result.Error)
	safeSummary := redactNavigationText(result.Summary)
	safeVerificationError := redactNavigationText(result.VerificationError)
	safeExtractedData := redactNavigationMap(result.ExtractedData, navigationMapSensitive(result.ExtractedData))
	n.log.WithFields(logrus.Fields{
		"navigation_id": result.NavigationID,
		"status":        result.Status,
		"total_steps":   result.TotalSteps,
		"total_tokens":  result.TotalTokens,
	}).Info("vision_navigation_callback: navigation completed")

	// Update navigation session
	n.mu.Lock()
	session := n.activeNavigations[result.NavigationID]
	if session == nil {
		n.mu.Unlock()
		return nil
	}
	// Awaiting human is a resumable pause, so it is intentionally not treated
	// as a committed completion here. Actual terminal results are immutable once
	// the first completion callback has been applied.
	switch session.Status {
	case StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected:
		n.mu.Unlock()
		return nil
	}
	session.VerifiedSuccess = result.VerifiedSuccess
	session.ExtractedData = safeExtractedData
	session.VerificationError = safeVerificationError
	session.FinalURL = safeFinalURL
	session.Error = safeError
	session.Summary = safeSummary
	session.TotalDurationMs = result.TotalDurationMs
	session.SetStatus(result.Status)
	session.StepCount = result.TotalSteps
	session.TotalTokens = result.TotalTokens
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
			"finalUrl":        safeFinalURL,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		}
		if safeError != "" {
			wsEvent["error"] = safeError
		}
		if safeSummary != "" {
			wsEvent["summary"] = safeSummary
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

// GetSession returns a snapshot of a navigation session by ID. The snapshot
// carries the session's Changed() channel so callers can wait on the next
// status transition without holding the navigator lock.
func (n *PlaywrightVisionNavigator) GetSession(navigationID string) (*NavigationSession, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	session, exists := n.activeNavigations[navigationID]
	if !exists {
		return nil, false
	}
	return session.Snapshot(), true
}

// stepRecordFromEvent maps an already-sanitized driver step callback onto the
// bounded history entry kept on the NavigationSession. Callback admission owns
// redaction so history does not traverse the same payload a second time.
func stepRecordFromEvent(event *NavigationStep, safeAction map[string]interface{}) NavigationStepRecord {
	rec := NavigationStepRecord{
		Index:       event.StepNumber,
		URL:         event.CurrentURL,
		Description: event.Reasoning,
		Success:     event.Error == "",
		Error:       event.Error,
		At:          time.Now(),
	}
	if t, ok := safeAction["type"].(string); ok {
		rec.ActionType = t
	}
	if sel, ok := safeAction["selector"].(string); ok {
		rec.Selector = sel
	}
	for _, key := range []string{"value", "text", "key"} {
		if v, ok := safeAction[key].(string); ok && v != "" {
			rec.Value = v
			break
		}
	}
	if u, ok := safeAction["url"].(string); ok && u != "" && (rec.ActionType == "navigate" || rec.URL == "") {
		rec.URL = u
	}
	return rec
}

// AbortNavigation sends an abort request to playwright-driver.
func (n *PlaywrightVisionNavigator) AbortNavigation(ctx context.Context, navigationID string) error {
	session, exists := n.GetSession(navigationID)
	if !exists {
		return fmt.Errorf("navigation not found: %s", navigationID)
	}

	resp, err := n.routeSession(ctx, session.SessionID, http.MethodPost, "/ai-navigate/abort", nil)
	if err != nil {
		return fmt.Errorf("abort request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update status locally
	n.mu.Lock()
	if s := n.activeNavigations[navigationID]; s != nil {
		s.SetStatus(StatusAborted)
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

	resp, err := n.routeSession(ctx, session.SessionID, http.MethodPost, "/ai-navigate/resume", nil)
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
		s.SetStatus(StatusNavigating)
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
	// Block on the session's status-change broadcast; awaiting_human is not
	// completion from a handle's point of view, so keep waiting through it.
	for {
		session, exists := h.navigator.GetSession(h.session.NavigationID)
		if !exists {
			return nil // Session cleaned up, assume completed
		}
		switch session.Status {
		case StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected:
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-session.Changed():
		}
	}
}

func (h *playwrightNavigationHandle) Abort(ctx context.Context) error {
	return h.navigator.AbortNavigation(ctx, h.session.NavigationID)
}

func (h *playwrightNavigationHandle) Resume(ctx context.Context) error {
	return h.navigator.ResumeNavigation(ctx, h.session.NavigationID)
}
