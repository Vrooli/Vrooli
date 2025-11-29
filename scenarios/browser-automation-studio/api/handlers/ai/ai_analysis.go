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
)

// DOMExtractor defines the interface for extracting DOM trees.
// This abstraction enables testing AI analysis without requiring browser automation.
type DOMExtractor interface {
	ExtractDOMTree(ctx context.Context, url string) (string, error)
}

// AIAnalysisHandler handles AI-powered element analysis using Ollama
type AIAnalysisHandler struct {
	log          *logrus.Logger
	domExtractor DOMExtractor
	ollamaClient OllamaClient
	model        string
}

// AIAnalysisOption configures the AIAnalysisHandler.
type AIAnalysisOption func(*AIAnalysisHandler)

// WithDOMExtractor sets a custom DOM extractor.
func WithDOMExtractor(extractor DOMExtractor) AIAnalysisOption {
	return func(h *AIAnalysisHandler) {
		h.domExtractor = extractor
	}
}

// WithAIAnalysisOllamaClient sets a custom Ollama client for AI analysis.
func WithAIAnalysisOllamaClient(client OllamaClient) AIAnalysisOption {
	return func(h *AIAnalysisHandler) {
		h.ollamaClient = client
	}
}

// WithAIAnalysisModel sets the Ollama model to use for AI analysis.
func WithAIAnalysisModel(model string) AIAnalysisOption {
	return func(h *AIAnalysisHandler) {
		h.model = model
	}
}

// NewAIAnalysisHandler creates a new AI analysis handler with optional configuration.
func NewAIAnalysisHandler(log *logrus.Logger, domHandler *DOMHandler, opts ...AIAnalysisOption) *AIAnalysisHandler {
	handler := &AIAnalysisHandler{
		log:   log,
		model: "llama3.2:3b",
	}

	// Apply options first
	for _, opt := range opts {
		opt(handler)
	}

	// Use domHandler as the extractor if not provided via options
	if handler.domExtractor == nil && domHandler != nil {
		handler.domExtractor = domHandler
	}

	// Create default Ollama client if not provided
	if handler.ollamaClient == nil {
		handler.ollamaClient = NewDefaultOllamaClient(log)
	}

	return handler
}

// AIAnalyzeElements handles POST /api/v1/ai-analyze-elements
func (h *AIAnalysisHandler) AIAnalyzeElements(w http.ResponseWriter, r *http.Request) {
	var req AIAnalyzeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode AI analyze request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" || req.Intent == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"fields": "url, intent"}))
		return
	}

	h.log.WithFields(logrus.Fields{
		"url":    req.URL,
		"intent": req.Intent,
	}).Info("AI analyzing elements")

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	suggestions, err := h.analyzeElementsWithAI(ctx, req.URL, req.Intent)
	if err != nil {
		h.log.WithError(err).Error("Failed to analyze elements with AI")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "ai_analyze_elements", "error": err.Error()}))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// analyzeElementsWithAI uses Ollama to analyze the DOM and suggest elements
