package runtime

// ExecutionResponse mirrors the payload returned by Browserless execution script.
type ExecutionResponse struct {
	Success         bool         `json:"success"`
	Error           string       `json:"error,omitempty"`
	Steps           []StepResult `json:"steps"`
	TotalDurationMs int          `json:"totalDurationMs,omitempty"`
}

// StepResult captures the outcome of a single instruction.
type StepResult struct {
	Index              int               `json:"index"`
	NodeID             string            `json:"nodeId"`
	Type               string            `json:"type"`
	StepName           string            `json:"stepName,omitempty"`
	Success            bool              `json:"success"`
	Error              string            `json:"error,omitempty"`
	Stack              string            `json:"stack,omitempty"`
	DurationMs         int               `json:"durationMs,omitempty"`
	ScreenshotBase64   string            `json:"screenshotBase64,omitempty"`
	FinalURL           string            `json:"finalUrl,omitempty"`
	Scenario           string            `json:"scenario,omitempty"`        // Scenario name for navigate nodes
	ScenarioPath       string            `json:"scenarioPath,omitempty"`    // Scenario path for navigate nodes
	DestinationType    string            `json:"destinationType,omitempty"` // url or scenario
	ConsoleLogs        []ConsoleLog      `json:"consoleLogs,omitempty"`
	NetworkEvents      []NetworkEvent    `json:"networkEvents,omitempty"`
	ElementBoundingBox *BoundingBox      `json:"elementBoundingBox,omitempty"`
	ClickPosition      *Point            `json:"clickPosition,omitempty"`
	ExtractedData      any               `json:"extractedData,omitempty"`
	Shortcuts          []string          `json:"shortcuts,omitempty"`
	ProbeResult        map[string]any    `json:"probeResult,omitempty"`
	FocusedElement     *ElementFocus     `json:"focusedElement,omitempty"`
	HighlightRegions   []HighlightRegion `json:"highlightRegions,omitempty"`
	MaskRegions        []MaskRegion      `json:"maskRegions,omitempty"`
	ZoomFactor         float64           `json:"zoomFactor,omitempty"`
	Assertion          *AssertionResult  `json:"assertion,omitempty"`
	DOMSnapshot        string            `json:"domSnapshot,omitempty"`
}

// AssertionResult captures the outcome of an assert step.
type AssertionResult struct {
	Mode          string `json:"mode,omitempty"`
	Selector      string `json:"selector,omitempty"`
	Expected      any    `json:"expected,omitempty"`
	Actual        any    `json:"actual,omitempty"`
	Success       bool   `json:"success"`
	Negated       bool   `json:"negated,omitempty"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
	Message       string `json:"message,omitempty"`
}

// ConsoleLog represents a console message emitted during a step.
type ConsoleLog struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

// NetworkEvent captures network activity for a step.
type NetworkEvent struct {
	Type         string `json:"type"`
	URL          string `json:"url"`
	Method       string `json:"method,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	Status       int    `json:"status,omitempty"`
	OK           bool   `json:"ok,omitempty"`
	Failure      string `json:"failure,omitempty"`
	Timestamp    int64  `json:"timestamp"`
}

// BoundingBox captures the position of an element within the viewport.
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Point represents a 2D coordinate used for pointer highlights.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ElementFocus captures focus metadata for screenshot framing.
type ElementFocus struct {
	Selector    string       `json:"selector"`
	BoundingBox *BoundingBox `json:"boundingBox,omitempty"`
}

// HighlightRegion describes an overlay applied to the screenshot for emphasis.
type HighlightRegion struct {
	Selector    string       `json:"selector"`
	BoundingBox *BoundingBox `json:"boundingBox,omitempty"`
	Padding     int          `json:"padding,omitempty"`
	Color       string       `json:"color,omitempty"`
}

// MaskRegion describes areas that were dimmed or masked during capture.
type MaskRegion struct {
	Selector    string       `json:"selector"`
	BoundingBox *BoundingBox `json:"boundingBox,omitempty"`
	Opacity     float64      `json:"opacity,omitempty"`
}
