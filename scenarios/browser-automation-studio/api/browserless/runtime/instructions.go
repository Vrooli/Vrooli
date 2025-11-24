package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

const (
	defaultLoopMaxIterations      = 100
	maxLoopMaxIterations          = 5000
	defaultLoopItemVariable       = "loop.item"
	defaultLoopIndexVariable      = "loop.index"
	minLoopIterationTimeoutMs     = 250
	maxLoopIterationTimeoutMs     = 600000
	defaultLoopIterationTimeoutMs = 45000
	minLoopTotalTimeoutMs         = 1000
	maxLoopTotalTimeoutMs         = 1800000
	defaultLoopTotalTimeoutMs     = 300000
)

const (
	defaultResilienceMaxAttempts                   = 3
	defaultResilienceRetryDelayMs                  = 750
	defaultResilienceBackoffFactor         float64 = 1.5
	defaultResiliencePreconditionTimeoutMs         = 15000
)

const (
	rotateOrientationPortrait  = "portrait"
	rotateOrientationLandscape = "landscape"
)

const (
	defaultGestureTimeoutMs  = 15000
	defaultGestureDurationMs = 350
	minGestureDurationMs     = 50
	maxGestureDurationMs     = 10000
	defaultGestureDistance   = 150
	minGestureDistance       = 10
	maxGestureDistance       = 5000
	defaultGestureSteps      = 15
	minGestureSteps          = 2
	maxGestureSteps          = 60
	minGestureScale          = 0.2
	maxGestureScale          = 4.0
)

// Instruction represents a normalized execution step that can be shipped to Browserless.
type Instruction struct {
	Index       int              `json:"index"`
	NodeID      string           `json:"nodeId"`
	Type        string           `json:"type"`
	Params      InstructionParam `json:"params"`
	PreloadHTML string           `json:"preloadHtml,omitempty"`
	Context     map[string]any   `json:"context,omitempty"`
}

// InstructionParam captures the parameter payload for a Browserless instruction.
type InstructionParam struct {
	URL                     string            `json:"url,omitempty"`
	Scenario                string            `json:"scenario,omitempty"`        // Scenario name for navigate nodes
	ScenarioPath            string            `json:"scenarioPath,omitempty"`    // Scenario path for navigate nodes
	DestinationType         string            `json:"destinationType,omitempty"` // url or scenario
	WaitUntil               string            `json:"waitUntil,omitempty"`
	TimeoutMs               int               `json:"timeoutMs,omitempty"`
	WaitForMs               int               `json:"waitForMs,omitempty"`
	WaitType                string            `json:"waitType,omitempty"`
	DurationMs              int               `json:"durationMs,omitempty"`
	MovementSteps           int               `json:"movementSteps,omitempty"`
	DragSourceSelector      string            `json:"dragSourceSelector,omitempty"`
	DragTargetSelector      string            `json:"dragTargetSelector,omitempty"`
	DragHoldMs              int               `json:"dragHoldMs,omitempty"`
	DragSteps               int               `json:"dragSteps,omitempty"`
	DragDurationMs          int               `json:"dragDurationMs,omitempty"`
	DragOffsetX             int               `json:"dragOffsetX,omitempty"`
	DragOffsetY             int               `json:"dragOffsetY,omitempty"`
	Selector                string            `json:"selector,omitempty"`
	WaitForSelector         string            `json:"waitForSelector,omitempty"`
	FilePaths               []string          `json:"filePaths,omitempty"`
	FilePath                string            `json:"filePath,omitempty"`
	SelectionMode           string            `json:"selectionMode,omitempty"`
	Name                    string            `json:"name,omitempty"`
	FullPage                *bool             `json:"fullPage,omitempty"`
	ViewportWidth           int               `json:"viewportWidth,omitempty"`
	ViewportHeight          int               `json:"viewportHeight,omitempty"`
	RotateOrientation       string            `json:"rotateOrientation,omitempty"`
	RotateAngle             int               `json:"rotateAngle,omitempty"`
	GestureType             string            `json:"gestureType,omitempty"`
	GestureDirection        string            `json:"gestureDirection,omitempty"`
	GestureSelector         string            `json:"gestureSelector,omitempty"`
	GestureStartX           int               `json:"gestureStartX,omitempty"`
	GestureStartY           int               `json:"gestureStartY,omitempty"`
	GestureEndX             int               `json:"gestureEndX,omitempty"`
	GestureEndY             int               `json:"gestureEndY,omitempty"`
	GestureDistance         int               `json:"gestureDistance,omitempty"`
	GestureScale            float64           `json:"gestureScale,omitempty"`
	GestureDurationMs       int               `json:"gestureDurationMs,omitempty"`
	GestureHoldMs           int               `json:"gestureHoldMs,omitempty"`
	GestureSteps            int               `json:"gestureSteps,omitempty"`
	GestureHasStart         bool              `json:"gestureHasStart,omitempty"`
	GestureHasEnd           bool              `json:"gestureHasEnd,omitempty"`
	GestureHasStartX        bool              `json:"gestureHasStartX,omitempty"`
	GestureHasStartY        bool              `json:"gestureHasStartY,omitempty"`
	GestureHasEndX          bool              `json:"gestureHasEndX,omitempty"`
	GestureHasEndY          bool              `json:"gestureHasEndY,omitempty"`
	Button                  string            `json:"button,omitempty"`
	ClickCount              int               `json:"clickCount,omitempty"`
	Text                    string            `json:"text,omitempty"`
	DelayMs                 int               `json:"delayMs,omitempty"`
	Clear                   *bool             `json:"clear,omitempty"`
	Submit                  *bool             `json:"submit,omitempty"`
	ExtractType             string            `json:"extractType,omitempty"`
	Attribute               string            `json:"attribute,omitempty"`
	AllMatches              *bool             `json:"allMatches,omitempty"`
	FocusSelector           string            `json:"focusSelector,omitempty"`
	HighlightSelectors      []string          `json:"highlightSelectors,omitempty"`
	HighlightColor          string            `json:"highlightColor,omitempty"`
	HighlightPadding        int               `json:"highlightPadding,omitempty"`
	HighlightBorderRadius   int               `json:"highlightBorderRadius,omitempty"`
	MaskSelectors           []string          `json:"maskSelectors,omitempty"`
	MaskOpacity             float64           `json:"maskOpacity,omitempty"`
	Background              string            `json:"background,omitempty"`
	ZoomFactor              float64           `json:"zoomFactor,omitempty"`
	CaptureDomSnapshot      bool              `json:"captureDomSnapshot,omitempty"`
	AssertMode              string            `json:"assertMode,omitempty"`
	ExpectedValue           any               `json:"expectedValue,omitempty"`
	FailureMessage          string            `json:"failureMessage,omitempty"`
	Expression              string            `json:"expression,omitempty"`
	StoreResult             string            `json:"storeResult,omitempty"`
	VariableName            string            `json:"variableName,omitempty"`
	VariableSource          string            `json:"variableSource,omitempty"`
	VariableValue           any               `json:"variableValue,omitempty"`
	VariableStoreAs         string            `json:"variableStoreAs,omitempty"`
	VariableTransform       string            `json:"variableTransform,omitempty"`
	VariableRequired        *bool             `json:"variableRequired,omitempty"`
	ConditionType           string            `json:"conditionType,omitempty"`
	ConditionExpression     string            `json:"conditionExpression,omitempty"`
	ConditionSelector       string            `json:"conditionSelector,omitempty"`
	ConditionVariable       string            `json:"conditionVariable,omitempty"`
	ConditionOperator       string            `json:"conditionOperator,omitempty"`
	ConditionValue          any               `json:"conditionValue,omitempty"`
	ConditionNegate         *bool             `json:"conditionNegate,omitempty"`
	ConditionPollIntervalMs int               `json:"conditionPollIntervalMs,omitempty"`
	CaseSensitive           *bool             `json:"caseSensitive,omitempty"`
	Negate                  *bool             `json:"negate,omitempty"`
	ContinueOnFailure       *bool             `json:"continueOnFailure,omitempty"`
	RetryAttempts           int               `json:"retryAttempts,omitempty"`
	RetryDelayMs            int               `json:"retryDelayMs,omitempty"`
	RetryBackoffFactor      float64           `json:"retryBackoffFactor,omitempty"`
	PreconditionSelector    string            `json:"preconditionSelector,omitempty"`
	PreconditionTimeoutMs   int               `json:"preconditionTimeoutMs,omitempty"`
	PreconditionWaitMs      int               `json:"preconditionWaitMs,omitempty"`
	SuccessSelector         string            `json:"successSelector,omitempty"`
	SuccessTimeoutMs        int               `json:"successTimeoutMs,omitempty"`
	SuccessWaitMs           int               `json:"successWaitMs,omitempty"`
	ProbeX                  int               `json:"probeX,omitempty"`
	ProbeY                  int               `json:"probeY,omitempty"`
	ProbeRadius             int               `json:"probeRadius,omitempty"`
	ProbeSamples            int               `json:"probeSamples,omitempty"`
	ShortcutKeys            []string          `json:"shortcutKeys,omitempty"`
	ShortcutDelayMs         int               `json:"shortcutDelayMs,omitempty"`
	KeyValue                string            `json:"keyValue,omitempty"`
	KeyEventType            string            `json:"keyEventType,omitempty"`
	KeyModifiers            []string          `json:"keyModifiers,omitempty"`
	ScrollType              string            `json:"scrollType,omitempty"`
	ScrollDirection         string            `json:"scrollDirection,omitempty"`
	ScrollAmount            int               `json:"scrollAmount,omitempty"`
	ScrollBehavior          string            `json:"scrollBehavior,omitempty"`
	ScrollX                 int               `json:"scrollX,omitempty"`
	ScrollY                 int               `json:"scrollY,omitempty"`
	ScrollTargetSelector    string            `json:"scrollTargetSelector,omitempty"`
	ScrollMaxAttempts       int               `json:"scrollMaxAttempts,omitempty"`
	OptionValue             string            `json:"optionValue,omitempty"`
	OptionText              string            `json:"optionText,omitempty"`
	OptionIndex             int               `json:"optionIndex,omitempty"`
	OptionValues            []string          `json:"optionValues,omitempty"`
	MultiSelect             bool              `json:"multiSelect,omitempty"`
	TabSwitchBy             string            `json:"tabSwitchBy,omitempty"`
	TabIndex                int               `json:"tabIndex,omitempty"`
	TabTitleMatch           string            `json:"tabTitleMatch,omitempty"`
	TabURLMatch             string            `json:"tabUrlMatch,omitempty"`
	TabWaitForNew           bool              `json:"tabWaitForNew,omitempty"`
	TabCloseOld             bool              `json:"tabCloseOld,omitempty"`
	FrameSwitchBy           string            `json:"frameSwitchBy,omitempty"`
	FrameIndex              int               `json:"frameIndex,omitempty"`
	FrameName               string            `json:"frameName,omitempty"`
	FrameSelector           string            `json:"frameSelector,omitempty"`
	FrameURLMatch           string            `json:"frameUrlMatch,omitempty"`
	LoopType                string            `json:"loopType,omitempty"`
	LoopItems               []any             `json:"loopItems,omitempty"`
	LoopArraySource         string            `json:"loopArraySource,omitempty"`
	LoopCount               int               `json:"loopCount,omitempty"`
	LoopConditionExpression string            `json:"loopConditionExpression,omitempty"`
	LoopConditionType       string            `json:"loopConditionType,omitempty"`
	LoopConditionVariable   string            `json:"loopConditionVariable,omitempty"`
	LoopConditionOperator   string            `json:"loopConditionOperator,omitempty"`
	LoopConditionValue      any               `json:"loopConditionValue,omitempty"`
	LoopMaxIterations       int               `json:"loopMaxIterations,omitempty"`
	LoopItemVariable        string            `json:"loopItemVariable,omitempty"`
	LoopIndexVariable       string            `json:"loopIndexVariable,omitempty"`
	LoopIterationTimeoutMs  int               `json:"loopIterationTimeoutMs,omitempty"`
	LoopTotalTimeoutMs      int               `json:"loopTotalTimeoutMs,omitempty"`
	WorkflowCallID          string            `json:"workflowCallId,omitempty"`
	WorkflowCallName        string            `json:"workflowCallName,omitempty"`
	WorkflowCallWait        *bool             `json:"workflowCallWait,omitempty"`
	WorkflowCallParams      map[string]any    `json:"workflowCallParams,omitempty"`
	WorkflowCallOutputs     map[string]string `json:"workflowCallOutputs,omitempty"`
	WorkflowCallDefinition  map[string]any    `json:"workflowCallDefinition,omitempty"`
	CookieName              string            `json:"cookieName,omitempty"`
	CookieValue             string            `json:"cookieValue,omitempty"`
	CookieURL               string            `json:"cookieUrl,omitempty"`
	CookieDomain            string            `json:"cookieDomain,omitempty"`
	CookiePath              string            `json:"cookiePath,omitempty"`
	CookieSameSite          string            `json:"cookieSameSite,omitempty"`
	CookieExpiresAt         string            `json:"cookieExpiresAt,omitempty"`
	CookieResultFormat      string            `json:"cookieResultFormat,omitempty"`
	CookieTTLSeconds        int               `json:"cookieTtlSeconds,omitempty"`
	CookieHTTPOnly          *bool             `json:"cookieHttpOnly,omitempty"`
	CookieSecure            *bool             `json:"cookieSecure,omitempty"`
	CookieClearAll          bool              `json:"cookieClearAll,omitempty"`
	StorageType             string            `json:"storageType,omitempty"`
	StorageKey              string            `json:"storageKey,omitempty"`
	StorageValue            string            `json:"storageValue,omitempty"`
	StorageValueType        string            `json:"storageValueType,omitempty"`
	StorageResultFormat     string            `json:"storageResultFormat,omitempty"`
	StorageClearAll         bool              `json:"storageClearAll,omitempty"`
	NetworkURLPattern       string            `json:"networkUrlPattern,omitempty"`
	NetworkMethod           string            `json:"networkMethod,omitempty"`
	NetworkMockType         string            `json:"networkMockType,omitempty"`
	NetworkStatusCode       int               `json:"networkStatusCode,omitempty"`
	NetworkHeaders          map[string]string `json:"networkHeaders,omitempty"`
	NetworkBody             any               `json:"networkBody,omitempty"`
	NetworkDelayMs          int               `json:"networkDelayMs,omitempty"`
	NetworkAbortReason      string            `json:"networkAbortReason,omitempty"`
}

