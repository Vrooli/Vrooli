package setup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/provider"
	"connectrpc.com/connect"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup"
	_ "modernc.org/sqlite"
)

func TestCatalogueHasSecretFreeReadOnlyProviderEntries(t *testing.T) {
	entries := Catalogue()
	if len(entries) < 2 {
		t.Fatalf("catalogue entries = %d, want Hetzner and DigitalOcean", len(entries))
	}
	for _, entry := range entries {
		if entry.CredentialField == "" || entry.CredentialLogicalID == "" || (entry.ID != "akash" && len(entry.Probes) == 0) {
			t.Fatalf("incomplete provider entry: %#v", entry)
		}
		for _, probe := range entry.Probes {
			if probe.Method != "GET" || !probe.ReadOnly {
				t.Fatalf("unsafe probe in %s: %#v", entry.ID, probe)
			}
		}
	}
}

func TestProviderCatalogueExposesPhoneDecisionPromptToClient(t *testing.T) {
	item := findProvider("hetzner")
	if item == nil {
		t.Fatal("Hetzner missing from provider catalogue")
	}
	client := providerProto(*item)
	if len(client.GetConsoleSteps()) != len(item.ConsoleSteps) {
		t.Fatalf("client steps = %d, catalogue steps = %d", len(client.GetConsoleSteps()), len(item.ConsoleSteps))
	}
	for i, step := range client.GetConsoleSteps() {
		if step.GetDecision() == "" {
			t.Errorf("client console step %q has no decision prompt; step %d = %#v", item.ConsoleSteps[i].ID, i, step)
		}
	}
}

func TestHetznerCatalogueSourcesPriceSizeRegionAndPaymentFacts(t *testing.T) {
	item := findProvider("hetzner")
	if item == nil {
		t.Fatal("Hetzner missing from provider catalogue")
	}
	for key, want := range map[string]string{
		"price_source":               "GET https://api.hetzner.cloud/v1/pricing",
		"sizes_source":               "GET https://api.hetzner.cloud/v1/server_types",
		"regions_source":             "GET https://api.hetzner.cloud/v1/locations",
		"payment_rails":              "Credit card",
		"payment_source":             "https://docs.hetzner.com/general/billing-and-account-management/billing-at-hetzner/payment-overview/",
		"price_observation_lifetime": "5 minutes; expired observations are refused",
	} {
		if !strings.Contains(item.BillingFacts[key], want) {
			t.Errorf("Hetzner billing fact %q = %q, want it to contain %q", key, item.BillingFacts[key], want)
		}
	}
	var paymentStep *ConsoleStep
	for i := range item.ConsoleSteps {
		if item.ConsoleSteps[i].ID == "payment" {
			paymentStep = &item.ConsoleSteps[i]
		}
	}
	if paymentStep == nil || paymentStep.URL != "https://accounts.hetzner.com/" || !strings.Contains(paymentStep.Instructions, "vary by country") {
		t.Fatalf("Hetzner payment step does not direct the operator to the provider account page or explain availability: %#v", paymentStep)
	}
}

func TestAkashCatalogueDeclaresWalletWithoutChainProbeOrEffect(t *testing.T) {
	item := findProvider("akash")
	if item == nil || item.CredentialLogicalID != "vrooli/compute-manager" || item.CredentialField != "akash-testnet-wallet" || item.BillingFacts["network"] != "sandbox-2" || item.BillingFacts["effect_mode"] != "stub" || len(item.Probes) != 0 || len(item.ConsoleSteps) != 2 || item.ConsoleSteps[0].ID != "console-api-key" || item.ConsoleSteps[1].ID != "wallet" || item.BillingFacts["market_source_credential_reference"] != "credential-authority://vrooli/compute-manager/akash-console-api-key" || item.BillingFacts["market_source_auth"] != "Akash Console API x-api-key required" || !strings.Contains(item.BillingFacts["blocked_reason"], "existing deployment dseq") || item.ConsoleSteps[1].URL != "/embedded/vrooli-onboarding/?target=local&step=credentials" {
		t.Fatalf("unsafe or incomplete Akash phone setup descriptor: %+v", item)
	}
}

