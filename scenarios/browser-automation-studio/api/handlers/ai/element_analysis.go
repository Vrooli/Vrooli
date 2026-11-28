package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	modelai "github.com/vrooli/browser-automation-studio/services/ai"
)

// ElementAnalysisHandler handles element analysis and coordinate-based operations.
// It coordinates between element extraction, AI suggestions, and coordinate probing.
type ElementAnalysisHandler struct {
	log                 *logrus.Logger
	runner              AutomationRunner
	suggestionGenerator *aiSuggestionGenerator
	modelClient         modelai.RolePromptClient
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
func WithSuggestionGenerator(gen *aiSuggestionGenerator) ElementAnalysisOption {
	return func(h *ElementAnalysisHandler) {
		h.suggestionGenerator = gen
	}
}

// WithAIModelClient sets the shared model service used for suggestions.
func WithAIModelClient(client modelai.RolePromptClient) ElementAnalysisOption {
	return func(h *ElementAnalysisHandler) {
		if client != nil {
			h.modelClient = client
		}
	}
}

// NewElementAnalysisHandler creates a new element analysis handler with optional configuration.
func NewElementAnalysisHandler(log *logrus.Logger, opts ...ElementAnalysisOption) *ElementAnalysisHandler {
	handler := &ElementAnalysisHandler{log: log}

	// Apply options first
	for _, opt := range opts {
		opt(handler)
	}

	// Create default suggestion generator if not provided
	if handler.modelClient == nil {
		handler.modelClient = newGatewayRolePromptClient()
	}
	if handler.suggestionGenerator == nil {
		handler.suggestionGenerator = newAISuggestionGenerator(log, WithAISuggestionModelClient(handler.modelClient))
	}

	return handler
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
func (h *ElementAnalysisHandler) RunAnalyzeElements(ctx context.Context, url string) (*AnalyzeElementsResult, error) {
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
		return nil, fmt.Errorf("generate AI suggestions through AI Gateway: %w", err)
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

// Test helpers - these delegate to the internal components for testing

// generateAISuggestions is a test helper that delegates to the suggestion generator.
func (h *ElementAnalysisHandler) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	return h.suggestionGenerator.generateAISuggestions(ctx, elements, pageContext)
}

// buildElementAnalysisPrompt is a test helper that delegates to the suggestion generator.
func (h *ElementAnalysisHandler) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
	return h.suggestionGenerator.buildElementAnalysisPrompt(elements, pageContext)
}
