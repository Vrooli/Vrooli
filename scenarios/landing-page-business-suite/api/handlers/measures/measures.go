// Package measures exposes landing-page-business-suite's typed analytical
// measures. It uses fixed, reviewed SQL aggregates over authoritative domain
// state; callers never provide a table name or SQL fragment.
package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	measurelib "github.com/vrooli/measures-go"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures/measuresv1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	measurestore "landing-page-business-suite-api/internal/measures"
)

const (
	MeasureSubscriptionsCreated           = "subscriptions.created"
	MeasureCreditTransactionsCreated      = "credit_transactions.created"
	MeasureCheckoutSessionsCreated        = "checkout_sessions.created"
	MeasureBundleProductsCreated          = "bundle_products.created"
	MeasureBundlePricesCreated            = "bundle_prices.created"
	MeasureSubscriptionSchedulesCreated   = "subscription_schedules.created"
	MeasureIntroCouponUsageCreated        = "intro_coupon_usage.created"
	MeasurePaymentAnomaliesCreated        = "payment_anomaly_log.created"
	MeasureUsersCreated                   = "users.created"
	MeasureUserSessionsCreated            = "user_sessions.created"
	MeasureAuthTokensCreated              = "auth_tokens.created" // #nosec G101 -- fixed database table identifier, never a credential
	MeasureProviderKeysCreated            = "api_keys.created"
	MeasureCreditReservationsCreated      = "credit_reservations.created"
	MeasureSubscriptionTierLimitsCreated  = "subscription_tier_limits.created"
	MeasureUsageRecordsCreated            = "usage_records.created"
	MeasureAdminSessionsCreated           = "admin_sessions.created"
	MeasureAdminUsersCreated              = "admin_users.created"
	MeasureAssetsCreated                  = "assets.created"
	MeasureDownloadAppsCreated            = "download_apps.created"
	MeasureDownloadArtifactsCreated       = "download_artifacts.created"
	MeasureDownloadAssetsCreated          = "download_assets.created"
	MeasureDownloadStorageSettingsCreated = "download_storage_settings.created"
	MeasureFeedbackRequestsCreated        = "feedback_requests.created"
	MeasureMetricsEventsCreated           = "metrics_events.created"
	MeasureRemoteProfilesCreated          = "remote_profiles.created"
	MeasureWaitlistEmailsCreated          = "waitlist_emails.created"
)

type spec struct {
	declaration measurelib.MeasureDeclaration
}

