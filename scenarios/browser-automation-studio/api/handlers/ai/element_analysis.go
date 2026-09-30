package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
)

// ElementAnalysisHandler handles element analysis and coordinate-based operations.
// It coordinates between element extraction, AI suggestions, and coordinate probing.
type ElementAnalysisHandler struct {
	log                 *logrus.Logger
	runner              AutomationRunner
	suggestionGenerator *ollamaSuggestionGenerator
	creditService       credits.CreditService
}

// ElementAnalysisOption configures the ElementAnalysisHandler.
type ElementAnalysisOption func(*ElementAnalysisHandler)

// WithElementRunner sets a custom automation runner.
func WithElementRunner(runner AutomationRunner) ElementAnalysisOption {
	return func(h *ElementAnalysisHandler) {
		h.runner = runner
	}
}

// WithSuggestionGenerator sets a custom suggestion generator.
func WithSuggestionGenerator(gen *ollamaSuggestionGenerator) ElementAnalysisOption {
	return func(h *ElementAnalysisHandler) {
		h.suggestionGenerator = gen
	}
}

// WithElementAnalysisCreditService sets the credit service for AI usage tracking.
func WithElementAnalysisCreditService(svc credits.CreditService) ElementAnalysisOption {
	return func(h *ElementAnalysisHandler) {
		h.creditService = svc
	}
}

// NewElementAnalysisHandler creates a new element analysis handler with optional configuration.
func NewElementAnalysisHandler(log *logrus.Logger, opts ...ElementAnalysisOption) *ElementAnalysisHandler {
	handler := &ElementAnalysisHandler{log: log}

	// Apply options first
	for _, opt := range opts {
		opt(handler)
	}

	// Create default runner if not provided
	if handler.runner == nil {
		runner, err := newAutomationRunner(log)
		if err != nil && log != nil {
			log.WithError(err).Warn("Failed to initialize automation runner for element analysis; requests will fail")
		}
		handler.runner = runner
	}

	// Create default suggestion generator if not provided
	if handler.suggestionGenerator == nil {
		handler.suggestionGenerator = newOllamaSuggestionGenerator(log)
	}

	return handler
}

// CreditCheckError signals that the user is not entitled to run the
// requested AI operation (tier/credit gate). Transports map this onto
// PermissionDenied / ResourceExhausted.
type CreditCheckError struct {
	Code      string
	Message   string
	Remaining int
}

func (e *CreditCheckError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// IsInsufficientCredits returns true when the gate failed for the
// out-of-credits reason rather than a tier reason.
func (e *CreditCheckError) IsInsufficientCredits() bool {
	return e != nil && e.Code == "INSUFFICIENT_CREDITS"
}

// AnalyzeElementsResult is the Go-typed return shape of
// ElementAnalysisHandler.RunAnalyzeElements.
type AnalyzeElementsResult struct {
	Elements      []ElementInfo
	AISuggestions []AISuggestion
	PageContext   PageContext
	// Screenshot is a base64 data URL (legacy UI consumes it as <img src>).
	Screenshot string
	CapturedAt time.Time
}

// RunAnalyzeElements runs the page-elements-and-context extraction script
// (with AI suggestion overlay) and returns a Go-typed result. The Connect
// handler is a thin adapter onto this method.
//
// `userID` is used only for credit accounting; "" means anonymous. `hasBYOK`
// signals the caller supplied an OpenRouter key (free of charge).
func (h *ElementAnalysisHandler) RunAnalyzeElements(ctx context.Context, url, userID string, hasBYOK bool) (*AnalyzeElementsResult, error) {
	if strings.TrimSpace(url) == "" {
		return nil, ErrMissingURL
	}
	if h.runner == nil {
		return nil, ErrAutomationRunnerNotReady
	}

	if h.creditService != nil {
		if userID == "" {
			userID = "anonymous"
		}
		canProceed, errCode, errMsg, remaining, err := h.creditService.CanPerformAIOperation(ctx, userID, credits.OpAIElementAnalyze, hasBYOK)
		if err != nil && h.log != nil {
			h.log.WithError(err).Warn("element_analysis: failed to check AI operation permission")
		} else if !canProceed {
			return nil, &CreditCheckError{Code: errCode, Message: errMsg, Remaining: remaining}
		}
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if h.log != nil {
		h.log.WithField("url", url).Info("Analyzing page elements")
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	elements, pageContext, screenshot, err := h.extractPageElements(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("extract page elements: %w", err)
	}

	aiSuggestions, err := h.suggestionGenerator.generateAISuggestions(ctx, elements, pageContext)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Warn("Failed to generate AI suggestions, continuing without them")
		}
		aiSuggestions = []AISuggestion{}
	}

	if h.creditService != nil && userID != "" && len(aiSuggestions) > 0 {
		if _, err := h.creditService.Charge(ctx, credits.ChargeRequest{
			UserIdentity: userID,
			Operation:    credits.OpAIElementAnalyze,
			IsBYOK:       hasBYOK,
		}); err != nil && h.log != nil {
			h.log.WithError(err).Warn("element_analysis: failed to charge credits")
		}
	}

	return &AnalyzeElementsResult{
		Elements:      elements,
		AISuggestions: aiSuggestions,
		PageContext:   pageContext,
		Screenshot:    screenshot,
		CapturedAt:    time.Now(),
	}, nil
}

// RunGetElementAtCoordinate probes the DOM at (x, y) and returns the
// resolved selection. Public wrapper over the unexported coordinate-probe
// helper; used by the AIService Connect handler.
func (h *ElementAnalysisHandler) RunGetElementAtCoordinate(ctx context.Context, url string, x, y int) (*ElementSelectionResult, error) {
	if strings.TrimSpace(url) == "" {
		return nil, ErrMissingURL
	}
	if h.runner == nil {
		return nil, ErrAutomationRunnerNotReady
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if h.log != nil {
		h.log.WithFields(logrus.Fields{
			"url": url, "x": x, "y": y,
		}).Info("Getting element at coordinate")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return h.getElementAtCoordinate(ctx, url, x, y)
}

// Compile-time check that the package's sentinel errors are wired up.
var _ = errors.New

// Test helpers - these delegate to the internal components for testing

// generateAISuggestions is a test helper that delegates to the suggestion generator.
func (h *ElementAnalysisHandler) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	return h.suggestionGenerator.generateAISuggestions(ctx, elements, pageContext)
}

// buildElementAnalysisPrompt is a test helper that delegates to the suggestion generator.
func (h *ElementAnalysisHandler) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
	return h.suggestionGenerator.buildElementAnalysisPrompt(elements, pageContext)
}
