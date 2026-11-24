package provision

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	"compute-manager/internal/provider/adoptedvps"
	"compute-manager/internal/provider/akash"
	"compute-manager/internal/setup"
	internalstore "compute-manager/internal/store"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
	_ "modernc.org/sqlite"
)

// syntheticAkashPriceFact builds test-only normalized facts; all URLs and
// versions are fixture data and are never market evidence.
func syntheticAkashPriceFact(now time.Time, region, size, offer, bidProvider, amount, denom string) provider.PriceFact {
	raw := akash.PriceFact{Network: "sandbox-2", ChainID: "synthetic-sandbox", DSeq: "42", GSeq: 1, OSeq: 1, Provider: bidProvider, Region: region, OfferID: offer, Amount: amount, Denom: denom, Period: "block", SourceURL: "https://fixture.invalid/akash-bid", SourceVersion: "synthetic-bid-v1", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), BlockSeconds: 6, BlockSourceURL: "https://fixture.invalid/block-time", BlockSourceVersion: "synthetic-block-v1", BlockObservedAt: now.Add(-time.Minute), BlockExpiresAt: now.Add(time.Hour), BaseUnitsPerToken: "1000000", DenomSourceURL: "https://fixture.invalid/denom-scale", DenomSourceVersion: "synthetic-denom-v1", DenomObservedAt: now.Add(-time.Minute), DenomExpiresAt: now.Add(time.Hour), USDPerToken: "1", FXSourceURL: "https://fixture.invalid/fx", FXSourceVersion: "synthetic-fx-v1", FXObservedAt: now.Add(-time.Minute), FXExpiresAt: now.Add(time.Hour)}
	fact, err := raw.NormalizeCatalogueFact(now, 30*24*60*60)
	if err != nil {
		panic(err)
	}
	fact.Size = size
	return fact
}

func syntheticAkashTags() map[string]string {
	return map[string]string{"akash-dseq": "42", "akash-gseq": "1", "akash-oseq": "1", "akash-provider": "akash1fixture", "akash-rate-amount": "1000000", "akash-rate-denom": "uakt", "akash-lifetime-seconds": "3600"}
}

func TestAdmissionConsumesFreshPriceFromSetupCatalogue(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(setup.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	repo := setup.NewRepository(db)
	if err := repo.PutPrice(context.Background(), provider.PriceFact{Provider: "hetzner", OfferID: "cx23", Region: "nbg1", Size: "cx23", Currency: "USD", ServerMinor: 649, IPv4Minor: 60, TaxBasis: "pre-tax", SourceURL: "https://fixture.invalid/hetzner/pricing", SourceVersion: "fixture-only", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(4 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	catalog := setup.Catalog{Repo: repo, Now: func() time.Time { return now }}
	fake := &provider.Fake{Now: func() time.Time { return now }}
	selected := &sqliteAdmissionProvider{fake: fake, db: db}
	credits := &sqliteAdmissionCredits{}
	svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: credits}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
	record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "setup-catalog-cx23", RequestedBy: "cd3-test", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "cx23"}}, 60)
	if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || selected.createCalls != 1 || !selected.admissionObserved {
		t.Fatalf("record=%+v instance=%+v create=%d admitted-before-create=%t err=%v", record, instance, selected.createCalls, selected.admissionObserved, err)
	}
}