// InstructionsFromPlan converts a compiled execution plan into Browserless instructions.
func InstructionsFromPlan(ctx context.Context, plan *compiler.ExecutionPlan) ([]Instruction, error) {
	if plan == nil {
		return nil, fmt.Errorf("execution plan is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	instructions := make([]Instruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		instr, err := instructionFromStep(ctx, step)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, instr)
	}

	return instructions, nil
}

type navigateConfig struct {
	URL             string `json:"url"`
	WaitUntil       string `json:"waitUntil"`
	TimeoutMs       int    `json:"timeoutMs"`
	WaitForMs       int    `json:"waitForMs"`
	Scenario        string `json:"scenario"`
	ScenarioName    string `json:"scenarioName"`
	ScenarioPath    string `json:"scenarioPath"`
	ScenarioPort    string `json:"scenarioPort"`
	DestinationType string `json:"destinationType"`
}

type waitConfig struct {
	Type      string `json:"type"`
	WaitType  string `json:"waitType"`
	Duration  int    `json:"duration"`
	Selector  string `json:"selector"`
	TimeoutMs int    `json:"timeoutMs"`
}

type screenshotConfig struct {
	Name                  string   `json:"name"`
	FullPage              *bool    `json:"fullPage"`
	ViewportWidth         int      `json:"viewportWidth"`
	ViewportHeight        int      `json:"viewportHeight"`
	WaitForMs             int      `json:"waitForMs"`
	FocusSelector         string   `json:"focusSelector"`
	HighlightSelectors    []string `json:"highlightSelectors"`
	HighlightColor        string   `json:"highlightColor"`
	HighlightPadding      int      `json:"highlightPadding"`
	HighlightBorderRadius int      `json:"highlightBorderRadius"`
	MaskSelectors         []string `json:"maskSelectors"`
	MaskOpacity           float64  `json:"maskOpacity"`
	Background            string   `json:"background"`
	ZoomFactor            float64  `json:"zoomFactor"`
	CaptureDomSnapshot    bool     `json:"captureDomSnapshot"`
}

type rotateConfig struct {
	Orientation string `json:"orientation"`
	Angle       int    `json:"angle"`
	WaitForMs   int    `json:"waitForMs"`
}

type gestureConfig struct {
	GestureType string  `json:"gestureType"`
	Direction   string  `json:"direction"`
	StartX      float64 `json:"startX"`
	StartY      float64 `json:"startY"`
	EndX        float64 `json:"endX"`
	EndY        float64 `json:"endY"`
	Distance    int     `json:"distance"`
	Scale       float64 `json:"scale"`
	Selector    string  `json:"selector"`
	DurationMs  int     `json:"durationMs"`
	HoldMs      int     `json:"holdMs"`
	Steps       int     `json:"steps"`
	TimeoutMs   int     `json:"timeoutMs"`
}

type clickConfig struct {
	Selector        string `json:"selector"`
	Button          string `json:"button"`
	ClickCount      int    `json:"clickCount"`
	TimeoutMs       int    `json:"timeoutMs"`
	WaitForMs       int    `json:"waitForMs"`
	WaitForSelector string `json:"waitForSelector"`
}

type typeConfig struct {
	Selector  string `json:"selector"`
	Text      string `json:"text"`
	DelayMs   int    `json:"delayMs"`
	Clear     *bool  `json:"clear"`
	Submit    *bool  `json:"submit"`
	TimeoutMs int    `json:"timeoutMs"`
}

type shortcutConfig struct {
	Shortcuts     []string `json:"shortcuts"`
	Shortcut      string   `json:"shortcut"`
	Sequence      []string `json:"sequence"`
	Keys          []string `json:"keys"`
	DelayMs       int      `json:"delayMs"`
	TimeoutMs     int      `json:"timeoutMs"`
	FocusSelector string   `json:"focusSelector"`
}

type hoverConfig struct {
	Selector   string `json:"selector"`
	TimeoutMs  int    `json:"timeoutMs"`
	WaitForMs  int    `json:"waitForMs"`
	Steps      int    `json:"steps"`
	DurationMs int    `json:"durationMs"`
}

type dragDropConfig struct {
	SourceSelector string `json:"sourceSelector"`
	TargetSelector string `json:"targetSelector"`
	HoldMs         int    `json:"holdMs"`
	Steps          int    `json:"steps"`
	DurationMs     int    `json:"durationMs"`
	OffsetX        int    `json:"offsetX"`
	OffsetY        int    `json:"offsetY"`
	TimeoutMs      int    `json:"timeoutMs"`
	WaitForMs      int    `json:"waitForMs"`
}

type focusConfig struct {
	Selector  string `json:"selector"`
	TimeoutMs int    `json:"timeoutMs"`
	WaitForMs int    `json:"waitForMs"`
}

const (
	defaultHoverSteps      = 10
	minHoverSteps          = 1
	maxHoverSteps          = 50
	defaultHoverDurationMs = 350
	minHoverDurationMs     = 50
	maxHoverDurationMs     = 10000
)

const (
	defaultDragHoldMs     = 150
	minDragHoldMs         = 0
	maxDragHoldMs         = 5000
	defaultDragSteps      = 18
	minDragSteps          = 1
	maxDragSteps          = 60
	defaultDragDurationMs = 600
	minDragDurationMs     = 50
	maxDragDurationMs     = 20000
	minDragOffset         = -5000
	maxDragOffset         = 5000
)

type scrollConfig struct {
	ScrollType     string `json:"scrollType"`
	Selector       string `json:"selector"`
	TargetSelector string `json:"targetSelector"`
	Direction      string `json:"direction"`
	Amount         int    `json:"amount"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	Behavior       string `json:"behavior"`
	TimeoutMs      int    `json:"timeoutMs"`
	WaitForMs      int    `json:"waitForMs"`
	MaxScrolls     int    `json:"maxScrolls"`
}

const (
	defaultScrollAmount   = 400
	minScrollAmount       = 10
	maxScrollAmount       = 5000
	defaultScrollAttempts = 12
	minScrollAttempts     = 1
	maxScrollAttempts     = 200
	minScrollCoordinate   = -500000
	maxScrollCoordinate   = 500000
)

const (
	defaultConditionalTimeoutMs    = 10000
	minConditionalTimeoutMs        = 100
	maxConditionalTimeoutMs        = 60000
	defaultConditionalPollInterval = 250
	minConditionalPollInterval     = 50
	maxConditionalPollInterval     = 2000
)

const (
	defaultNetworkMockStatus = 200
	minHTTPStatusCode        = 100
	maxHTTPStatusCode        = 599
	maxNetworkDelayMs        = 600000
)

type extractConfig struct {
	Selector    string `json:"selector"`
	ExtractType string `json:"extractType"`
	Attribute   string `json:"attribute"`
	AllMatches  *bool  `json:"allMatches"`
	StoreResult string `json:"storeResult"`
	StoreIn     string `json:"storeIn"`
}

type selectConfig struct {
	Selector  string   `json:"selector"`
	SelectBy  string   `json:"selectBy"`
	Value     string   `json:"value"`
	Text      string   `json:"text"`
	Index     int      `json:"index"`
	Multiple  bool     `json:"multiple"`
	Values    []string `json:"values"`
	TimeoutMs int      `json:"timeoutMs"`
	WaitForMs int      `json:"waitForMs"`
}

type tabSwitchConfig struct {
	SwitchBy   string `json:"switchBy"`
	Index      int    `json:"index"`
	TitleMatch string `json:"titleMatch"`
	URLMatch   string `json:"urlMatch"`
	WaitForNew bool   `json:"waitForNew"`
	TimeoutMs  int    `json:"timeoutMs"`
	CloseOld   bool   `json:"closeOld"`
}

type frameSwitchConfig struct {
	SwitchBy  string `json:"switchBy"`
	Index     int    `json:"index"`
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	URLMatch  string `json:"urlMatch"`
	TimeoutMs int    `json:"timeoutMs"`
}

type evaluateConfig struct {
	Expression  string `json:"expression"`
	TimeoutMs   int    `json:"timeoutMs"`
	StoreResult string `json:"storeResult"`
}

type uploadFileConfig struct {
	Selector  string   `json:"selector"`
	FilePath  string   `json:"filePath"`
	FilePaths []string `json:"filePaths"`
	TimeoutMs int      `json:"timeoutMs"`
	WaitForMs int      `json:"waitForMs"`
}

type keyboardConfig struct {
	Key       string               `json:"key"`       // Legacy singular format (deprecated)
	Keys      []string             `json:"keys"`      // Modern array format (preferred)
	Sequence  string               `json:"sequence"`  // Alternative sequence format
	EventType string               `json:"eventType"`
	Modifiers keyboardModifierSpec `json:"modifiers"`
	DelayMs   int                  `json:"delayMs"`
	TimeoutMs int                  `json:"timeoutMs"`
}

type setVariableConfig struct {
	Name        string `json:"name"`
	SourceType  string `json:"sourceType"`
	Value       any    `json:"value"`
	ValueType   string `json:"valueType"`
	Expression  string `json:"expression"`
	Selector    string `json:"selector"`
	ExtractType string `json:"extractType"`
	Attribute   string `json:"attribute"`
	AllMatches  *bool  `json:"allMatches"`
	TimeoutMs   int    `json:"timeoutMs"`
	StoreAs     string `json:"storeAs"`
}

type useVariableConfig struct {
	Name      string `json:"name"`
	StoreAs   string `json:"storeAs"`
	Transform string `json:"transform"`
	Required  *bool  `json:"required"`
}

type conditionalConfig struct {
	ConditionType  string `json:"conditionType"`
	Expression     string `json:"expression"`
	Selector       string `json:"selector"`
	Variable       string `json:"variable"`
	Operator       string `json:"operator"`
	Value          any    `json:"value"`
	Negate         bool   `json:"negate"`
	TimeoutMs      int    `json:"timeoutMs"`
	PollIntervalMs int    `json:"pollIntervalMs"`
}

type loopConfig struct {
	LoopType            string `json:"loopType"`
	ArraySource         string `json:"arraySource"`
	LoopItems           []any  `json:"loopItems"`
	Count               int    `json:"count"`
	MaxIterations       int    `json:"maxIterations"`
	IterationTimeoutMs  int    `json:"iterationTimeoutMs"`
	TotalTimeoutMs      int    `json:"totalTimeoutMs"`
	ItemVariable        string `json:"itemVariable"`
	IndexVariable       string `json:"indexVariable"`
	ConditionType       string `json:"conditionType"`
	ConditionExpression string `json:"conditionExpression"`
	ConditionVariable   string `json:"conditionVariable"`
	ConditionOperator   string `json:"conditionOperator"`
	ConditionValue      any    `json:"conditionValue"`
}

type workflowCallConfig struct {
	WorkflowID        string            `json:"workflowId"`
	WorkflowName      string            `json:"workflowName"`
	WaitForCompletion *bool             `json:"waitForCompletion"`
	Parameters        map[string]any    `json:"parameters"`
	OutputMapping     map[string]string `json:"outputMapping"`
	InlineDefinition  map[string]any    `json:"workflowDefinition"`
}

type setCookieConfig struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	URL        string `json:"url"`
	Domain     string `json:"domain"`
	Path       string `json:"path"`
	SameSite   string `json:"sameSite"`
	Secure     *bool  `json:"secure"`
	HTTPOnly   *bool  `json:"httpOnly"`
	ExpiresAt  string `json:"expiresAt"`
	TTLSeconds int    `json:"ttlSeconds"`
	MaxAge     int    `json:"maxAgeSeconds"`
	TimeoutMs  int    `json:"timeoutMs"`
	WaitForMs  int    `json:"waitForMs"`
}

type getCookieConfig struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	ResultAs    string `json:"resultFormat"`
	StoreResult string `json:"storeResult"`
	StoreAs     string `json:"storeAs"`
	TimeoutMs   int    `json:"timeoutMs"`
	WaitForMs   int    `json:"waitForMs"`
}

type clearCookieConfig struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Domain    string `json:"domain"`
	Path      string `json:"path"`
	ClearAll  bool   `json:"clearAll"`
	TimeoutMs int    `json:"timeoutMs"`
	WaitForMs int    `json:"waitForMs"`
}

type setStorageConfig struct {
	StorageType string `json:"storageType"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	TimeoutMs   int    `json:"timeoutMs"`
	WaitForMs   int    `json:"waitForMs"`
}

type getStorageConfig struct {
	StorageType string `json:"storageType"`
	Key         string `json:"key"`
	ResultAs    string `json:"resultFormat"`
	StoreResult string `json:"storeResult"`
	StoreAs     string `json:"storeAs"`
	TimeoutMs   int    `json:"timeoutMs"`
	WaitForMs   int    `json:"waitForMs"`
}

type clearStorageConfig struct {
	StorageType string `json:"storageType"`
	Key         string `json:"key"`
	ClearAll    bool   `json:"clearAll"`
	TimeoutMs   int    `json:"timeoutMs"`
	WaitForMs   int    `json:"waitForMs"`
}

type networkMockConfig struct {
	URLPattern  string            `json:"urlPattern"`
	Method      string            `json:"method"`
	MockType    string            `json:"mockType"`
	StatusCode  int               `json:"statusCode"`
	Headers     map[string]string `json:"headers"`
	Body        any               `json:"body"`
	DelayMs     int               `json:"delayMs"`
	AbortReason string            `json:"abortReason"`
}

type keyboardModifierSpec struct {
	Ctrl  bool `json:"ctrl"`
	Shift bool `json:"shift"`
	Alt   bool `json:"alt"`
	Meta  bool `json:"meta"`
}

type assertConfig struct {
	AssertMode        string `json:"assertMode"`
	Mode              string `json:"mode"`
	Comparison        string `json:"comparison"`
	Selector          string `json:"selector"`
	ExpectedValue     any    `json:"expectedValue"`
	Expected          any    `json:"expected"`
	Text              string `json:"text"`
	TimeoutMs         int    `json:"timeoutMs"`
	Expression        string `json:"expression"`
	Attribute         string `json:"attribute"`
	FailureMessage    string `json:"failureMessage"`
	CaseSensitive     *bool  `json:"caseSensitive"`
	Negate            *bool  `json:"negate"`
	ContinueOnFailure *bool  `json:"continueOnFailure"`
}

func instructionFromStep(ctx context.Context, step compiler.ExecutionStep) (Instruction, error) {
	base := Instruction{
		Index:  step.Index,
		NodeID: step.NodeID,
		Type:   string(step.Type),
		Params: InstructionParam{},
	}

	if attempts, ok := getIntParam(step.Params, "retryAttempts", "retry_attempts", "retry"); ok {
		if attempts < 0 {
			attempts = 0
		}
		base.Params.RetryAttempts = attempts
	}
	if delay, ok := getIntParam(step.Params, "retryDelayMs", "retry_delay_ms", "retryDelay"); ok && delay > 0 {
		base.Params.RetryDelayMs = delay
	}
	if factor, ok := getFloatParam(step.Params, "retryBackoffFactor", "retry_backoff_factor", "retryBackoff"); ok && factor > 0 {
		base.Params.RetryBackoffFactor = factor
	}
	if allowFailure, ok := getBoolParam(
		step.Params,
		"continueOnError",
		"continue_on_error",
		"continueOnFailure",
		"continue_on_failure",
	); ok {
		base.Params.ContinueOnFailure = boolPtr(allowFailure)
	}

	resilienceState := applyResilienceConfig(step.Params, &base)

	switch step.Type {
	case compiler.StepNavigate:
		var cfg navigateConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("navigate node %s has invalid data: %w", step.NodeID, err)
		}
		destinationType := strings.ToLower(strings.TrimSpace(cfg.DestinationType))
		scenarioName := strings.TrimSpace(cfg.Scenario)
		if scenarioName == "" {
			scenarioName = strings.TrimSpace(cfg.ScenarioName)
		}
		scenarioSelected := destinationType == "scenario" || (destinationType == "" && scenarioName != "")

		if scenarioSelected {
			if scenarioName == "" {
				return Instruction{}, fmt.Errorf("navigate node %s missing scenario name", step.NodeID)
			}

			portCandidates := []string{}
			if strings.TrimSpace(cfg.ScenarioPort) != "" {
				portCandidates = append(portCandidates, cfg.ScenarioPort)
			}

			resolvedURL, _, err := scenarioport.ResolveURL(ctx, scenarioName, cfg.ScenarioPath, portCandidates...)
			if err != nil {
				return Instruction{}, fmt.Errorf("navigate node %s failed to resolve scenario %s: %w", step.NodeID, scenarioName, err)
			}
			base.Params.URL = resolvedURL
			base.Params.Scenario = scenarioName
			base.Params.ScenarioPath = strings.TrimSpace(cfg.ScenarioPath)
			base.Params.DestinationType = "scenario"
			log.Printf("[BAS navigate] node=%s scenario=%s path=%s url=%s", step.NodeID, scenarioName, strings.TrimSpace(cfg.ScenarioPath), resolvedURL)
		} else {
			trimmedURL := strings.TrimSpace(cfg.URL)
			if trimmedURL == "" {
				return Instruction{}, fmt.Errorf("navigate node %s missing url", step.NodeID)
			}
			base.Params.URL = trimmedURL
			base.Params.DestinationType = "url"
		}
		if wait := strings.TrimSpace(cfg.WaitUntil); wait != "" {
			base.Params.WaitUntil = wait
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepWait:
		var cfg waitConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("wait node %s has invalid data: %w", step.NodeID, err)
		}
		waitType := strings.ToLower(strings.TrimSpace(firstNonEmpty(cfg.Type, cfg.WaitType)))
		// "duration" is an alias for "time" - accept both
		if waitType == "" || waitType == "time" || waitType == "duration" {
			base.Params.WaitType = "time"
			if cfg.Duration > 0 {
				base.Params.DurationMs = cfg.Duration
			} else {
				base.Params.DurationMs = 1000
			}
		} else if waitType == "element" {
			base.Params.WaitType = "element"
			base.Params.Selector = strings.TrimSpace(cfg.Selector)
			base.Params.TimeoutMs = cfg.TimeoutMs
			if base.Params.Selector == "" {
				return Instruction{}, fmt.Errorf("wait node %s requires selector for element wait", step.NodeID)
			}
		} else if waitType == "navigation" {
			base.Params.WaitType = "navigation"
			if cfg.TimeoutMs > 0 {
				base.Params.TimeoutMs = cfg.TimeoutMs
			}
		} else {
			return Instruction{}, fmt.Errorf("wait node %s has unsupported type %q", step.NodeID, waitType)
		}
	case compiler.StepScreenshot:
		var cfg screenshotConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("screenshot node %s has invalid data: %w", step.NodeID, err)
		}
		base.Params.Name = cfg.Name
		base.Params.ViewportWidth = cfg.ViewportWidth
		base.Params.ViewportHeight = cfg.ViewportHeight
		base.Params.WaitForMs = cfg.WaitForMs
		base.Params.FullPage = cfg.FullPage
		if trimmed := strings.TrimSpace(cfg.FocusSelector); trimmed != "" {
			base.Params.FocusSelector = trimmed
		}
		if selectors := normalizeStringSlice(cfg.HighlightSelectors); len(selectors) > 0 {
			base.Params.HighlightSelectors = selectors
		}
		if color := strings.TrimSpace(cfg.HighlightColor); color != "" {
			base.Params.HighlightColor = color
		}
		if cfg.HighlightPadding > 0 {
			base.Params.HighlightPadding = cfg.HighlightPadding
		}
		if cfg.HighlightBorderRadius > 0 {
			base.Params.HighlightBorderRadius = cfg.HighlightBorderRadius
		}
		if selectors := normalizeStringSlice(cfg.MaskSelectors); len(selectors) > 0 {
			base.Params.MaskSelectors = selectors
		}
		if cfg.MaskOpacity > 0 {
			base.Params.MaskOpacity = cfg.MaskOpacity
		}
		if background := strings.TrimSpace(cfg.Background); background != "" {
			base.Params.Background = background
		}
		if cfg.ZoomFactor > 0 {
			base.Params.ZoomFactor = cfg.ZoomFactor
		}
		if cfg.CaptureDomSnapshot {
			base.Params.CaptureDomSnapshot = true
		} else if capture, ok := getBoolParam(step.Params, "captureDomSnapshot", "capture_dom_snapshot", "captureDom"); ok && capture {
			base.Params.CaptureDomSnapshot = true
		}
	case compiler.StepClick:
		var cfg clickConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("click node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("click node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		if button := strings.ToLower(strings.TrimSpace(cfg.Button)); button != "" && button != "left" {
			base.Params.Button = button
		}
		if cfg.ClickCount > 1 {
			base.Params.ClickCount = cfg.ClickCount
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		if trimmed := strings.TrimSpace(cfg.WaitForSelector); trimmed != "" {
			base.Params.WaitForSelector = trimmed
		}
		preconditionSelector := base.Params.WaitForSelector
		if preconditionSelector == "" {
			preconditionSelector = selector
		}
		applyDefaultInteractiveResilience(&base, preconditionSelector, resilienceState)
	case compiler.StepFocus, compiler.StepBlur:
		// [REQ:BAS-NODE-FOCUS-INPUT] [REQ:BAS-NODE-BLUR-VALIDATION]
		var cfg focusConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("%s node %s has invalid data: %w", step.Type, step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("%s node %s missing selector", step.Type, step.NodeID)
		}
		base.Params.Selector = selector
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepTypeInput:
		var cfg typeConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("type node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("type node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		base.Params.Text = cfg.Text
		if cfg.DelayMs > 0 {
			base.Params.DelayMs = cfg.DelayMs
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.Clear != nil {
			base.Params.Clear = cfg.Clear
		}
		if cfg.Submit != nil {
			base.Params.Submit = cfg.Submit
		}
		applyDefaultInteractiveResilience(&base, selector, resilienceState)
	case compiler.StepShortcut:
		var cfg shortcutConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("shortcut node %s has invalid data: %w", step.NodeID, err)
		}

		combos := collectShortcutCombos(cfg.Shortcuts, cfg.Sequence, cfg.Keys, cfg.Shortcut)
		if len(combos) == 0 {
			return Instruction{}, fmt.Errorf("shortcut node %s missing shortcuts", step.NodeID)
		}

		normalized := make([]string, 0, len(combos))
		for _, combo := range combos {
			if normalizedCombo := normalizeShortcutCombo(combo); normalizedCombo != "" {
				normalized = append(normalized, normalizedCombo)
			}
		}

		if len(normalized) == 0 {
			return Instruction{}, fmt.Errorf("shortcut node %s produced no valid shortcuts", step.NodeID)
		}

		base.Params.ShortcutKeys = normalized
		if cfg.DelayMs > 0 {
			base.Params.ShortcutDelayMs = cfg.DelayMs
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if trimmed := strings.TrimSpace(cfg.FocusSelector); trimmed != "" {
			base.Params.FocusSelector = trimmed
		}
	case compiler.StepKeyboard:
		// [REQ:BAS-NODE-KEYBOARD-DISPATCH]
		var cfg keyboardConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("keyboard node %s has invalid data: %w", step.NodeID, err)
		}

		// Support multiple keyboard input formats: keys (array), key (singular), or sequence
		var keyValue string
		if len(cfg.Keys) > 0 {
			// Modern array format - use first key
			keyValue = strings.TrimSpace(cfg.Keys[0])
		} else if trimmed := strings.TrimSpace(cfg.Key); trimmed != "" {
			// Legacy singular format (deprecated but still supported)
			keyValue = trimmed
		} else if trimmed := strings.TrimSpace(cfg.Sequence); trimmed != "" {
			// Sequence format - use entire sequence
			keyValue = trimmed
		}

		if keyValue == "" {
			return Instruction{}, fmt.Errorf("keyboard node %s missing key input (expected keys[], key, or sequence)", step.NodeID)
		}

		eventType := strings.ToLower(strings.TrimSpace(cfg.EventType))
		switch eventType {
		case "keydown", "keyup":
		default:
			eventType = "keypress"
		}
		base.Params.KeyValue = keyValue
		base.Params.KeyEventType = eventType
		if modifiers := collectKeyboardModifiers(cfg.Modifiers); len(modifiers) > 0 {
			base.Params.KeyModifiers = modifiers
		}
		if cfg.DelayMs > 0 {
			base.Params.DelayMs = cfg.DelayMs
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
	case compiler.StepHover:
		// [REQ:BAS-NODE-HOVER-INTERACTION]
		var cfg hoverConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("hover node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("hover node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		base.Params.MovementSteps = clampHoverSteps(cfg.Steps)
		base.Params.DurationMs = clampHoverDuration(cfg.DurationMs)
		applyDefaultInteractiveResilience(&base, selector, resilienceState)
	case compiler.StepDragDrop:
		// [REQ:BAS-NODE-DRAG-DROP]
		var cfg dragDropConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("dragDrop node %s has invalid data: %w", step.NodeID, err)
		}
		source := strings.TrimSpace(cfg.SourceSelector)
		if source == "" {
			return Instruction{}, fmt.Errorf("dragDrop node %s missing source selector", step.NodeID)
		}
		target := strings.TrimSpace(cfg.TargetSelector)
		if target == "" {
			return Instruction{}, fmt.Errorf("dragDrop node %s missing target selector", step.NodeID)
		}
		base.Params.DragSourceSelector = source
		base.Params.DragTargetSelector = target
		base.Params.DragHoldMs = clampDragHold(cfg.HoldMs)
		base.Params.DragSteps = clampDragSteps(cfg.Steps)
		base.Params.DragDurationMs = clampDragDuration(cfg.DurationMs)
		base.Params.DragOffsetX = clampDragOffset(cfg.OffsetX)
		base.Params.DragOffsetY = clampDragOffset(cfg.OffsetY)
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		applyDefaultInteractiveResilience(&base, source, resilienceState)
	case compiler.StepScroll:
		// [REQ:BAS-NODE-SCROLL-NAVIGATION]
		var cfg scrollConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("scroll node %s has invalid data: %w", step.NodeID, err)
		}
		scrollType := normalizeScrollType(cfg.ScrollType)
		base.Params.ScrollType = scrollType
		behavior := normalizeScrollBehavior(cfg.Behavior)
		base.Params.ScrollBehavior = behavior
		if direction := normalizeScrollDirection(cfg.Direction); direction != "" {
			base.Params.ScrollDirection = direction
		}
		base.Params.ScrollAmount = clampScrollAmount(cfg.Amount)
		base.Params.ScrollX = clampScrollCoordinate(cfg.X)
		base.Params.ScrollY = clampScrollCoordinate(cfg.Y)
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		selector := strings.TrimSpace(cfg.Selector)
		targetSelector := strings.TrimSpace(cfg.TargetSelector)
		scrollPreconditionSelector := ""
		switch scrollType {
		case "element":
			if selector == "" {
				return Instruction{}, fmt.Errorf("scroll node %s requires selector for element mode", step.NodeID)
			}
			base.Params.Selector = selector
			scrollPreconditionSelector = selector
		case "position":
			// coordinates already captured
		case "untilVisible":
			if targetSelector == "" {
				targetSelector = selector
			}
			if targetSelector == "" {
				return Instruction{}, fmt.Errorf("scroll node %s requires target selector for untilVisible mode", step.NodeID)
			}
			base.Params.ScrollTargetSelector = targetSelector
			if base.Params.ScrollDirection == "" {
				base.Params.ScrollDirection = "down"
			}
			scrollPreconditionSelector = targetSelector
			attempts := clampScrollAttempts(cfg.MaxScrolls)
			if attempts == 0 {
				attempts = defaultScrollAttempts
			}
			base.Params.ScrollMaxAttempts = attempts
		default:
			if base.Params.ScrollDirection == "" {
				base.Params.ScrollDirection = "down"
			}
		}
		applyDefaultInteractiveResilience(&base, scrollPreconditionSelector, resilienceState)
	case compiler.StepSelect:
		// [REQ:BAS-NODE-SELECT-DROPDOWN]
		var cfg selectConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("select node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("select node %s missing selector", step.NodeID)
		}
		mode := normalizeSelectMode(cfg.SelectBy)
		value := strings.TrimSpace(cfg.Value)
		text := strings.TrimSpace(cfg.Text)
		values := normalizeStringSlice(cfg.Values)
		indexProvided := hasParam(step.Params, "index", "optionIndex")
		multi := cfg.Multiple
		if mode == "" {
			switch {
			case multi && len(values) > 0:
				mode = "value"
			case value != "":
				mode = "value"
			case text != "":
				mode = "text"
			case indexProvided:
				mode = "index"
			default:
				mode = "value"
			}
		}

		if multi && mode == "index" {
			return Instruction{}, fmt.Errorf("select node %s cannot use index mode with multi-select", step.NodeID)
		}

		switch mode {
		case "value":
			if multi {
				if len(values) == 0 && value != "" {
					values = []string{value}
				}
				if len(values) == 0 {
					return Instruction{}, fmt.Errorf("select node %s requires values for multi-select", step.NodeID)
				}
			} else {
				if value == "" {
					if len(values) > 0 {
						value = values[0]
					}
				}
				if value == "" {
					return Instruction{}, fmt.Errorf("select node %s missing value", step.NodeID)
				}
			}
		case "text":
			if multi {
				if len(values) == 0 && text != "" {
					values = []string{text}
				}
				if len(values) == 0 {
					return Instruction{}, fmt.Errorf("select node %s requires text values for multi-select", step.NodeID)
				}
			} else {
				if text == "" {
					if len(values) > 0 {
						text = values[0]
					} else if value != "" {
						text = value
					}
				}
				if text == "" {
					return Instruction{}, fmt.Errorf("select node %s missing text target", step.NodeID)
				}
			}
		case "index":
			if !indexProvided {
				return Instruction{}, fmt.Errorf("select node %s missing index", step.NodeID)
			}
			if cfg.Index < 0 {
				return Instruction{}, fmt.Errorf("select node %s requires non-negative index", step.NodeID)
			}
		default:
			return Instruction{}, fmt.Errorf("select node %s has unsupported select mode %q", step.NodeID, cfg.SelectBy)
		}

		base.Params.Selector = selector
		base.Params.SelectionMode = mode
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		if multi {
			base.Params.MultiSelect = true
			if len(values) > 0 {
				base.Params.OptionValues = values
			}
		} else {
			if mode == "value" && value != "" {
				base.Params.OptionValue = value
			}
			if mode == "text" && text != "" {
				base.Params.OptionText = text
			}
			if mode == "index" {
				base.Params.OptionIndex = cfg.Index
			}
		}
		applyDefaultInteractiveResilience(&base, selector, resilienceState)
	case compiler.StepConditional:
		var cfg conditionalConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("conditional node %s has invalid data: %w", step.NodeID, err)
		}
		mode := strings.ToLower(strings.TrimSpace(cfg.ConditionType))
		if mode == "" {
			mode = "expression"
		}
		base.Params.ConditionType = mode
		var pollInterval int
		if cfg.PollIntervalMs > 0 {
			pollInterval = clampInt(cfg.PollIntervalMs, minConditionalPollInterval, maxConditionalPollInterval)
		}
		if pollInterval > 0 {
			base.Params.ConditionPollIntervalMs = pollInterval
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = clampInt(cfg.TimeoutMs, minConditionalTimeoutMs, maxConditionalTimeoutMs)
		} else {
			base.Params.TimeoutMs = defaultConditionalTimeoutMs
		}

		switch mode {
		case "expression", "script", "js":
			expression := strings.TrimSpace(cfg.Expression)
			if expression == "" {
				return Instruction{}, fmt.Errorf("conditional node %s missing expression", step.NodeID)
			}
			base.Params.ConditionExpression = expression
		case "element", "selector":
			selector := strings.TrimSpace(cfg.Selector)
			if selector == "" {
				return Instruction{}, fmt.Errorf("conditional node %s missing selector", step.NodeID)
			}
			base.Params.ConditionSelector = selector
		case "variable":
			variableName := strings.TrimSpace(cfg.Variable)
			if variableName == "" {
				return Instruction{}, fmt.Errorf("conditional node %s missing variable name", step.NodeID)
			}
			operator := strings.ToLower(strings.TrimSpace(cfg.Operator))
			if operator == "" {
				operator = "equals"
			}
			base.Params.ConditionVariable = variableName
			base.Params.ConditionOperator = operator
			if cfg.Value != nil {
				base.Params.ConditionValue = cfg.Value
			} else if raw, ok := step.Params["value"]; ok {
				base.Params.ConditionValue = raw
			}
		default:
			return Instruction{}, fmt.Errorf("conditional node %s has unsupported condition type %q", step.NodeID, cfg.ConditionType)
		}

		if cfg.Negate {
			negate := true
			base.Params.ConditionNegate = &negate
		}
	case compiler.StepLoop:
		var cfg loopConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("loop node %s has invalid data: %w", step.NodeID, err)
		}
		loopType := normalizeLoopType(cfg.LoopType)
		if loopType == "" {
			return Instruction{}, fmt.Errorf("loop node %s requires a loopType", step.NodeID)
		}
		params := base.Params
		params.LoopType = loopType
		params.LoopMaxIterations = clampLoopIterations(cfg.MaxIterations)
		params.LoopItemVariable = normalizeLoopVariable(cfg.ItemVariable, defaultLoopItemVariable)
		params.LoopIndexVariable = normalizeLoopVariable(cfg.IndexVariable, defaultLoopIndexVariable)
		params.LoopIterationTimeoutMs = clampLoopIterationTimeout(cfg.IterationTimeoutMs)
		params.LoopTotalTimeoutMs = clampLoopTotalTimeout(cfg.TotalTimeoutMs)
		switch loopType {
		case "foreach":
			source := strings.TrimSpace(cfg.ArraySource)
			// Permit inline loopItems when arraySource is not provided to align with new executor.
			if source != "" {
				params.LoopArraySource = source
			} else if len(cfg.LoopItems) > 0 {
				params.LoopItems = cfg.LoopItems
			} else {
				return Instruction{}, fmt.Errorf("loop node %s requires an arraySource or loopItems for forEach loops", step.NodeID)
			}
		case "repeat":
			if cfg.Count <= 0 {
				return Instruction{}, fmt.Errorf("loop node %s requires count > 0 for repeat loops", step.NodeID)
			}
			params.LoopCount = cfg.Count
			if cfg.Count > params.LoopMaxIterations {
				params.LoopMaxIterations = clampLoopIterations(cfg.Count)
			}
		case "while":
			conditionType := normalizeLoopConditionType(cfg.ConditionType)
			if conditionType == "" {
				conditionType = "variable"
			}
			params.LoopConditionType = conditionType
			switch conditionType {
			case "expression":
				expr := strings.TrimSpace(cfg.ConditionExpression)
				if expr == "" {
					return Instruction{}, fmt.Errorf("loop node %s requires conditionExpression for expression-based while loops", step.NodeID)
				}
				params.LoopConditionExpression = expr
			case "variable":
				variable := strings.TrimSpace(cfg.ConditionVariable)
				if variable == "" {
					return Instruction{}, fmt.Errorf("loop node %s requires conditionVariable for variable-based while loops", step.NodeID)
				}
				params.LoopConditionVariable = variable
				operator := normalizeLoopOperator(cfg.ConditionOperator)
				params.LoopConditionOperator = operator
				if cfg.ConditionValue != nil {
					params.LoopConditionValue = cfg.ConditionValue
				}
			default:
				return Instruction{}, fmt.Errorf("loop node %s has unsupported conditionType %q", step.NodeID, cfg.ConditionType)
			}
		default:
			return Instruction{}, fmt.Errorf("loop node %s has unsupported loopType %q", step.NodeID, loopType)
		}
		base.Params = params
		return base, nil
	case compiler.StepWorkflowCall:
		var cfg workflowCallConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("workflow call node %s has invalid data: %w", step.NodeID, err)
		}
		workflowID := strings.TrimSpace(cfg.WorkflowID)
		inlineDefinition := cfg.InlineDefinition
		if len(inlineDefinition) == 0 {
			inlineDefinition = nil
		}
		if workflowID == "" && inlineDefinition == nil {
			return Instruction{}, fmt.Errorf("workflow call node %s requires workflowId or workflowDefinition", step.NodeID)
		}
		waitForCompletion := true
		if cfg.WaitForCompletion != nil {
			waitForCompletion = *cfg.WaitForCompletion
		}
		if !waitForCompletion {
			return Instruction{}, fmt.Errorf("workflow call node %s must wait for completion (async execution not supported yet)", step.NodeID)
		}
		if workflowID != "" {
			base.Params.WorkflowCallID = workflowID
		}
		if trimmedName := strings.TrimSpace(cfg.WorkflowName); trimmedName != "" {
			base.Params.WorkflowCallName = trimmedName
		}
		base.Params.WorkflowCallWait = &waitForCompletion
		if cleaned := sanitizeWorkflowCallParams(cfg.Parameters); len(cleaned) > 0 {
			base.Params.WorkflowCallParams = cleaned
		}
		if outputs := normalizeWorkflowCallOutputs(cfg.OutputMapping); len(outputs) > 0 {
			base.Params.WorkflowCallOutputs = outputs
		}
		if inlineDefinition != nil {
			base.Params.WorkflowCallDefinition = inlineDefinition
		}
	case compiler.StepRotate:
		// [REQ:BAS-NODE-ROTATE-MOBILE]
		var cfg rotateConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("rotate node %s has invalid data: %w", step.NodeID, err)
		}
		orientation, err := normalizeRotateOrientation(cfg.Orientation)
		if err != nil {
			return Instruction{}, fmt.Errorf("rotate node %s has invalid orientation: %w", step.NodeID, err)
		}
		angle := defaultRotateAngleFor(orientation)
		if hasParam(step.Params, "angle") {
			normalizedAngle, ok := normalizeRotateAngle(cfg.Angle)
			if !ok {
				return Instruction{}, fmt.Errorf("rotate node %s has unsupported angle %d", step.NodeID, cfg.Angle)
			}
			if !angleMatchesOrientation(orientation, normalizedAngle) {
				return Instruction{}, fmt.Errorf("rotate node %s angle %d does not match %s orientation", step.NodeID, normalizedAngle, orientation)
			}
			angle = normalizedAngle
		}
		base.Params.RotateOrientation = orientation
		base.Params.RotateAngle = angle
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepGesture:
		// [REQ:BAS-NODE-GESTURE-MOBILE]
		var cfg gestureConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("gesture node %s has invalid data: %w", step.NodeID, err)
		}
		base.Params.GestureType = normalizeGestureType(cfg.GestureType)
		base.Params.GestureDirection = normalizeGestureDirection(cfg.Direction)
		base.Params.GestureSelector = strings.TrimSpace(cfg.Selector)
		base.Params.GestureDistance = clampGestureDistance(cfg.Distance)
		base.Params.GestureScale = clampGestureScale(cfg.Scale)
		base.Params.GestureDurationMs = clampGestureDuration(cfg.DurationMs)
		base.Params.GestureHoldMs = clampGestureHold(cfg.HoldMs)
		base.Params.GestureSteps = clampGestureSteps(cfg.Steps)
		base.Params.GestureStartX = clampGestureCoordinate(cfg.StartX)
		base.Params.GestureStartY = clampGestureCoordinate(cfg.StartY)
		base.Params.GestureEndX = clampGestureCoordinate(cfg.EndX)
		base.Params.GestureEndY = clampGestureCoordinate(cfg.EndY)
		startXProvided := hasParam(step.Params, "startX")
		startYProvided := hasParam(step.Params, "startY")
		endXProvided := hasParam(step.Params, "endX")
		endYProvided := hasParam(step.Params, "endY")
		base.Params.GestureHasStartX = startXProvided
		base.Params.GestureHasStartY = startYProvided
		base.Params.GestureHasEndX = endXProvided
		base.Params.GestureHasEndY = endYProvided
		base.Params.GestureHasStart = startXProvided || startYProvided
		base.Params.GestureHasEnd = endXProvided || endYProvided
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
	case compiler.StepTabSwitch:
		var cfg tabSwitchConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("tabSwitch node %s has invalid data: %w", step.NodeID, err)
		}
		mode := strings.ToLower(strings.TrimSpace(cfg.SwitchBy))
		if mode == "" {
			mode = "newest"
		}
		switch mode {
		case "index":
			if cfg.Index < 0 {
				return Instruction{}, fmt.Errorf("tabSwitch node %s requires non-negative index", step.NodeID)
			}
			base.Params.TabIndex = cfg.Index
		case "title":
			title := strings.TrimSpace(cfg.TitleMatch)
			if title == "" {
				return Instruction{}, fmt.Errorf("tabSwitch node %s requires titleMatch when switchBy=title", step.NodeID)
			}
			base.Params.TabTitleMatch = title
		case "url":
			pattern := strings.TrimSpace(cfg.URLMatch)
			if pattern == "" {
				return Instruction{}, fmt.Errorf("tabSwitch node %s requires urlMatch when switchBy=url", step.NodeID)
			}
			base.Params.TabURLMatch = pattern
		case "newest", "oldest":
			// no-op
		default:
			return Instruction{}, fmt.Errorf("tabSwitch node %s has unsupported switchBy %q", step.NodeID, cfg.SwitchBy)
		}
		base.Params.TabSwitchBy = mode
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForNew {
			base.Params.TabWaitForNew = true
		}
		if cfg.CloseOld {
			base.Params.TabCloseOld = true
		}
	case compiler.StepFrameSwitch:
		// [REQ:BAS-NODE-FRAME-SWITCH]
		var cfg frameSwitchConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("frameSwitch node %s has invalid data: %w", step.NodeID, err)
		}
		mode := strings.ToLower(strings.TrimSpace(cfg.SwitchBy))
		if mode == "" {
			mode = "selector"
		}
		switch mode {
		case "selector":
			selector := strings.TrimSpace(cfg.Selector)
			if selector == "" {
				return Instruction{}, fmt.Errorf("frameSwitch node %s requires selector when switchBy=selector", step.NodeID)
			}
			base.Params.FrameSelector = selector
		case "index":
			if cfg.Index < 0 {
				return Instruction{}, fmt.Errorf("frameSwitch node %s requires non-negative index", step.NodeID)
			}
			base.Params.FrameIndex = cfg.Index
		case "name":
			name := strings.TrimSpace(cfg.Name)
			if name == "" {
				return Instruction{}, fmt.Errorf("frameSwitch node %s requires name when switchBy=name", step.NodeID)
			}
			base.Params.FrameName = name
		case "url":
			pattern := strings.TrimSpace(cfg.URLMatch)
			if pattern == "" {
				return Instruction{}, fmt.Errorf("frameSwitch node %s requires urlMatch when switchBy=url", step.NodeID)
			}
			base.Params.FrameURLMatch = pattern
		case "parent", "main":
			// no-op
		default:
			return Instruction{}, fmt.Errorf("frameSwitch node %s has unsupported switchBy %q", step.NodeID, cfg.SwitchBy)
		}
		base.Params.FrameSwitchBy = mode
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
	case compiler.StepSetCookie:
		var cfg setCookieConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("setCookie node %s has invalid data: %w", step.NodeID, err)
		}
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			return Instruction{}, fmt.Errorf("setCookie node %s requires name", step.NodeID)
		}
		value := cfg.Value
		if value == "" {
			return Instruction{}, fmt.Errorf("setCookie node %s requires value", step.NodeID)
		}
		targetURL := strings.TrimSpace(cfg.URL)
		domain := strings.TrimSpace(cfg.Domain)
		if targetURL == "" && domain == "" {
			return Instruction{}, fmt.Errorf("setCookie node %s requires url or domain", step.NodeID)
		}
		path := strings.TrimSpace(cfg.Path)
		if path == "" && domain != "" {
			path = "/"
		}
		base.Params.CookieName = name
		base.Params.CookieValue = value
		if targetURL != "" {
			base.Params.CookieURL = targetURL
		}
		if domain != "" {
			base.Params.CookieDomain = domain
		}
		if path != "" {
			base.Params.CookiePath = path
		}
		if sameSite := normalizeCookieSameSite(cfg.SameSite); sameSite != "" {
			base.Params.CookieSameSite = sameSite
		}
		if cfg.Secure != nil {
			base.Params.CookieSecure = cfg.Secure
		}
		if cfg.HTTPOnly != nil {
			base.Params.CookieHTTPOnly = cfg.HTTPOnly
		}
		if expires := strings.TrimSpace(cfg.ExpiresAt); expires != "" {
			base.Params.CookieExpiresAt = expires
		}
		ttl := cfg.TTLSeconds
		if ttl == 0 && cfg.MaxAge > 0 {
			ttl = cfg.MaxAge
		}
		if ttl > 0 {
			base.Params.CookieTTLSeconds = ttl
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepGetCookie:
		var cfg getCookieConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("getCookie node %s has invalid data: %w", step.NodeID, err)
		}
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			return Instruction{}, fmt.Errorf("getCookie node %s requires name", step.NodeID)
		}
		base.Params.CookieName = name
		if url := strings.TrimSpace(cfg.URL); url != "" {
			base.Params.CookieURL = url
		}
		if domain := strings.TrimSpace(cfg.Domain); domain != "" {
			base.Params.CookieDomain = domain
		}
		if path := strings.TrimSpace(cfg.Path); path != "" {
			base.Params.CookiePath = path
		}
		base.Params.CookieResultFormat = normalizeCookieResultFormat(cfg.ResultAs)
		store := strings.TrimSpace(firstNonEmpty(cfg.StoreAs, cfg.StoreResult))
		if store == "" {
			store = name
		}
		base.Params.StoreResult = store
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepClearCookie:
		var cfg clearCookieConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("clearCookie node %s has invalid data: %w", step.NodeID, err)
		}
		name := strings.TrimSpace(cfg.Name)
		targetURL := strings.TrimSpace(cfg.URL)
		domain := strings.TrimSpace(cfg.Domain)
		path := strings.TrimSpace(cfg.Path)
		clearAll := cfg.ClearAll || (name == "" && targetURL == "" && domain == "")
		if clearAll {
			base.Params.CookieClearAll = true
		} else {
			if name == "" {
				return Instruction{}, fmt.Errorf("clearCookie node %s requires name when not clearing all", step.NodeID)
			}
			if targetURL == "" && domain == "" {
				return Instruction{}, fmt.Errorf("clearCookie node %s requires url or domain when targeting a specific cookie", step.NodeID)
			}
			base.Params.CookieName = name
		}
		if targetURL != "" {
			base.Params.CookieURL = targetURL
		}
		if domain != "" {
			base.Params.CookieDomain = domain
		}
		if path != "" {
			base.Params.CookiePath = path
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepSetStorage:
		var cfg setStorageConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("setStorage node %s has invalid data: %w", step.NodeID, err)
		}
		key := strings.TrimSpace(cfg.Key)
		if key == "" {
			return Instruction{}, fmt.Errorf("setStorage node %s requires key", step.NodeID)
		}
		base.Params.StorageType = normalizeStorageType(cfg.StorageType)
		base.Params.StorageKey = key
		base.Params.StorageValue = cfg.Value
		base.Params.StorageValueType = normalizeStorageValueType(cfg.ValueType)
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepGetStorage:
		var cfg getStorageConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("getStorage node %s has invalid data: %w", step.NodeID, err)
		}
		key := strings.TrimSpace(cfg.Key)
		if key == "" {
			return Instruction{}, fmt.Errorf("getStorage node %s requires key", step.NodeID)
		}
		base.Params.StorageType = normalizeStorageType(cfg.StorageType)
		base.Params.StorageKey = key
		base.Params.StorageResultFormat = normalizeStorageResultFormat(cfg.ResultAs)
		store := strings.TrimSpace(firstNonEmpty(cfg.StoreAs, cfg.StoreResult))
		if store == "" {
			store = key
		}
		base.Params.StoreResult = store
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepClearStorage:
		var cfg clearStorageConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("clearStorage node %s has invalid data: %w", step.NodeID, err)
		}
		base.Params.StorageType = normalizeStorageType(cfg.StorageType)
		key := strings.TrimSpace(cfg.Key)
		clearAll := cfg.ClearAll || key == ""
		if clearAll {
			base.Params.StorageClearAll = true
		} else {
			base.Params.StorageKey = key
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepNetworkMock:
		var cfg networkMockConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("networkMock node %s has invalid data: %w", step.NodeID, err)
		}
		pattern := strings.TrimSpace(cfg.URLPattern)
		if pattern == "" {
			return Instruction{}, fmt.Errorf("networkMock node %s requires urlPattern", step.NodeID)
		}
		mockType := normalizeNetworkMockType(cfg.MockType)
		if mockType == "" {
			mockType = "response"
		}
		base.Params.NetworkURLPattern = pattern
		base.Params.NetworkMockType = mockType
		if method := normalizeHTTPMethod(cfg.Method); method != "" {
			base.Params.NetworkMethod = method
		}
		if delay := clampNetworkDelayMs(cfg.DelayMs); delay > 0 {
			base.Params.NetworkDelayMs = delay
		}
		switch mockType {
		case "response":
			status := cfg.StatusCode
			if status == 0 {
				status = defaultNetworkMockStatus
			}
			base.Params.NetworkStatusCode = clampHTTPStatusCode(status)
			if normalized := normalizeHeaderMap(cfg.Headers); len(normalized) > 0 {
				base.Params.NetworkHeaders = normalized
			}
			if cfg.Body != nil {
				base.Params.NetworkBody = cfg.Body
			}
		case "abort":
			base.Params.NetworkAbortReason = normalizeNetworkAbortReason(cfg.AbortReason)
		case "delay":
			if base.Params.NetworkDelayMs == 0 {
				return Instruction{}, fmt.Errorf("networkMock node %s requires delayMs for delay mocks", step.NodeID)
			}
		default:
			return Instruction{}, fmt.Errorf("networkMock node %s has unsupported mockType %q", step.NodeID, mockType)
		}
	case compiler.StepExtract:
		var cfg extractConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("extract node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("extract node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		typeValue := strings.ToLower(strings.TrimSpace(cfg.ExtractType))
		if typeValue == "" {
			typeValue = "text"
		}
		base.Params.ExtractType = typeValue
		if strings.TrimSpace(cfg.Attribute) != "" {
			base.Params.Attribute = strings.TrimSpace(cfg.Attribute)
		}
		if cfg.AllMatches != nil {
			base.Params.AllMatches = cfg.AllMatches
		}
		if storeTarget := strings.TrimSpace(firstNonEmpty(cfg.StoreResult, cfg.StoreIn)); storeTarget != "" {
			base.Params.StoreResult = storeTarget
		}
	case compiler.StepEvaluate:
		// [REQ:BAS-NODE-SCRIPT-EXECUTE]
		var cfg evaluateConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("evaluate node %s has invalid data: %w", step.NodeID, err)
		}
		expression := strings.TrimSpace(cfg.Expression)
		if expression == "" {
			return Instruction{}, fmt.Errorf("evaluate node %s missing expression", step.NodeID)
		}
		base.Params.Expression = expression
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if storeName := strings.TrimSpace(cfg.StoreResult); storeName != "" {
			base.Params.StoreResult = storeName
		}
	case compiler.StepUploadFile:
		// [REQ:BAS-NODE-UPLOAD-FILE]
		var cfg uploadFileConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("uploadFile node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("uploadFile node %s missing selector", step.NodeID)
		}
		candidatePaths := make([]string, 0, len(cfg.FilePaths)+1)
		candidatePaths = append(candidatePaths, cfg.FilePaths...)
		if trimmedSingle := strings.TrimSpace(cfg.FilePath); trimmedSingle != "" {
			candidatePaths = append(candidatePaths, trimmedSingle)
		}
		validatedPaths := make([]string, 0, len(candidatePaths))
		for _, rawPath := range candidatePaths {
			trimmed := strings.TrimSpace(rawPath)
			if trimmed == "" {
				continue
			}
			cleaned := filepath.Clean(trimmed)
			if !filepath.IsAbs(cleaned) {
				return Instruction{}, fmt.Errorf("uploadFile node %s requires absolute file path, got %s", step.NodeID, trimmed)
			}
			info, err := os.Stat(cleaned)
			if err != nil {
				if os.IsNotExist(err) {
					return Instruction{}, fmt.Errorf("uploadFile node %s cannot find file %s", step.NodeID, cleaned)
				}
				return Instruction{}, fmt.Errorf("uploadFile node %s failed to read file %s: %w", step.NodeID, cleaned, err)
			}
			if info.IsDir() {
				return Instruction{}, fmt.Errorf("uploadFile node %s path %s is a directory", step.NodeID, cleaned)
			}
			validatedPaths = append(validatedPaths, cleaned)
		}
		if len(validatedPaths) == 0 {
			return Instruction{}, fmt.Errorf("uploadFile node %s requires at least one file path", step.NodeID)
		}
		base.Params.Selector = selector
		base.Params.FilePaths = validatedPaths
		if len(validatedPaths) == 1 {
			base.Params.FilePath = validatedPaths[0]
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepSetVariable:
		// [REQ:BAS-NODE-VARIABLE-SET]
		var cfg setVariableConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("setVariable node %s has invalid data: %w", step.NodeID, err)
		}
		variableName := strings.TrimSpace(cfg.Name)
		if variableName == "" {
			return Instruction{}, fmt.Errorf("setVariable node %s missing variable name", step.NodeID)
		}
		source := normalizeVariableSource(cfg.SourceType)
		if source == "" {
			return Instruction{}, fmt.Errorf("setVariable node %s missing sourceType", step.NodeID)
		}
		base.Params.VariableName = variableName
		base.Params.VariableSource = source
		if storeAs := strings.TrimSpace(cfg.StoreAs); storeAs != "" {
			base.Params.StoreResult = storeAs
		}
		switch source {
		case "static":
			value, err := convertStaticVariableValue(cfg.Value, cfg.ValueType)
			if err != nil {
				return Instruction{}, fmt.Errorf("setVariable node %s has invalid static value: %w", step.NodeID, err)
			}
			base.Params.VariableValue = value
		case "expression":
			expression := strings.TrimSpace(cfg.Expression)
			if expression == "" {
				return Instruction{}, fmt.Errorf("setVariable node %s requires expression for expression source", step.NodeID)
			}
			base.Params.Expression = expression
			if cfg.TimeoutMs > 0 {
				base.Params.TimeoutMs = cfg.TimeoutMs
			}
		case "extract":
			selector := strings.TrimSpace(cfg.Selector)
			if selector == "" {
				return Instruction{}, fmt.Errorf("setVariable node %s requires selector for extract source", step.NodeID)
			}
			extractType := normalizeExtractMode(cfg.ExtractType)
			base.Params.Selector = selector
			base.Params.ExtractType = extractType
			if attr := strings.TrimSpace(cfg.Attribute); attr != "" {
				base.Params.Attribute = attr
			}
			if cfg.AllMatches != nil {
				base.Params.AllMatches = cfg.AllMatches
			}
			if cfg.TimeoutMs > 0 {
				base.Params.TimeoutMs = cfg.TimeoutMs
			}
		default:
			return Instruction{}, fmt.Errorf("setVariable node %s has unsupported source %s", step.NodeID, source)
		}
	case compiler.StepUseVariable:
		// [REQ:BAS-NODE-VARIABLE-USE]
		var cfg useVariableConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("useVariable node %s has invalid data: %w", step.NodeID, err)
		}
		variableName := strings.TrimSpace(cfg.Name)
		if variableName == "" {
			return Instruction{}, fmt.Errorf("useVariable node %s missing variable name", step.NodeID)
		}
		storeAs := strings.TrimSpace(cfg.StoreAs)
		if storeAs == "" {
			storeAs = variableName
		}
		base.Params.VariableName = variableName
		base.Params.StoreResult = storeAs
		if trimmed := strings.TrimSpace(cfg.Transform); trimmed != "" {
			base.Params.VariableTransform = trimmed
		}
		if cfg.Required != nil {
			base.Params.VariableRequired = cfg.Required
		}
	case compiler.StepAssert:
		var cfg assertConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("assert node %s has invalid data: %w", step.NodeID, err)
		}

		mode := normalizeAssertMode(firstNonEmpty(cfg.AssertMode, cfg.Mode, cfg.Comparison))
		if mode == "" {
			mode = "exists"
		}

		switch mode {
		case "exists", "not_exists", "exists_or_not", "text_equals", "text_contains", "attribute_equals", "attribute_contains", "expression":
		default:
			return Instruction{}, fmt.Errorf("assert node %s has unsupported mode %q", step.NodeID, mode)
		}

		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" && mode != "expression" {
			return Instruction{}, fmt.Errorf("assert node %s requires selector for mode %s", step.NodeID, mode)
		}

		base.Params.AssertMode = mode
		if selector != "" {
			base.Params.Selector = selector
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.CaseSensitive != nil {
			base.Params.CaseSensitive = cfg.CaseSensitive
		}
		if cfg.Negate != nil {
			base.Params.Negate = cfg.Negate
		}
		if cfg.ContinueOnFailure != nil {
			base.Params.ContinueOnFailure = cfg.ContinueOnFailure
		}
		if trimmed := strings.TrimSpace(cfg.FailureMessage); trimmed != "" {
			base.Params.FailureMessage = trimmed
		}
		if trimmed := strings.TrimSpace(cfg.Expression); trimmed != "" {
			base.Params.Expression = trimmed
		}
		if attr := strings.TrimSpace(cfg.Attribute); attr != "" {
			base.Params.Attribute = attr
		}

		expected := cfg.ExpectedValue
		if expected == nil && cfg.Expected != nil {
			expected = cfg.Expected
		}
		if expected == nil && strings.TrimSpace(cfg.Text) != "" {
			expected = strings.TrimSpace(cfg.Text)
		}

		if needsExpectedValue(mode) && expected == nil {
			return Instruction{}, fmt.Errorf("assert node %s requires expected value for mode %s", step.NodeID, mode)
		}

		if expected != nil {
			base.Params.ExpectedValue = expected
		}
	case compiler.StepCustom:
		return Instruction{}, fmt.Errorf("custom node %s is not yet supported", step.NodeID)
	default:
		return Instruction{}, fmt.Errorf("step type %q is not supported", step.Type)
	}

	return base, nil
}

