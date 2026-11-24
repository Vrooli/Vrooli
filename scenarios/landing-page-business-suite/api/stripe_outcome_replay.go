package main

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/monetization"
)

const py1MoneyLedgerCredentialIdentity = "scenario-authenticator/service-principal/landing-page-business-suite-money-ledger-writer/money-ledger"

var stripeOutcomeNames = []string{
	"real_webhook_delivery", "checkout_success", "decline", "3ds", "cancel", "upgrade", "downgrade",
	"renewal", "past_due", "credit_topup", "donation", "audio_tools_to_bundle_upgrade",
}

// stripeOutcomeReplay starts the phase with a provider-side catalog reconcile.
// It returns NOT_EVALUATED until each behavioral outcome has a durable receipt;
// a successful Stripe price lookup is never treated as a payment proof.
func (s *Server) stripeOutcomeReplay(ctx context.Context) (*monetization.StripeOutcomeReport, error) {
	environment := strings.ToLower(strings.TrimSpace(runtimeEnvironment()))
	report := &monetization.StripeOutcomeReport{
		Status: "not_evaluated", HostEnvironment: environment, StripeMode: strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE"))),
		LocalStandIn: stripeOutcomeLocalStandIn(environment),
		Catalog:      monetization.CatalogReplay{Status: "not_evaluated", DonationPrice: "missing"},
		Limitations: []string{
			"Catalog verification proves Stripe test prices exist and match LPBS metadata; it does not prove checkout, webhook, or lifecycle behavior.",
			"The customer.updated delivery demonstrates webhook ingress only; payment and money-ledger lifecycle outcomes remain unknown.",
		},
	}
	if report.LocalStandIn {
		report.Limitations = append(report.Limitations, "This host is a local stand-in until CL2 staging exists.")
	}
	for _, name := range stripeOutcomeNames {
		report.Outcomes = append(report.Outcomes, monetization.OutcomeObservation{Name: name, Status: "unknown"})
	}
	if !stripeOutcomeEnvironmentAllowed(environment) {
		report.Status = "failed"
		report.Catalog.Status = "failed"
		report.Limitations = append(report.Limitations, "Stripe outcome execution is refused unless the host explicitly identifies as local, development, test, or staging.")
		return report, nil
	}
	if report.StripeMode != "test" {
		report.Status = "failed"
		report.Catalog.Status = "failed"
		report.Limitations = append(report.Limitations, "Stripe outcome execution is refused unless STRIPE_MODE=test.")
		return report, nil
	}
	if s.stripeService == nil || s.planService == nil || s.paymentSettings == nil {
		return report, nil
	}
	settings, err := s.paymentSettings.GetStripeSettings(ctx)
	if err != nil || settings == nil {
		return report, nil
	}
	config := s.stripeService.getConfig()
	if !strings.HasPrefix(config.secretKey, "sk_test_") && !strings.HasPrefix(config.secretKey, "rk_test_") {
		report.Status = "failed"
		report.Catalog.Status = "failed"
		report.Limitations = append(report.Limitations, "The authority-resolved Stripe server key is not recognizable as a test-mode key.")
		return report, nil
	}
	if err := commerce.ValidateStripeKeyModePair(settings.GetPublishableKey(), config.secretKey); err != nil {
		report.Status = "failed"
		report.Catalog.Status = "failed"
		report.Limitations = append(report.Limitations, "The authority-resolved Stripe key pair has a mode mismatch.")
		return report, nil
	}

	if strings.EqualFold(strings.TrimSpace(envx.Get("PY1_STRIPE_TEST_CATALOG_RECONCILE")), "true") {
		changes, reconcileErr := s.reconcileTestStripeCatalog(ctx, environment)
		report.Catalog.Changes = append(report.Catalog.Changes, changes...)
		if reconcileErr != nil {
			report.Status = "failed"
			report.Catalog.Status = "failed"
			report.Limitations = append(report.Limitations, "Explicit test catalog reconciliation failed: "+reconcileErr.Error())
			return report, nil
		}
	}

	plans := s.planService.GetPlanStore().GetPlans()
	if len(plans) == 0 {
		report.Catalog.Status = "failed"
		report.Limitations = append(report.Limitations, "LPBS has no configured Stripe prices to reconcile.")
		return report, nil
	}
	mismatches := make([]string, 0)
	for _, plan := range plans {
		if plan == nil {
			mismatches = append(mismatches, "catalog contains an empty plan")
			continue
		}
		if strings.TrimSpace(plan.StripePriceId) == "" {
			mismatches = append(mismatches, "catalog entry without Stripe price ID")
			report.Catalog.PriceReconciliations = append(report.Catalog.PriceReconciliations, monetization.PriceReconciliation{
				PlanName: plan.GetPlanName(), PlanTier: plan.GetPlanTier(), Kind: plan.GetKind().String(), Status: "missing_price_id",
				ExpectedAmountCents: plan.GetAmountCents(), ExpectedCurrency: plan.GetCurrency(), ExpectedInterval: commerce.BillingIntervalLabel(plan.GetBillingInterval()),
			})
			continue
		}
		reconciliation := monetization.PriceReconciliation{
			PlanName: plan.GetPlanName(), PlanTier: plan.GetPlanTier(), Kind: plan.GetKind().String(), PriceID: plan.GetStripePriceId(),
			ExpectedAmountCents: plan.GetAmountCents(), ExpectedCurrency: plan.GetCurrency(),
			ExpectedInterval: commerce.BillingIntervalLabel(plan.GetBillingInterval()),
		}
		if reconciliation.ExpectedInterval == "unspecified" {
			if plan.GetKind().String() == "PLAN_KIND_CREDITS_TOPUP" || plan.GetKind().String() == "PLAN_KIND_SUPPORTER_CONTRIBUTION" || strings.EqualFold(strings.TrimSpace(plan.GetPlanTier()), "donation") {
				reconciliation.ExpectedInterval = "one_time"
			} else {
				reconciliation.ExpectedInterval = ""
			}
		}
		actual, verifyErr := s.stripeService.VerifyStripePrice(plan.StripePriceId)
		if verifyErr != nil || actual == nil {
			reconciliation.Status = "lookup_failed"
			reconciliation.LookupFailureStatus, reconciliation.LookupFailureCode = stripeLookupFailure(verifyErr)
			mismatches = append(mismatches, fmt.Sprintf("%s (%s): provider lookup failed (status=%d code=%s)", plan.GetPlanName(), plan.StripePriceId, reconciliation.LookupFailureStatus, reconciliation.LookupFailureCode))
			report.Catalog.PriceReconciliations = append(report.Catalog.PriceReconciliations, reconciliation)
			continue
		}
		report.Catalog.VerifiedPrices++
		reconciliation.ActualFieldsAvailable = true
		reconciliation.ActualPriceID, _ = actual["id"].(string)
		reconciliation.ActualAmountCents, _ = actual["amount_cents"].(int64)
		reconciliation.ActualCurrency, _ = actual["currency"].(string)
		reconciliation.ActualInterval, _ = actual["interval"].(string)
		reconciliation.ActualActive, _ = actual["active"].(bool)
		fieldDiffs := make([]string, 0, 5)
		if reconciliation.ActualPriceID != plan.StripePriceId {
			fieldDiffs = append(fieldDiffs, fmt.Sprintf("id expected=%s actual=%s", plan.StripePriceId, reconciliation.ActualPriceID))
		}
		if reconciliation.ActualAmountCents != plan.AmountCents {
			fieldDiffs = append(fieldDiffs, fmt.Sprintf("amount_cents expected=%d actual=%d", plan.AmountCents, reconciliation.ActualAmountCents))
		}
		if !strings.EqualFold(reconciliation.ActualCurrency, plan.Currency) {
			fieldDiffs = append(fieldDiffs, fmt.Sprintf("currency expected=%s actual=%s", plan.Currency, reconciliation.ActualCurrency))
		}
		if reconciliation.ActualInterval != reconciliation.ExpectedInterval {
			fieldDiffs = append(fieldDiffs, fmt.Sprintf("interval expected=%s actual=%s", reconciliation.ExpectedInterval, reconciliation.ActualInterval))
		}
		if !reconciliation.ActualActive {
			fieldDiffs = append(fieldDiffs, "active expected=true actual=false")
		}
		reconciliation.Status = "matched"
		if len(fieldDiffs) > 0 {
			reconciliation.Status = "mismatch"
			mismatches = append(mismatches, fmt.Sprintf("%s (%s): %s", plan.GetPlanName(), plan.StripePriceId, strings.Join(fieldDiffs, "; ")))
		}
		if strings.EqualFold(strings.TrimSpace(plan.PlanTier), "donation") {
			report.Catalog.DonationPrice = "verified"
		}
		report.Catalog.PriceReconciliations = append(report.Catalog.PriceReconciliations, reconciliation)
	}
	report.Catalog.Mismatches = mismatches
	if len(mismatches) > 0 {
		report.Catalog.Status = "failed"
		report.Status = "failed"
		return report, nil
	}
	report.Catalog.Status = "passed"
	if report.Catalog.DonationPrice != "verified" {
		report.Catalog.Status = "incomplete"
	}

	if port := strings.TrimSpace(envx.Get("API_PORT")); port != "" {
		ledgerWriter, ledgerTarget, releaseLedgerFixture, ledgerErr := s.py1TestLedgerWriter(ctx)
		if releaseLedgerFixture != nil {
			defer releaseLedgerFixture()
		}
		if ledgerErr != nil {
			report.Status = "failed"
			report.Limitations = append(report.Limitations, "No Stripe test checkout was created because the isolated PY1 Money Ledger fixture is not ready: "+ledgerErr.Error())
			return report, nil
		}
		executionID, idErr := newStripeOutcomeExecutionID()
		if idErr != nil {
			report.Status = "failed"
			report.Limitations = append(report.Limitations, "Could not create a unique Stripe test outcome execution ID.")
			return report, nil
		}
		report.ExecutionID = executionID
		forwardTo := "http://127.0.0.1:" + port + stripeWebhookPath
		listener, listenErr := startStripeTestListenerForEvents(ctx, config.secretKey, forwardTo,
			"checkout.session.completed", "checkout.session.async_payment_succeeded", "invoice.paid", "invoice.payment_failed",
			"customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted")
		if listenErr != nil {
			report.Limitations = append(report.Limitations, listenErr.Error())
			return report, nil
		}
		restoreWebhookSecret := s.stripeService.useEphemeralWebhookSecret(listener.webhookSecret)
		defer restoreWebhookSecret()
		defer listener.close()
		for _, outcome := range []string{"real_webhook_delivery", "checkout_success"} {
			if err := s.replayPaidTestCheckout(ctx, executionID, outcome, port, listener, ledgerWriter, ledgerTarget, report); err != nil {
				setStripeOutcome(report, outcome, "failed", err.Error())
				report.Status = "failed"
				return report, nil
			}
		}
		if err := s.replayPY1TestClockSubscriptionLifecycle(ctx, executionID, listener, ledgerWriter, ledgerTarget, report); err != nil {
			report.Status = "failed"
			report.Limitations = append(report.Limitations, "Clocked subscription lifecycle did not complete with signed webhook and durable ledger receipts: "+err.Error())
			return report, nil
		}
	} else {
		report.Limitations = append(report.Limitations, "API_PORT is unavailable; the local Stripe webhook listener was not started.")
	}
	if !stripeOutcomeBehaviorsPassed(report) {
		report.Status = "failed"
	}
	return report, nil
}

