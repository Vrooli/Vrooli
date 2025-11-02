package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AIAnalysisHandler handles AI-powered element analysis using Ollama
type AIAnalysisHandler struct {
	log        *logrus.Logger
	domHandler *DOMHandler
}

// NewAIAnalysisHandler creates a new AI analysis handler
func NewAIAnalysisHandler(log *logrus.Logger, domHandler *DOMHandler) *AIAnalysisHandler {
	return &AIAnalysisHandler{
		log:        log,
		domHandler: domHandler,
	}
}

// AIAnalyzeElements handles POST /api/v1/ai-analyze-elements
func (h *AIAnalysisHandler) AIAnalyzeElements(w http.ResponseWriter, r *http.Request) {
	var req AIAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	// First, extract the DOM tree from the page
	domData, err := h.domHandler.ExtractDOMTree(ctx, url)
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

	// Call Ollama API directly using curl
	ollamaPayload := map[string]any{
		"model":  "llama3.2:3b", // Use a fast text model
		"prompt": prompt,
		"stream": false,
		// Don't use format: json as it causes double-escaping issues
	}

	payloadBytes, err := json.Marshal(ollamaPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama payload: %w", err)
	}

	// Call Ollama API
	curlCmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST",
		"http://localhost:11434/api/generate",
		"-H", "Content-Type: application/json",
		"-d", string(payloadBytes))

	output, err := curlCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama API: %w, output: %s", err, string(output))
	}

	// Parse Ollama response
	var ollamaResp struct {
		Model    string `json:"model"`
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.Unmarshal(output, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	// Log the raw Ollama response for debugging
	previewLen := 200
	if len(ollamaResp.Response) < previewLen {
		previewLen = len(ollamaResp.Response)
	}
	h.log.WithFields(logrus.Fields{
		"model":            ollamaResp.Model,
		"done":             ollamaResp.Done,
		"response_length":  len(ollamaResp.Response),
		"response_preview": ollamaResp.Response[:previewLen],
	}).Info("Received Ollama response")

	// Parse the JSON response from the model
	var suggestions []ElementInfo
	responseText := ollamaResp.Response

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
				origPreview := ollamaResp.Response
				if len(ollamaResp.Response) > 300 {
					origPreview = ollamaResp.Response[:300]
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
