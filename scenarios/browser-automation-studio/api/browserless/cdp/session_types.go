package cdp

import (
	"time"

	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// StepResult represents the outcome of executing a workflow step.
type StepResult struct {
	Success            bool                     `json:"success"`
	Error              string                   `json:"error,omitempty"`
	DurationMs         int                      `json:"durationMs"`
	Screenshot         string                   `json:"screenshot,omitempty"`
	URL                string                   `json:"url"`
	Title              string                   `json:"title"`
	ConsoleLogs        []ConsoleLog             `json:"consoleLogs,omitempty"`
	NetworkEvents      []NetworkEvent           `json:"networkEvents,omitempty"`
	DebugContext       map[string]interface{}   `json:"debugContext,omitempty"`
	ElementBoundingBox *runtime.BoundingBox     `json:"elementBoundingBox,omitempty"`
	ExtractedData      any                      `json:"extractedData,omitempty"`
	Condition          *runtime.ConditionResult `json:"conditionResult,omitempty"`
}

const (
	defaultScrollAmountPixels      = 400
	defaultScrollVisibilityChecks  = 12
	scrollVisibilityDelay          = 150 * time.Millisecond
	defaultVariableTimeoutMs       = 30000
	defaultConditionalTimeoutMs    = 10000
	defaultConditionalPollInterval = 250
	rotateOrientationPortrait      = "portrait"
	rotateOrientationLandscape     = "landscape"
	defaultGestureTimeoutMs        = 15000
	defaultGestureDurationMs       = 350
	defaultGestureSteps            = 15
	defaultGestureDistance         = 150
	minGestureDurationMs           = 50
	minGestureSteps                = 2
)

type scrollOptions struct {
	scrollType     string
	selector       string
	targetSelector string
	direction      string
	behavior       string
	amount         int
	x              int
	y              int
	maxAttempts    int
	waitAfterMs    int
}

type dragDropOptions struct {
	sourceSelector string
	targetSelector string
	holdMs         int
	steps          int
	durationMs     int
	offsetX        int
	offsetY        int
	waitAfterMs    int
}

type gestureOptions struct {
	Type             string
	Direction        string
	StartX           float64
	StartY           float64
	EndX             float64
	EndY             float64
	Distance         int
	Scale            float64
	DurationMs       int
	HoldMs           int
	Steps            int
	Selector         string
	TimeoutMs        int
	HasExplicitStart bool
	HasExplicitEnd   bool
	HasStartX        bool
	HasStartY        bool
	HasEndX          bool
	HasEndY          bool
}

type selectEvalPayload struct {
	Selector string   `json:"selector"`
	Mode     string   `json:"mode"`
	Value    string   `json:"value"`
	Text     string   `json:"text"`
	Index    int      `json:"index"`
	Values   []string `json:"values"`
	Multi    bool     `json:"multi"`
}

type selectEvalResult struct {
	SelectedValues []string `json:"selectedValues"`
	SelectedTexts  []string `json:"selectedTexts"`
	Multiple       bool     `json:"multiple"`
}

type tabSwitchOptions struct {
	SwitchBy   string
	Index      int
	TitleMatch string
	URLMatch   string
	WaitForNew bool
	TimeoutMs  int
	CloseOld   bool
}

type conditionalOptions struct {
	Type           string
	Expression     string
	Selector       string
	Variable       string
	Operator       string
	Value          any
	Negate         bool
	TimeoutMs      int
	PollIntervalMs int
	Variables      map[string]any
}