func clampHoverSteps(raw int) int {
	if raw < minHoverSteps {
		return defaultHoverSteps
	}
	if raw > maxHoverSteps {
		return maxHoverSteps
	}
	return raw
}

func clampHoverDuration(raw int) int {
	if raw < minHoverDurationMs {
		if raw == 0 {
			return defaultHoverDurationMs
		}
		return minHoverDurationMs
	}
	if raw > maxHoverDurationMs {
		return maxHoverDurationMs
	}
	return raw
}

func clampDragHold(raw int) int {
	if raw <= 0 {
		return defaultDragHoldMs
	}
	if raw > maxDragHoldMs {
		return maxDragHoldMs
	}
	return raw
}

func clampDragSteps(raw int) int {
	if raw <= 0 {
		return defaultDragSteps
	}
	if raw < minDragSteps {
		return minDragSteps
	}
	if raw > maxDragSteps {
		return maxDragSteps
	}
	return raw
}

func clampDragDuration(raw int) int {
	if raw <= 0 {
		return defaultDragDurationMs
	}
	if raw < minDragDurationMs {
		return minDragDurationMs
	}
	if raw > maxDragDurationMs {
		return maxDragDurationMs
	}
	return raw
}

func clampDragOffset(raw int) int {
	if raw == 0 {
		return 0
	}
	if raw < minDragOffset {
		return minDragOffset
	}
	if raw > maxDragOffset {
		return maxDragOffset
	}
	return raw
}

