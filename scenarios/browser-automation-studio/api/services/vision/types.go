package vision

// NavigationRequest contains all parameters needed to start an AI navigation.
type NavigationRequest struct {
	// SessionID is the browser session to navigate.
	SessionID string `json:"session_id"`

	// Prompt is the natural language instruction for the AI.
	Prompt string `json:"prompt"`

	// Model is the vision model to use (e.g., "gpt-4o", "claude-sonnet-4").
	Model string `json:"model"`

	// MaxSteps limits the number of navigation steps.
	MaxSteps int `json:"max_steps,omitempty"`

	// APIKey is an optional BYOK key for the AI provider.
	APIKey string `json:"api_key,omitempty"`

	// NavigatorType optionally specifies which navigator to use.
	// If not specified, the registry will auto-select.
	NavigatorType NavigatorType `json:"navigator_type,omitempty"`

	// UserID is the user identity for credit tracking.
	UserID string `json:"-"`

	// CallbackURL is the URL for step event callbacks.
	CallbackURL string `json:"-"`
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
	NavigationID    string           `json:"navigationId"`
	Status          NavigationStatus `json:"status"`
	TotalSteps      int              `json:"totalSteps"`
	TotalTokens     int              `json:"totalTokens"`
	TotalDurationMs int64            `json:"totalDurationMs"`
	FinalURL        string           `json:"finalUrl"`
	Error           string           `json:"error,omitempty"`
	Summary         string           `json:"summary,omitempty"`
}

// NavigatorInfo provides information about a navigator for the list endpoint.
type NavigatorInfo struct {
	Type              NavigatorType    `json:"type"`
	Available         bool             `json:"available"`
	Description       string           `json:"description"`
	CreditPolicy      CreditPolicyInfo `json:"credit_policy"`
	AllowedSources    []ClientSource   `json:"allowed_sources"`
	UnavailableReason string           `json:"unavailable_reason,omitempty"`
}

// CreditPolicyInfo is the JSON-serializable form of CreditPolicy.
type CreditPolicyInfo struct {
	RequiresCredits  bool              `json:"requires_credits"`
	CreditsPerStep   int               `json:"credits_per_step"`
	BypassConditions []BypassCondition `json:"bypass_conditions"`
}

// NavigatorsResponse is the response for GET /api/v1/ai-navigate/navigators.
type NavigatorsResponse struct {
	Navigators []NavigatorInfo `json:"navigators"`
	Default    NavigatorType   `json:"default"`
}
