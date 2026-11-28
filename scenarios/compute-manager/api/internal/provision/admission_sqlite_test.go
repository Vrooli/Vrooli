package provision

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	internalstore "compute-manager/internal/store"
	_ "modernc.org/sqlite"
)

const fixtureOnlyPriceSource = "https://fixture.invalid/compute-manager/prices"

var fixtureOnlyObservedAt = time.Date(2026, 11, 23, 11, 0, 0, 0, time.UTC)

// sqliteFixtureCatalogue is deliberately detached from provider APIs. Its
// values exist only to exercise the A2 cap and are not live price evidence.
type sqliteFixtureCatalogue struct {
	facts map[string]provider.PriceFact
}

func (c sqliteFixtureCatalogue) LookupPrice(_ context.Context, providerName, region, size string, includeIPv4 bool) (provider.PriceFact, error) {
	fact, ok := c.facts[size]
	if !ok {
		return provider.PriceFact{}, fmt.Errorf("fixture-only price unavailable for %s", size)
	}
	if !includeIPv4 {
		fact.IPv4Minor = 0
	}
	fact.Provider = providerName
	fact.Region = region
	fact.Size = size
	return fact, nil
}

// sqliteAdmissionProvider wraps the repository fake while checking durable
// admission records at the exact instant Create would cross the provider seam.
type sqliteAdmissionProvider struct {
	fake              *provider.Fake
	db                *sql.DB
	providerName      string
	createCalls       int
	admissionObserved bool
}

func (p *sqliteAdmissionProvider) Name() string {
	if p.providerName != "" {
		return p.providerName
	}
	return "hetzner"
}
func (*sqliteAdmissionProvider) Facts() provider.BillingFacts {
	return provider.BillingFacts{RoundingUnit: time.Hour, MinimumBillable: time.Hour, StoppedStillBills: true}
}

func (p *sqliteAdmissionProvider) Create(ctx context.Context, spec provider.Spec) (provider.Instance, error) {
	p.createCalls++
	var quoteState, admissionDecision, intentState string
	if err := p.db.QueryRowContext(ctx, `
		SELECT q.state, a.decision, i.state
		FROM spend_quotes q
		JOIN spend_admissions a ON a.idempotency_key=q.idempotency_key
		JOIN instance_intents i ON i.idempotency_key=q.idempotency_key
		WHERE i.id=?`, spec.Tags["vrooli-intent-id"]).Scan(&quoteState, &admissionDecision, &intentState); err != nil {
		return provider.Instance{}, fmt.Errorf("read durable admission before provider create: %w", err)
	}
	if quoteState != "approved" || admissionDecision != "approved" || intentState != string(intent.StateOpen) {
		return provider.Instance{}, fmt.Errorf("provider create observed quote=%q admission=%q intent=%q; want approved/approved/open", quoteState, admissionDecision, intentState)
	}
	p.admissionObserved = true
	return p.fake.Create(ctx, spec)
}

func (p *sqliteAdmissionProvider) Describe(ctx context.Context, id string) (provider.Instance, error) {
	return p.fake.Describe(ctx, id)
}
func (p *sqliteAdmissionProvider) List(ctx context.Context) ([]provider.Instance, error) {
	return p.fake.List(ctx)
}
func (p *sqliteAdmissionProvider) Destroy(ctx context.Context, id string) error {
	return p.fake.Destroy(ctx, id)
}

type sqliteAdmissionCredits struct{ calls int }

func (c *sqliteAdmissionCredits) ReserveCredits(_ context.Context, _, _ string, _ float64, _ time.Duration) (meter.Reservation, error) {
	c.calls++
	return meter.Reservation{ID: fmt.Sprintf("fixture-reservation-%d", c.calls)}, nil
}
func (*sqliteAdmissionCredits) ReleaseReservation(context.Context, string) error { return nil }
func (*sqliteAdmissionCredits) FinalizeReservation(context.Context, string, float64) error {
	return nil
}

func newSQLiteAdmissionService(t *testing.T) (*sql.DB, Service, *sqliteAdmissionProvider, time.Time) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatalf("apply compute-manager schema: %v", err)
	}

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	prices := sqliteFixtureCatalogue{facts: map[string]provider.PriceFact{
		"cx23": {
			Provider: "hetzner", OfferID: "fixture-only-cx23-nbg1", Region: "nbg1", Size: "cx23",
			Currency: "USD", ServerMinor: 649, IPv4Minor: 60, TaxBasis: "pre-tax",
			SourceURL: fixtureOnlyPriceSource, SourceVersion: "fixture-only-2026-11-23",
			ObservedAt: fixtureOnlyObservedAt, ExpiresAt: now.Add(time.Hour),
		},
		"cx33": {
			Provider: "hetzner", OfferID: "fixture-only-cx33-nbg1", Region: "nbg1", Size: "cx33",
			Currency: "USD", ServerMinor: 899, IPv4Minor: 60, TaxBasis: "pre-tax",
			SourceURL: fixtureOnlyPriceSource, SourceVersion: "fixture-only-2026-11-23",
			ObservedAt: fixtureOnlyObservedAt, ExpiresAt: now.Add(time.Hour),
		},
	}}
	fake := &provider.Fake{Now: func() time.Time { return now }}
	selected := &sqliteAdmissionProvider{fake: fake, db: db}
	credits := &sqliteAdmissionCredits{}
	svc := Service{
		Intents:  intent.Service{Store: intent.SQLStore{DB: db}, Provider: selected, Now: func() time.Time { return now }},
		Meter:    meter.Service{Credits: credits},
		Prices:   prices,
		MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"},
		Provider: selected,
		Effects:  fixtureLiveEffects(),
		Window:   time.Hour,
		DB:       db,
		Now:      func() time.Time { return now },
	}
	return db, svc, selected, now
}