func TestAdoptedVPSDeclaredFactUsesPersistedQuoteAdmissionBeforeAdapterCreate(t *testing.T) {
	for _, tc := range []struct {
		name         string
		monthlyMinor int64
		wantDecision string
		wantCreate   int
	}{
		{name: "within cap", monthlyMinor: 599, wantDecision: "approved", wantCreate: 1},
		{name: "over cap", monthlyMinor: 710, wantDecision: "refused_over_cap", wantCreate: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			if _, err := db.Exec(internalstore.Schema()); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(setup.Schema()); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
			catalog := setup.Catalog{Repo: setup.NewRepository(db), Now: func() time.Time { return now }}
			key := "adopted-vps-" + strings.ReplaceAll(tc.name, " ", "-")
			fact, err := catalog.RecordAdoptedVPSOffer(context.Background(), key, tc.monthlyMinor, "USD", "pre-tax")
			if err != nil {
				t.Fatal(err)
			}
			adapter := &adoptedVPSCountingAdapter{adapter: &adoptedvps.Adapter{Now: func() time.Time { return now }}}
			svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: adapter, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: adapter, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
			record, instance, createErr := svc.Create(context.Background(), intent.Request{IdempotencyKey: key, RequestedBy: "cd4-test", Provider: adoptedvps.Name, Spec: provider.Spec{Region: "manual", Size: fact.Size, AdoptedTarget: &provider.AdoptedTarget{Host: "vps.example.net", User: "deploy", DisplayName: "staging"}}}, 60)
			if tc.wantDecision == "approved" && (createErr != nil || record.State != intent.StateFulfilled || instance.Address != "vps.example.net") {
				t.Fatalf("record=%+v instance=%+v err=%v", record, instance, createErr)
			}
			if tc.wantDecision != "approved" && (createErr == nil || record.State != intent.StateRefused) {
				t.Fatalf("over-cap adoption should be refused: record=%+v err=%v", record, createErr)
			}
			if adapter.createCalls != tc.wantCreate {
				t.Fatalf("adapter Create calls=%d, want %d", adapter.createCalls, tc.wantCreate)
			}
			receipt, err := svc.GetSpendReceipt(context.Background(), key)
			if err != nil || receipt.Decision != tc.wantDecision || receipt.Provider != adoptedvps.Name || receipt.OfferID != fact.OfferID || receipt.MonthlyMinor != tc.monthlyMinor || receipt.Currency != "USD" || receipt.TaxBasis != "pre-tax" || receipt.SourceURL != fact.SourceURL || receipt.SourceVersion != fact.SourceVersion || receipt.ObservedAt.IsZero() || receipt.ExpiresAt.IsZero() {
				t.Fatalf("receipt=%+v err=%v", receipt, err)
			}
		})
	}
}

type adoptedVPSCountingAdapter struct {
	adapter     *adoptedvps.Adapter
	createCalls int
}

func (p *adoptedVPSCountingAdapter) Name() string                 { return p.adapter.Name() }
func (p *adoptedVPSCountingAdapter) Facts() provider.BillingFacts { return p.adapter.Facts() }
func (p *adoptedVPSCountingAdapter) Describe(ctx context.Context, id string) (provider.Instance, error) {
	return p.adapter.Describe(ctx, id)
}
func (p *adoptedVPSCountingAdapter) List(ctx context.Context) ([]provider.Instance, error) {
	return p.adapter.List(ctx)
}
func (p *adoptedVPSCountingAdapter) Destroy(ctx context.Context, id string) error {
	return p.adapter.Destroy(ctx, id)
}
func (p *adoptedVPSCountingAdapter) Create(ctx context.Context, spec provider.Spec) (provider.Instance, error) {
	p.createCalls++
	return p.adapter.Create(ctx, spec)
}

func TestAdmissionConsumesPersistedDigitalOceanCatalogueFact(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(setup.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	repo := setup.NewRepository(db)
	// Fixture-only observation modeled on the official /v2/sizes contract. It is
	// a parser/admission fixture, not a live DigitalOcean quote.
	fact := provider.PriceFact{Provider: "digitalocean", OfferID: "s-2vcpu-4gb", Region: "nyc1", Size: "s-2vcpu-4gb", Currency: "USD", ServerMinor: 709, IPv4Minor: 0, TaxBasis: "pre-tax", SourceURL: "https://api.digitalocean.com/v2/sizes?per_page=200", SourceVersion: "fixture-only-etag", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(4 * time.Minute)}
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	catalog := setup.Catalog{Repo: repo, Now: func() time.Time { return now }}
	fake := &provider.Fake{Now: func() time.Time { return now }}
	selected := &sqliteAdmissionProvider{fake: fake, db: db, providerName: "digitalocean"}
	credits := &sqliteAdmissionCredits{}
	svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: credits}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
	record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "setup-catalog-do-bundled", RequestedBy: "cd3-test", Provider: "digitalocean", Spec: provider.Spec{Region: "nyc1", Size: "s-2vcpu-4gb", IPv4: true}}, 60)
	if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || selected.createCalls != 1 || !selected.admissionObserved {
		t.Fatalf("record=%+v instance=%+v create=%d admitted-before-create=%t err=%v", record, instance, selected.createCalls, selected.admissionObserved, err)
	}
}

