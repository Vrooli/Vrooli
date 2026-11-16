// Package credits provides BAS account usage read models and its durable
// LPBS outbox journey adapter. AI billing is owned by AI Gateway.
package credits

import (
	"context"
	"time"
)

// CreditService exposes BAS's account usage read surfaces.
type CreditService interface {
	// GetUsage returns the usage summary for a user in the current billing period.
	GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error)

	// GetUsageHistory returns usage summaries for multiple billing periods.
	// months specifies how many months to retrieve, offset specifies months to skip from current.
	// Returns (summaries, hasMore, error).
	GetUsageHistory(ctx context.Context, userIdentity string, months, offset int) ([]UsageSummary, bool, error)

	// GetOperationLog returns paginated operation log entries for a billing period.
	// month is in YYYY-MM format, category filters by operation category (ai, execution, export).
	// Empty category returns all operations.
	GetOperationLog(ctx context.Context, userIdentity, month, category string, limit, offset int) (*OperationLogPage, error)
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
