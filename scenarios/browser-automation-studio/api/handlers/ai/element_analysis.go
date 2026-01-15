package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
	"github.com/vrooli/browser-automation-studio/services/credits"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
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

// AnalyzeElements handles POST /api/v1/analyze-elements
func (h *ElementAnalysisHandler) AnalyzeElements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ElementAnalysisRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode element analysis request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	// Check AI operation permission (tier + credits)
	var userID string
	if h.creditService != nil {
		userID = entitlement.UserIdentityFromContext(ctx)
		if userID == "" {
			userID = "anonymous"
		}

		canProceed, errCode, errMsg, remaining, err := h.creditService.CanPerformAIOperation(ctx, userID, credits.OpAIElementAnalyze, false)
		if err != nil {
			h.log.WithError(err).Warn("element_analysis: failed to check AI operation permission")
		} else if !canProceed {
			status := http.StatusForbidden
			if errCode == "INSUFFICIENT_CREDITS" {
				status = http.StatusPaymentRequired
			}
			RespondError(w, &APIError{
				Status:  status,
				Code:    errCode,
				Message: errMsg,
				Details: map[string]string{"remaining": fmt.Sprintf("%d", remaining)},
			})
			return
		}
	}

	// Normalize URL - add protocol if missing
	url := req.URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	h.log.WithField("url", url).Info("Analyzing page elements")

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second) // Extended timeout for analysis
	defer cancel()

	// Step 1: Extract elements and take screenshot
	elements, pageContext, screenshot, err := h.extractPageElements(ctx, url)
	if err != nil {
		h.log.WithError(err).Error("Failed to extract page elements")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "extract_page_elements", "error": err.Error()}))
		return
	}

	// Step 2: Generate AI suggestions using Ollama
	aiSuggestions, err := h.suggestionGenerator.generateAISuggestions(ctx, elements, pageContext)
	if err != nil {
		h.log.WithError(err).Warn("Failed to generate AI suggestions, continuing without them")
		aiSuggestions = []AISuggestion{} // Continue without AI suggestions
	}

	// Charge credits after successful AI operation (if AI suggestions were generated)
	if h.creditService != nil && userID != "" && len(aiSuggestions) > 0 {
		if _, err := h.creditService.Charge(ctx, credits.ChargeRequest{
			UserIdentity: userID,
			Operation:    credits.OpAIElementAnalyze,
		}); err != nil {
			h.log.WithError(err).Warn("element_analysis: failed to charge credits")
		}
	}

	response := ElementAnalysisResponse{
		Success:       true,
		Elements:      elements,
		AISuggestions: aiSuggestions,
		PageContext:   pageContext,
		Screenshot:    screenshot,
		Timestamp:     time.Now().Unix(),
	}

	respondJSON(w, response)
}

// GetElementAtCoordinate handles POST /api/v1/element-at-coordinate
func (h *ElementAnalysisHandler) GetElementAtCoordinate(w http.ResponseWriter, r *http.Request) {
	var req ElementAtCoordinateRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode element at coordinate request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	// Normalize URL - add protocol if missing
	url := req.URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	h.log.WithFields(logrus.Fields{
		"url": url,
		"x":   req.X,
		"y":   req.Y,
	}).Info("Getting element at coordinate")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	selection, err := h.getElementAtCoordinate(ctx, url, req.X, req.Y)
	if err != nil {
		h.log.WithError(err).Error("Failed to get element at coordinate")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_element_at_coordinate", "error": err.Error()}))
		return
	}

	respondJSON(w, selection)
}

// respondJSON is a helper function to write JSON responses.
func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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