func TestDigitalOceanCatalogueDeclaresExactLifecycleAndBillingScopesAndSourcedFacts(t *testing.T) {
	do := findProvider("digitalocean")
	if do == nil {
		t.Fatal("DigitalOcean missing from the shared provider catalogue")
	}
	wantScopes := []string{"droplet:create", "droplet:delete", "droplet:read", "regions:read", "sizes:read", "actions:read", "image:read", "snapshot:read", "vpc:read", "tag:create", "tag:read", "billing:read"}
	if len(do.CredentialScopes) != len(wantScopes) {
		t.Fatalf("DigitalOcean scopes = %#v, want exact set %#v", do.CredentialScopes, wantScopes)
	}
	gotScopes := map[string]bool{}
	for _, scope := range do.CredentialScopes {
		gotScopes[scope] = true
	}
	for _, scope := range wantScopes {
		if !gotScopes[scope] {
			t.Errorf("DigitalOcean token scope %q missing", scope)
		}
	}
	if do.CredentialField != "digitalocean-api-token" || len(do.ConsoleSteps) != 3 || len(do.Offers) == 0 {
		t.Fatalf("DigitalOcean setup descriptor incomplete: %#v", do)
	}
	wantProbes := map[string]string{"droplets": "/v2/droplets", "sizes": "/v2/sizes", "regions": "/v2/regions", "invoices": "/v2/customers/my/invoices?per_page=1"}
	if len(do.Probes) != len(wantProbes) {
		t.Fatalf("DigitalOcean probe set = %#v", do.Probes)
	}
	for _, probe := range do.Probes {
		if probe.Method != http.MethodGet || !probe.ReadOnly || wantProbes[probe.ID] != probe.Path {
			t.Fatalf("DigitalOcean setup probe is not read-only: %#v", probe)
		}
	}
	if do.BillingFacts["minimum"] != "60 seconds or US$0.01, whichever is higher" || do.BillingFacts["bundled_cpu_monthly_cap"] != "672 hours (28 days)" || do.BillingFacts["price_observation_lifetime"] != "5 minutes; expired observations are refused" {
		t.Fatalf("DigitalOcean sourced billing/freshness facts incomplete: %#v", do.BillingFacts)
	}
}

func TestDigitalOceanVerificationWithMissingCredentialDoesNoProviderRequest(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-do-no-token?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	providerRequests := 0
	repo := NewRepository(db)
	completeConsoleSteps(t, repo, *findProvider("digitalocean"), time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	svc := &service{
		repo: repo,
		resolve: func(_ context.Context, field string) (string, error) {
			if field != "digitalocean-api-token" {
				t.Fatalf("resolved credential field = %q", field)
			}
			return "", errors.New("fixture credential missing")
		},
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			providerRequests++
			return nil, errors.New("unexpected provider request")
		})},
		now: func() time.Time { return time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC) },
	}
	response, err := svc.VerifyProvider(context.Background(), connect.NewRequest(&setupv1.VerifyProviderRequest{ProviderId: "digitalocean", Region: "nyc1", Size: "s-1vcpu-1gb"}))
	if err != nil {
		t.Fatal(err)
	}
	if providerRequests != 0 {
		t.Fatalf("missing credential caused %d provider requests", providerRequests)
	}
	if response.Msg.GetReadiness().GetPurchaseEnabled() || response.Msg.GetReadiness().GetCredentialStatus() != "missing" {
		t.Fatalf("missing credential returned unsafe readiness: %#v", response.Msg.GetReadiness())
	}
}

