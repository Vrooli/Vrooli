package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
	_ "modernc.org/sqlite"
)

func usageTestDependencies(t *testing.T, user string) (UsageDependencies, *int, *string) {
	t.Helper()
	status := 0
	message := ""
	return UsageDependencies{
		UserEmail: func(context.Context) string { return user },
		WriteError: func(_ http.ResponseWriter, gotStatus int, gotMessage, _ string) {
			status, message = gotStatus, gotMessage
		},
		LogError: func(string, map[string]any) {},
	}, &status, &message
}

func TestReportUsageRequiresVerifiedIdentityBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "")
	req := httptest.NewRequest(http.MethodPost, "/usage/report", nil)
	ReportUsage(nil, deps).ServeHTTP(httptest.NewRecorder(), req)
	if *status != http.StatusUnauthorized || *message != "Authentication required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestReportUsageRejectsMalformedJSONBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "customer@example.com")
	req := httptest.NewRequest(http.MethodPost, "/usage/report", nil)
	ReportUsage(nil, deps).ServeHTTP(httptest.NewRecorder(), req)
	if *status != http.StatusBadRequest || *message != "Invalid request body" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestUsageSummaryRequiresAuthenticatedIdentity(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "")
	GetUsageSummary(nil, nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/usage/summary", nil))
	if *status != http.StatusUnauthorized || *message != "Authentication required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestCheckLimitValidatesLimitKeyBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "customer@example.com")
	CheckLimit(nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/usage/check", nil))
	if *status != http.StatusBadRequest || *message != "limit_key is required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestCountQuotaTransportPreservesWindowAndAuthenticatedSubject(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, query := range []string{
		`CREATE TABLE subscription_tier_limits(id TEXT,tier_id TEXT,limit_type TEXT,limit_key TEXT,limit_value INTEGER,cost_multiplier INTEGER,app_bundle_key TEXT,reset_period TEXT,created_at DATETIME,updated_at DATETIME)`,
		`INSERT INTO subscription_tier_limits VALUES('count','pro','count_based','compute_minutes',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('cost','pro','cost_based','ai_credits',100,1000000,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`CREATE TABLE usage_records(user_identity TEXT,billing_period TEXT,limit_key TEXT,usage_amount INTEGER,app_bundle_key TEXT,last_operation_at DATETIME,updated_at DATETIME,UNIQUE(user_identity,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE credit_reservations(id TEXT PRIMARY KEY,user_identity TEXT,billing_period TEXT,limit_key TEXT,reserved_amount INTEGER,status TEXT,expires_at DATETIME,created_at DATETIME,finalized_at DATETIME)`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	service := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: db, LimitsService: commerce.NewLimitsService(db, "sqlite", nil), Dialect: "sqlite"})
	deps, status, message := usageTestDependencies(t, "owner@example.test")
	deps.SubscriptionTier = func(context.Context, string) (string, error) { return "pro", nil }
	response := httptest.NewRecorder()
	before := time.Now()
	ReserveCredits(service, nil, deps).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/usage/reservations", strings.NewReader(`{"limit_key":"compute_minutes","amount":60,"window_seconds":3600,"tier":"business"}`)))
	if *status != 0 {
		t.Fatalf("quota reserve status=%d message=%s", *status, *message)
	}
	var reservation struct {
		ID     string `json:"reservation_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &reservation); err != nil {
		t.Fatal(err)
	}
	if reservation.ID == "" || reservation.Status != "pending" {
		t.Fatalf("reserve response=%s", response.Body)
	}
	var actor string
	var expires time.Time
	if err := db.QueryRow(`SELECT user_identity,expires_at FROM credit_reservations WHERE id=?`, reservation.ID).Scan(&actor, &expires); err != nil {
		t.Fatal(err)
	}
	if actor != "owner@example.test" || expires.Sub(before) < 59*time.Minute || expires.Sub(before) > 61*time.Minute {
		t.Fatalf("actor=%s custom expiration=%s", actor, expires.Sub(before))
	}
	ReserveCredits(service, nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/reservations", strings.NewReader(`{"limit_key":"compute_minutes","amount":41}`)))
	if *status != http.StatusPaymentRequired {
		t.Fatalf("pending quota hold must enforce server tier, status=%d message=%s", *status, *message)
	}
	*status = 0
	deps.UserEmail = func(context.Context) string { return "other@example.test" }
	finalize := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/finalize", strings.NewReader(`{"actual_amount":30}`)), map[string]string{"reservationID": reservation.ID})
	FinalizeReservation(service, deps).ServeHTTP(httptest.NewRecorder(), finalize)
	if *status != http.StatusForbidden {
		t.Fatalf("cross-subject quota finalization status=%d", *status)
	}
	*status = 0
	deps.UserEmail = func(context.Context) string { return "owner@example.test" }
	finalize = mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/finalize", strings.NewReader(`{"actual_amount":30}`)), map[string]string{"reservationID": reservation.ID})
	FinalizeReservation(service, deps).ServeHTTP(httptest.NewRecorder(), finalize)
	if *status != 0 {
		t.Fatalf("quota finalization status=%d message=%s", *status, *message)
	}
	var amount int64
	if err := db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity='owner@example.test'`).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 30 {
		t.Fatalf("count usage=%d", amount)
	}
	// Releasing another count reservation preserves the authenticated subject
	// and leaves finalized usage unchanged.
	response = httptest.NewRecorder()
	ReserveCredits(service, nil, deps).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/usage/reservations", strings.NewReader(`{"limit_key":"compute_minutes","amount":40}`)))
	if *status != 0 {
		t.Fatalf("second count reserve status=%d message=%s", *status, *message)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &reservation); err != nil {
		t.Fatal(err)
	}
	release := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/release", nil), map[string]string{"reservationID": reservation.ID})
	ReleaseReservation(service, deps).ServeHTTP(httptest.NewRecorder(), release)
	if *status != 0 {
		t.Fatalf("count release status=%d message=%s", *status, *message)
	}
	var released string
	if err := db.QueryRow(`SELECT status FROM credit_reservations WHERE id=?`, reservation.ID).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != "released" {
		t.Fatalf("count release state=%s", released)
	}
	// No funding tables exist in this fixture: either touching lots or routing
	// cost units to quota accounting would violate this observable contract.
	*status = 0
	ReserveCredits(service, nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/reservations", strings.NewReader(`{"limit_key":"ai_credits","amount":1}`)))
	if *status != http.StatusServiceUnavailable || *message != "Verified billing scope is unavailable" {
		t.Fatalf("cost-based request status=%d message=%s", *status, *message)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_reservations WHERE limit_key='ai_credits'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("credit request reached non-credit quota state")
	}
}

// Direct reports must establish the entire batch's non-credit authority before
// writing. Funding tables deliberately do not exist: denied requests must not
// attempt either account or personal credit mutations.
func TestDirectUsageAdmissionHasNoCallerBillingAuthority(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, query := range []string{
		`CREATE TABLE subscription_tier_limits(id TEXT,tier_id TEXT,limit_type TEXT,limit_key TEXT,limit_value INTEGER,cost_multiplier INTEGER,app_bundle_key TEXT,reset_period TEXT,created_at DATETIME,updated_at DATETIME)`,
		`INSERT INTO subscription_tier_limits VALUES('count','pro','count_based','compute_minutes',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('cost','pro','cost_based','ai_credits',100,1000000,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('empty','pro','','empty_kind',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('unknown','pro','mystery','unknown',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('scoped','pro','count_based','scoped_count',100,0,'console','monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('global-scoped','pro','count_based','scoped_count',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('dup1','pro','count_based','duplicate',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('dup2','pro','count_based','duplicate',100,0,NULL,'monthly',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`CREATE TABLE usage_records(user_identity TEXT,billing_period TEXT,limit_key TEXT,usage_amount INTEGER,app_bundle_key TEXT,operation_id TEXT,last_operation_at DATETIME,updated_at DATETIME,UNIQUE(user_identity,billing_period,limit_key,app_bundle_key))`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	service := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: db, LimitsService: commerce.NewLimitsService(db, "sqlite", nil), Dialect: "sqlite"})
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"forged account", `{"limit_key":"ai_credits","amount":1,"business_account_id":"victim"}`, 503},
		{"count account is not authority", `{"limit_key":"compute_minutes","amount":1,"business_account_id":"victim"}`, 503},
		{"selected app", `{"limit_key":"compute_minutes","amount":1,"app_bundle_key":"console"}`, 503},
		{"absent credit app", `{"limit_key":"ai_credits","amount":1}`, 503},
		{"byok cannot bypass", `{"limit_key":"ai_credits","amount":0,"is_byok":true}`, 503},
		{"absent count app binding", `{"limit_key":"scoped_count","amount":1}`, 503},
		{"missing kind", `{"limit_key":"empty_kind","amount":1}`, 400},
		{"unknown kind", `{"limit_key":"unknown","amount":1}`, 400},
		{"duplicate definition", `{"limit_key":"duplicate","amount":1}`, 400},
		{"missing definition", `{"limit_key":"missing","amount":1}`, 400},
		{"missing key", `{"amount":1}`, 400},
		{"mixed cost batch", `[{"limit_key":"compute_minutes","amount":1},{"limit_key":"ai_credits","amount":1}]`, 503},
		{"mixed app batch", `[{"limit_key":"compute_minutes","amount":1},{"limit_key":"scoped_count","amount":1}]`, 503},
		{"mixed unknown batch", `[{"limit_key":"compute_minutes","amount":1},{"limit_key":"unknown","amount":1}]`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, status, _ := usageTestDependencies(t, "owner@example.test")
			deps.SubscriptionTier = func(context.Context, string) (string, error) { return "pro", nil }
			deps.WriteError = func(_ http.ResponseWriter, s int, _ string, code string) {
				*status = s
				if (s == 503 && code != "billing_scope_unavailable") || (s == 400 && code != "validation") {
					t.Errorf("code=%s", code)
				}
			}
			ReportUsage(service, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(tc.body)))
			if *status != tc.status {
				t.Fatalf("status=%d want=%d", *status, tc.status)
			}
			var count int
			if err := db.QueryRow(`SELECT COUNT(*) FROM usage_records`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("rejected batch mutated usage")
			}
			if !strings.HasPrefix(tc.body, "[") {
				var direct commerce.UsageReportRequest
				if err := json.Unmarshal([]byte(tc.body), &direct); err != nil {
					t.Fatal(err)
				}
				direct.UserIdentity, direct.Tier = "owner@example.test", "pro"
				if err := service.RecordUsage(context.Background(), direct); err == nil {
					t.Fatal("service accepted unauthorized direct usage")
				}
				if err := db.QueryRow(`SELECT COUNT(*) FROM usage_records`).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatal("service denial mutated usage")
				}
			}
		})
	}
	// A configured global count meter retains the established user quota owner.
	deps, status, message := usageTestDependencies(t, "owner@example.test")
	deps.SubscriptionTier = func(_ context.Context, subject string) (string, error) {
		if subject != "owner@example.test" {
			t.Fatal(subject)
		}
		return "pro", nil
	}
	ReportUsage(service, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(`{"limit_key":"compute_minutes","amount":3,"user_identity":"victim@example.test","tier":"business"}`)))
	if *status != 0 {
		t.Fatalf("status=%d message=%s", *status, *message)
	}
	var subject string
	var amount int64
	var app sql.NullString
	if err := db.QueryRow(`SELECT user_identity,usage_amount,app_bundle_key FROM usage_records`).Scan(&subject, &amount, &app); err != nil {
		t.Fatal(err)
	}
	if subject != "owner@example.test" || amount != 3 || app.Valid {
		t.Fatalf("subject=%s amount=%d app=%v", subject, amount, app)
	}
	// BYOK retains a zero-amount global count observation and its metadata.
	deps, status, message = usageTestDependencies(t, "byok@example.test")
	deps.SubscriptionTier = func(context.Context, string) (string, error) { return "pro", nil }
	var observed bool
	byokService := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: db, LimitsService: commerce.NewLimitsService(db, "sqlite", nil), Dialect: "sqlite", Log: func(event string, fields map[string]interface{}) {
		if event == "usage_recorded" {
			observed = fields["is_byok"] == true && fields["amount"] == int64(0) && fields["user_identity"] == "byok@example.test" && fields["limit_key"] == "compute_minutes"
		}
	}})
	response := httptest.NewRecorder()
	ReportUsage(byokService, deps).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(`{"limit_key":"compute_minutes","amount":7,"is_byok":true}`)))
	if *status != 0 || response.Code != http.StatusOK || !observed {
		t.Fatalf("status=%d response=%d observed=%v message=%s", *status, response.Code, observed, *message)
	}
	if err := db.QueryRow(`SELECT usage_amount,app_bundle_key FROM usage_records WHERE user_identity='byok@example.test'`).Scan(&amount, &app); err != nil {
		t.Fatal(err)
	}
	if amount != 0 || app.Valid {
		t.Fatalf("BYOK amount=%d app=%v", amount, app)
	}
	// The service boundary independently denies a forged cost request.
	if err := service.RecordUsage(context.Background(), commerce.UsageReportRequest{UserIdentity: "owner@example.test", Tier: "pro", LimitKey: "ai_credits", Amount: 1, BusinessAccountID: "victim"}); err != commerce.ErrDirectUsageScopeUnavailable {
		t.Fatalf("service denial=%v", err)
	}
}

