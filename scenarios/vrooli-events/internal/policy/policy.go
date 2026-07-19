// Package policy implements the policy enforcement engine for vrooli-events:
// access control rules, rate limiting, circuit breakers, and violation logging.
//
// DOC: docs/guides/managing-policies.md
// DOC: docs/concepts/ARCHITECTURE.md#policy-engine
package policy

import (
	"context"
	"strings"
	"time"
)

// RuleType classifies policy rules.
type RuleType string

const (
	RuleTypeAccess         RuleType = "access"
	RuleTypeRateLimit      RuleType = "rate_limit"
	RuleTypeCircuitBreaker RuleType = "circuit_breaker"
)

// Effect is the outcome of an access control rule.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// Rule represents a policy rule in the store.
type Rule struct {
	ID               int64     `json:"id"`
	RuleType         RuleType  `json:"rule_type"`
	SourceScenario   string    `json:"source_scenario"`
	TargetScenario   string    `json:"target_scenario"`
	EndpointPattern  string    `json:"endpoint_pattern,omitempty"`
	Effect           Effect    `json:"effect,omitempty"`
	Priority         int       `json:"priority"`
	Enabled          bool      `json:"enabled"`
	MaxRequests      int       `json:"max_requests,omitempty"`
	WindowSeconds    int       `json:"window_seconds,omitempty"`
	BurstAllowance   int       `json:"burst_allowance,omitempty"`
	FailureThreshold int       `json:"failure_threshold,omitempty"`
	CooldownSeconds  int       `json:"cooldown_seconds,omitempty"`
	SuccessThreshold int       `json:"success_threshold,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ReceiptProjectionRule centrally authorizes which typed unary operations may
// create a durable receipt, and bounds the projection retained for each one.
// Scenario code may provide candidate fields, but this rule is the authority
// for whether anything is emitted or retained.
type ReceiptProjectionRule struct {
	ID               int64     `json:"id"`
	SourceScenario   string    `json:"source_scenario"`
	TargetScenario   string    `json:"target_scenario"`
	OperationPattern string    `json:"operation_pattern"`
	ResponseFields   []string  `json:"response_fields"`
	RedactFields     []string  `json:"redact_fields,omitempty"`
	MaxBytes         int       `json:"max_bytes"`
	SamplePerTenK    int       `json:"sample_per_ten_k"`
	RetentionDays    int       `json:"retention_days"`
	Priority         int       `json:"priority"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ReceiptProjectionFilters struct {
	Source  string
	Target  string
	Enabled *bool
}

// Violation records a policy denial.
type Violation struct {
	ID             int64    `json:"id"`
	Timestamp      string   `json:"timestamp"`
	SourceScenario string   `json:"source_scenario"`
	TargetScenario string   `json:"target_scenario"`
	Endpoint       string   `json:"endpoint"`
	RuleID         int64    `json:"rule_id"`
	RuleType       RuleType `json:"rule_type"`
	Reason         string   `json:"reason"`
}

// Decision is the result of evaluating a policy.
type Decision struct {
	Allowed      bool         `json:"allowed"`
	RuleID       int64        `json:"rule_id,omitempty"`
	RuleType     RuleType     `json:"rule_type,omitempty"`
	Reason       string       `json:"reason"`
	RetryAfter   int          `json:"retry_after,omitempty"`
	CircuitState CircuitState `json:"circuit_state,omitempty"`
}

// ListFilters defines filters for listing policy rules.
type ListFilters struct {
	RuleType RuleType
	Source   string
	Target   string
	Enabled  *bool
}

// ViolationFilters defines filters for listing violations.
type ViolationFilters struct {
	Source   string
	Target   string
	RuleType RuleType
	Since    string
	Until    string
	Limit    int
}

// CircuitBreakerOverride records a manual state override for a circuit breaker rule.
type CircuitBreakerOverride struct {
	RuleID    int64        `json:"rule_id"`
	State     CircuitState `json:"state"`
	ExpiresAt time.Time    `json:"expires_at"`
	CreatedAt time.Time    `json:"created_at"`
}

// ComputeSpecificity scores an access rule's pattern specificity.
// Scoring per pattern: exact match = 3pts, prefix glob (contains * but isn't
// just "*") = 2pts, wildcard ("*" or "**" or empty) = 1pt.
// Sum across source + target + endpoint (max 9).
func ComputeSpecificity(source, target, endpoint string) int {
	return patternScore(source) + patternScore(target) + patternScore(endpoint)
}

func patternScore(p string) int {
	if p == "" || p == "*" || p == "**" {
		return 1
	}
	if strings.Contains(p, "*") {
		return 2
	}
	return 3
}

// Store defines the policy storage interface.
type Store interface {
	CreateRule(ctx context.Context, r Rule) (int64, error)
	GetRule(ctx context.Context, id int64) (Rule, error)
	ListRules(ctx context.Context, f ListFilters) ([]Rule, error)
	UpdateRule(ctx context.Context, r Rule) error
	DeleteRule(ctx context.Context, id int64) error
	LogViolation(ctx context.Context, v Violation) error
	ListViolations(ctx context.Context, f ViolationFilters) ([]Violation, error)
	SetCircuitBreakerOverride(ctx context.Context, ruleID int64, state CircuitState, ttlSeconds int) error
	GetCircuitBreakerOverride(ctx context.Context, ruleID int64) (*CircuitBreakerOverride, error)
	CreateReceiptProjection(ctx context.Context, r ReceiptProjectionRule) (int64, error)
	GetReceiptProjection(ctx context.Context, id int64) (ReceiptProjectionRule, error)
	ListReceiptProjections(ctx context.Context, f ReceiptProjectionFilters) ([]ReceiptProjectionRule, error)
	UpdateReceiptProjection(ctx context.Context, r ReceiptProjectionRule) error
	DeleteReceiptProjection(ctx context.Context, id int64) error
	MatchReceiptProjection(ctx context.Context, source, target, operation string) (*ReceiptProjectionRule, error)
	Close() error
}