func specs() []spec {
	return []spec{
		newCreatedCountSpec(MeasureSubscriptionsCreated, "subscriptions", "CountSubscriptionsCreated", "How many subscriptions were created in a time window.", []string{"how many subscriptions were created this week", "new subscriptions last month", "subscription signups in the last 30 days"}),
		newCreatedCountSpec(MeasureCreditTransactionsCreated, "credit_transactions", "CountCreditTransactionsCreated", "How many credit transactions were recorded in a time window.", []string{"how many credit transactions occurred this week", "credit transactions last month", "credit activity in the last 30 days"}),
		newCreatedCountSpec(MeasureCheckoutSessionsCreated, "checkout_sessions", "CountCheckoutSessionsCreated", "How many checkout sessions were created in a time window.", []string{"how many checkout sessions were created this week", "checkout starts last month", "checkout sessions in the last 30 days"}),
		newCreatedCountSpec(MeasureBundleProductsCreated, "bundle_products", "CountBundleProductsCreated", "How many bundle products were added to the commercial catalog in a time window.", []string{"how many bundle products were added this week", "new bundle products last month", "catalog products created in the last 30 days"}),
		newCreatedCountSpec(MeasureBundlePricesCreated, "bundle_prices", "CountBundlePricesCreated", "How many bundle prices were added to the commercial catalog in a time window.", []string{"how many bundle prices were added this week", "new pricing options last month", "catalog prices created in the last 30 days"}),
		newCreatedCountSpec(MeasureSubscriptionSchedulesCreated, "subscription_schedules", "CountSubscriptionSchedulesCreated", "How many subscription schedules were created in a time window.", []string{"how many subscription schedules were created this week", "scheduled subscriptions last month", "subscription schedules in the last 30 days"}),
		newCreatedCountSpec(MeasureIntroCouponUsageCreated, "intro_coupon_usage", "CountIntroCouponUsageCreated", "How many introductory coupons were used in a time window.", []string{"how many introductory coupons were used this week", "intro coupon usage last month", "coupon redemptions in the last 30 days"}),
		newCreatedCountSpec(MeasurePaymentAnomaliesCreated, "payment_anomaly_log", "CountPaymentAnomaliesCreated", "How many payment anomalies were recorded in a time window.", []string{"how many payment anomalies occurred this week", "payment anomaly count last month", "payment issues in the last 30 days"}),
		newCreatedCountSpec(MeasureUsersCreated, "users", "CountUsersCreated", "How many customer accounts were created in a time window.", []string{"how many users signed up this week", "new customer accounts last month", "user registrations in the last 30 days"}),
		newCreatedCountSpec(MeasureUserSessionsCreated, "user_sessions", "CountUserSessionsCreated", "How many user sessions were created in a time window.", []string{"how many user sessions were created this week", "new customer sessions last month", "sessions issued in the last 30 days"}),
		newCreatedCountSpec(MeasureAuthTokensCreated, "auth_tokens", "CountAuthTokensCreated", "How many authentication tokens were issued in a time window.", []string{"how many auth tokens were issued this week", "authentication tokens last month", "tokens issued in the last 30 days"}),
		newCreatedCountSpec(MeasureProviderKeysCreated, "api_keys", "CountAPIKeysCreated", "How many provider API keys were registered in a time window.", []string{"how many API keys were registered this week", "new provider keys last month", "API key registrations in the last 30 days"}),
		newCreatedCountSpec(MeasureCreditReservationsCreated, "credit_reservations", "CountCreditReservationsCreated", "How many credit reservations were created in a time window.", []string{"how many credit reservations were created this week", "credit reservations last month", "credit holds in the last 30 days"}),
		newCreatedCountSpec(MeasureSubscriptionTierLimitsCreated, "subscription_tier_limits", "CountSubscriptionTierLimitsCreated", "How many subscription tier limits were configured in a time window.", []string{"how many tier limits were configured this week", "new subscription limits last month", "tier limits created in the last 30 days"}),
		newCreatedCountSpec(MeasureUsageRecordsCreated, "usage_records", "CountUsageRecordsCreated", "How many usage records were created in a time window.", []string{"how many usage records were created this week", "usage records last month", "usage records in the last 30 days"}),
		newCreatedCountSpec(MeasureAdminSessionsCreated, "admin_sessions", "CountAdminSessionsCreated", "How many administrator sessions were created in a time window.", []string{"how many admin sessions were created this week", "administrator sessions last month", "admin sign-ins in the last 30 days"}),
		newCreatedCountSpec(MeasureAdminUsersCreated, "admin_users", "CountAdminUsersCreated", "How many administrator accounts were created in a time window.", []string{"how many admin users were created this week", "new administrators last month", "admin account registrations in the last 30 days"}),
		newCreatedCountSpec(MeasureAssetsCreated, "assets", "CountAssetsCreated", "How many content assets were created in a time window.", []string{"how many content assets were created this week", "new assets last month", "assets created in the last 30 days"}),
		newCreatedCountSpec(MeasureDownloadAppsCreated, "download_apps", "CountDownloadAppsCreated", "How many downloadable applications were created in a time window.", []string{"how many download apps were created this week", "new downloadable applications last month", "download apps created in the last 30 days"}),
		newCreatedCountSpec(MeasureDownloadArtifactsCreated, "download_artifacts", "CountDownloadArtifactsCreated", "How many downloadable artifacts were created in a time window.", []string{"how many download artifacts were created this week", "new download artifacts last month", "release artifacts created in the last 30 days"}),
		newCreatedCountSpec(MeasureDownloadAssetsCreated, "download_assets", "CountDownloadAssetsCreated", "How many downloadable assets were created in a time window.", []string{"how many download assets were created this week", "new downloadable assets last month", "download assets created in the last 30 days"}),
		newCreatedCountSpec(MeasureDownloadStorageSettingsCreated, "download_storage_settings", "CountDownloadStorageSettingsCreated", "How many download storage settings were created in a time window.", []string{"how many download storage settings were created this week", "new download storage settings last month", "download storage settings created in the last 30 days"}),
		newCreatedCountSpec(MeasureFeedbackRequestsCreated, "feedback_requests", "CountFeedbackRequestsCreated", "How many feedback requests were created in a time window.", []string{"how many feedback requests were created this week", "new feedback requests last month", "feedback requests in the last 30 days"}),
		newCreatedCountSpec(MeasureMetricsEventsCreated, "metrics_events", "CountMetricsEventsCreated", "How many analytics events were recorded in a time window.", []string{"how many metrics events were recorded this week", "analytics events last month", "metrics events in the last 30 days"}),
		newCreatedCountSpec(MeasureRemoteProfilesCreated, "remote_profiles", "CountRemoteProfilesCreated", "How many remote profiles were created in a time window.", []string{"how many remote profiles were created this week", "new remote profiles last month", "remote profiles created in the last 30 days"}),
		newCreatedCountSpec(MeasureWaitlistEmailsCreated, "waitlist_emails", "CountWaitlistEmailsCreated", "How many waitlist signups were created in a time window.", []string{"how many waitlist signups were created this week", "new waitlist signups last month", "waitlist emails in the last 30 days"}),
	}
}

