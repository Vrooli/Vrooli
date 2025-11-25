package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

// ElementAnalysisHandler handles element analysis and coordinate-based operations.
// It coordinates between element extraction, AI suggestions, and coordinate probing.
type ElementAnalysisHandler struct {
	log                 *logrus.Logger
	runner              *automationRunner
	suggestionGenerator *ollamaSuggestionGenerator
}

// NewElementAnalysisHandler creates a new element analysis handler.
func NewElementAnalysisHandler(log *logrus.Logger) *ElementAnalysisHandler {
	runner, err := newAutomationRunner(log)
	if err != nil && log != nil {
		log.WithError(err).Warn("Failed to initialize automation runner for element analysis; requests will fail")
	}
	return &ElementAnalysisHandler{
		log:                 log,
		runner:              runner,
		suggestionGenerator: newOllamaSuggestionGenerator(log),
	}
}

// AnalyzeElements handles POST /api/v1/analyze-elements
func (h *ElementAnalysisHandler) AnalyzeElements(w http.ResponseWriter, r *http.Request) {
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

	// Normalize URL - add protocol if missing
	url := req.URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	h.log.WithField("url", url).Info("Analyzing page elements")

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second) // Extended timeout for analysis
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
