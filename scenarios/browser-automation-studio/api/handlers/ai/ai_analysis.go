package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

// DOMExtractor defines the interface for extracting DOM trees.
// This abstraction enables testing AI analysis without requiring browser automation.
type DOMExtractor interface {
	ExtractDOMTree(ctx context.Context, url string) (string, error)
}

// ElementAnalyzer owns the domain logic for interpreting DOM snapshots and user intent.
// It is intentionally transport-agnostic so HTTP handlers remain thin.
type ElementAnalyzer interface {
	Analyze(ctx context.Context, url, intent string) ([]ElementInfo, error)
}

// AIAnalysisHandler handles AI-powered element analysis using Ollama.
// It focuses on HTTP concerns and delegates domain logic to an ElementAnalyzer.
type AIAnalysisHandler struct {
	log      *logrus.Logger
	analyzer ElementAnalyzer
	timeout  time.Duration
}

type aiAnalysisConfig struct {
	analyzer     ElementAnalyzer
	domExtractor DOMExtractor
	ollamaClient OllamaClient
	model        string
	timeout      time.Duration
}

// AIAnalysisOption configures the AIAnalysisHandler.
type AIAnalysisOption func(*aiAnalysisConfig)

// WithDOMExtractor sets a custom DOM extractor.
func WithDOMExtractor(extractor DOMExtractor) AIAnalysisOption {
	return func(cfg *aiAnalysisConfig) {
		cfg.domExtractor = extractor
	}
}

// WithAIAnalysisOllamaClient sets a custom Ollama client for AI analysis.
func WithAIAnalysisOllamaClient(client OllamaClient) AIAnalysisOption {
	return func(cfg *aiAnalysisConfig) {
		cfg.ollamaClient = client
	}
}

// WithAIAnalysisModel sets the Ollama model to use for AI analysis.
func WithAIAnalysisModel(model string) AIAnalysisOption {
	return func(cfg *aiAnalysisConfig) {
		cfg.model = model
	}
}

// WithElementAnalyzer injects a fully configured analyzer, bypassing default wiring.
func WithElementAnalyzer(analyzer ElementAnalyzer) AIAnalysisOption {
	return func(cfg *aiAnalysisConfig) {
		cfg.analyzer = analyzer
	}
}

// WithAIAnalysisTimeout overrides the default analysis timeout, primarily for tests.
func WithAIAnalysisTimeout(timeout time.Duration) AIAnalysisOption {
	return func(cfg *aiAnalysisConfig) {
		cfg.timeout = timeout
	}
}

// NewAIAnalysisHandler creates a new AI analysis handler with optional configuration.
func NewAIAnalysisHandler(log *logrus.Logger, domHandler *DOMHandler, opts ...AIAnalysisOption) *AIAnalysisHandler {
	cfg := aiAnalysisConfig{
		model:   "llama3.2:3b",
		timeout: constants.AIAnalysisTimeout,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	analyzer := cfg.analyzer
	if analyzer == nil {
		domExtractor := cfg.domExtractor
		if domExtractor == nil && domHandler != nil {
			domExtractor = domHandler
		}
		ollamaClient := cfg.ollamaClient
		if ollamaClient == nil {
			ollamaClient = NewDefaultOllamaClient(log)
		}

		analyzer = &AIElementAnalyzer{
			log:          log,
			domExtractor: domExtractor,
			ollamaClient: ollamaClient,
			model:        cfg.model,
		}
	}

	return &AIAnalysisHandler{
		log:      log,
		analyzer: analyzer,
		timeout:  cfg.timeout,
	}
}

// AIAnalyzeElements handles POST /api/v1/ai-analyze-elements.
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

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
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

// analyzeElementsWithAI delegates to the injected analyzer.
func (h *AIAnalysisHandler) analyzeElementsWithAI(ctx context.Context, url, intent string) ([]ElementInfo, error) {
	if h.analyzer == nil {
		return nil, fmt.Errorf("element analyzer not configured")
	}
	return h.analyzer.Analyze(ctx, url, intent)
}

// AIElementAnalyzer owns the domain logic for DOM extraction and Ollama prompting.
type AIElementAnalyzer struct {
	log          *logrus.Logger
	domExtractor DOMExtractor
	ollamaClient OllamaClient
	model        string
}

// Analyze extracts the DOM for the given URL and asks the Ollama model for element suggestions.
func (a *AIElementAnalyzer) Analyze(ctx context.Context, url, intent string) ([]ElementInfo, error) {
	if a.domExtractor == nil {
		return nil, fmt.Errorf("DOM extractor not configured")
	}

	// First, extract the DOM tree from the page
	domData, err := a.domExtractor.ExtractDOMTree(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract DOM tree: %w", err)
	}

	// Log DOM extraction results
	domPreview := domData
	if len(domData) > 300 {
		domPreview = domData[:300]
	}
	if a.log != nil {
		a.log.WithFields(logrus.Fields{
			"url":         url,
			"dom_length":  len(domData),
			"dom_preview": domPreview,
		}).Info("Extracted DOM tree")
	}

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
	responseText, err := a.ollamaClient.Query(ctx, a.model, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama API: %w", err)
	}

	// Log the raw Ollama response for debugging
	previewLen := 200
	if len(responseText) < previewLen {
		previewLen = len(responseText)
	}
	if a.log != nil {
		a.log.WithFields(logrus.Fields{
			"model":            a.model,
			"response_length":  len(responseText),
			"response_preview": responseText[:previewLen],
		}).Info("Received Ollama response")
	}

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
				if a.log != nil {
					a.log.WithError(err).WithFields(logrus.Fields{
						"original_response": origPreview,
						"cleaned_json":      cleanedPreview,
					}).Error("Failed to parse AI response as JSON after cleaning")
				}

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
			if a.log != nil {
				a.log.WithField("response", responseText).Error("No valid JSON found in AI response")
			}
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
