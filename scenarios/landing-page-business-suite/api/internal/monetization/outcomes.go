package monetization

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// StripeOutcomeReport contains bounded, secret-free evidence from a real
// Stripe test-mode execution. Unrun outcomes remain explicitly unknown.
type StripeOutcomeReport struct {
	Status          string               `json:"status"`
	ExecutionID     string               `json:"execution_id,omitempty"`
	HostEnvironment string               `json:"host_environment"`
	StripeMode      string               `json:"stripe_mode"`
	LocalStandIn    bool                 `json:"local_stand_in"`
	Catalog         CatalogReplay        `json:"catalog"`
	Outcomes        []OutcomeObservation `json:"outcomes"`
	Limitations     []string             `json:"limitations"`
}

type CatalogReplay struct {
	Status               string                `json:"status"`
	VerifiedPrices       int                   `json:"verified_prices"`
	Mismatches           []string              `json:"mismatches,omitempty"`
	DonationPrice        string                `json:"donation_price_status"`
	PriceReconciliations []PriceReconciliation `json:"price_reconciliations,omitempty"`
	Changes              []string              `json:"test_mode_changes,omitempty"`
}

type PriceReconciliation struct {
	PlanName              string `json:"plan_name"`
	PlanTier              string `json:"plan_tier"`
	Kind                  string `json:"kind"`
	PriceID               string `json:"price_id"`
	Status                string `json:"status"`
	ExpectedAmountCents   int64  `json:"expected_amount_cents"`
	ExpectedCurrency      string `json:"expected_currency"`
	ExpectedInterval      string `json:"expected_interval"`
	ActualPriceID         string `json:"actual_price_id,omitempty"`
	ActualFieldsAvailable bool   `json:"actual_fields_available"`
	ActualAmountCents     int64  `json:"actual_amount_cents,omitempty"`
	ActualCurrency        string `json:"actual_currency,omitempty"`
	ActualInterval        string `json:"actual_interval,omitempty"`
	ActualActive          bool   `json:"actual_active"`
	LookupFailureStatus   int    `json:"lookup_failure_status,omitempty"`
	LookupFailureCode     string `json:"lookup_failure_code,omitempty"`
}

type OutcomeObservation struct {
	Name                   string   `json:"name"`
	Status                 string   `json:"status"`
	Detail                 string   `json:"detail,omitempty"`
	StripeEventID          string   `json:"stripe_event_id,omitempty"`
	RelatedStripeEventIDs  []string `json:"related_stripe_event_ids,omitempty"`
	StripeEventType        string   `json:"stripe_event_type,omitempty"`
	StripeObjectID         string   `json:"stripe_object_id,omitempty"`
	PostgresReceiptStatus  string   `json:"postgres_receipt_status,omitempty"`
	CheckoutSessionID      string   `json:"checkout_session_id,omitempty"`
	PaymentIntentID        string   `json:"payment_intent_id,omitempty"`
	SubscriptionID         string   `json:"subscription_id,omitempty"`
	SubscriptionStatus     string   `json:"subscription_status,omitempty"`
	PaymentStatus          string   `json:"payment_status,omitempty"`
	BillingReason          string   `json:"billing_reason,omitempty"`
	LifecycleReceiptStatus string   `json:"postgres_lifecycle_receipt_status,omitempty"`
	EconomicReceiptStatus  string   `json:"postgres_economic_receipt_status,omitempty"`
	AmountMinor            int64    `json:"amount_minor,omitempty"`
	Currency               string   `json:"currency,omitempty"`
	MoneyLedgerExternalID  string   `json:"money_ledger_external_id,omitempty"`
	MoneyLedgerStatus      string   `json:"money_ledger_status,omitempty"`
	MoneyLedgerPostingID   string   `json:"money_ledger_posting_id,omitempty"`
	MoneyLedgerReceiptID   string   `json:"money_ledger_receipt_id,omitempty"`
}