func newCreatedCountSpec(name, domain, method, intent string, questions []string) spec {
	return spec{declaration: measurelib.MeasureDeclaration{
		Name:        name,
		Scenario:    "landing-page-business-suite",
		Domain:      domain,
		Intent:      intent,
		Questions:   questions,
		Params:      map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:      measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "count", Unit: "records", SummaryTemplate: "{count} " + domain + " created ({window})"},
		Effect:      measurelib.EffectRead,
		RunEligible: true,
		Service:     "MeasuresService",
		Method:      method,
	}}
}

// NewRegistry returns the measure serve registry using the standard SQL-backed
// repository. NewRegistryWithRepository is available to composition roots that
// already own a repository implementation.
func NewRegistry(db measurestore.Counter, now func() time.Time) (*measurelib.Registry, error) {
	return NewRegistryWithRepository(measurestore.NewSQLRepository(db), now)
}

// NewRegistryWithRepository preserves registry and Connect semantic parity
// while keeping query selection and execution outside transport code.
func NewRegistryWithRepository(repository measurestore.Repository, now func() time.Time) (*measurelib.Registry, error) {
	if now == nil {
		now = time.Now
	}
	registry := measurelib.NewRegistry(measurelib.WithClock(now))
	for _, item := range specs() {
		item := item
		if err := registry.Register(item.declaration, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			window, err := resolveToken(request.Params["window"], now())
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			count, err := repository.CountCreated(ctx, item.declaration.Name, window.From, window.To)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			query, ok := repository.QueryFor(item.declaration.Name)
			if !ok {
				return measurelib.MeasureResult{}, fmt.Errorf("catalog entry missing for %s", item.declaration.Name)
			}
			return measurelib.MeasureResult{
				Value:      strconv.FormatInt(count, 10),
				Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf("%s; window=[%s,%s)", query, window.From.UTC().Format(time.RFC3339), window.To.UTC().Format(time.RFC3339))},
			}, nil
		}); err != nil {
			return nil, fmt.Errorf("register %s: %w", item.declaration.Name, err)
		}
	}
	return registry, nil
}

func resolveToken(token string, now time.Time) (measurelib.Range, error) {
	if token == "" {
		token = string(measurelib.TokenThisWeek)
	}
	return measurelib.ResolveToken(measurelib.TimeWindowToken(token), now, time.UTC)
}

func resolveProtoWindow(window *measuresv1.TimeWindow, now time.Time) (measurelib.Range, error) {
	if window == nil || window.GetWindow() == nil {
		return measurelib.ResolveToken(measurelib.TokenThisWeek, now, time.UTC)
	}
	return measurelib.ResolveTimeWindow(window, now, time.UTC)
}