func normalizeScrollType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "element":
		return "element"
	case "position":
		return "position"
	case "untilvisible", "until_visible", "untilVisible":
		return "untilVisible"
	default:
		return "page"
	}
}

func normalizeScrollBehavior(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "smooth":
		return "smooth"
	default:
		return "auto"
	}
}

func normalizeCookieSameSite(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lax":
		return "lax"
	case "strict":
		return "strict"
	case "none", "no_restriction", "no-restriction":
		return "none"
	default:
		return ""
	}
}

func normalizeCookieResultFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "object", "full", "cookie":
		return "object"
	default:
		return "value"
	}
}

func normalizeStorageType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "session", "sessionstorage", "session-storage":
		return "sessionStorage"
	default:
		return "localStorage"
	}
}

func normalizeStorageValueType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json", "object":
		return "json"
	default:
		return "text"
	}
}

func normalizeStorageResultFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json", "object":
		return "json"
	default:
		return "text"
	}
}

func normalizeScrollDirection(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "up", "down", "left", "right", "top", "bottom":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func normalizeSelectMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "value", "values":
		return "value"
	case "text", "label", "innertext":
		return "text"
	case "index", "position":
		return "index"
	default:
		return ""
	}
}

func clampScrollAmount(raw int) int {
	if raw <= 0 {
		return defaultScrollAmount
	}
	if raw < minScrollAmount {
		return minScrollAmount
	}
	if raw > maxScrollAmount {
		return maxScrollAmount
	}
	return raw
}

