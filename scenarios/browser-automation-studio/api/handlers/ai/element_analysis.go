package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

// ElementAnalysisHandler handles element analysis and coordinate-based operations
type ElementAnalysisHandler struct {
	log *logrus.Logger
}

// NewElementAnalysisHandler creates a new element analysis handler
func NewElementAnalysisHandler(log *logrus.Logger) *ElementAnalysisHandler {
	return &ElementAnalysisHandler{log: log}
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
	aiSuggestions, err := h.generateAISuggestions(ctx, elements, pageContext)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(selection)
}

// getElementAtCoordinate uses browserless to get element candidates at specific coordinates
func (h *ElementAnalysisHandler) getElementAtCoordinate(ctx context.Context, url string, x, y int) (*ElementSelectionResult, error) {
	browserlessURL := resolveBrowserlessURL()
	if browserlessURL == "" {
		return nil, fmt.Errorf("browserless URL not configured")
	}

	httpClient := &http.Client{Timeout: constants.AIRequestTimeout}
	session := runtime.NewSession(browserlessURL, httpClient, h.log)

	navigateInstruction := runtime.Instruction{
		Index:  0,
		NodeID: "probe.navigate",
		Type:   "navigate",
		Params: runtime.InstructionParam{
			URL:       url,
			WaitUntil: "networkidle2",
			TimeoutMs: 45000,
		},
	}

	navResponse, err := session.ExecuteInstruction(ctx, navigateInstruction)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to URL: %w", err)
	}

	if navResponse == nil || len(navResponse.Steps) == 0 || !navResponse.Steps[0].Success {
		errMsg := "navigation failed"
		if navResponse != nil && len(navResponse.Steps) > 0 {
			if trimmed := strings.TrimSpace(navResponse.Steps[0].Error); trimmed != "" {
				errMsg = trimmed
			}
		}
		return nil, errors.New(errMsg)
	}

	probeInstruction := runtime.Instruction{
		Index:  1,
		NodeID: "probe.element",
		Type:   "probeElements",
		Params: runtime.InstructionParam{
			ProbeX:       x,
			ProbeY:       y,
			ProbeRadius:  8,
			ProbeSamples: 36,
		},
	}

	probeResponse, err := session.ExecuteInstruction(ctx, probeInstruction)
	if err != nil {
		return nil, fmt.Errorf("element probe failed: %w", err)
	}

	if probeResponse == nil || len(probeResponse.Steps) == 0 {
		return nil, fmt.Errorf("element probe returned no results")
	}

	probeStep := probeResponse.Steps[0]
	if !probeStep.Success {
		if trimmed := strings.TrimSpace(probeStep.Error); trimmed != "" {
			return nil, errors.New(trimmed)
		}
		return nil, fmt.Errorf("element probe unsuccessful")
	}

	if probeStep.ProbeResult == nil {
		return nil, fmt.Errorf("element probe did not return data")
	}

	data, err := json.Marshal(probeStep.ProbeResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal probe result: %w", err)
	}

	var selection ElementSelectionResult
	if err := json.Unmarshal(data, &selection); err != nil {
		return nil, fmt.Errorf("failed to decode probe result: %w", err)
	}

	if len(selection.Candidates) == 0 {
		return nil, fmt.Errorf("no qualifying elements found at coordinates")
	}

	if selection.SelectedIndex < 0 || selection.SelectedIndex >= len(selection.Candidates) {
		selection.SelectedIndex = 0
	}

	if selection.Element == nil {
		selection.Element = selection.Candidates[selection.SelectedIndex].Element
	}

	return &selection, nil
}