func TestSQLiteAdmissionPersistsCX23QuoteAndApprovalBeforeOneProviderCreate(t *testing.T) {
	db, svc, selected, _ := newSQLiteAdmissionService(t)
	record, created, err := svc.Create(context.Background(), intent.Request{
		IdempotencyKey: "fixture-cx23-approved",
		RequestedBy:    "cd3-test-owner",
		Provider:       "hetzner",
		Spec:           provider.Spec{Region: "nbg1", Size: "cx23", Image: "ubuntu-24.04"},
	}, 60)
	if err != nil {
		t.Fatalf("create admitted fixture CX23: %v", err)
	}
	if record.State != intent.StateFulfilled || created.ID == "" {
		t.Fatalf("intent=%+v created=%+v; want fulfilled intent and instance", record, created)
	}
	if selected.createCalls != 1 || selected.fake.CreateCalls != 1 || !selected.admissionObserved {
		t.Fatalf("provider creates wrapper=%d fake=%d admission-observed=%t; want exactly one after persisted admission", selected.createCalls, selected.fake.CreateCalls, selected.admissionObserved)
	}

	var quoteState, currency, taxBasis, sourceURL, sourceVersion, offerID string
	var serverMinor, ipv4Minor, monthlyMinor int64
	if err := db.QueryRow(`SELECT state,currency,server_minor,ipv4_minor,monthly_minor,tax_basis,source_url,source_version,offer_id FROM spend_quotes WHERE idempotency_key=?`, "fixture-cx23-approved").Scan(&quoteState, &currency, &serverMinor, &ipv4Minor, &monthlyMinor, &taxBasis, &sourceURL, &sourceVersion, &offerID); err != nil {
		t.Fatalf("read persisted CX23 quote: %v", err)
	}
	if quoteState != "approved" || currency != "USD" || serverMinor != 649 || ipv4Minor != 60 || monthlyMinor != 709 || taxBasis != "pre-tax" || sourceURL != fixtureOnlyPriceSource || sourceVersion != "fixture-only-2026-11-23" || offerID != "fixture-only-cx23-nbg1" {
		t.Fatalf("persisted quote = state %q, %s server=%d ipv4=%d total=%d tax %q offer %q source %q/%q; want approved fixture quote totaling cap", quoteState, currency, serverMinor, ipv4Minor, monthlyMinor, taxBasis, offerID, sourceURL, sourceVersion)
	}
	var decision, capCurrency, capTaxBasis string
	var capMonthlyMinor int64
	if err := db.QueryRow(`SELECT decision,cap_currency,cap_monthly_minor,cap_tax_basis FROM spend_admissions WHERE idempotency_key=?`, "fixture-cx23-approved").Scan(&decision, &capCurrency, &capMonthlyMinor, &capTaxBasis); err != nil {
		t.Fatalf("read persisted CX23 admission: %v", err)
	}
	if decision != "approved" || capCurrency != "USD" || capMonthlyMinor != 709 || capTaxBasis != "pre-tax" {
		t.Fatalf("persisted admission = %q cap %s %d tax %q; want approved USD 709 pre-tax", decision, capCurrency, capMonthlyMinor, capTaxBasis)
	}
}

