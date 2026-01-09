// Package policy provides the policy evaluation interface and implementations.
//
// This package defines the SEAM for policy decisions. Policies control:
// - Whether sandbox is required for a run
// - Whether approval is required
// - Concurrency limits
// - Resource limits
// - Runner restrictions
//
// The Evaluator interface allows policy logic to be swapped, extended,
// or mocked without changing the orchestration code.
package policy

import (
	"context"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Evaluator Interface - The primary seam for policy decisions
// -----------------------------------------------------------------------------

// Evaluator evaluates policies and makes decisions about run execution.
type Evaluator interface {
	// EvaluateRunRequest decides whether a run can be created and how.
	// Returns a Decision containing the evaluated policy constraints.
	EvaluateRunRequest(ctx context.Context, req EvaluateRequest) (*Decision, error)

	// EvaluateApproval decides whether a run can be approved.
	EvaluateApproval(ctx context.Context, runID uuid.UUID, actor string) (*ApprovalDecision, error)

	// CheckConcurrency verifies concurrency limits are not exceeded.
	CheckConcurrency(ctx context.Context, req ConcurrencyRequest) (*ConcurrencyDecision, error)

	// GetEffectivePolicies returns policies that apply to a given scope.
	GetEffectivePolicies(ctx context.Context, scopePath, projectRoot string) ([]*domain.Policy, error)
}

// -----------------------------------------------------------------------------
// Request Types
// -----------------------------------------------------------------------------

// EvaluateRequest contains context for policy evaluation.
type EvaluateRequest struct {
	// Task being executed
	Task *domain.Task

	// Profile being used
	Profile *domain.AgentProfile

	// Requested run mode
	RequestedMode domain.RunMode

	// Actor requesting the run
	Actor string

	// Override flags
	ForceInPlace bool // Request to skip sandbox despite policy
}

// ConcurrencyRequest contains context for concurrency check.
type ConcurrencyRequest struct {
	ScopePath   string
	ProjectRoot string
	RunnerType  domain.RunnerType
}

// -----------------------------------------------------------------------------
// Decision Types
// -----------------------------------------------------------------------------

// Decision contains the policy evaluation result for a run request.
type Decision struct {
	// Allowed indicates whether the run is permitted.
	Allowed bool

	// DenialReason explains why the run was denied (if Allowed is false).
	DenialReason string

	// DenialPolicy is the policy that caused denial (if any).
	DenialPolicy *domain.Policy

	// RequiresSandbox indicates sandbox mode must be used.
	RequiresSandbox bool

	// RequiresApproval indicates the run results need approval.
	RequiresApproval bool

	// EffectiveTimeout is the maximum execution time allowed.
	EffectiveTimeout int64 // milliseconds

	// EffectiveMaxFiles is the maximum files that can be changed.
	EffectiveMaxFiles int

	// EffectiveMaxSize is the maximum total size of changes.
	EffectiveMaxSize int64 // bytes

	// AppliedPolicies lists policies that contributed to this decision.
	AppliedPolicies []AppliedPolicy
}

// AppliedPolicy records which policy contributed to a decision.
type AppliedPolicy struct {
	PolicyID   uuid.UUID
	PolicyName string
	Effect     string // "allow", "deny", "require_sandbox", etc.
}

// ApprovalDecision contains the policy evaluation for an approval request.
type ApprovalDecision struct {
	// Allowed indicates whether approval is permitted.
	Allowed bool

	// DenialReason explains why approval was denied.
	DenialReason string

	// RequiresAdditionalApprover indicates multi-party approval is needed.
	RequiresAdditionalApprover bool

	// AutoApproved indicates this was automatically approved by policy.
	AutoApproved bool

	// AutoApproveReason explains why it was auto-approved.
	AutoApproveReason string
}

// ConcurrencyDecision contains the result of a concurrency check.
type ConcurrencyDecision struct {
	// Allowed indicates whether a new run can start.
	Allowed bool

	// DenialReason explains why concurrency was denied.
	DenialReason string

	// CurrentRuns is the number of currently active runs.
	CurrentRuns int

	// MaxRuns is the maximum allowed concurrent runs.
	MaxRuns int

	// WaitEstimate is an estimate of when capacity might be available.
	WaitEstimate string
}

// -----------------------------------------------------------------------------
// Repository Interface
// -----------------------------------------------------------------------------

// Repository provides persistence for policies.
type Repository interface {
	// Create stores a new policy.
	Create(ctx context.Context, policy *domain.Policy) error

	// Get retrieves a policy by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Policy, error)

	// Update modifies an existing policy.
	Update(ctx context.Context, policy *domain.Policy) error

	// Delete removes a policy.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves policies matching a filter.
	List(ctx context.Context, filter ListFilter) ([]*domain.Policy, error)

	// FindByScope finds policies matching a scope pattern.
	FindByScope(ctx context.Context, scopePath string) ([]*domain.Policy, error)
}

// ListFilter specifies criteria for listing policies.
type ListFilter struct {
	Enabled *bool
	Limit   int
	Offset  int
}

// -----------------------------------------------------------------------------
// Default Policy Provider
// -----------------------------------------------------------------------------

// DefaultPolicyProvider returns default policies when none are configured.
type DefaultPolicyProvider interface {
	// GetDefaults returns the default policy rules.
	GetDefaults() *domain.PolicyRules
}