func (h *AIAnalysisHandler) analyzeElementsWithAI(ctx context.Context, url, intent string) ([]ElementInfo, error) {
	if h.domExtractor == nil {
		return nil, fmt.Errorf("DOM extractor not configured")
	}

	// First, extract the DOM tree from the page
	domData, err := h.domExtractor.ExtractDOMTree(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract DOM tree: %w", err)
	}

	// Log DOM extraction results
	domPreview := domData
	if len(domData) > 300 {
		domPreview = domData[:300]
	}
	h.log.WithFields(logrus.Fields{
		"url":         url,
		"dom_length":  len(domData),
		"dom_preview": domPreview,
	}).Info("Extracted DOM tree")

	// Create a prompt for Ollama to analyze the DOM and suggest elements
	prompt := fmt.Sprintf(`You are an expert web automation assistant. Analyze this DOM tree and help identify the best elements to interact with based on the user's intent.

URL: %s
User Intent: %s

DOM Tree:
%s

Based on the DOM structure, suggest the most relevant elements for the user's intent. For each suggestion, provide:
1. The exact text content of the element
2. The tag name (e.g., "BUTTON", "INPUT", "A")
3. A confidence score (0.0 to 1.0)
4. The best CSS selector from the DOM data
5. The element category (e.g., "actions", "navigation", "data-entry")

Return ONLY a JSON array with no additional text. Focus on elements that match the user's intent. If the user wants a "search button", prioritize buttons with search-related text or aria-labels.

Example format:
[
  {
    "text": "Search",
    "tagName": "BUTTON",
    "type": "button",
    "confidence": 0.9,
    "category": "actions",
    "selectors": [
      {
        "selector": "button[type='submit']",
        "type": "css",
        "robustness": 0.8,
        "fallback": false
      }
    ],
    "boundingBox": {"x": 0, "y": 0, "width": 0, "height": 0},
    "attributes": {}
  }
]`, url, intent, domData)

	// Query Ollama via the client interface
	responseText, err := h.ollamaClient.Query(ctx, h.model, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama API: %w", err)
	}

	// Log the raw Ollama response for debugging
	previewLen := 200
	if len(responseText) < previewLen {
		previewLen = len(responseText)
	}
	h.log.WithFields(logrus.Fields{
		"model":            h.model,
		"response_length":  len(responseText),
		"response_preview": responseText[:previewLen],
	}).Info("Received Ollama response")

	// Parse the JSON response from the model
	var suggestions []ElementInfo

	// First try direct parsing
	if err := json.Unmarshal([]byte(responseText), &suggestions); err != nil {
		// Clean up common issues in the response
		// Remove any escape sequences
		responseText = strings.ReplaceAll(responseText, "\\r", "")
		responseText = strings.ReplaceAll(responseText, "\\n", "\n")
		responseText = strings.ReplaceAll(responseText, "\\\"", "\"")
		responseText = strings.ReplaceAll(responseText, "\\\\", "\\")

		// Try to find and extract just the JSON array
		startIdx := strings.Index(responseText, "[")
		endIdx := strings.LastIndex(responseText, "]")

		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			jsonStr := responseText[startIdx : endIdx+1]

			// Clean up the extracted JSON
			jsonStr = strings.TrimSpace(jsonStr)

			// Try parsing the cleaned JSON
			if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
				origPreview := responseText
				if len(responseText) > 300 {
					origPreview = responseText[:300]
				}
				cleanedPreview := jsonStr
				if len(jsonStr) > 300 {
					cleanedPreview = jsonStr[:300]
				}
				h.log.WithError(err).WithFields(logrus.Fields{
					"original_response": origPreview,
					"cleaned_json":      cleanedPreview,
				}).Error("Failed to parse AI response as JSON after cleaning")

				// Return a single fallback suggestion
				return []ElementInfo{{
					Text:       "Search",
					TagName:    "BUTTON",
					Type:       "button",
					Confidence: 0.5,
					Category:   "actions",
					Selectors: []SelectorOption{{
						Selector:   "button",
						Type:       "css",
						Robustness: 0.3,
						Fallback:   true,
					}},
					BoundingBox: Rectangle{X: 0, Y: 0, Width: 0, Height: 0},
					Attributes:  map[string]string{"note": "AI response parsing failed, using fallback"},
				}}, nil
			}
		} else {
			h.log.WithField("response", responseText).Error("No valid JSON found in AI response")
			// Return debug info about what we received
			respPreview := responseText
			if len(responseText) > 100 {
				respPreview = responseText[:100]
			}
			return []ElementInfo{{
				Text:       "No JSON found in response",
				TagName:    "DEBUG",
				Type:       "debug",
				Confidence: 0.1,
				Category:   "error",
				Selectors: []SelectorOption{{
					Selector:   "body",
					Type:       "css",
					Robustness: 0.1,
					Fallback:   true,
				}},
				BoundingBox: Rectangle{X: 0, Y: 0, Width: 0, Height: 0},
				Attributes:  map[string]string{"response": respPreview},
			}}, nil
		}
	}

	return suggestions, nil
}