func TestSQLiteAdmissionPersistsCX33RefusalAndCapWithoutProviderCreate(t *testing.T) {
	db, svc, selected, _ := newSQLiteAdmissionService(t)
	record, _, err := svc.Create(context.Background(), intent.Request{
		IdempotencyKey: "fixture-cx33-refused",
		RequestedBy:    "cd3-test-owner",
		Provider:       "hetzner",
		Spec:           provider.Spec{Region: "nbg1", Size: "cx33", Image: "ubuntu-24.04"},
	}, 60)
	if err == nil || !strings.Contains(err.Error(), "exceeds cap") {
		t.Fatalf("create over-cap fixture CX33 error = %v; want explicit over-cap refusal", err)
	}
	if record.State != intent.StateRefused {
		t.Fatalf("intent state = %q; want refused", record.State)
	}
	if selected.createCalls != 0 || selected.fake.CreateCalls != 0 || selected.admissionObserved {
		t.Fatalf("provider creates wrapper=%d fake=%d admission-observed=%t; want zero", selected.createCalls, selected.fake.CreateCalls, selected.admissionObserved)
	}

	var quoteState, reason, currency, taxBasis, sourceURL, sourceVersion, offerID string
	var serverMinor, ipv4Minor, monthlyMinor int64
	if err := db.QueryRow(`SELECT state,reason,currency,server_minor,ipv4_minor,monthly_minor,tax_basis,source_url,source_version,offer_id FROM spend_quotes WHERE idempotency_key=?`, "fixture-cx33-refused").Scan(&quoteState, &reason, &currency, &serverMinor, &ipv4Minor, &monthlyMinor, &taxBasis, &sourceURL, &sourceVersion, &offerID); err != nil {
		t.Fatalf("read persisted CX33 quote: %v", err)
	}
	if quoteState != "refused_over_cap" || !strings.Contains(reason, "exceeds cap") || currency != "USD" || serverMinor != 899 || ipv4Minor != 60 || monthlyMinor != 959 || taxBasis != "pre-tax" || sourceURL != fixtureOnlyPriceSource || sourceVersion != "fixture-only-2026-11-23" || offerID != "fixture-only-cx33-nbg1" {
		t.Fatalf("persisted quote = state %q reason %q %s server=%d ipv4=%d total=%d tax %q offer %q source %q/%q; want refused fixture quote above cap", quoteState, reason, currency, serverMinor, ipv4Minor, monthlyMinor, taxBasis, offerID, sourceURL, sourceVersion)
	}
	var decision, admissionReason, capCurrency, capTaxBasis string
	var capMonthlyMinor int64
	if err := db.QueryRow(`SELECT decision,reason,cap_currency,cap_monthly_minor,cap_tax_basis FROM spend_admissions WHERE idempotency_key=?`, "fixture-cx33-refused").Scan(&decision, &admissionReason, &capCurrency, &capMonthlyMinor, &capTaxBasis); err != nil {
		t.Fatalf("read persisted CX33 admission: %v", err)
	}
	if decision != "refused_over_cap" || !strings.Contains(admissionReason, "exceeds cap") || capCurrency != "USD" || capMonthlyMinor != 709 || capTaxBasis != "pre-tax" {
		t.Fatalf("persisted admission = %q reason %q cap %s %d tax %q; want refused against USD 709 pre-tax", decision, admissionReason, capCurrency, capMonthlyMinor, capTaxBasis)
	}
}

func TestSQLiteAdmissionLostCreateResponseRetryRequiresReconciliationWithoutReorder(t *testing.T) {
	db, svc, selected, _ := newSQLiteAdmissionService(t)
	selected.fake.LoseCreateResponse = true
	request := intent.Request{
		IdempotencyKey: "fixture-cx23-lost-response",
		RequestedBy:    "cd3-test-owner",
		Provider:       "hetzner",
		Spec:           provider.Spec{Region: "nbg1", Size: "cx23", Image: "ubuntu-24.04"},
	}

	record, _, err := svc.Create(context.Background(), request, 60)
	if err != provider.ErrCreateResponseLost {
		t.Fatalf("first create error = %v; want lost provider response", err)
	}
	if record.State != intent.StateOpen || selected.createCalls != 1 || selected.fake.CreateCalls != 1 || !selected.admissionObserved {
		t.Fatalf("first attempt intent=%+v wrapper creates=%d fake creates=%d admission-observed=%t; want open intent and one create after persisted admission", record, selected.createCalls, selected.fake.CreateCalls, selected.admissionObserved)
	}

	retryRecord, _, err := svc.Create(context.Background(), request, 60)
	if err == nil || !strings.Contains(err.Error(), "requires reconciliation") {
		t.Fatalf("retry error = %v; want reconciliation-required refusal", err)
	}
	if retryRecord.State != intent.StateOpen || selected.createCalls != 1 || selected.fake.CreateCalls != 1 {
		t.Fatalf("retry intent=%+v wrapper creates=%d fake creates=%d; want open intent and exactly one total provider create", retryRecord, selected.createCalls, selected.fake.CreateCalls)
	}
	instances, err := selected.fake.List(context.Background())
	if err != nil || len(instances) != 1 {
		t.Fatalf("fake provider instances=%d err=%v; want one persisted instance awaiting reconciliation", len(instances), err)
	}
	var quoteCount, admissionCount, reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM spend_quotes WHERE idempotency_key=?`, request.IdempotencyKey).Scan(&quoteCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM spend_admissions WHERE idempotency_key=?`, request.IdempotencyKey).Scan(&admissionCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM reservations WHERE intent_id=?`, retryRecord.ID).Scan(&reservationCount); err != nil {
		t.Fatal(err)
	}
	if quoteCount != 1 || admissionCount != 1 || reservationCount != 1 {
		t.Fatalf("persisted quote/admission/reservation counts = %d/%d/%d; retry must not reorder spend or reserve again", quoteCount, admissionCount, reservationCount)
	}
}