// Handler implements the typed Connect surface with the same aggregate path as
// the registry. It intentionally exposes only fixed methods, never an entity
// selector that could blur domains or introduce SQL injection risk.
type Handler struct {
	repository measurestore.Repository
	now        func() time.Time
	byName     map[string]spec
}

func NewHandler(db measurestore.Counter, now func() time.Time) *Handler {
	return NewHandlerWithRepository(measurestore.NewSQLRepository(db), now)
}

func NewHandlerWithRepository(repository measurestore.Repository, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	byName := make(map[string]spec, len(specs()))
	for _, item := range specs() {
		byName[item.declaration.Name] = item
	}
	return &Handler{repository: repository, now: now, byName: byName}
}

func (h *Handler) countWindow(ctx context.Context, name string, window *measuresv1.TimeWindow) (int64, error) {
	rangeValue, err := resolveProtoWindow(window, h.now())
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	item, ok := h.byName[name]
	if !ok {
		return 0, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown measure %q", name))
	}
	result, err := h.repository.CountCreated(ctx, item.declaration.Name, rangeValue.From, rangeValue.To)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, err)
	}
	return result, nil
}

func (h *Handler) CountSubscriptionsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountSubscriptionsCreatedRequest]) (*connect.Response[lpbsv1.CountSubscriptionsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureSubscriptionsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountSubscriptionsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountCreditTransactionsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountCreditTransactionsCreatedRequest]) (*connect.Response[lpbsv1.CountCreditTransactionsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureCreditTransactionsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountCreditTransactionsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountCheckoutSessionsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountCheckoutSessionsCreatedRequest]) (*connect.Response[lpbsv1.CountCheckoutSessionsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureCheckoutSessionsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountCheckoutSessionsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountBundleProductsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountBundleProductsCreatedRequest]) (*connect.Response[lpbsv1.CountBundleProductsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureBundleProductsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountBundleProductsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountBundlePricesCreated(ctx context.Context, req *connect.Request[lpbsv1.CountBundlePricesCreatedRequest]) (*connect.Response[lpbsv1.CountBundlePricesCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureBundlePricesCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountBundlePricesCreatedResponse{Count: result}), nil
}

func (h *Handler) CountSubscriptionSchedulesCreated(ctx context.Context, req *connect.Request[lpbsv1.CountSubscriptionSchedulesCreatedRequest]) (*connect.Response[lpbsv1.CountSubscriptionSchedulesCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureSubscriptionSchedulesCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountSubscriptionSchedulesCreatedResponse{Count: result}), nil
}

func (h *Handler) CountIntroCouponUsageCreated(ctx context.Context, req *connect.Request[lpbsv1.CountIntroCouponUsageCreatedRequest]) (*connect.Response[lpbsv1.CountIntroCouponUsageCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureIntroCouponUsageCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountIntroCouponUsageCreatedResponse{Count: result}), nil
}

func (h *Handler) CountPaymentAnomaliesCreated(ctx context.Context, req *connect.Request[lpbsv1.CountPaymentAnomaliesCreatedRequest]) (*connect.Response[lpbsv1.CountPaymentAnomaliesCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasurePaymentAnomaliesCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountPaymentAnomaliesCreatedResponse{Count: result}), nil
}

func (h *Handler) CountUsersCreated(ctx context.Context, req *connect.Request[lpbsv1.CountUsersCreatedRequest]) (*connect.Response[lpbsv1.CountUsersCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureUsersCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountUsersCreatedResponse{Count: result}), nil
}

func (h *Handler) CountUserSessionsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountUserSessionsCreatedRequest]) (*connect.Response[lpbsv1.CountUserSessionsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureUserSessionsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountUserSessionsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountAuthTokensCreated(ctx context.Context, req *connect.Request[lpbsv1.CountAuthTokensCreatedRequest]) (*connect.Response[lpbsv1.CountAuthTokensCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureAuthTokensCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountAuthTokensCreatedResponse{Count: result}), nil
}

func (h *Handler) CountAPIKeysCreated(ctx context.Context, req *connect.Request[lpbsv1.CountAPIKeysCreatedRequest]) (*connect.Response[lpbsv1.CountAPIKeysCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureProviderKeysCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountAPIKeysCreatedResponse{Count: result}), nil
}

func (h *Handler) CountCreditReservationsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountCreditReservationsCreatedRequest]) (*connect.Response[lpbsv1.CountCreditReservationsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureCreditReservationsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountCreditReservationsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountSubscriptionTierLimitsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountSubscriptionTierLimitsCreatedRequest]) (*connect.Response[lpbsv1.CountSubscriptionTierLimitsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureSubscriptionTierLimitsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountSubscriptionTierLimitsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountUsageRecordsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountUsageRecordsCreatedRequest]) (*connect.Response[lpbsv1.CountUsageRecordsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureUsageRecordsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountUsageRecordsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountAdminSessionsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountAdminSessionsCreatedRequest]) (*connect.Response[lpbsv1.CountAdminSessionsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureAdminSessionsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountAdminSessionsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountAdminUsersCreated(ctx context.Context, req *connect.Request[lpbsv1.CountAdminUsersCreatedRequest]) (*connect.Response[lpbsv1.CountAdminUsersCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureAdminUsersCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountAdminUsersCreatedResponse{Count: result}), nil
}

func (h *Handler) CountAssetsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountAssetsCreatedRequest]) (*connect.Response[lpbsv1.CountAssetsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureAssetsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountAssetsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountDownloadAppsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountDownloadAppsCreatedRequest]) (*connect.Response[lpbsv1.CountDownloadAppsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureDownloadAppsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountDownloadAppsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountDownloadArtifactsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountDownloadArtifactsCreatedRequest]) (*connect.Response[lpbsv1.CountDownloadArtifactsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureDownloadArtifactsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountDownloadArtifactsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountDownloadAssetsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountDownloadAssetsCreatedRequest]) (*connect.Response[lpbsv1.CountDownloadAssetsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureDownloadAssetsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountDownloadAssetsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountDownloadStorageSettingsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountDownloadStorageSettingsCreatedRequest]) (*connect.Response[lpbsv1.CountDownloadStorageSettingsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureDownloadStorageSettingsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountDownloadStorageSettingsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountFeedbackRequestsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountFeedbackRequestsCreatedRequest]) (*connect.Response[lpbsv1.CountFeedbackRequestsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureFeedbackRequestsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountFeedbackRequestsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountMetricsEventsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountMetricsEventsCreatedRequest]) (*connect.Response[lpbsv1.CountMetricsEventsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureMetricsEventsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountMetricsEventsCreatedResponse{Count: result}), nil
}

