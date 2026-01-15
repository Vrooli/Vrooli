// Package credits provides unified credit management for all billable operations.
//
// This package replaces the scattered usage tracking (UsageTracker, AICreditsTracker)
// with a single, type-safe service that handles all credit operations.
//
// # Architecture
//
// The credit system uses a unified pool model where all operations (AI, executions,
// exports) draw from the same credit balance. Operation types are tracked for
// analytics and UI display.
//
// # Usage
//
//	svc := credits.NewService(opts)
//
//	// Check if user can afford operation
//	canCharge, remaining, err := svc.CanCharge(ctx, userID, credits.OpAIWorkflowGenerate)
//
//	// Charge after successful operation
//	result, err := svc.Charge(ctx, credits.ChargeRequest{
//	    UserIdentity: userID,
//	    Operation:    credits.OpAIWorkflowGenerate,
//	    Metadata:     credits.ChargeMetadata{Model: "gpt-4"},
//	})
package credits

import (
	"context"
	"errors"
	"time"
)

// CreditService is the unified interface for all credit operations.
// This replaces UsageTracker and AICreditsTracker with a single, type-safe API.
type CreditService interface {
	// CanCharge checks if the user has sufficient credits for the operation.
	// Returns (canCharge, remainingCredits, error).
	// remainingCredits is -1 for unlimited tiers.
	// This should be called BEFORE performing the operation.
	CanCharge(ctx context.Context, userIdentity string, op OperationType) (bool, int, error)

	// Charge deducts credits for a completed operation.
	// Call this AFTER the operation succeeds.
	// Returns the charge result with remaining balance.
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)

	// ChargeIfAllowed combines CanCharge and Charge atomically.
	// Useful when you want to reserve credits before starting an operation.
	// Returns ErrInsufficientCredits if not allowed.
	ChargeIfAllowed(ctx context.Context, req ChargeRequest) (*ChargeResult, error)

	// GetUsage returns the usage summary for a user in the current billing period.
	GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error)

	// GetOperationCost returns the credit cost for an operation type.
	// Returns 0 for free operations.
	GetOperationCost(op OperationType) int

	// LogFailedOperation logs an operation that failed (no credits charged).
	// Useful for audit trails and debugging.
	LogFailedOperation(ctx context.Context, req ChargeRequest, opErr error) error

	// GetUsageHistory returns usage summaries for multiple billing periods.
	// months specifies how many months to retrieve, offset specifies months to skip from current.
	// Returns (summaries, hasMore, error).
	GetUsageHistory(ctx context.Context, userIdentity string, months, offset int) ([]UsageSummary, bool, error)

	// GetOperationLog returns paginated operation log entries for a billing period.
	// month is in YYYY-MM format, category filters by operation category (ai, execution, export).
	// Empty category returns all operations.
	GetOperationLog(ctx context.Context, userIdentity, month, category string, limit, offset int) (*OperationLogPage, error)

	// CanPerformAIOperation checks if user can perform an AI operation.
	// Combines BYOK bypass, tier check, and credit check in one call.
	// Returns (canProceed, errorCode, errorMessage, remaining, error).
	// errorCode is empty string if canProceed is true.
	// If hasBYOK is true, bypasses tier and credit checks (user pays their own way).
	CanPerformAIOperation(ctx context.Context, userIdentity string, op OperationType, hasBYOK bool) (bool, string, string, int, error)
}

// ChargeRequest encapsulates all information needed to charge credits.
// This design makes it impossible to forget required fields.
type ChargeRequest struct {
	// UserIdentity is required: who to charge (email or customer ID)
	UserIdentity string

	// Operation is required: what operation type
	Operation OperationType

	// Metadata contains optional operation-specific details for logging/auditing
	Metadata ChargeMetadata
}