func (h Handler) validateStripeOutcomes(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if !req.Msg.GetIncludeExecution() {
		return stripeOutcomeResponse(req.Msg.GetScenario(), &StripeOutcomeReport{
			Status: "not_evaluated", HostEnvironment: "unknown", StripeMode: "unknown", Catalog: CatalogReplay{Status: "not_evaluated", DonationPrice: "not_evaluated"},
			Limitations: []string{"This phase performs no Stripe calls unless Test Genie requests execution."},
		}, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED), nil
	}
	if h.outcomeRunner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("LPBS Stripe outcome runner is unavailable"))
	}
	report, err := h.outcomeRunner(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("LPBS Stripe outcome replay failed: %w", err))
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED
	switch report.Status {
	case "passed":
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	case "failed":
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	case "not_evaluated", "":
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED
	default:
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("LPBS Stripe outcome runner returned invalid status"))
	}
	return stripeOutcomeResponse(req.Msg.GetScenario(), report, status), nil
}

func stripeOutcomeResponse(scenario string, report *StripeOutcomeReport, status scenariovalidationv1.ValidationStatus) *connect.Response[scenariovalidationv1.ValidateScenarioResponse] {
	fields, err := structpb.NewStruct(map[string]any{
		"scenario": scenario,
		"stripe_test_outcomes": map[string]any{
			"status": report.Status, "execution_id": report.ExecutionID, "host_environment": report.HostEnvironment, "stripe_mode": report.StripeMode, "local_stand_in": report.LocalStandIn,
			"catalog": map[string]any{"status": report.Catalog.Status, "verified_prices": report.Catalog.VerifiedPrices,
				"mismatches": stringValues(report.Catalog.Mismatches), "donation_price_status": report.Catalog.DonationPrice,
				"price_reconciliations": priceReconciliationMaps(report.Catalog.PriceReconciliations), "test_mode_changes": stringValues(report.Catalog.Changes)},
			"outcomes": outcomeMaps(report.Outcomes), "limitations": stringValues(report.Limitations),
		},
	})
	if err != nil {
		// The report contains only JSON-safe scalars and slices. Preserve a
		// typed provider error rather than silently dropping its evidence.
		return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
			Scenario: scenario, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR,
			FailureClassification: "system",
		})
	}
	native, _ := anypb.New(fields)
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario: scenario, Status: status, NativeDetail: native,
		Assessment: buildStripeOutcomeAssessment(scenario, report), FailureClassification: failureClass(status),
	})
}

func priceReconciliationMaps(items []PriceReconciliation) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"plan_name": item.PlanName, "plan_tier": item.PlanTier, "kind": item.Kind, "price_id": item.PriceID,
			"status": item.Status, "expected_amount_cents": item.ExpectedAmountCents, "expected_currency": item.ExpectedCurrency,
			"expected_interval": item.ExpectedInterval, "actual_price_id": item.ActualPriceID, "actual_amount_cents": item.ActualAmountCents,
			"actual_fields_available": item.ActualFieldsAvailable, "actual_currency": item.ActualCurrency,
			"actual_interval": item.ActualInterval, "actual_active": item.ActualActive,
			"lookup_failure_status": item.LookupFailureStatus, "lookup_failure_code": item.LookupFailureCode,
		})
	}
	return result
}