func TestLocalQEMUSetupCatalogueZeroFactIsStoredBeforeCreate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(setup.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	repo := setup.NewRepository(db)
	catalog := setup.Catalog{Repo: repo, Now: func() time.Time { return now }}
	if _, err := catalog.RecordLocalQEMUOffer(context.Background(), "headless-linux"); err != nil {
		t.Fatal(err)
	}
	fake := &provider.Fake{Now: func() time.Time { return now }}
	selected := &sqliteAdmissionProvider{fake: fake, db: db, providerName: "local-qemu"}
	svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
	record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "local-qemu-headless-quote", RequestedBy: "cd5-test", Provider: "local-qemu", Spec: provider.Spec{Region: "local", Size: "headless-linux", Profile: "headless-linux"}}, 60)
	if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || selected.createCalls != 1 || !selected.admissionObserved {
		t.Fatalf("record=%+v instance=%+v create=%d admitted-before-create=%t err=%v", record, instance, selected.createCalls, selected.admissionObserved, err)
	}
	facts, err := repo.Prices(context.Background(), "local-qemu")
	if err != nil || len(facts) != 1 || facts[0].OfferID != "host-local-qemu:headless-linux" || facts[0].SourceVersion == "" || facts[0].SourceURL == "" {
		t.Fatalf("persisted catalogue facts = %#v, err=%v", facts, err)
	}
	receipt, err := svc.GetSpendReceipt(context.Background(), "local-qemu-headless-quote")
	if err != nil || receipt.Decision != "approved" || receipt.QuoteState != "approved" || receipt.Provider != "local-qemu" || receipt.OfferID != facts[0].OfferID || receipt.Region != "local" || receipt.Size != "headless-linux" || !receipt.IPv4 || receipt.ServerMinor != 0 || receipt.IPv4Minor != 0 || receipt.MonthlyMinor != 0 || receipt.SourceURL != facts[0].SourceURL || receipt.SourceVersion != facts[0].SourceVersion {
		t.Fatalf("persisted local-QEMU receipt=%+v err=%v", receipt, err)
	}
}

func TestLocalQEMUInvalidProfileOrRegionPersistsRefusalBeforeCreate(t *testing.T) {
	cases := []struct{ name, region, size, profile string }{
		{name: "remote region", region: "nbg1", size: "headless-linux", profile: "headless-linux"},
		{name: "size profile mismatch", region: "local", size: "headless-linux", profile: "desktop-linux"},
		{name: "unsupported profile", region: "local", size: "other", profile: "other"},
		{name: "missing profile", region: "local", size: "headless-linux", profile: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			if _, err := db.Exec(internalstore.Schema()); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(setup.Schema()); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
			selected := &sqliteAdmissionProvider{fake: &provider.Fake{Now: func() time.Time { return now }}, db: db, providerName: "local-qemu"}
			svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: setup.Catalog{Repo: setup.NewRepository(db), Now: func() time.Time { return now }}, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
			key := "local-qemu-invalid-" + strings.ReplaceAll(tc.name, " ", "-")
			record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: key, RequestedBy: "cd5-test", Provider: "local-qemu", Spec: provider.Spec{Region: tc.region, Size: tc.size, Profile: tc.profile}}, 60)
			if err == nil || record.State != intent.StateRefused || selected.createCalls != 0 {
				t.Fatalf("record=%+v create=%d err=%v; expected fail-closed before provider create", record, selected.createCalls, err)
			}
			var quoteState, admissionDecision string
			if err := db.QueryRow(`SELECT q.state,a.decision FROM spend_quotes q JOIN spend_admissions a ON a.idempotency_key=q.idempotency_key WHERE q.idempotency_key=?`, key).Scan(&quoteState, &admissionDecision); err != nil {
				t.Fatal(err)
			}
			if quoteState != "refused_billing_facts" || admissionDecision != quoteState {
				t.Fatalf("quote=%q admission=%q; expected stored billing-facts refusal", quoteState, admissionDecision)
			}
		})
	}
}

