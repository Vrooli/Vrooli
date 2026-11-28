package vision

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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
	// BrowserSession is resolved by the BAS broker from the authenticated AM
	// identity and existing active session; callers cannot populate it directly.
	BrowserSession *BrowserInferenceSessionBinding `json:"-"`
}

type BrowserInferenceSessionBinding struct {
	RunID         string `json:"run_id"`
	SessionID     string `json:"session_id"`
	GrantReceipt  string `json:"grant_receipt"`
	ReservationID string `json:"reservation_id,omitempty"`
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
	InferenceReceipt    *InferenceReceipt      `json:"inferenceReceipt,omitempty"`
	DurationMs          int64                  `json:"durationMs"`
	GoalAchieved        bool                   `json:"goalAchieved"`
	Error               string                 `json:"error,omitempty"`
	ElementLabels       []interface{}          `json:"elementLabels,omitempty"`
	AwaitingHuman       bool                   `json:"awaitingHuman,omitempty"`
	HumanIntervention   *HumanInterventionInfo `json:"humanIntervention,omitempty"`
}

// InferenceReceipt retains Gateway accounting returned with one browser model
// call. This is display/reconciliation evidence from the driver callback; it
// is not a billing authority. LPBS settlement still requires owner-attested
// reservation evidence.
type InferenceReceipt struct {
	RootOperationID        string                    `json:"rootOperationId"`
	BillingOperationID     string                    `json:"billingOperationId,omitempty"`
	Source                 string                    `json:"source"`
	Provider               string                    `json:"provider,omitempty"`
	Model                  string                    `json:"model,omitempty"`
	ProviderCallsMade      int32                     `json:"providerCallsMade,omitempty"`
	InputTokens            int64                     `json:"inputTokens"`
	OutputTokens           int64                     `json:"outputTokens"`
	InputTokensProvenance  string                    `json:"inputTokensProvenance,omitempty"`
	OutputTokensProvenance string                    `json:"outputTokensProvenance,omitempty"`
	CostMicros             ExactUsage                `json:"costMicros,omitempty"`
	CostProvenance         string                    `json:"costProvenance,omitempty"`
	CreditsCharged         ExactUsage                `json:"creditsCharged,omitempty"`
	AccountingStatus       string                    `json:"accountingStatus,omitempty"`
	Applied                *InferenceAppliedSettings `json:"applied,omitempty"`
}

// ExactUsage accepts Gateway JSON that may encode exact int64 values as either
// decimal strings or safe JSON integers. It never routes money through float64.
type ExactUsage string

func (u *ExactUsage) UnmarshalJSON(data []byte) error {
	var value string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
	} else {
		value = string(data)
	}
	if value == "" {
		*u = ""
		return nil
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("exact usage must be a non-negative integer")
		}
	}
	*u = ExactUsage(value)
	return nil
}

type InferenceAppliedSettings struct {
	MaxOutputTokensEffective int32    `json:"maxOutputTokensEffective,omitempty"`
	MaxOutputTokensSource    string   `json:"maxOutputTokensSource,omitempty"`
	TemperatureSent          *float64 `json:"temperatureSent,omitempty"`
	TemperatureSupport       string   `json:"temperatureSupport,omitempty"`
}

func (r *InferenceReceipt) Clone() *InferenceReceipt {
	if r == nil {
		return nil
	}
	copy := *r
	if r.Applied != nil {
		applied := *r.Applied
		if r.Applied.TemperatureSent != nil {
			temperature := *r.Applied.TemperatureSent
			applied.TemperatureSent = &temperature
		}
		copy.Applied = &applied
	}
	return &copy
}

func (r *InferenceReceipt) Matches(tokens TokenUsage) bool {
	if r == nil || strings.TrimSpace(r.RootOperationID) == "" {
		return false
	}
	switch r.Source {
	case "local", "own-key", "vrooli-credits":
	default:
		return false
	}
	if r.ProviderCallsMade < 0 || r.InputTokens < 0 || r.OutputTokens < 0 || tokens.PromptTokens < 0 || tokens.CompletionTokens < 0 {
		return false
	}
	if r.InputTokens != int64(tokens.PromptTokens) || r.OutputTokens != int64(tokens.CompletionTokens) {
		return false
	}
	for _, value := range []ExactUsage{r.CostMicros, r.CreditsCharged} {
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil || parsed < 0 {
			return false
		}
	}
	if r.Applied != nil && r.Applied.MaxOutputTokensEffective < 0 {
		return false
	}
	return true
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
