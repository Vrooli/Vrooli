package runtime

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
	Keys                    []string          `json:"keys,omitempty"`     // Keyboard keys array (preferred)
	Sequence                string            `json:"sequence,omitempty"` // Keyboard sequence string
	KeyValue                string            `json:"keyValue,omitempty"` // Deprecated: use Keys instead
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
	LoopItemVariable        string            `json:"loopItemVariable,omitempty"`
	LoopIndexVariable       string            `json:"loopIndexVariable,omitempty"`
	LoopMaxIterations       int               `json:"loopMaxIterations,omitempty"`
	LoopIterationTimeoutMs  int               `json:"loopIterationTimeoutMs,omitempty"`
	LoopTotalTimeoutMs      int               `json:"loopTotalTimeoutMs,omitempty"`
	LoopCondition           string            `json:"loopCondition,omitempty"`
	LoopMaxDurationMs       int               `json:"loopMaxDurationMs,omitempty"`
	LoopIterationDelayMs    int               `json:"loopIterationDelayMs,omitempty"`
	SetVariableName         string            `json:"setVariableName,omitempty"`
	SetVariableSource       string            `json:"setVariableSource,omitempty"`
	SetVariableValue        any               `json:"setVariableValue,omitempty"`
	SetVariableStoreAs      string            `json:"setVariableStoreAs,omitempty"`
	SetVariableTransform    string            `json:"setVariableTransform,omitempty"`
	SetVariableRequired     *bool             `json:"setVariableRequired,omitempty"`
	NetworkMockURL          string            `json:"networkMockUrl,omitempty"`
	NetworkMockMethod       string            `json:"networkMockMethod,omitempty"`
	NetworkMockStatus       int               `json:"networkMockStatus,omitempty"`
	NetworkMockHeaders      map[string]string `json:"networkMockHeaders,omitempty"`
	NetworkMockBody         string            `json:"networkMockBody,omitempty"`
	AssertSelector          string            `json:"assertSelector,omitempty"`
	AssertAttribute         string            `json:"assertAttribute,omitempty"`
	AssertValue             any               `json:"assertValue,omitempty"`
	AssertComparison        string            `json:"assertComparison,omitempty"`
	AssertNormalized        *bool             `json:"assertNormalized,omitempty"`
	AssertTimeoutMs         int               `json:"assertTimeoutMs,omitempty"`
	AssertPollIntervalMs    int               `json:"assertPollIntervalMs,omitempty"`
	UseVariableName         string            `json:"useVariableName,omitempty"`
	UseVariableStoreAs      string            `json:"useVariableStoreAs,omitempty"`
	UseVariableTransform    string            `json:"useVariableTransform,omitempty"`
	UseVariableRequired     *bool             `json:"useVariableRequired,omitempty"`
	Condition               any               `json:"condition,omitempty"`
	FrameWaitForSelector    string            `json:"frameWaitForSelector,omitempty"`
	FrameTimeoutMs          int               `json:"frameTimeoutMs,omitempty"`
	TabTimeoutMs            int               `json:"tabTimeoutMs,omitempty"`
	SubmitSelector          string            `json:"submitSelector,omitempty"`
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