func (h *Handler) CountRemoteProfilesCreated(ctx context.Context, req *connect.Request[lpbsv1.CountRemoteProfilesCreatedRequest]) (*connect.Response[lpbsv1.CountRemoteProfilesCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureRemoteProfilesCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountRemoteProfilesCreatedResponse{Count: result}), nil
}

func (h *Handler) CountWaitlistEmailsCreated(ctx context.Context, req *connect.Request[lpbsv1.CountWaitlistEmailsCreatedRequest]) (*connect.Response[lpbsv1.CountWaitlistEmailsCreatedResponse], error) {
	result, err := h.countWindow(ctx, MeasureWaitlistEmailsCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.CountWaitlistEmailsCreatedResponse{Count: result}), nil
}

// RegisterRoutes mounts the typed Connect surface. The caller owns the access
// policy so commercial aggregates can remain admin/service-only.
func RegisterRoutes(router *mux.Router, db measurestore.Counter, now func() time.Time, middleware func(http.HandlerFunc) http.HandlerFunc) error {
	repository := measurestore.NewSQLRepository(db)
	registry, err := NewRegistryWithRepository(repository, now)
	if err != nil {
		return err
	}
	path, connectHandler := lpbsconnect.NewMeasuresServiceHandler(NewHandlerWithRepository(repository, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: middleware(connectHandler.ServeHTTP)})
	router.PathPrefix("/api/v1/measures/").Handler(middleware(http.StripPrefix("/api/v1/measures", registry.Handler()).ServeHTTP))
	return nil
}