// ChargeMetadata contains optional operation-specific details for logging.
// Use the appropriate fields based on the operation type.
type ChargeMetadata struct {
	// Common fields
	WorkflowID  string `json:"workflow_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
	DurationMs  int    `json:"duration_ms,omitempty"`

	// AI operation fields
	Model            string `json:"model,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`

	// Export operation fields
	ExportFormat  string `json:"export_format,omitempty"`
	FileSizeBytes int64  `json:"file_size_bytes,omitempty"`
}

// ChargeResult contains the result of a charge operation.
type ChargeResult struct {
	// Charged is the number of credits charged (may be 0 if free operation)
	Charged int

	// RemainingCredits is the user's remaining balance after the charge.
	// -1 indicates unlimited.
	RemainingCredits int

	// WasCharged is true if credits were actually deducted
	WasCharged bool
}

// UsageSummary contains aggregated usage for a billing period.
type UsageSummary struct {
	// UserIdentity is the user this summary is for
	UserIdentity string `json:"user_identity"`

	// BillingMonth in YYYY-MM format
	BillingMonth string `json:"billing_month"`

	// TotalCreditsUsed is the total credits consumed this period
	TotalCreditsUsed int `json:"total_credits_used"`

	// TotalOperations is the count of all operations this period
	TotalOperations int `json:"total_operations"`

	// ByOperation breaks down credits by operation type
	// e.g., {"ai.workflow_generate": 15, "execution.run": 42}
	ByOperation map[OperationType]int `json:"by_operation"`

	// OperationCounts breaks down operation counts by type
	OperationCounts map[OperationType]int `json:"operation_counts"`

	// CreditsLimit is the user's credit limit for this period.
	// -1 indicates unlimited.
	CreditsLimit int `json:"credits_limit"`

	// CreditsRemaining is the credits remaining this period.
	// -1 indicates unlimited.
	CreditsRemaining int `json:"credits_remaining"`

	// PeriodStart is the start of the billing period
	PeriodStart time.Time `json:"period_start"`

	// PeriodEnd is the end of the billing period
	PeriodEnd time.Time `json:"period_end"`

	// ResetDate is when the credits reset (first of next month)
	ResetDate time.Time `json:"reset_date"`
}

// OperationLogPage contains paginated operation log entries.
type OperationLogPage struct {
	// UserIdentity is the user this page is for
	UserIdentity string `json:"user_identity"`

	// BillingMonth in YYYY-MM format
	BillingMonth string `json:"billing_month"`

	// Operations is the list of operations in this page
	Operations []OperationLogEntry `json:"operations"`

	// Total is the total number of operations matching the query
	Total int `json:"total"`

	// Limit is the page size
	Limit int `json:"limit"`

	// Offset is the starting position
	Offset int `json:"offset"`

	// HasMore indicates if there are more results
	HasMore bool `json:"has_more"`
}

// OperationLogEntry represents a single operation in the log.
type OperationLogEntry struct {
	// ID is the unique identifier for this entry
	ID string `json:"id"`

	// OperationType is the type of operation performed
	OperationType OperationType `json:"operation_type"`

	// CreditsCharged is the number of credits charged (0 for free ops)
	CreditsCharged int `json:"credits_charged"`

	// Success indicates if the operation completed successfully
	Success bool `json:"success"`

	// CreatedAt is when the operation was performed
	CreatedAt time.Time `json:"created_at"`

	// Metadata contains operation-specific details
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ErrorMessage contains the error if Success is false
	ErrorMessage string `json:"error_message,omitempty"`
}

// Sentinel errors for credit operations.
var (
	// ErrInsufficientCredits indicates the user doesn't have enough credits.
	ErrInsufficientCredits = errors.New("insufficient credits for this operation")

	// ErrNoCreditsAccess indicates the user's tier doesn't have access to credits.
	ErrNoCreditsAccess = errors.New("your subscription tier does not include this feature")

	// ErrInvalidOperation indicates an unknown operation type.
	ErrInvalidOperation = errors.New("unknown operation type")

	// ErrUserIdentityRequired indicates the user identity is missing.
	ErrUserIdentityRequired = errors.New("user identity is required")
)