func buildStripeOutcomeAssessment(scenario string, report *StripeOutcomeReport) *commonv1.MaturityAssessment {
	level, next := "L0", "L1"
	if report != nil && report.Catalog.Status == "passed" && report.Catalog.DonationPrice == "verified" {
		level, next = "L1", "L2"
	}
	if report != nil && outcomeStatus(report, "real_webhook_delivery") == "passed" {
		level, next = "L2", "L3"
	}
	findings := make([]*commonv1.AssessmentFinding, 0, 2)
	blockingCodes := make([]string, 0, 2)
	if report == nil || report.Catalog.Status != "passed" || report.Catalog.DonationPrice != "verified" {
		findings = append(findings, &commonv1.AssessmentFinding{
			Code: "stripe.catalog_mismatch", Severity: "SEVERITY_ERROR", Title: "Stripe test catalog does not match LPBS",
			Message:     "At least one configured LPBS price is missing, unavailable, or differs from its Stripe test-mode fields.",
			Remediation: "Reconcile each test-mode Stripe price with its LPBS plan and configure a donation price before replaying payment outcomes.",
			FixClass:    "manual",
			Maturity: &commonv1.FindingMaturity{
				CapabilityId: "stripe_outcomes", LocalLevel: "L1",
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
				Dimension:    "business", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED,
			},
		})
		blockingCodes = append(blockingCodes, "stripe.catalog_mismatch")
	}
	if !allOutcomesPassed(report) {
		findings = append(findings, &commonv1.AssessmentFinding{
			Code: "stripe.behavior_not_replayed", Severity: "SEVERITY_ERROR", Title: "Required Stripe outcomes remain unverified",
			Message:     "At least one required payment lifecycle outcome has no retained Stripe test receipt.",
			Remediation: "Replay the missing Stripe test-mode outcomes and retain their Postgres and money-ledger receipts.",
			FixClass:    "manual",
			Maturity: &commonv1.FindingMaturity{
				CapabilityId: "stripe_outcomes", LocalLevel: "L1",
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
				Dimension:    "business", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED,
			},
		})
		blockingCodes = append(blockingCodes, "stripe.behavior_not_replayed")
	}
	return &commonv1.MaturityAssessment{
		Scenario: scenario, Provider: "landing-page-business-suite", Phase: "stripe-test-outcomes", Version: "0.1.1",
		Findings: findings, FindingsBySeverity: map[string]int32{"FINDING_SEVERITY_ERROR": int32(len(findings))},
		Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, BlockingFindingCodes: blockingCodes, Clean: len(findings) == 0},
	}
}

func outcomeStatus(report *StripeOutcomeReport, name string) string {
	for _, outcome := range report.Outcomes {
		if outcome.Name == name {
			return outcome.Status
		}
	}
	return "unknown"
}

func allOutcomesPassed(report *StripeOutcomeReport) bool {
	if report == nil {
		return false
	}
	required := []string{
		"real_webhook_delivery", "checkout_success", "decline", "3ds", "cancel", "upgrade", "downgrade",
		"renewal", "past_due", "credit_topup", "donation", "audio_tools_to_bundle_upgrade",
	}
	for _, name := range required {
		if outcomeStatus(report, name) != "passed" {
			return false
		}
	}
	return true
}

func stringValues(items []string) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func outcomeMaps(items []OutcomeObservation) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"name": item.Name, "status": item.Status, "detail": item.Detail,
			"stripe_event_id": item.StripeEventID, "related_stripe_event_ids": stringValues(item.RelatedStripeEventIDs), "stripe_event_type": item.StripeEventType,
			"stripe_object_id": item.StripeObjectID, "postgres_receipt_status": item.PostgresReceiptStatus,
			"checkout_session_id": item.CheckoutSessionID, "payment_intent_id": item.PaymentIntentID,
			"subscription_id": item.SubscriptionID, "subscription_status": item.SubscriptionStatus,
			"payment_status": item.PaymentStatus, "billing_reason": item.BillingReason,
			"postgres_lifecycle_receipt_status": item.LifecycleReceiptStatus,
			"postgres_economic_receipt_status":  item.EconomicReceiptStatus,
			"amount_minor":                      item.AmountMinor, "currency": item.Currency,
			"money_ledger_external_id": item.MoneyLedgerExternalID, "money_ledger_status": item.MoneyLedgerStatus,
			"money_ledger_posting_id": item.MoneyLedgerPostingID, "money_ledger_receipt_id": item.MoneyLedgerReceiptID,
		})
	}
	return result
}
