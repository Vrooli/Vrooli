package vision

// NavigationRequest contains all parameters needed to start an AI navigation.
type NavigationPostcondition struct {
	Selector string `json:"selector"`
	Mode     string `json:"mode"`
	Expected string `json:"expected"`
}
type NavigationExtraction struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Limit     int    `json:"limit"`
}
type NavigationRequest struct {
	EffectPolicy   string                    `json:"effect_policy,omitempty"`
	Postconditions []NavigationPostcondition `json:"postconditions,omitempty"`
	Extraction     []NavigationExtraction    `json:"extraction,omitempty"`
	// SessionID is the browser session to navigate.
	SessionID string `json:"session_id"`

	// Prompt is the natural language instruction for the AI.
	Prompt string `json:"prompt"`

	// Model is the provider-neutral AI Gateway route profile (local_first or
	// remote_only). It is retained as a wire field for UI compatibility.
	Model string `json:"model"`

	// MaxSteps limits the number of navigation steps.
	MaxSteps int `json:"max_steps,omitempty"`

	// NavigatorType optionally specifies which navigator to use.
	// If not specified, the registry will auto-select.
	NavigatorType NavigatorType `json:"navigator_type,omitempty"`

	// CallbackURL is the URL for step event callbacks.
	CallbackURL string `json:"-"`

	// RootOperationID is the caller's stable idempotency root for this
	// navigation. The sidecar derives a distinct stable Gateway ID per step.
	RootOperationID string `json:"-"`
	// AccessToken is the verified shared session token, carried only over the
	// owned local sidecar route and never persisted in navigation state.
	AccessToken string `json:"-"`
	// AISource is the selected shared billing source: local, own-key, or
	// vrooli-credits. Gateway remains authoritative for authorization and usage.
	AISource string `json:"-"`
	// PrivacyConsentGrant is an opaque Gateway-signed source-scoped receipt.
	// It is carried in process memory only and never persisted or logged.
	PrivacyConsentGrant string `json:"-"`
}

// NavigationStep represents a single step in the navigation process.
type NavigationStep struct {
	NavigationID        string                 `json:"navigationId"`
	StepNumber          int                    `json:"stepNumber"`
	Action              map[string]interface{} `json:"action"`
	Reasoning           string                 `json:"reasoning"`
	Screenshot          string                 `json:"screenshot"` // base64
	AnnotatedScreenshot string                 `json:"annotatedScreenshot,omitempty"`
	CurrentURL          string                 `json:"currentUrl"`
	TokensUsed          TokenUsage             `json:"tokensUsed"`
	DurationMs          int64                  `json:"durationMs"`
	GoalAchieved        bool                   `json:"goalAchieved"`
	Error               string                 `json:"error,omitempty"`
	ElementLabels       []interface{}          `json:"elementLabels,omitempty"`
	AwaitingHuman       bool                   `json:"awaitingHuman,omitempty"`
	HumanIntervention   *HumanInterventionInfo `json:"humanIntervention,omitempty"`
}

// TokenUsage tracks token consumption for a step.
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// NavigationResult represents the outcome of a completed navigation.
type NavigationResult struct {
	VerifiedSuccess   bool                   `json:"verifiedSuccess"`
	ExtractedData     map[string]interface{} `json:"extractedData,omitempty"`
	VerificationError string                 `json:"verificationError,omitempty"`
	NavigationID      string                 `json:"navigationId"`
	Status            NavigationStatus       `json:"status"`
	TotalSteps        int                    `json:"totalSteps"`
	TotalTokens       int                    `json:"totalTokens"`
	TotalDurationMs   int64                  `json:"totalDurationMs"`
	FinalURL          string                 `json:"finalUrl"`
	Error             string                 `json:"error,omitempty"`
	Summary           string                 `json:"summary,omitempty"`
}

// NavigatorInfo provides information about a navigator for the list endpoint.
type NavigatorInfo struct {
	Type              NavigatorType  `json:"type"`
	Available         bool           `json:"available"`
	Description       string         `json:"description"`
	CreditPolicy      CreditPolicy   `json:"credit_policy"`
	AllowedSources    []ClientSource `json:"allowed_sources"`
	UnavailableReason string         `json:"unavailable_reason,omitempty"`
}

// NavigatorsResponse is the response for GET /api/v1/ai-navigate/navigators.
type NavigatorsResponse struct {
	Navigators []NavigatorInfo `json:"navigators"`
	Default    NavigatorType   `json:"default"`
}