func newStripeOutcomeExecutionID() (string, error) {
	var suffix [16]byte
	if _, err := cryptorand.Read(suffix[:]); err != nil {
		return "", err
	}
	return "py1-" + hex.EncodeToString(suffix[:]), nil
}

func (s *Server) replayPaidTestCheckout(ctx context.Context, runID, outcome, apiPort string, listener *stripeTestListener, ledgerWriter commerce.ServiceOrderLedgerWriter, ledgerTarget commerce.PY1TestLedgerTarget, report *monetization.StripeOutcomeReport) error {
	priceID := s.lowestConfiguredSubscriptionTestPrice()
	if priceID == "" {
		return errors.New("LPBS has no reconciled subscription price for the test Checkout")
	}
	base := "http://localhost:" + apiPort
	session, err := s.stripeService.CreateTestOutcomeCheckoutSession(ctx, priceID, base+"/?py1_checkout=success&session_id={CHECKOUT_SESSION_ID}", base+"/?py1_checkout=cancel", runID+"-"+outcome+"@example.test", runID, outcome)
	if err != nil || session == nil || strings.TrimSpace(session.GetSessionId()) == "" || strings.TrimSpace(session.GetUrl()) == "" {
		return errors.New("LPBS could not create a correlated Stripe test Checkout Session")
	}
	parsedCheckout, parseErr := url.Parse(session.GetUrl())
	if parseErr != nil || parsedCheckout.Scheme != "https" || (parsedCheckout.Hostname() != "checkout.stripe.com" && parsedCheckout.Hostname() != "checkout.stripe.dev") || !strings.HasPrefix(session.GetSessionId(), "cs_test_") {
		return errors.New("Stripe returned a Checkout Session outside the test-hosted Checkout boundary")
	}
	scriptPath, err := stripeCheckoutBrowserScriptPath()
	if err != nil {
		return err
	}
	browserReceipt, err := runStripeCheckoutBrowser(ctx, scriptPath, session.GetUrl(), "success")
	if err != nil || browserReceipt.SessionID != session.GetSessionId() || browserReceipt.Status != "succeeded" {
		return errors.New("hosted Stripe test Checkout did not reach its success redirect")
	}
	deliveryStatus, eventID, eventType, err := listener.waitForEventDelivery(ctx, map[string]bool{"checkout.session.completed": true, "checkout.session.async_payment_succeeded": true})
	if err != nil || deliveryStatus != "passed" || eventID == "" || (eventType != "checkout.session.completed" && eventType != "checkout.session.async_payment_succeeded") {
		return errors.New("Stripe Checkout completed without a successful signed webhook delivery to LPBS")
	}
	webhook, err := s.stripeService.webhookService().LoadTestOutcomeWebhookReceipt(ctx, runID, outcome)
	if err != nil || webhook == nil || webhook.EventID != eventID || webhook.EventType != "checkout.session.completed" || webhook.ObjectID != session.GetSessionId() || webhook.Status != "processed" || webhook.Livemode {
		return errors.New("LPBS did not retain the matching processed Checkout webhook receipt in Postgres")
	}
	economic, err := s.stripeService.webhookService().LoadTestEconomicReceipt(ctx, runID, outcome)
	if err != nil || economic == nil || economic.EventID != eventID || economic.CheckoutSessionID != session.GetSessionId() || economic.PaymentStatus != "paid" || economic.AmountMinor <= 0 || economic.Currency == "" || economic.MoneyLedgerExternalID == "" {
		return errors.New("LPBS did not retain the matching paid Checkout economic receipt in Postgres")
	}
	if economic.AmountMinor != session.GetAmountCents() || !strings.EqualFold(economic.Currency, session.GetCurrency()) {
		return errors.New("Stripe Checkout amount or currency did not match the durable LPBS economic receipt")
	}
	for index := range report.Outcomes {
		if report.Outcomes[index].Name == outcome {
			item := &report.Outcomes[index]
			item.StripeEventID = webhook.EventID
			item.StripeEventType = webhook.EventType
			item.StripeObjectID = webhook.ObjectID
			item.PostgresReceiptStatus = webhook.Status
			item.CheckoutSessionID = economic.CheckoutSessionID
			item.PaymentIntentID = economic.PaymentIntentID
			item.SubscriptionID = economic.SubscriptionID
			item.EconomicReceiptStatus = "persisted"
			item.AmountMinor = economic.AmountMinor
			item.Currency = economic.Currency
			item.MoneyLedgerExternalID = economic.MoneyLedgerExternalID
		}
	}
	ledger, ledgerErr := s.dispatchPY1TestEconomicReceipt(ctx, runID, outcome, ledgerWriter, ledgerTarget)
	if ledger != nil {
		for index := range report.Outcomes {
			if report.Outcomes[index].Name == outcome {
				report.Outcomes[index].MoneyLedgerStatus = ledger.MoneyLedgerStatus
				report.Outcomes[index].MoneyLedgerPostingID = ledger.MoneyLedgerPostingID
				report.Outcomes[index].MoneyLedgerReceiptID = ledger.MoneyLedgerReceiptID
			}
		}
	}
	if ledgerErr != nil || ledger == nil || ledger.MoneyLedgerStatus != "posted" {
		setStripeOutcome(report, outcome, "unknown", "Stripe Checkout was paid and linked to its Postgres webhook/economic receipts; accepted Money Ledger posting is still pending.")
		if ledgerErr != nil {
			report.Limitations = append(report.Limitations, ledgerErr.Error())
		}
		return nil
	}
	setStripeOutcome(report, outcome, "passed", fmt.Sprintf("Stripe test Checkout %s completed and delivered event %s; Postgres retains a paid %d %s economic receipt and Money Ledger accepted posting %s with receipt %s.", economic.CheckoutSessionID, economic.EventID, economic.AmountMinor, strings.ToUpper(economic.Currency), ledger.MoneyLedgerPostingID, ledger.MoneyLedgerReceiptID))
	return nil
}