func TestHetznerVerificationWithMissingCredentialDoesNoProviderRequest(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-hetzner-no-token?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	providerRequests := 0
	repo := NewRepository(db)
	completeConsoleSteps(t, repo, *findProvider("hetzner"), time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC))
	svc := &service{
		repo: repo,
		resolve: func(_ context.Context, field string) (string, error) {
			if field != "hetzner-api-token" {
				t.Fatalf("resolved credential field = %q", field)
			}
			return "", errors.New("fixture credential missing")
		},
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			providerRequests++
			return nil, errors.New("unexpected provider request")
		})},
		now: func() time.Time { return time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC) },
	}
	response, err := svc.VerifyProvider(context.Background(), connect.NewRequest(&setupv1.VerifyProviderRequest{ProviderId: "hetzner", Region: "nbg1", Size: "cx23"}))
	if err != nil {
		t.Fatal(err)
	}
	if providerRequests != 0 {
		t.Fatalf("missing credential caused %d provider requests", providerRequests)
	}
	if response.Msg.GetReadiness().GetPurchaseEnabled() || response.Msg.GetReadiness().GetCredentialStatus() != "missing" {
		t.Fatalf("missing credential returned unsafe readiness: %#v", response.Msg.GetReadiness())
	}
}

func TestVerificationRequiresEveryPhoneStepBeforeCredentialResolution(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-verify-before-phone-steps?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	providerRequests, credentialResolutions := 0, 0
	svc := &service{
		repo: NewRepository(db),
		resolve: func(context.Context, string) (string, error) {
			credentialResolutions++
			return "synthetic-test-value", nil
		},
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			providerRequests++
			return nil, errors.New("unexpected provider request")
		})},
	}
	_, err = svc.VerifyProvider(context.Background(), connect.NewRequest(&setupv1.VerifyProviderRequest{ProviderId: "hetzner", Region: "nbg1", Size: "cx23"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("verification error = %v, want failed precondition", err)
	}
	if credentialResolutions != 0 || providerRequests != 0 {
		t.Fatalf("incomplete phone flow resolved credential %d times and issued %d provider requests", credentialResolutions, providerRequests)
	}
}

func TestReadinessStopsAtFirstUnknown(t *testing.T) {
	verified := func(state State) Evidence { return Evidence{State: state, Verified: true} }
	evidence := map[State]Evidence{Authenticated: verified(Authenticated), AccountUsable: verified(AccountUsable), OfferAvailable: {State: OfferAvailable, Detail: "not available"}}
	got := Evaluate("hetzner", true, true, evidence)
	if got.Current != AccountUsable {
		t.Fatalf("state = %s, want account_usable", got.Current)
	}
	if got := Evaluate("hetzner", false, true, evidence); got.Current != NeedsConsoleSteps {
		t.Fatalf("without console steps state = %s", got.Current)
	}
	if got := Evaluate("hetzner", true, false, evidence); got.Current != CredentialNeeded {
		t.Fatalf("without credential state = %s", got.Current)
	}
}

func TestConfiguredCredentialNeedsFreshVerifiedProbeBeforeAuthentication(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-credential-not-authenticated?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	item := findProvider("hetzner")
	if item == nil {
		t.Fatal("Hetzner missing from provider catalogue")
	}
	now := time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)
	repo := NewRepository(db)
	for i, step := range item.ConsoleSteps {
		_, err := repo.StartStep(context.Background(), StepRecord{ProviderID: item.ID, StepID: step.ID, AskID: fmt.Sprintf("ask-%d", i), EventTime: now, State: "complete", Answer: "complete", UpdatedAt: now})
		if err != nil {
			t.Fatal(err)
		}
	}
	svc := &service{
		repo:       repo,
		configured: func(context.Context, string) (bool, error) { return true, nil },
		now:        func() time.Time { return now },
	}
	readiness := func() *setupv1.ProviderReadiness {
		t.Helper()
		response, err := svc.GetReadiness(context.Background(), connect.NewRequest(&setupv1.GetReadinessRequest{ProviderId: item.ID}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetReadiness()
	}
	if got := readiness().GetCurrent(); got != string(CredentialNeeded) {
		t.Fatalf("configured credential without a provider probe reported %q, want %q", got, CredentialNeeded)
	}
	if err := repo.PutEvidence(context.Background(), item.ID, Evidence{ID: "probe:locations", State: Authenticated, Verified: false, CheckedAt: now}); err != nil {
		t.Fatal(err)
	}
	if got := readiness().GetCurrent(); got != string(CredentialNeeded) {
		t.Fatalf("failed probe reported %q, want %q", got, CredentialNeeded)
	}
	if err := repo.PutEvidence(context.Background(), item.ID, Evidence{ID: "probe:locations", State: Authenticated, Verified: true, CheckedAt: now}); err != nil {
		t.Fatal(err)
	}
	if got := readiness().GetCurrent(); got != string(Authenticated) {
		t.Fatalf("fresh successful probe reported %q, want %q", got, Authenticated)
	}
	if err := repo.PutEvidence(context.Background(), item.ID, Evidence{ID: "probe:locations", State: Authenticated, Verified: true, CheckedAt: now.Add(-6 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if got := readiness().GetCurrent(); got != string(CredentialNeeded) {
		t.Fatalf("stale probe reported %q, want %q", got, CredentialNeeded)
	}
}

func TestPurchaseEnabledRequiresFreshAccountOfferAndPaymentEvidence(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-purchase-enabled-evidence?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	item := findProvider("hetzner")
	if item == nil {
		t.Fatal("Hetzner missing from provider catalogue")
	}
	now := time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)
	repo := NewRepository(db)
	completeConsoleSteps(t, repo, *item, now)
	if err := repo.PutEvidence(context.Background(), item.ID, Evidence{ID: "probe:servers", State: AccountUsable, Verified: true, Detail: "fixture list success", CheckedAt: now}); err != nil {
		t.Fatal(err)
	}
	fact := provider.PriceFact{Provider: item.ID, OfferID: "cx23", Region: "nbg1", Size: "cx23", Currency: "USD", ServerMinor: 649, IPv4Minor: 60, TaxBasis: "pre-tax", SourceURL: "https://api.hetzner.cloud/v1/pricing", SourceVersion: "fixture-version", ObservedAt: now, ExpiresAt: now.Add(5 * time.Minute)}
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	svc := &service{
		repo:       repo,
		configured: func(context.Context, string) (bool, error) { return true, nil },
		now:        func() time.Time { return now },
	}
	getReadiness := func() *setupv1.ProviderReadiness {
		t.Helper()
		response, err := svc.GetReadiness(context.Background(), connect.NewRequest(&setupv1.GetReadinessRequest{ProviderId: item.ID}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetReadiness()
	}
	if ready := getReadiness(); ready.GetCurrent() != string(OfferAvailable) || ready.GetPurchaseEnabled() {
		t.Fatalf("account and fresh offer without payment confirmation = state %q, purchase=%t", ready.GetCurrent(), ready.GetPurchaseEnabled())
	}
	if err := repo.PutEvidence(context.Background(), item.ID, Evidence{ID: "console:payment", State: PaymentValid, Verified: true, Detail: "operator phone confirmation", CheckedAt: now}); err != nil {
		t.Fatal(err)
	}
	ready := getReadiness()
	if ready.GetCurrent() != string(PurchaseEnabled) || !ready.GetPurchaseEnabled() {
		t.Fatalf("complete verified setup = state %q, purchase=%t; want purchase_enabled", ready.GetCurrent(), ready.GetPurchaseEnabled())
	}
	foundPurchaseEvidence := false
	for _, evidence := range ready.GetEvidence() {
		if evidence.GetId() == "readiness:purchase-enabled" && evidence.GetVerified() && strings.Contains(evidence.GetDetail(), "spend-cap approval") {
			foundPurchaseEvidence = true
		}
	}
	if !foundPurchaseEvidence {
		t.Fatal("purchase-enabled evidence did not explain the separate quote and spend-cap gate")
	}
	fact.ExpiresAt = now.Add(-time.Second)
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	ready = getReadiness()
	if ready.GetPurchaseEnabled() || ready.GetCurrent() != string(AccountUsable) {
		t.Fatalf("expired price left purchase enabled: state=%q purchase=%t", ready.GetCurrent(), ready.GetPurchaseEnabled())
	}
}

func completeConsoleSteps(t *testing.T, repo Repository, item Provider, now time.Time) {
	t.Helper()
	for _, step := range item.ConsoleSteps {
		_, err := repo.StartStep(context.Background(), StepRecord{
			ProviderID: item.ID, StepID: step.ID, AskID: fmt.Sprintf("fixture-%s-%s", item.ID, step.ID),
			EventTime: now, State: "complete", Answer: "complete", UpdatedAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