// extractPageElements extracts all interactive elements from a page
func (h *ElementAnalysisHandler) extractPageElements(ctx context.Context, url string) ([]ElementInfo, PageContext, string, error) {
	// Create temporary files
	tmpElementsFile, err := os.CreateTemp("", "elements-*.json")
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to create temp elements file: %w", err)
	}
	defer os.Remove(tmpElementsFile.Name())
	defer tmpElementsFile.Close()

	tmpScreenshotFile, err := os.CreateTemp("", "analysis-screenshot-*.png")
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to create temp screenshot file: %w", err)
	}
	defer os.Remove(tmpScreenshotFile.Name())
	defer tmpScreenshotFile.Close()

	// Use browserless to extract elements with a custom script
	elementScript := `
// Find all interactive elements
const interactiveElements = Array.from(document.querySelectorAll('a, button, input, select, textarea, [role="button"], [onclick], [tabindex]'));

// Helper function to calculate element confidence based on visibility and interactivity
function calculateConfidence(element) {
  let confidence = 0.1; // Base confidence

  // Visible elements get higher confidence
  const rect = element.getBoundingClientRect();
  if (rect.width > 0 && rect.height > 0) confidence += 0.3;

  // Elements with text content get higher confidence
  if (element.textContent && element.textContent.trim()) confidence += 0.2;

  // Common interactive elements get higher confidence
  if (['button', 'input', 'select', 'textarea'].includes(element.tagName.toLowerCase())) confidence += 0.3;
  if (element.tagName.toLowerCase() === 'a' && element.href) confidence += 0.2;

  // Elements with specific roles get higher confidence
  if (element.getAttribute('role') === 'button') confidence += 0.2;

  return Math.min(confidence, 1.0);
}

// Helper function to generate robust selectors
function generateSelectors(element) {
  const selectors = [];

  // ID selector (highest priority)
  if (element.id) {
    selectors.push({
      selector: '#' + element.id,
      type: 'id',
      robustness: 0.9,
      fallback: false
    });
  }

  // Data attribute selectors
  for (const attr of element.attributes) {
    if (attr.name.startsWith('data-')) {
      selectors.push({
        selector: '[' + attr.name + '="' + attr.value + '"]',
        type: 'data-attr',
        robustness: 0.8,
        fallback: false
      });
    }
  }

  // Class selectors (for semantic classes)
  if (element.className) {
    const classes = element.className.split(/\s+/);
    const semanticClasses = classes.filter(cls =>
      /^(btn|button|link|nav|menu|form|input|submit|login|search)/.test(cls)
    );
    if (semanticClasses.length > 0) {
      selectors.push({
        selector: '.' + semanticClasses[0],
        type: 'class',
        robustness: 0.6,
        fallback: false
      });
    }
  }

  // CSS selector based on tag and attributes
  let cssSelector = element.tagName.toLowerCase();
  if (element.type) cssSelector += '[type="' + element.type + '"]';
  if (element.name) cssSelector += '[name="' + element.name + '"]';

  selectors.push({
    selector: cssSelector,
    type: 'css',
    robustness: 0.4,
    fallback: true
  });

  // XPath as fallback
  function getXPath(element) {
    if (element.id) return '//*[@id="' + element.id + '"]';

    let path = '';
    while (element && element.nodeType === Node.ELEMENT_NODE) {
      let index = 0;
      for (let sibling = element.previousSibling; sibling; sibling = sibling.previousSibling) {
        if (sibling.nodeType === Node.ELEMENT_NODE && sibling.tagName === element.tagName) index++;
      }
      const tagName = element.tagName.toLowerCase();
      path = '/' + tagName + '[' + (index + 1) + ']' + path;
      element = element.parentNode;
    }
    return path;
  }

  selectors.push({
    selector: getXPath(element),
    type: 'xpath',
    robustness: 0.3,
    fallback: true
  });

  return selectors.sort((a, b) => b.robustness - a.robustness);
}

// Helper function to categorize elements
function categorizeElement(element) {
  const text = element.textContent?.toLowerCase() || '';
  const type = element.type?.toLowerCase() || '';
  const tagName = element.tagName.toLowerCase();

  if (type === 'password' || text.includes('password') || text.includes('login') || text.includes('sign in')) {
    return 'authentication';
  }
  if (type === 'search' || text.includes('search') || element.name?.includes('search')) {
    return 'data-entry';
  }
  if (tagName === 'a' || text.includes('menu') || text.includes('nav')) {
    return 'navigation';
  }
  if (type === 'submit' || text.includes('submit') || text.includes('save') || text.includes('send')) {
    return 'actions';
  }
  if (tagName === 'input' || tagName === 'textarea' || tagName === 'select') {
    return 'data-entry';
  }

  return 'general';
}

// Extract element information
const elements = interactiveElements.map(element => {
  const rect = element.getBoundingClientRect();
  const text = element.textContent?.trim() || element.value || element.placeholder || '';

  return {
    text: text.substring(0, 100), // Limit text length
    tagName: element.tagName,
    type: element.type || element.tagName.toLowerCase(),
    selectors: generateSelectors(element),
    boundingBox: {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height
    },
    confidence: calculateConfidence(element),
    category: categorizeElement(element),
    attributes: {
      id: element.id || '',
      className: element.className || '',
      name: element.name || '',
      placeholder: element.placeholder || '',
      'aria-label': element.getAttribute('aria-label') || '',
      title: element.title || ''
    }
  };
}).filter(el => el.boundingBox.width > 0 && el.boundingBox.height > 0); // Only visible elements

// Extract page context
const pageContext = {
  title: document.title,
  url: window.location.href,
  hasLogin: document.querySelector('input[type="password"]') !== null ||
           Array.from(document.querySelectorAll('button, a')).some(el =>
             el.textContent.toLowerCase().includes('login') || el.textContent.toLowerCase().includes('sign in')),
  hasSearch: document.querySelector('input[type="search"], input[name*="search"], input[placeholder*="search"]') !== null,
  formCount: document.querySelectorAll('form').length,
  buttonCount: document.querySelectorAll('button, input[type="button"], input[type="submit"]').length,
  linkCount: document.querySelectorAll('a[href]').length,
  viewport: {
    width: window.innerWidth,
    height: window.innerHeight
  }
};

return {
  elements: elements,
  pageContext: pageContext
};`

	elementsCmd := exec.CommandContext(ctx, "resource-browserless", "extract", url,
		"--script", elementScript,
		"--output", tmpElementsFile.Name())

	output, err := elementsCmd.CombinedOutput()
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to extract elements: %w, output: %s", err, string(output))
	}

	// Take screenshot separately
	screenshotCmd := exec.CommandContext(ctx, "resource-browserless", "screenshot",
		"--url", url,
		"--output", tmpScreenshotFile.Name())

	screenshotOutput, err := screenshotCmd.CombinedOutput()
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to take screenshot: %w, output: %s", err, string(screenshotOutput))
	}

	// Read and parse elements data
	elementsData, err := os.ReadFile(tmpElementsFile.Name())
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to read elements file: %w", err)
	}

	var result struct {
		Elements    []ElementInfo `json:"elements"`
		PageContext PageContext   `json:"pageContext"`
	}

	if err := json.Unmarshal(elementsData, &result); err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to parse elements JSON: %w", err)
	}

	// Read screenshot
	screenshotData, err := os.ReadFile(tmpScreenshotFile.Name())
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("failed to read screenshot: %w", err)
	}

	// Encode screenshot to base64
	base64Screenshot := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(screenshotData))

	return result.Elements, result.PageContext, base64Screenshot, nil
}