func TestLocalQEMUMissingStaleOrMismatchedCatalogueFactsRefuseBeforeCreate(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		fact    *provider.PriceFact
	}{
		{name: "missing fact", profile: "headless-linux"},
		{name: "stale fact", profile: "headless-linux", fact: &provider.PriceFact{Provider: "local-qemu", OfferID: "host-local-qemu:headless-linux", Region: "local", Size: "headless-linux", Currency: "USD", TaxBasis: "pre-tax", SourceURL: "urn:vrooli:compute-manager:host-local-qemu-zero-provider-compute-ipv4-charge", SourceVersion: "declared-host-local-zero-provider-compute-ipv4-charge-v1"}},
		{name: "mismatched offer identity", profile: "headless-linux", fact: &provider.PriceFact{Provider: "local-qemu", OfferID: "host-local-qemu:desktop-linux", Region: "local", Size: "headless-linux", Currency: "USD", TaxBasis: "pre-tax", SourceURL: "urn:vrooli:compute-manager:host-local-qemu-zero-provider-compute-ipv4-charge", SourceVersion: "declared-host-local-zero-provider-compute-ipv4-charge-v1"}},
		{name: "profile missing from catalogue", profile: "desktop-linux", fact: &provider.PriceFact{Provider: "local-qemu", OfferID: "host-local-qemu:headless-linux", Region: "local", Size: "headless-linux", Currency: "USD", TaxBasis: "pre-tax", SourceURL: "urn:vrooli:compute-manager:host-local-qemu-zero-provider-compute-ipv4-charge", SourceVersion: "declared-host-local-zero-provider-compute-ipv4-charge-v1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			if _, err := db.Exec(internalstore.Schema()); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(setup.Schema()); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
			repo := setup.NewRepository(db)
			if tc.fact != nil {
				fact := *tc.fact
				if tc.name == "stale fact" {
					fact.ObservedAt, fact.ExpiresAt = now.Add(-2*time.Hour), now.Add(-time.Hour)
				} else {
					fact.ObservedAt, fact.ExpiresAt = now.Add(-time.Minute), now.Add(time.Hour)
				}
				if err := repo.PutPrice(context.Background(), fact); err != nil {
					t.Fatal(err)
				}
			}
			selected := &sqliteAdmissionProvider{fake: &provider.Fake{Now: func() time.Time { return now }}, db: db, providerName: "local-qemu"}
			svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: setup.Catalog{Repo: repo, Now: func() time.Time { return now }}, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
			key := "local-qemu-catalog-invalid-" + strings.ReplaceAll(tc.name, " ", "-")
			record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: key, RequestedBy: "cd5-test", Provider: "local-qemu", Spec: provider.Spec{Region: "local", Size: tc.profile, Profile: tc.profile}}, 60)
			if err == nil || record.State != intent.StateRefused || selected.createCalls != 0 {
				t.Fatalf("record=%+v create=%d err=%v; expected refusal before provider create", record, selected.createCalls, err)
			}
			var quoteState, admissionDecision string
			if err := db.QueryRow(`SELECT q.state,a.decision FROM spend_quotes q JOIN spend_admissions a ON a.idempotency_key=q.idempotency_key WHERE q.idempotency_key=?`, key).Scan(&quoteState, &admissionDecision); err != nil {
				t.Fatal(err)
			}
			if quoteState != "refused_billing_facts" || admissionDecision != quoteState {
				t.Fatalf("quote=%q admission=%q; expected stored billing-facts refusal", quoteState, admissionDecision)
			}
		})
	}
}

func TestAkashStubEffectGatePersistsQuoteAndRefusesBeforeProvider(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(setup.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	repo := setup.NewRepository(db)
	// Fixture-only quote proves that even priceable Akash remains stub-gated.
	fact := syntheticAkashPriceFact(now, "testnet", "lease", "fixture-testnet-offer", "akash1fixture", "1000000", "uakt")
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	catalog := setup.Catalog{Repo: repo, Now: func() time.Time { return now }}
	fake := &provider.Fake{Now: func() time.Time { return now }}
	selected := &sqliteAdmissionProvider{fake: fake, db: db, providerName: "akash"}
	svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: selected, Effects: fixtureStubEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }}
	request := intent.Request{IdempotencyKey: "akash-stub-receipt", RequestedBy: "cd9-test", Provider: "akash", Spec: provider.Spec{Region: "testnet", Size: "lease", Tags: syntheticAkashTags()}}
	record, _, err := svc.Create(context.Background(), request, 60)
	if err == nil || record.State != intent.StateRefused || selected.createCalls != 0 {
		t.Fatalf("record=%+v creates=%d err=%v", record, selected.createCalls, err)
	}
	var quoteState, admissionDecision, quoteReason string
	if err := db.QueryRow(`SELECT q.state,a.decision,q.reason FROM spend_quotes q JOIN spend_admissions a ON a.idempotency_key=q.idempotency_key WHERE q.idempotency_key=?`, "akash-stub-receipt").Scan(&quoteState, &admissionDecision, &quoteReason); err != nil {
		t.Fatal(err)
	}
	if quoteState != "refused_effect_gate" || admissionDecision != quoteState || !strings.Contains(quoteReason, "stub") {
		t.Fatalf("quote=%q admission=%q reason=%q", quoteState, admissionDecision, quoteReason)
	}
	var intentState string
	if err := db.QueryRow(`SELECT state FROM instance_intents WHERE idempotency_key=?`, "akash-stub-receipt").Scan(&intentState); err != nil {
		t.Fatal(err)
	}
	if intentState != "refused" {
		t.Fatalf("intent state=%q", intentState)
	}
	if _, _, err := svc.Create(context.Background(), request, 60); err == nil || !strings.Contains(err.Error(), "requires reconciliation") || selected.createCalls != 0 {
		t.Fatalf("replay error=%v provider create calls=%d", err, selected.createCalls)
	}
}

