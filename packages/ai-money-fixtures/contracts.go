// Package aimoneyfixtures contains standard-library-only contract types and
// deterministic fakes for the AI money and front-door integration tests.
// Nothing in this package establishes production integration or live-payment
// behavior; callers must also test the real owning components.
package aimoneyfixtures

import (
	"context"
	"time"
)

const (
	ErrorCreditsRequired = "credits_required"
	ErrorNoEligibleRoute = "no_eligible_route"
	ErrorUnauthorized    = "unauthorized"
)

// PublicError is the stable public error body for paid AI requests.
type PublicError struct {
	Error     string `json:"error"`
	ErrorType string `json:"error_type"`
	Retryable bool   `json:"retryable"`
}

// FeatureRequest is deliberately bounded: one declared instruction, one
// user turn and a JSON schema. It does not accept a caller-selected use class.
type FeatureRequest struct {
	Instruction string         `json:"instruction"`
	Turn        FeatureTurn    `json:"turn"`
	Schema      map[string]any `json:"schema"`
}

// ChatRequest is the bounded public coding-agent request: exactly one user
// message is accepted by this contract fixture; the caller cannot set a use class.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []FeatureTurn `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type FeatureTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// UsageReceipt is the public per-call billing record. Provider cost is kept
// separate from customer charge; lots preserve funding attribution.
type UsageReceipt struct {
	OperationID          string   `json:"operation_id"`
	RequestID            string   `json:"request_id"`
	FeatureID            string   `json:"feature_id"`
	RouteID              string   `json:"route_id"`
	Supplier             string   `json:"supplier"`
	Model                string   `json:"model"`
	PermissionVersion    string   `json:"permission_version"`
	PricingVersion       string   `json:"pricing_version"`
	TermsReference       string   `json:"terms_reference"`
	InputTokens          int64    `json:"input_tokens"`
	OutputTokens         int64    `json:"output_tokens"`
	UpstreamCostMicros   int64    `json:"upstream_cost_micros"`
	CustomerChargeMicros int64    `json:"customer_charge_micros"`
	LotIDs               []string `json:"lot_ids"`
	SourceKinds          []string `json:"source_kinds"`
}

// GatewayPlanRequest is the private LPBS-to-Gateway planning contract. It is
// intentionally distinct from a public GatewayRequest.
type GatewayPlanRequest struct {
	OperationID  string `json:"operation_id"`
	FeatureID    string `json:"feature_id"`
	UseClass     string `json:"use_class"`
	Model        string `json:"model"`
	BudgetMicros int64  `json:"budget_micros"`
	PrivacyClass string `json:"privacy_class"`
}

type GatewayPlan struct {
	RouteID               string `json:"route_id"`
	Supplier              string `json:"supplier"`
	Model                 string `json:"model"`
	MaxProviderCostMicros int64  `json:"max_provider_cost_micros"`
}

type GatewayExecuteRequest struct {
	OperationID  string         `json:"operation_id"`
	Plan         GatewayPlan    `json:"plan"`
	Feature      FeatureRequest `json:"feature"`
	Funded       bool           `json:"funded"`
	BudgetMicros int64          `json:"budget_micros"`
}

type GatewayResult struct {
	Output             string `json:"output"`
	InputTokens        int64  `json:"input_tokens"`
	OutputTokens       int64  `json:"output_tokens"`
	ProviderCostMicros int64  `json:"provider_cost_micros"`
}

type GatewayReceiptRequest struct {
	OperationID string `json:"operation_id"`
}

type GatewayReceipt struct {
	OperationID string        `json:"operation_id"`
	State       string        `json:"state"`
	Result      GatewayResult `json:"result"`
}

// GatewayPrivate is the private contract consumed by the LPBS fake front door.
type GatewayPrivate interface {
	Plan(context.Context, GatewayPlanRequest) (GatewayPlan, error)
	Execute(context.Context, GatewayExecuteRequest) (GatewayResult, error)
	Receipt(context.Context, GatewayReceiptRequest) (GatewayReceipt, error)
}

// RuleRecord is fixture input only. Approval is explicit data and never
// performed by fixture code.
type RuleRecord struct {
	Supplier      string            `json:"supplier"`
	Plan          string            `json:"plan"`
	AccessTier    string            `json:"access_tier"`
	UseClasses    map[string]string `json:"use_classes"`
	Evidence      Evidence          `json:"evidence"`
	Review        Review            `json:"review"`
	EffectiveFrom time.Time         `json:"effective_from"`
	ExpiresAt     *time.Time        `json:"expires_at"`
}

type Evidence struct {
	URL          string `json:"url"`
	DocumentDate string `json:"document_date"`
	ReadAt       string `json:"read_at"`
	SHA256       string `json:"sha256"`
}

type Review struct {
	Status      string `json:"status"`
	DecisionRef string `json:"decision_ref,omitempty"`
}

// EligibleFunc is a test seam. Production policy remains owned by
// ai-provider-go/policy; this fixture does not implement eligibility.
type EligibleFunc func(request any, account any) (bool, string)

// FixtureKeyIssuer issues deterministic test identities; it is not a
// production key issuer and does not provide revocation or signing guarantees.
type FixtureKeyIssuer interface {
	Issue(subject string, expires time.Time) (string, error)
	Revoke(token string)
	Valid(token string, now time.Time) bool
}
