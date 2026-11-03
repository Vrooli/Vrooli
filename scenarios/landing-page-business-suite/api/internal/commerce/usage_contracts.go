package commerce

import (
	"context"
	cryptoRand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrBusinessAccountRequired = errors.New("a verified business account is required for Vrooli credits")

// MeterLimit validates the authoritative data kind before selecting an owner.
func requireMeterKind(ctx context.Context, limits LimitsServicer, tier, key, expected string) (*TierLimit, error) {
	if limits == nil || strings.TrimSpace(tier) == "" {
		return nil, errors.New("meter limit authority is unavailable")
	}
	limit, err := limits.GetLimit(ctx, tier, key, nil)
	if err != nil {
		return nil, err
	}
	if limit == nil || (limit.LimitType != "count_based" && limit.LimitType != "cost_based") {
		return nil, errors.New("meter limit kind is unavailable")
	}
	if expected != "" && limit.LimitType != expected {
		return nil, fmt.Errorf("%s meter requires its %s accounting owner", key, limit.LimitType)
	}
	return limit, nil
}

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
	BusinessAccountID string            `json:"business_account_id,omitempty"`
	Tier              string            `json:"-"`
	UserIdentity      string            `json:"user_identity"`
	LimitKey          string            `json:"limit_key"`
	Amount            int64             `json:"amount"`
	AppBundleKey      string            `json:"app_bundle_key"`
	Operation         string            `json:"operation,omitempty"`
	IsBYOK            bool              `json:"is_byok,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	OperationID       *string           `json:"operation_id,omitempty"`
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

func newReservationID() string {
	b := make([]byte, 16)
	if _, err := cryptoRand.Read(b); err != nil {
		return fmt.Sprintf("reservation-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func usageEventMetadata(metadata map[string]string) (int64, int, int, string) {
	var cost int64
	var prompt, completion int
	if metadata != nil {
		cost, _ = strconv.ParseInt(strings.TrimSpace(metadata["cost_micros"]), 10, 64)
		prompt, _ = strconv.Atoi(strings.TrimSpace(metadata["prompt_tokens"]))
		completion, _ = strconv.Atoi(strings.TrimSpace(metadata["completion_tokens"]))
	}
	provider := firstNonEmpty(metadata["provider"], "local")
	return cost, prompt, completion, provider
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