// generateAISuggestions uses Ollama to generate intelligent automation suggestions
func (h *ElementAnalysisHandler) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	// Build prompt for Ollama
	prompt := h.buildElementAnalysisPrompt(elements, pageContext)

	// Create temporary file for Ollama output
	tmpOllamaFile, err := os.CreateTemp("", "ollama-suggestions-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp ollama file: %w", err)
	}
	defer os.Remove(tmpOllamaFile.Name())
	defer tmpOllamaFile.Close()

	// Call resource-ollama using the query command
	ollamaCmd := exec.CommandContext(ctx, "resource-ollama", "query",
		"--model", "llama3.2",
		"--prompt", prompt)

	output, err := ollamaCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w, output: %s", err, string(output))
	}

	// Write ollama output to temp file for parsing
	if err := os.WriteFile(tmpOllamaFile.Name(), output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write ollama output: %w", err)
	}

	// Read and parse Ollama response
	ollamaData, err := os.ReadFile(tmpOllamaFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read ollama output: %w", err)
	}

	var ollamaResponse struct {
		Suggestions []AISuggestion `json:"suggestions"`
	}

	if err := json.Unmarshal(ollamaData, &ollamaResponse); err != nil {
		// If JSON parsing fails, try to extract from the raw response
		h.log.WithError(err).Warn("Failed to parse Ollama JSON response, trying fallback parsing")
		return h.parseOllamaFallback(string(ollamaData))
	}

	return ollamaResponse.Suggestions, nil
}

// buildElementAnalysisPrompt creates a structured prompt for element analysis
func (h *ElementAnalysisHandler) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
	// Build elements summary
	elementsJSON, _ := json.MarshalIndent(elements, "  ", "  ")
	contextJSON, _ := json.MarshalIndent(pageContext, "  ", "  ")

	return fmt.Sprintf(`Analyze this webpage and suggest the most likely automation actions a user would want to perform.

Page Information:
%s

Available Interactive Elements:
%s

Provide suggestions in this exact JSON format:
{
  "suggestions": [
    {
      "action": "Login to account",
      "description": "Click the login button to authenticate user",
      "elementText": "Login",
      "selector": "#login-btn",
      "confidence": 0.95,
      "category": "authentication",
      "reasoning": "Page has password input and login button, indicating authentication workflow"
    }
  ]
}

Categories to use:
- "authentication": Login, logout, register, password reset
- "navigation": Menu items, page links, tabs, breadcrumbs
- "data-entry": Forms, search, input fields, text areas
- "actions": Submit, save, delete, edit, download buttons
- "content": Read more, expand, filter, sort options

Focus on:
1. Common user workflows and practical automation scenarios
2. Element text and semantic meaning over position
3. Rank by likelihood of user intent (0.0 to 1.0 confidence)
4. Provide clear reasoning for each suggestion
5. Prefer robust selectors (ID > data attributes > semantic classes)

Return only valid JSON without additional text.`, string(contextJSON), string(elementsJSON))
}

// parseOllamaFallback attempts to extract suggestions from malformed Ollama response
func (h *ElementAnalysisHandler) parseOllamaFallback(response string) ([]AISuggestion, error) {
	h.log.WithField("response", response).Debug("Attempting fallback parsing of Ollama response")

	// For now, return empty suggestions if we can't parse
	// In a real implementation, you might try regex extraction or other parsing strategies
	return []AISuggestion{}, nil
}