func clampScrollCoordinate(raw int) int {
	if raw < minScrollCoordinate {
		return minScrollCoordinate
	}
	if raw > maxScrollCoordinate {
		return maxScrollCoordinate
	}
	return raw
}

func clampScrollAttempts(raw int) int {
	if raw <= 0 {
		return 0
	}
	if raw < minScrollAttempts {
		return minScrollAttempts
	}
	if raw > maxScrollAttempts {
		return maxScrollAttempts
	}
	return raw
}

func normalizeGestureType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "swipe":
		return "swipe"
	case "pinch":
		return "pinch"
	case "doubletap", "double_tap", "double-tap":
		return "doubleTap"
	case "longpress", "long_press", "long-press", "press":
		return "longPress"
	default:
		return "tap"
	}
}

func normalizeGestureDirection(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "up", "north":
		return "up"
	case "left", "west":
		return "left"
	case "right", "east":
		return "right"
	case "down", "south":
		return "down"
	default:
		return "down"
	}
}

func clampGestureDistance(raw int) int {
	if raw <= 0 {
		return defaultGestureDistance
	}
	if raw < minGestureDistance {
		return minGestureDistance
	}
	if raw > maxGestureDistance {
		return maxGestureDistance
	}
	return raw
}

func clampGestureDuration(raw int) int {
	if raw <= 0 {
		return defaultGestureDurationMs
	}
	if raw < minGestureDurationMs {
		return minGestureDurationMs
	}
	if raw > maxGestureDurationMs {
		return maxGestureDurationMs
	}
	return raw
}

