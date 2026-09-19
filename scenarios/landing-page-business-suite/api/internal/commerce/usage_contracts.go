package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// UsageStore is the context-aware persistence contract for usage records and
// credit reservations.
//
// seam: UsageStore keeps metering persistence independent of a concrete pool
// and preserves request-scoped test isolation.
type UsageStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PingContext(context.Context) error
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// UsageRecord represents a single persisted usage record.
type UsageRecord struct {
	ID              string     `json:"id"`
	UserIdentity    string     `json:"user_identity"`
	BillingPeriod   string     `json:"billing_period"`
	LimitKey        string     `json:"limit_key"`
	UsageAmount     int64      `json:"usage_amount"`
	AppBundleKey    *string    `json:"app_bundle_key"`
	LastOperationAt *time.Time `json:"last_operation_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UsageSummary is the customer-facing usage view for one billing period.
type UsageSummary struct {
	UserIdentity   string             `json:"user_identity"`
	BillingPeriod  string             `json:"billing_period"`
	Tier           string             `json:"tier,omitempty"`
	Limits         map[string]int64   `json:"limits"`
	Usage          map[string]int64   `json:"usage"`
	Remaining      map[string]int64   `json:"remaining"`
	DisplayCredits map[string]float64 `json:"display_credits"`
	ResetDate      time.Time          `json:"reset_date"`
	ByApp          map[string]int64   `json:"by_app,omitempty"`
}

// UsageHealthStatus contains metering health and its most recent activity.
type UsageHealthStatus struct {
	Healthy               bool       `json:"healthy"`
	DatabaseConnected     bool       `json:"database_connected"`
	ServiceAuthConfigured bool       `json:"service_auth_configured"`
	ServiceAuthMode       string     `json:"service_auth_mode"`
	LastRecordAt          *time.Time `json:"last_record_at,omitempty"`
	RecordsThisPeriod     int64      `json:"records_this_period"`
}

// UsageReportRequest is a metering report from an entitled application.
type UsageReportRequest struct {
	UserIdentity string            `json:"user_identity"`
	LimitKey     string            `json:"limit_key"`
	Amount       int64             `json:"amount"`
	AppBundleKey string            `json:"app_bundle_key"`
	Operation    string            `json:"operation,omitempty"`
	IsBYOK       bool              `json:"is_byok,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	OperationID  *string           `json:"operation_id,omitempty"`
}

// NormalizedUsageReport is a validated report ready for persistence.
type NormalizedUsageReport struct {
	UsageReportRequest
	Amount int64
}

// NormalizeUsageReport validates external metering input and canonicalizes its
// identity fields. BYOK reports consume zero platform credits.
func NormalizeUsageReport(request UsageReportRequest) (NormalizedUsageReport, error) {
	request.UserIdentity = strings.ToLower(strings.TrimSpace(request.UserIdentity))
	request.LimitKey = strings.ToLower(strings.TrimSpace(request.LimitKey))
	request.AppBundleKey = strings.ToLower(strings.TrimSpace(request.AppBundleKey))
	if request.UserIdentity == "" {
		return NormalizedUsageReport{}, fmt.Errorf("user_identity is required")
	}
	if request.LimitKey == "" {
		return NormalizedUsageReport{}, fmt.Errorf("limit_key is required")
	}
	if request.Amount <= 0 && !request.IsBYOK {
		return NormalizedUsageReport{}, fmt.Errorf("amount must be positive")
	}
	amount := request.Amount
	if request.IsBYOK {
		amount = 0
	}
	return NormalizedUsageReport{UsageReportRequest: request, Amount: amount}, nil
}