func TestDirectUsageRequiresServerTier(t *testing.T) {
	for _, tier := range []string{"", "error"} {
		t.Run(tier, func(t *testing.T) {
			deps, status, _ := usageTestDependencies(t, "owner@example.test")
			deps.WriteError = func(_ http.ResponseWriter, s int, _ string, code string) {
				*status = s
				if code != "meter_authority_unavailable" {
					t.Errorf("code=%s", code)
				}
			}
			if tier == "error" {
				deps.SubscriptionTier = func(context.Context, string) (string, error) { return "", sql.ErrNoRows }
			}
			ReportUsage(nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(`{"limit_key":"compute_minutes","amount":1,"tier":"pro"}`)))
			if *status != http.StatusServiceUnavailable {
				t.Fatalf("status=%d", *status)
			}
		})
	}
}

// Fail each authoritative read, including the resolver and RecordUsage recheck.
type failingDirectLimits struct {
	commerce.LimitsServicer
	calls, failAt int
	cause         error
}

func (f *failingDirectLimits) GetTierLimits(context.Context, string) ([]commerce.TierLimit, error) {
	f.calls++
	if f.calls == f.failAt {
		if f.cause != nil {
			return nil, f.cause
		}
		return nil, errors.New("private database failure")
	}
	return []commerce.TierLimit{{LimitKey: "compute_minutes", LimitType: "count_based"}}, nil
}
func TestDirectUsageAuthorityReadFailuresAreRetryable(t *testing.T) {
	for failAt := 1; failAt <= 4; failAt++ {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			authority := &failingDirectLimits{failAt: failAt}
			service := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{LimitsService: authority})
			deps, status, _ := usageTestDependencies(t, "owner@example.test")
			deps.SubscriptionTier = func(context.Context, string) (string, error) { return "pro", nil }
			deps.WriteError = func(_ http.ResponseWriter, s int, message, code string) {
				*status = s
				if code != "meter_authority_unavailable" || message != commerce.ErrDirectUsageAuthorityUnavailable.Error() {
					t.Errorf("message=%q code=%s", message, code)
				}
			}
			ReportUsage(service, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(`{"limit_key":"compute_minutes","amount":1}`)))
			if *status != http.StatusServiceUnavailable || authority.calls != failAt {
				t.Fatalf("status=%d reads=%d", *status, authority.calls)
			}
			// A nil usage store would panic on any persistence attempt.
		})
	}
}