func clampGestureHold(raw int) int {
	if raw <= 0 {
		return 0
	}
	if raw < minGestureDurationMs {
		return minGestureDurationMs
	}
	if raw > maxGestureDurationMs {
		return maxGestureDurationMs
	}
	return raw
}

func clampGestureSteps(raw int) int {
	if raw <= 0 {
		return defaultGestureSteps
	}
	if raw < minGestureSteps {
		return minGestureSteps
	}
	if raw > maxGestureSteps {
		return maxGestureSteps
	}
	return raw
}

func clampGestureScale(raw float64) float64 {
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 1
	}
	if raw < minGestureScale {
		return minGestureScale
	}
	if raw > maxGestureScale {
		return maxGestureScale
	}
	return raw
}

func clampGestureCoordinate(raw float64) int {
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0
	}
	value := int(math.Round(raw))
	if value < minScrollCoordinate {
		return minScrollCoordinate
	}
	if value > maxScrollCoordinate {
		return maxScrollCoordinate
	}
	return value
}

func normalizeRotateOrientation(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "", "portrait", "portrait-primary", "portrait_primary", "portraitprimary", "portraitsecondary", "portrait-secondary":
		return rotateOrientationPortrait, nil
	case "landscape", "landscape-primary", "landscape_primary", "landscapeprimary", "landscapesecondary", "landscape-secondary":
		return rotateOrientationLandscape, nil
	default:
		return "", fmt.Errorf("unsupported orientation %q", raw)
	}
}