type akashProvisionClient struct {
	calls int
	req   akash.LeaseRequest
}

func (c *akashProvisionClient) CreateLease(_ context.Context, req akash.LeaseRequest) (akash.LeaseReceipt, error) {
	c.calls++
	c.req = req
	return akash.LeaseReceipt{DeploymentID: "fixture-deployment", LeaseID: "fixture-lease", Provider: req.ProviderAddress, Endpoint: "https://fixture.invalid/app", IntentID: req.IntentID}, nil
}
func (*akashProvisionClient) GetLease(context.Context, string) (akash.LeaseReceipt, error) {
	return akash.LeaseReceipt{}, nil
}
func (*akashProvisionClient) ListLeases(context.Context) ([]akash.LeaseReceipt, error) {
	return nil, nil
}
func (*akashProvisionClient) ReleaseLease(context.Context, string) error { return nil }

func TestAkashLiveFixtureBindsPersistedIntentToProviderTargetReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(setup.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	catalog := setup.Catalog{Repo: setup.NewRepository(db), Now: func() time.Time { return now }}
	if err := catalog.Repo.PutPrice(context.Background(), syntheticAkashPriceFact(now, "testnet", "lease", "fixture-testnet-offer", "akash1fixture", "1000000", "uakt")); err != nil {
		t.Fatal(err)
	}
	effects, err := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{"akash": effectgate.Live}})
	if err != nil {
		t.Fatal(err)
	}
	client := &akashProvisionClient{}
	adapter := &akash.Adapter{Config: akash.Config{Network: "testnet", ChainID: "fixture-sandbox", Endpoint: "https://rpc.fixture.invalid", ProviderAddress: "akash1fixture", WalletCredentialRef: "credential-authority://compute-manager/akash-testnet-wallet"}, Gate: effects, Client: client, Now: func() time.Time { return now }}
	svc := Service{Intents: intent.Service{Store: intent.SQLStore{DB: db}, Provider: adapter, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: &sqliteAdmissionCredits{}}, Prices: catalog, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 50_000_000, TaxBasis: "pre-tax"}, Provider: adapter, Effects: effects, Window: time.Hour, DB: db, Now: func() time.Time { return now }}
	request := intent.Request{IdempotencyKey: "fixture-akash-live", RequestedBy: "cd9-test", Provider: "akash", Spec: provider.Spec{Region: "testnet", Size: "lease", Tags: syntheticAkashTags()}}
	record, instance, err := svc.Create(context.Background(), request, 60)
	if err != nil {
		t.Fatal(err)
	}
	wantRef := akash.TargetReference("fixture-sandbox", "fixture-deployment", "fixture-lease")
	if record.State != intent.StateFulfilled || record.InstanceID != wantRef || instance.ID != wantRef || client.calls != 1 || client.req.IntentID != record.ID {
		t.Fatalf("record=%+v instance=%+v client calls=%d request=%+v", record, instance, client.calls, client.req)
	}
	var storedState, storedInstanceID string
	if err := db.QueryRow(`SELECT state,instance_id FROM instance_intents WHERE idempotency_key=?`, request.IdempotencyKey).Scan(&storedState, &storedInstanceID); err != nil {
		t.Fatal(err)
	}
	if storedState != "fulfilled" || storedInstanceID != wantRef {
		t.Fatalf("stored intent state=%q instance=%q", storedState, storedInstanceID)
	}
	quoteReceipt, err := svc.GetSpendReceipt(context.Background(), request.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	if quoteReceipt.Currency != "USD" || quoteReceipt.MonthlyMinor != 43_200_000 || quoteReceipt.PriceProvenanceJSON == "" || quoteReceipt.PriceProvenanceFingerprint == "" {
		t.Fatalf("persisted quote lost normalized amount or source/conversion provenance: %+v", quoteReceipt)
	}
	if _, _, err := svc.Create(context.Background(), request, 60); err == nil || !strings.Contains(err.Error(), "requires reconciliation") || client.calls != 1 {
		t.Fatalf("replay error=%v client calls=%d", err, client.calls)
	}
}