func (s *Server) lowestConfiguredSubscriptionTestPrice() string {
	if s == nil || s.planService == nil || s.planService.GetPlanStore() == nil {
		return ""
	}
	var selected string
	var selectedAmount int64
	for _, plan := range s.planService.GetPlanStore().GetPlans() {
		if plan == nil || plan.GetKind().String() != "PLAN_KIND_SUBSCRIPTION" || plan.GetAmountCents() <= 0 || strings.TrimSpace(plan.GetStripePriceId()) == "" || !strings.EqualFold(plan.GetCurrency(), "usd") {
			continue
		}
		if selected == "" || plan.GetAmountCents() < selectedAmount || (plan.GetAmountCents() == selectedAmount && plan.GetStripePriceId() < selected) {
			selected, selectedAmount = plan.GetStripePriceId(), plan.GetAmountCents()
		}
	}
	return selected
}

type stripeCheckoutBrowserReceipt struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

func stripeCheckoutBrowserScriptPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.New("could not resolve hosted Checkout browser runner")
	}
	for depth := 0; depth < 5; depth++ {
		candidate := filepath.Join(wd, "scripts", "complete-stripe-checkout.py")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", errors.New("hosted Checkout browser runner is unavailable")
}

func runStripeCheckoutBrowser(ctx context.Context, scriptPath, checkoutURL, outcome string) (stripeCheckoutBrowserReceipt, error) {
	if outcome != "success" || strings.TrimSpace(checkoutURL) == "" {
		return stripeCheckoutBrowserReceipt{}, errors.New("invalid hosted Checkout browser request")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return stripeCheckoutBrowserReceipt{}, errors.New("Python Playwright runner is unavailable")
	}
	cmd := exec.CommandContext(ctx, python, scriptPath, outcome)
	cmd.Stdin = strings.NewReader(checkoutURL + "\n")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	output, err := cmd.Output()
	if err != nil {
		return stripeCheckoutBrowserReceipt{}, errors.New("hosted Stripe Checkout browser flow failed")
	}
	var receipt stripeCheckoutBrowserReceipt
	if err := json.Unmarshal(output, &receipt); err != nil || strings.TrimSpace(receipt.SessionID) == "" || receipt.Status != "succeeded" {
		return stripeCheckoutBrowserReceipt{}, errors.New("hosted Checkout browser flow returned no accepted receipt")
	}
	return receipt, nil
}