func normalizeRotateAngle(raw int) (int, bool) {
	value := raw % 360
	if value < 0 {
		value += 360
	}
	switch value {
	case 0, 90, 180, 270:
		return value, true
	default:
		return 0, false
	}
}

func defaultRotateAngleFor(orientation string) int {
	switch orientation {
	case rotateOrientationLandscape:
		return 90
	default:
		return 0
	}
}

func angleMatchesOrientation(orientation string, angle int) bool {
	switch orientation {
	case rotateOrientationPortrait:
		return angle == 0 || angle == 180
	case rotateOrientationLandscape:
		return angle == 90 || angle == 270
	default:
		return true
	}
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

type resilienceConfigState struct {
	maxAttemptsConfigured bool
	delayConfigured       bool
	backoffConfigured     bool
}

func applyResilienceConfig(params map[string]any, instruction *Instruction) resilienceConfigState {
	state := resilienceConfigState{}
	if params == nil || instruction == nil {
		return state
	}
	raw, ok := params["resilience"]
	if !ok {
		return state
	}
	config, ok := normalizeMap(raw)
	if !ok || len(config) == 0 {
		return state
	}

	if maxAttempts, ok := toIntValue(config["maxAttempts"]); ok {
		if maxAttempts < 1 {
			maxAttempts = 1
		}
		instruction.Params.RetryAttempts = maxAttempts - 1
		state.maxAttemptsConfigured = true
	}
	if delayMs, ok := toIntValue(config["delayMs"]); ok && delayMs >= 0 {
		instruction.Params.RetryDelayMs = delayMs
		state.delayConfigured = true
	}
	if backoff, ok := toFloatValue(config["backoffFactor"]); ok && backoff > 0 {
		instruction.Params.RetryBackoffFactor = backoff
		state.backoffConfigured = true
	}

	if selector, ok := toStringValue(config["preconditionSelector"]); ok && selector != "" {
		instruction.Params.PreconditionSelector = selector
		if timeout, ok := toIntValue(config["preconditionTimeoutMs"]); ok && timeout >= 0 {
			instruction.Params.PreconditionTimeoutMs = timeout
		}
		if waitMs, ok := toIntValue(config["preconditionWaitMs"]); ok && waitMs > 0 {
			instruction.Params.PreconditionWaitMs = waitMs
		}
	}

	if selector, ok := toStringValue(config["successSelector"]); ok && selector != "" {
		instruction.Params.SuccessSelector = selector
		if timeout, ok := toIntValue(config["successTimeoutMs"]); ok && timeout >= 0 {
			instruction.Params.SuccessTimeoutMs = timeout
		}
		if waitMs, ok := toIntValue(config["successWaitMs"]); ok && waitMs > 0 {
			instruction.Params.SuccessWaitMs = waitMs
		}
	}

	return state
}

func applyDefaultInteractiveResilience(
	instruction *Instruction,
	fallbackSelector string,
	state resilienceConfigState,
) {
	if instruction == nil {
		return
	}
	if !state.maxAttemptsConfigured && instruction.Params.RetryAttempts <= 0 {
		instruction.Params.RetryAttempts = defaultResilienceMaxAttempts - 1
	}
	if !state.delayConfigured && instruction.Params.RetryDelayMs <= 0 {
		instruction.Params.RetryDelayMs = defaultResilienceRetryDelayMs
	}
	if !state.backoffConfigured && instruction.Params.RetryBackoffFactor <= 0 {
		instruction.Params.RetryBackoffFactor = defaultResilienceBackoffFactor
	}

	if trimmed := strings.TrimSpace(fallbackSelector); trimmed != "" && instruction.Params.PreconditionSelector == "" {
		instruction.Params.PreconditionSelector = trimmed
		if instruction.Params.PreconditionTimeoutMs <= 0 {
			timeout := instruction.Params.TimeoutMs
			if timeout <= 0 {
				timeout = defaultResiliencePreconditionTimeoutMs
			}
			instruction.Params.PreconditionTimeoutMs = timeout
		}
	}
}

func normalizeMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func toIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed), true
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func toFloatValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if parsed, err := v.Float64(); err == nil {
			return parsed, true
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func toStringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	case fmt.Stringer:
		trimmed := strings.TrimSpace(v.String())
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	default:
		return "", false
	}
}