func TestDirectUsageAuthorityFailurePreservesCause(t *testing.T) {
	cause := errors.New("private injected database failure")
	service := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		LimitsService: &failingDirectLimits{failAt: 1, cause: cause},
	})
	err := service.ValidateDirectUsage(context.Background(), commerce.UsageReportRequest{
		UserIdentity: "owner@example.test", Tier: "pro", LimitKey: "compute_minutes", Amount: 1,
	})
	if !errors.Is(err, commerce.ErrDirectUsageAuthorityUnavailable) || !errors.Is(err, cause) {
		t.Fatalf("authority wrapper did not preserve sentinel and cause: %v", err)
	}
}

func TestDirectUsagePreflightDiagnosticsAreBounded(t *testing.T) {
	for _, reason := range []string{"missing_callback", "empty_tier", "lookup_failed", "lookup_unavailable"} {
		t.Run(reason, func(t *testing.T) {
			deps, status, message := usageTestDependencies(t, "private-owner@example.test")
			authority := "subscription_tier"
			var service *commerce.UsageService
			switch reason {
			case "empty_tier":
				deps.SubscriptionTier = func(context.Context, string) (string, error) { return "", nil }
			case "lookup_failed":
				deps.SubscriptionTier = func(context.Context, string) (string, error) { return "", errors.New("private database failure") }
			case "lookup_unavailable":
				authority = "tier_limits"
				deps.SubscriptionTier = func(context.Context, string) (string, error) { return "pro", nil }
				service = commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{LimitsService: &failingDirectLimits{failAt: 1}})
			}
			logs := 0
			deps.LogError = func(event string, fields map[string]any) {
				logs++
				if event != "report_usage_preflight_failed" || len(fields) != 4 || fields["phase"] != "preflight" || fields["authority"] != authority || fields["reason"] != reason || fields["code"] != "meter_authority_unavailable" {
					t.Errorf("unexpected diagnostic: %s %#v", event, fields)
				}
			}
			ReportUsage(service, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/usage/report", strings.NewReader(`{"limit_key":"compute_minutes","amount":1,"operation_id":"private-request"}`)))
			if logs != 1 || *status != http.StatusServiceUnavailable || *message != commerce.ErrDirectUsageAuthorityUnavailable.Error() {
				t.Fatalf("logs=%d status=%d message=%q", logs, *status, *message)
			}
		})
	}
}
