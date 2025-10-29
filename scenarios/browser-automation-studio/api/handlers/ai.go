package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ElementAnalysisRequest represents the request to analyze page elements
type ElementAnalysisRequest struct {
	URL string `json:"url"`
}

// ElementAtCoordinateRequest represents the request to get element at coordinate
type ElementAtCoordinateRequest struct {
	URL string `json:"url"`
	X   int    `json:"x"`
	Y   int    `json:"y"`
}

// AIAnalyzeRequest represents the request for AI-powered element analysis
type AIAnalyzeRequest struct {
	URL        string `json:"url"`
	Screenshot string `json:"screenshot"`
	Intent     string `json:"intent"`
}

// SelectorOption represents a single selector option with metadata
type SelectorOption struct {
	Selector   string  `json:"selector"`
	Type       string  `json:"type"`       // "id", "class", "data-attr", "xpath", "css"
	Robustness float64 `json:"robustness"` // 0-1 score indicating reliability
	Fallback   bool    `json:"fallback"`   // true if this is a fallback option
}

// ElementInfo represents information about a single interactive element
type ElementInfo struct {
	Text        string            `json:"text"`
	TagName     string            `json:"tagName"`
	Type        string            `json:"type"` // "button", "input", "link", "form", etc.
	Selectors   []SelectorOption  `json:"selectors"`
	BoundingBox Rectangle         `json:"boundingBox"`
	Confidence  float64           `json:"confidence"` // 0-1 confidence that user would interact with this
	Category    string            `json:"category"`   // "authentication", "navigation", "data-entry", etc.
	Attributes  map[string]string `json:"attributes"` // relevant attributes like placeholder, aria-label, etc.
}

// Rectangle represents a bounding box
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// AISuggestion represents an AI-generated suggestion for automation
type AISuggestion struct {
	Action      string  `json:"action"`      // "Login to account", "Search for products", etc.
	Description string  `json:"description"` // Detailed description
	ElementText string  `json:"elementText"` // Text of the target element
	Selector    string  `json:"selector"`    // Recommended selector
	Confidence  float64 `json:"confidence"`  // 0-1 confidence score
	Category    string  `json:"category"`    // "authentication", "navigation", etc.
	Reasoning   string  `json:"reasoning"`   // Why this suggestion was made
}

// PageContext represents contextual information about the page
type PageContext struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	HasLogin    bool   `json:"hasLogin"`
	HasSearch   bool   `json:"hasSearch"`
	FormCount   int    `json:"formCount"`
	ButtonCount int    `json:"buttonCount"`
	LinkCount   int    `json:"linkCount"`
}

// ElementAnalysisResponse represents the complete response for element analysis
type ElementAnalysisResponse struct {
	Success       bool           `json:"success"`
	Elements      []ElementInfo  `json:"elements"`
	AISuggestions []AISuggestion `json:"aiSuggestions"`
	PageContext   PageContext    `json:"pageContext"`
	Screenshot    string         `json:"screenshot"` // base64 encoded
	Timestamp     int64          `json:"timestamp"`
}

