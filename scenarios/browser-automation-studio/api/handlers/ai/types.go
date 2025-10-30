package ai

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