func hasParam(params map[string]any, keys ...string) bool {
	if params == nil {
		return false
	}
	for _, key := range keys {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func collectKeyboardModifiers(spec keyboardModifierSpec) []string {
	modifiers := make([]string, 0, 4)
	if spec.Ctrl {
		modifiers = append(modifiers, "ctrl")
	}
	if spec.Shift {
		modifiers = append(modifiers, "shift")
	}
	if spec.Alt {
		modifiers = append(modifiers, "alt")
	}
	if spec.Meta {
		modifiers = append(modifiers, "meta")
	}
	if len(modifiers) == 0 {
		return nil
	}
	return modifiers
}

var shortcutKeyAliases = map[string]string{
	"ctrl":       "Control",
	"control":    "Control",
	"cmd":        "Meta",
	"command":    "Meta",
	"meta":       "Meta",
	"super":      "Meta",
	"windows":    "Meta",
	"win":        "Meta",
	"alt":        "Alt",
	"option":     "Alt",
	"shift":      "Shift",
	"enter":      "Enter",
	"return":     "Enter",
	"esc":        "Escape",
	"escape":     "Escape",
	"space":      "Space",
	"spacebar":   "Space",
	"tab":        "Tab",
	"backspace":  "Backspace",
	"delete":     "Delete",
	"del":        "Delete",
	"home":       "Home",
	"end":        "End",
	"pageup":     "PageUp",
	"pagedown":   "PageDown",
	"arrowup":    "ArrowUp",
	"arrowdown":  "ArrowDown",
	"arrowleft":  "ArrowLeft",
	"arrowright": "ArrowRight",
	"up":         "ArrowUp",
	"down":       "ArrowDown",
	"left":       "ArrowLeft",
	"right":      "ArrowRight",
	"plus":       "+",
}

func collectShortcutCombos(shortcuts, sequence, keys []string, single string) []string {
	combos := make([]string, 0, len(shortcuts)+len(sequence)+len(keys)+1)
	seen := make(map[string]struct{})
	add := func(values []string) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			combos = append(combos, trimmed)
		}
	}

	add(shortcuts)
	add(sequence)
	add(keys)

	if trimmed := strings.TrimSpace(single); trimmed != "" {
		if _, exists := seen[trimmed]; !exists {
			combos = append(combos, trimmed)
		}
	}

	return combos
}

func normalizeShortcutCombo(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	segments := strings.Split(trimmed, "+")
	result := make([]string, 0, len(segments))

	for _, segment := range segments {
		cleaned := strings.TrimSpace(segment)
		if cleaned == "" {
			continue
		}

		lower := strings.ToLower(cleaned)
		lower = strings.ReplaceAll(lower, "-", "")
		if alias, ok := shortcutKeyAliases[lower]; ok {
			result = append(result, alias)
			continue
		}

		if normalizedArrow := normalizeArrowKey(lower); normalizedArrow != "" {
			result = append(result, normalizedArrow)
			continue
		}

		if isFunctionKey(lower) {
			result = append(result, strings.ToUpper(lower))
			continue
		}

		if len(cleaned) == 1 {
			runeVal := []rune(cleaned)[0]
			if unicode.IsLetter(runeVal) {
				result = append(result, strings.ToUpper(cleaned))
				continue
			}
			if unicode.IsDigit(runeVal) || cleaned == "+" || cleaned == "/" || cleaned == "-" {
				result = append(result, cleaned)
				continue
			}
		}

		result = append(result, strings.ToUpper(lower[:1])+lower[1:])
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, "+")
}

func normalizeExtractMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "html":
		return "html"
	case "attribute":
		return "attribute"
	case "value":
		return "value"
	default:
		return "text"
	}
}

func normalizeVariableSource(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "static", "value", "text":
		return "static"
	case "expression", "script":
		return "expression"
	case "extract", "selector":
		return "extract"
	default:
		return ""
	}
}

func convertStaticVariableValue(value any, valueType string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "boolean", "bool":
		return coerceBool(value)
	case "number", "float", "int":
		return coerceNumber(value)
	case "json":
		return parseJSONValue(value)
	default:
		if str, ok := value.(string); ok {
			return str, nil
		}
		return value, nil
	}
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampNetworkDelayMs(value int) int {
	if value <= 0 {
		return 0
	}
	return clampInt(value, 0, maxNetworkDelayMs)
}

func clampHTTPStatusCode(value int) int {
	if value <= 0 {
		return defaultNetworkMockStatus
	}
	return clampInt(value, minHTTPStatusCode, maxHTTPStatusCode)
}

func normalizeHTTPMethod(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "*" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if lower == "any" || lower == "all" {
		return ""
	}
	return strings.ToUpper(trimmed)
}

func normalizeNetworkMockType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "response", "mock", "reply":
		return "response"
	case "abort", "fail", "block":
		return "abort"
	case "delay", "throttle", "slow":
		return "delay"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func normalizeNetworkAbortReason(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "aborted", "abort":
		return "Aborted"
	case "blocked", "blockedbyclient":
		return "BlockedByClient"
	case "blockedbyresponse":
		return "BlockedByResponse"
	case "timedout", "timeout":
		return "TimedOut"
	case "accessdenied":
		return "AccessDenied"
	case "connectionclosed":
		return "ConnectionClosed"
	case "connectionreset":
		return "ConnectionReset"
	case "connectionrefused":
		return "ConnectionRefused"
	case "connectionaborted":
		return "ConnectionAborted"
	case "connectionfailed":
		return "ConnectionFailed"
	case "namenotresolved":
		return "NameNotResolved"
	case "internetdisconnected":
		return "InternetDisconnected"
	case "addressunreachable":
		return "AddressUnreachable"
	default:
		return "Failed"
	}
}

func normalizeHeaderMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func coerceBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		if lower == "true" || lower == "1" {
			return true, nil
		}
		if lower == "false" || lower == "0" || lower == "" {
			return false, nil
		}
		return false, fmt.Errorf("value %q cannot be parsed as bool", typed)
	case float64:
		return typed != 0, nil
	case float32:
		return typed != 0, nil
	case int, int32, int64:
		return fmt.Sprintf("%v", typed) != "0", nil
	default:
		return false, fmt.Errorf("unsupported bool value type %T", value)
	}
}

func coerceNumber(value any) (any, error) {
	switch typed := value.(type) {
	case float64, float32, int, int32, int64:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, nil
		}
		if strings.Contains(trimmed, ".") {
			parsed, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return nil, err
			}
			return parsed, nil
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported number value type %T", value)
	}
}

func parseJSONValue(value any) (any, error) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, nil
		}
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		return value, nil
	}
}

func normalizeArrowKey(value string) string {
	switch value {
	case "arrowup", "up":
		return "ArrowUp"
	case "arrowdown", "down":
		return "ArrowDown"
	case "arrowleft", "left":
		return "ArrowLeft"
	case "arrowright", "right":
		return "ArrowRight"
	default:
		return ""
	}
}

func normalizeLoopType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "foreach", "for_each", "for-each":
		return "foreach"
	case "repeat", "count":
		return "repeat"
	case "while":
		return "while"
	default:
		return ""
	}
}

func clampLoopIterations(value int) int {
	if value <= 0 {
		return defaultLoopMaxIterations
	}
	if value > maxLoopMaxIterations {
		return maxLoopMaxIterations
	}
	return value
}

func clampLoopIterationTimeout(value int) int {
	if value <= 0 {
		return defaultLoopIterationTimeoutMs
	}
	if value < minLoopIterationTimeoutMs {
		return minLoopIterationTimeoutMs
	}
	if value > maxLoopIterationTimeoutMs {
		return maxLoopIterationTimeoutMs
	}
	return value
}

func clampLoopTotalTimeout(value int) int {
	if value <= 0 {
		return defaultLoopTotalTimeoutMs
	}
	if value < minLoopTotalTimeoutMs {
		return minLoopTotalTimeoutMs
	}
	if value > maxLoopTotalTimeoutMs {
		return maxLoopTotalTimeoutMs
	}
	return value
}

func normalizeLoopVariable(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func normalizeLoopConditionType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "variable":
		return "variable"
	case "expression", "script":
		return "expression"
	default:
		return ""
	}
}

func normalizeLoopOperator(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "truthy", "is_truthy":
		return "truthy"
	case "equals", "eq":
		return "equals"
	case "not_equals", "ne":
		return "not_equals"
	case "contains":
		return "contains"
	default:
		return value
	}
}

func sanitizeWorkflowCallParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	cleaned := make(map[string]any, len(params))
	for key, value := range params {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		cleaned[trimmed] = value
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func normalizeWorkflowCallOutputs(outputs map[string]string) map[string]string {
	if len(outputs) == 0 {
		return nil
	}
	cleaned := make(map[string]string, len(outputs))
	for source, target := range outputs {
		src := strings.TrimSpace(source)
		dst := strings.TrimSpace(target)
		if src == "" || dst == "" {
			continue
		}
		cleaned[src] = dst
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func isFunctionKey(value string) bool {
	if len(value) < 2 || value[0] != 'f' {
		return false
	}
	for _, r := range value[1:] {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func getIntParam(params map[string]any, keys ...string) (int, bool) {
	if params == nil {
		return 0, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case float32:
			return int(v), true
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return int(i), true
			}
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if i, err := strconv.Atoi(trimmed); err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

func getFloatParam(params map[string]any, keys ...string) (float64, bool) {
	if params == nil {
		return 0, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int32:
			return float64(v), true
		case int64:
			return float64(v), true
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f, true
			}
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func getBoolParam(params map[string]any, keys ...string) (bool, bool) {
	if params == nil {
		return false, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if parsed, err := strconv.ParseBool(trimmed); err == nil {
				return parsed, true
			}
		case int:
			return v != 0, true
		case int32:
			return v != 0, true
		case int64:
			return v != 0, true
		case float32:
			return v != 0, true
		case float64:
			return v != 0, true
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return i != 0, true
			}
		}
	}
	return false, false
}

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func decodeParams(src map[string]any, target any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeAssertMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "exists", "selector_exists", "success":
		return "exists"
	case "not_exists", "missing", "absent", "failure":
		return "not_exists"
	case "text", "text_equals", "equals", "equal", "text_equal":
		return "text_equals"
	case "text_contains", "contains", "text_contains_any":
		return "text_contains"
	case "attribute", "attribute_equals":
		return "attribute_equals"
	case "attribute_contains":
		return "attribute_contains"
	case "expression", "script":
		return "expression"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func needsExpectedValue(mode string) bool {
	switch mode {
	case "text_equals", "text_contains", "attribute_equals", "attribute_contains":
		return true
	default:
		return false
	}
}