func (h *Handler) TakePreviewScreenshot(w http.ResponseWriter, r *http.Request) {
	type PreviewRequest struct {
		URL string `json:"url"`
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode preview request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	h.log.WithField("url", req.URL).Info("Taking preview screenshot and capturing console logs")

	// Create temporary files for screenshot and console logs
	tmpScreenshotFile, err := os.CreateTemp("", "preview-*.png")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp screenshot file")
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpScreenshotFile.Name())
	defer tmpScreenshotFile.Close()

	tmpConsoleFile, err := os.CreateTemp("", "console-*.json")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp console file")
		http.Error(w, "Failed to create temporary console file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpConsoleFile.Name())
	defer tmpConsoleFile.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second) // Increased timeout for both operations
	defer cancel()

	// Take screenshot first with explicit viewport to match extract viewport
	screenshotCmd := exec.CommandContext(ctx, "resource-browserless", "screenshot",
		"--url", req.URL,
		"--output", tmpScreenshotFile.Name())

	screenshotOutput, err := screenshotCmd.CombinedOutput()
	if err != nil {
		h.log.WithError(err).WithField("output", string(screenshotOutput)).Error("Failed to take screenshot with resource-browserless")

		// Provide more helpful error messages
		errorMsg := "Failed to take screenshot"
		if strings.Contains(string(screenshotOutput), "HTTP 500") {
			errorMsg = "Unable to access the URL - it may not be reachable from the browser automation service"
		} else if strings.Contains(string(screenshotOutput), "timeout") {
			errorMsg = "Screenshot request timed out - the page may be taking too long to load"
		} else if strings.Contains(string(screenshotOutput), "connection") {
			errorMsg = "Cannot connect to the URL - please check if it's accessible"
		}

		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	// Capture console logs
	consoleCmd := exec.CommandContext(ctx, "resource-browserless", "console",
		req.URL,
		"--output", tmpConsoleFile.Name(),
		"--wait-ms", "3000") // Wait a bit longer for dynamic content

	consoleOutput, consoleErr := consoleCmd.CombinedOutput()
	var consoleLogs interface{}
	if consoleErr != nil {
		h.log.WithError(consoleErr).WithField("output", string(consoleOutput)).Warn("Failed to capture console logs, continuing with screenshot only")
		// Set empty console logs if capture fails
		consoleLogs = []interface{}{}
	} else {
		// Read and parse console logs
		consoleData, err := os.ReadFile(tmpConsoleFile.Name())
		if err != nil {
			h.log.WithError(err).Warn("Failed to read console log file")
			consoleLogs = []interface{}{}
		} else {
			var consoleResult map[string]interface{}
			if err := json.Unmarshal(consoleData, &consoleResult); err != nil {
				h.log.WithError(err).Warn("Failed to parse console log JSON")
				consoleLogs = []interface{}{}
			} else {
				if logs, ok := consoleResult["logs"]; ok {
					consoleLogs = logs
				} else {
					consoleLogs = []interface{}{}
				}
			}
		}
	}

	// Check if the screenshot file was actually created and is valid
	fileInfo, err := os.Stat(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Screenshot file was not created")
		http.Error(w, "Screenshot file was not created", http.StatusInternalServerError)
		return
	}

	// Check if file has content
	if fileInfo.Size() == 0 {
		h.log.Error("Screenshot file is empty")
		http.Error(w, "Screenshot file is empty", http.StatusInternalServerError)
		return
	}

	// Read the screenshot file
	screenshotData, err := os.ReadFile(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Failed to read screenshot file")
		http.Error(w, "Failed to read screenshot", http.StatusInternalServerError)
		return
	}

	// Validate that it's a PNG file by checking the magic bytes
	if len(screenshotData) < 8 || !bytes.Equal(screenshotData[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		h.log.Error("Generated file is not a valid PNG")
		http.Error(w, "Generated file is not a valid PNG image", http.StatusInternalServerError)
		return
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(screenshotData)

	// Return both screenshot and console logs
	response := map[string]interface{}{
		"success":     true,
		"screenshot":  fmt.Sprintf("data:image/png;base64,%s", base64Data),
		"consoleLogs": consoleLogs,
		"url":         req.URL,
		"timestamp":   time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeElements handles POST /api/v1/analyze-elements
func (h *Handler) AnalyzeElements(w http.ResponseWriter, r *http.Request) {
	var req ElementAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode element analysis request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
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
		http.Error(w, "Failed to analyze page elements", http.StatusInternalServerError)
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
func (h *Handler) GetElementAtCoordinate(w http.ResponseWriter, r *http.Request) {
	var req ElementAtCoordinateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode element at coordinate request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
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

	element, err := h.getElementAtCoordinate(ctx, url, req.X, req.Y)
	if err != nil {
		h.log.WithError(err).Error("Failed to get element at coordinate")
		http.Error(w, "Failed to get element at coordinate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(element)
}

// getElementAtCoordinate uses browserless to get element at specific coordinates
func (h *Handler) getElementAtCoordinate(ctx context.Context, url string, x, y int) (*ElementInfo, error) {
	// Create temporary file for results
	tmpFile, err := os.CreateTemp("", "element-at-coord-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// JavaScript script to get element at coordinates
	script := fmt.Sprintf(`
const element = document.elementFromPoint(%d, %d);

if (!element) {
  return { error: "No element found at coordinates" };
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

const rect = element.getBoundingClientRect();
const text = element.textContent?.trim() || element.value || element.placeholder || '';

return {
  text: text.substring(0, 100),
  tagName: element.tagName,
  type: element.type || element.tagName.toLowerCase(),
  selectors: generateSelectors(element),
  boundingBox: {
    x: rect.x,
    y: rect.y,
    width: rect.width,
    height: rect.height
  },
  confidence: 0.8,
  category: categorizeElement(element),
  attributes: {
    id: element.id || '',
    className: element.className || '',
    name: element.name || '',
    placeholder: element.placeholder || '',
    'aria-label': element.getAttribute('aria-label') || '',
    title: element.title || ''
  }
};`, x, y)

	// Try using navigate + extract approach to avoid bot detection
	// First navigate to the page
	navigateCmd := exec.CommandContext(ctx, "resource-browserless", "navigate", url)
	navigateOutput, err := navigateCmd.CombinedOutput()
	if err != nil {
		h.log.WithError(err).WithField("output", string(navigateOutput)).Error("Failed to navigate to URL")
	} else {
		h.log.Info("Successfully navigated to URL")
	}

	// Then try extract with script
	extractCmd := exec.CommandContext(ctx, "resource-browserless", "extract", url,
		"--script", script,
		"--output", tmpFile.Name())

	output, err := extractCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to extract element: %w, output: %s", err, string(output))
	}

	// Read and parse result
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read result: %w", err)
	}

	var element ElementInfo
	if err := json.Unmarshal(data, &element); err != nil {
		return nil, fmt.Errorf("failed to parse element JSON: %w", err)
	}

	return &element, nil
}

// AIAnalyzeElements handles POST /api/v1/ai-analyze-elements
func (h *Handler) AIAnalyzeElements(w http.ResponseWriter, r *http.Request) {
	var req AIAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode AI analyze request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.Intent == "" {
		http.Error(w, "URL and intent are required", http.StatusBadRequest)
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
		http.Error(w, "Failed to analyze elements with AI", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// extractDOMTree extracts the DOM structure from a page for analysis
func (h *Handler) extractDOMTree(ctx context.Context, url string) (string, error) {
	// Normalize URL - add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// JavaScript to extract DOM tree with selectors - simplified and robust
	script := `
	try {
		// Wait a bit for React to render
		await new Promise(resolve => setTimeout(resolve, 2000));
		
		const body = document.body || document.documentElement;
		const result = {
			tagName: "BODY",
			id: null,
			className: null,
			text: null,
			selector: "body",
			children: []
		};
		
		if (!body) {
			return result;
		}
		
		// Simple extraction - just get immediate children
		const children = Array.from(body.children || []);
		for (let i = 0; i < Math.min(children.length, 10); i++) {
			const child = children[i];
			if (child && child.tagName && !['SCRIPT', 'STYLE', 'NOSCRIPT'].includes(child.tagName)) {
				const childData = {
					tagName: child.tagName,
					id: child.id || null,
					className: child.className || null,
					text: (child.textContent || "").substring(0, 50).trim() || null,
					selector: child.id ? "#" + child.id : child.tagName.toLowerCase()
				};
				result.children.push(childData);
			}
		}
		
		return result;
	} catch (e) {
		return {
			tagName: "ERROR",
			selector: "error", 
			text: e.message,
			children: []
		};
	}`

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "dom-extract-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Use browserless extract to get the DOM
	extractCmd := exec.CommandContext(ctx, "resource-browserless", "extract", url,
		"--script", script,
		"--output", tmpFile.Name())

	output, err := extractCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to extract DOM: %w, output: %s", err, string(output))
	}

	// Read the extracted DOM data
	domData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read DOM data: %w", err)
	}

	// Return the extracted DOM data directly (browserless now returns proper JSON)
	return string(domData), nil
}

// GetDOMTree handles POST /api/v1/dom-tree - returns the DOM structure for Browser Inspector
func (h *Handler) GetDOMTree(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode DOM tree request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	h.log.WithField("url", req.URL).Info("Extracting DOM tree")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	domData, err := h.extractDOMTree(ctx, req.URL)
	if err != nil {
		h.log.WithError(err).Error("Failed to extract DOM tree")
		http.Error(w, "Failed to extract DOM tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(domData))
}

// analyzeElementsWithAI uses Ollama to analyze the DOM and suggest elements
func (h *Handler) analyzeElementsWithAI(ctx context.Context, url, intent string) ([]ElementInfo, error) {
	// First, extract the DOM tree from the page
	domData, err := h.extractDOMTree(ctx, url)
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
	ollamaPayload := map[string]interface{}{
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

// extractPageElements extracts interactive elements from a webpage
func (h *Handler) extractPageElements(ctx context.Context, url string) ([]ElementInfo, PageContext, string, error) {
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
func (h *Handler) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	// Build prompt for Ollama
	prompt := h.buildElementAnalysisPrompt(elements, pageContext)

	// Create temporary file for Ollama output
	tmpOllamaFile, err := os.CreateTemp("", "ollama-suggestions-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp ollama file: %w", err)
	}
	defer os.Remove(tmpOllamaFile.Name())
	defer tmpOllamaFile.Close()

	// Call resource-ollama
	ollamaCmd := exec.CommandContext(ctx, "resource-ollama", "chat",
		"--model", "llama3.2",
		"--prompt", prompt,
		"--format", "json",
		"--max-tokens", "1000",
		"--output", tmpOllamaFile.Name())

	output, err := ollamaCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w, output: %s", err, string(output))
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
func (h *Handler) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
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
func (h *Handler) parseOllamaFallback(response string) ([]AISuggestion, error) {
	h.log.WithField("response", response).Debug("Attempting fallback parsing of Ollama response")

	// For now, return empty suggestions if we can't parse
	// In a real implementation, you might try regex extraction or other parsing strategies
	return []AISuggestion{}, nil
}