func (s *Server) py1TestLedgerWriter(ctx context.Context) (commerce.ServiceOrderLedgerWriter, commerce.PY1TestLedgerTarget, func(), error) {
	endpoint := strings.TrimSpace(envx.Get("PY1_TEST_MONEY_LEDGER_URL"))
	fixtureID := strings.TrimSpace(envx.Get("PY1_TEST_MONEY_LEDGER_FIXTURE_ID"))
	var lease *py1LedgerFixtureLease
	if endpoint == "" && stripeOutcomeLocalStandIn(runtimeEnvironment()) {
		var err error
		lease, err = startLocalPY1LedgerFixture(ctx)
		if err != nil {
			return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("could not start the isolated CA-verifying Money Ledger fixture")
		}
		endpoint = lease.config.URL
		fixtureID = lease.config.FixtureID
	}
	release := func() {
		if lease != nil {
			lease.Close()
		}
	}
	if endpoint == "" || fixtureID == "" {
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("fixture URL and isolated fixture identity are required")
	}
	fixtureWriter, err := commerce.NewMoneyLedgerTestFixtureWriter(endpoint, fixtureID)
	if err != nil {
		release()
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("fixture identity check failed")
	}
	target, err := fixtureWriter.PY1Target()
	if err != nil {
		release()
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("Money Ledger fixture did not prove its posting scope")
	}
	// Optional expected IDs allow deployments to pin the fixture identity while
	// keeping the authoritative target sourced from the isolated fixture.
	expected := commerce.PY1TestLedgerTarget{
		AdapterID:        strings.TrimSpace(envx.Get("PY1_TEST_MONEY_LEDGER_ADAPTER_ID")),
		BookID:           strings.TrimSpace(envx.Get("PY1_TEST_MONEY_LEDGER_BOOK_ID")),
		RevenueAccountID: strings.TrimSpace(envx.Get("PY1_TEST_MONEY_LEDGER_REVENUE_ACCOUNT_ID")),
	}
	if expected.AdapterID != "" || expected.BookID != "" || expected.RevenueAccountID != "" {
		if err := fixtureWriter.ValidatePY1Target(expected); err != nil {
			release()
			return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("configured Money Ledger target does not match isolated fixture scope")
		}
	}
	authenticatorURL, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	if err != nil || strings.TrimSpace(authenticatorURL) == "" {
		release()
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("Scenario Authenticator service URL could not be resolved")
	}
	tokens := commerce.NewMoneyLedgerServiceOrderTokens(func(context.Context) (string, error) {
		return resolvePY1MoneyLedgerServiceCredential()
	}, authenticatorURL, nil)
	if _, err := tokens.Token(ctx); err != nil {
		release()
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("credential-authority service token exchange failed")
	}
	writer, err := commerce.NewMoneyLedgerPY1TestServiceWriter(endpoint, tokens, nil)
	if err != nil {
		release()
		return nil, commerce.PY1TestLedgerTarget{}, nil, errors.New("loopback PY1 test writer initialization failed")
	}
	return writer, target, release, nil
}

// resolvePY1MoneyLedgerServiceCredential reads the operator-issued credential
// at the same identity Scenario Authenticator writes for the declared principal.
// It never falls back to an environment variable or the legacy LPBS secret.
func resolvePY1MoneyLedgerServiceCredential() (string, error) {
	want := serviceprincipal.CredentialIdentity("landing-page-business-suite-money-ledger-writer", "money-ledger")
	if want != py1MoneyLedgerCredentialIdentity {
		return "", errors.New("Money Ledger service principal identity contract changed")
	}
	identity, err := credentialauthority.ParseIdentity(want)
	if err != nil {
		return "", err
	}
	binding, err := administration.ResolveBoundCredential(identity, serviceprincipal.CredentialField, "money-ledger:write")
	return binding.Value, err
}

func (s *Server) dispatchPY1TestEconomicReceipt(ctx context.Context, runID, outcome string, writer commerce.ServiceOrderLedgerWriter, target commerce.PY1TestLedgerTarget) (*commerce.StripeTestEconomicReceipt, error) {
	return s.stripeService.webhookService().DispatchTestEconomicReceipt(ctx, writer, target, runID, outcome)
}

func (s *Server) triggerStripeTestCustomerUpdate(ctx context.Context, executionID string) (string, error) {
	if s == nil || s.stripeService == nil || !strings.HasPrefix(executionID, "py1-") {
		return "", errors.New("Stripe test outcome execution is unavailable")
	}
	create := url.Values{}
	create.Set("email", executionID+"@example.com")
	create.Set("metadata[lpbs_py1_test_run_id]", executionID)
	create.Set("metadata[lpbs_py1_test_outcome]", "real_webhook_delivery")
	createHeaders := http.Header{}
	createHeaders.Set("Idempotency-Key", executionID+"-customer-create")
	body, err := s.stripeService.doStripeRequestWithHeaders(ctx, http.MethodPost, "/v1/customers", strings.NewReader(create.Encode()), "application/x-www-form-urlencoded", createHeaders)
	if err != nil {
		return "", errors.New("create test customer failed")
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil || strings.TrimSpace(created.ID) == "" {
		return "", errors.New("Stripe returned no test customer ID")
	}
	update := url.Values{}
	update.Set("description", "PY1 Test Genie webhook delivery "+executionID)
	updateHeaders := http.Header{}
	updateHeaders.Set("Idempotency-Key", executionID+"-customer-update")
	if _, err := s.stripeService.doStripeRequestWithHeaders(ctx, http.MethodPost, "/v1/customers/"+url.PathEscape(created.ID), strings.NewReader(update.Encode()), "application/x-www-form-urlencoded", updateHeaders); err != nil {
		return "", errors.New("update test customer failed")
	}
	return created.ID, nil
}

func stripeOutcomeBehaviorsPassed(report *monetization.StripeOutcomeReport) bool {
	if report == nil {
		return false
	}
	for _, name := range stripeOutcomeNames {
		found := false
		for _, outcome := range report.Outcomes {
			if outcome.Name == name && outcome.Status == "passed" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (s *Server) reconcileTestStripeCatalog(ctx context.Context, environment string) ([]string, error) {
	if !stripeOutcomeLocalStandIn(environment) || strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE"))) != "test" {
		return nil, fmt.Errorf("catalog reconciliation is restricted to local Stripe test mode")
	}
	planStore := s.planService.GetPlanStore()
	bundle := planStore.GetBundle()
	if bundle == nil || !strings.EqualFold(strings.TrimSpace(bundle.GetEnvironment()), "test") || strings.TrimSpace(bundle.GetStripeProductId()) == "" {
		return nil, fmt.Errorf("the active bundle must be a test catalog with a configured Stripe product")
	}
	fetcher := func(ctx context.Context, priceID string) (*commerce.StripePriceImport, error) {
		return s.stripeService.FetchStripePriceDetails(ctx, priceID)
	}
	changes := make([]string, 0, 4)
	for _, plan := range planStore.GetPlans() {
		if plan == nil || strings.TrimSpace(plan.GetStripePriceId()) == "" || plan.GetKind().String() != "PLAN_KIND_SUBSCRIPTION" {
			continue
		}
		_, lookupErr := s.stripeService.VerifyStripePrice(plan.GetStripePriceId())
		if !stripePriceIsMissing(lookupErr) {
			continue
		}
		interval := commerce.BillingIntervalLabel(plan.GetBillingInterval())
		if interval != "month" && interval != "year" {
			return changes, fmt.Errorf("cannot reconcile unsupported interval for plan %s", plan.GetPlanName())
		}
		newID, err := s.stripeService.createStripeTestPrice(ctx, bundle.GetStripeProductId(), plan.GetPlanName()+" "+interval, plan.GetAmountCents(), interval,
			"py1-test-reconcile-"+strings.ToLower(plan.GetPlanTier())+"-"+interval)
		if err != nil {
			return changes, fmt.Errorf("create test price for %s: %s", plan.GetPlanName(), safeStripeMutationError(err))
		}
		oldID := plan.GetStripePriceId()
		if _, err := s.planService.UpdateBundlePriceWithStripe(ctx, bundle.GetBundleKey(), oldID, commerce.UpdateBundlePriceInput{StripePriceID: &newID}, fetcher); err != nil {
			return changes, fmt.Errorf("map replacement test price for %s: %s", plan.GetPlanName(), safeStripeMutationError(err))
		}
		changes = append(changes, fmt.Sprintf("replaced missing test price %s with %s for %s", oldID, newID, plan.GetPlanName()))
	}
	donationPresent := false
	for _, plan := range planStore.GetPlans() {
		if plan != nil && strings.EqualFold(strings.TrimSpace(plan.GetPlanTier()), "donation") {
			donationPresent = true
			break
		}
	}
	if !donationPresent {
		newID, err := s.stripeService.createStripeTestPrice(ctx, bundle.GetStripeProductId(), "Donation test only", 100, "", "py1-test-donation-v1")
		if err != nil {
			return changes, fmt.Errorf("create test donation price: %s", safeStripeMutationError(err))
		}
		amount := int64(100)
		currency := "usd"
		weight := int32(30)
		display := true
		if _, err := s.planService.CreateBundlePrice(ctx, bundle.GetBundleKey(), commerce.CreateBundlePriceInput{
			StripePriceID: newID, PlanName: "Donation test only", PlanTier: "donation", BillingInterval: "one_time",
			AmountCents: &amount, Currency: &currency, DisplayWeight: &weight, DisplayEnabled: &display,
		}, fetcher); err != nil {
			return changes, fmt.Errorf("map test donation price: %s", safeStripeMutationError(err))
		}
		changes = append(changes, fmt.Sprintf("added fixed 100-cent test donation price %s", newID))
	}
	return changes, nil
}

func stripePriceIsMissing(err error) bool {
	var apiErr *StripeAPIError
	return errors.As(err, &apiErr) && apiErr.Status == 404 && strings.EqualFold(strings.TrimSpace(apiErr.Code), "resource_missing")
}

func safeStripeMutationError(err error) string {
	status, code := stripeLookupFailure(err)
	if status == 0 {
		return code
	}
	return fmt.Sprintf("status=%d code=%s", status, code)
}

func stripeLookupFailure(err error) (int, string) {
	if err == nil {
		return 0, "no_price_details"
	}
	var apiErr *StripeAPIError
	if errors.As(err, &apiErr) {
		code := strings.TrimSpace(apiErr.Code)
		if code == "" {
			code = "stripe_api_error"
		}
		return apiErr.Status, code
	}
	var configErr *StripeConfigError
	if errors.As(err, &configErr) {
		return 0, "credential_configuration"
	}
	return 0, "provider_error"
}

func stripeOutcomeEnvironmentAllowed(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "local", "test", "staging":
		return true
	default:
		return false
	}
}

func stripeOutcomeLocalStandIn(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "local", "test":
		return true
	default:
		return false
	}
}

func setStripeOutcome(report *monetization.StripeOutcomeReport, name, status, detail string) {
	for index := range report.Outcomes {
		if report.Outcomes[index].Name == name {
			report.Outcomes[index].Status = status
			report.Outcomes[index].Detail = detail
			return
		}
	}
}
