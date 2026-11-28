package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
	_ "modernc.org/sqlite"
)

type ledgerDB struct{ queries []string }

type affectedRows int64

func (r affectedRows) LastInsertId() (int64, error) { return 1, nil }
func (r affectedRows) RowsAffected() (int64, error) { return int64(r), nil }

func (d *ledgerDB) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	d.queries = append(d.queries, query)
	return affectedRows(1), nil
}

func (*ledgerDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) { return nil, nil }

type fixturePrices struct {
	prices  map[string]PriceFact
	missing bool
}

type exactPriceCatalogue struct{ fact provider.PriceFact }

func (c exactPriceCatalogue) LookupPrice(context.Context, string, string, string, bool) (provider.PriceFact, error) {
	return c.fact, nil
}

func (f fixturePrices) LookupPrice(_ context.Context, providerName, region, size string, ipv4 bool) (PriceFact, error) {
	if f.missing {
		return PriceFact{}, ErrQuoteUnavailable
	}
	item, ok := f.prices[size]
	if !ok {
		item = PriceFact{Provider: providerName, OfferID: providerName + "-" + size, Region: region, Size: size, Currency: "USD", ServerMinor: 100, IPv4Minor: 0, TaxBasis: "pre-tax", SourceURL: "https://fixture.invalid/prices", SourceVersion: "fixture-2026-11-23", ObservedAt: time.Date(2026, 11, 23, 11, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)}
	}
	item.Provider, item.Region, item.Size = providerName, region, size
	item.OfferID = providerName + "-" + size
	return item, nil
}

func fixtureService(st intent.Store, cr meter.Credits, p provider.Provider, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) Service {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	return Service{Intents: intent.Service{Store: st, Provider: p, Now: func() time.Time { return now }}, Meter: meter.Service{Credits: cr}, Provider: p, Effects: fixtureLiveEffects(), Window: time.Hour, DB: db, Now: func() time.Time { return now }, Prices: fixturePrices{prices: map[string]PriceFact{
		// Fixture-only split costs; these are not live Hetzner price evidence.
		"cx23": {Provider: "hetzner", OfferID: "cx23-nbg1", Region: "nbg1", Size: "cx23", Currency: "USD", ServerMinor: 649, IPv4Minor: 60, TaxBasis: "pre-tax", SourceURL: "https://fixture.invalid/prices", SourceVersion: "fixture-2026-11-23", ObservedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)},
		"cx33": {Provider: "hetzner", OfferID: "cx33-nbg1", Region: "nbg1", Size: "cx33", Currency: "USD", ServerMinor: 899, IPv4Minor: 60, TaxBasis: "pre-tax", SourceURL: "https://fixture.invalid/prices", SourceVersion: "fixture-2026-11-23", ObservedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)},
	}}, MoneyCap: MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}}
}

type store struct {
	records       map[string]intent.Record
	order         *[]string
	transitionErr error
}

func (s *store) Create(_ context.Context, r intent.Record) (intent.Record, error) {
	if r.ID == "" {
		r.ID = r.IdempotencyKey
	}
	s.records[r.IdempotencyKey] = r
	*s.order = append(*s.order, "intent")
	return r, nil
}
func (s *store) GetByIDOrKey(_ context.Context, key string) (intent.Record, error) {
	r, ok := s.records[key]
	if !ok {
		return intent.Record{}, intent.ErrNotFound
	}
	return r, nil
}
func (s *store) SaveOpen(_ context.Context, r intent.Record) error {
	s.records[r.IdempotencyKey] = r
	return nil
}
func (s *store) Transition(_ context.Context, from intent.State, r intent.Record) error {
	if s.transitionErr != nil {
		return s.transitionErr
	}
	if !intent.CanTransition(from, r.State) {
		return intent.ErrInvalidTransition
	}
	s.records[r.IdempotencyKey] = r
	return nil
}

type credits struct {
	order    *[]string
	refuse   bool
	released bool
}

func (c *credits) ReserveCredits(context.Context, string, string, float64, time.Duration) (meter.Reservation, error) {
	*c.order = append(*c.order, "reserve")
	if c.refuse {
		return meter.Reservation{}, meter.ErrInsufficientCredits
	}
	return meter.Reservation{ID: "r1"}, nil
}
func (c *credits) ReleaseReservation(context.Context, string) error         { c.released = true; return nil }
func (*credits) FinalizeReservation(context.Context, string, float64) error { return nil }

type orderedProvider struct {
	provider.Fake
	order *[]string
	fail  bool
	name  string
}

func (p *orderedProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "hetzner"
}

func (p *orderedProvider) Create(ctx context.Context, s provider.Spec) (provider.Instance, error) {
	*p.order = append(*p.order, "provider")
	if p.fail {
		return provider.Instance{}, errors.New("provider unavailable")
	}
	return p.Fake.Create(ctx, s)
}

func TestCreateOrdersIntentReservationProvider(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := fixtureService(st, cr, p, &ledgerDB{})
	_, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "one", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if got := order; len(got) != 3 || got[0] != "intent" || got[1] != "reserve" || got[2] != "provider" {
		t.Fatalf("order = %v", got)
	}
}

func TestUnknownProviderDefaultsClosedAfterPersistingIntentAndAdmission(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	credits := &credits{order: &order}
	selected := &orderedProvider{order: &order, name: "akash"}
	db := &ledgerDB{}
	svc := fixtureService(st, credits, selected, db)
	closed, err := effectgate.New(effectgate.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc.Effects = closed
	now := svc.Now()
	svc.Prices = exactPriceCatalogue{fact: syntheticAkashPriceFact(now, "testnet", "lease", "akash-lease", "akash1fixture", "1000000", "uakt")}
	svc.MoneyCap = MoneyCap{Currency: "USD", MonthlyMinor: 100_000_000, TaxBasis: "pre-tax"}

	request := intent.Request{IdempotencyKey: "akash-stub-intent", RequestedBy: "fixture-test", Provider: "akash", Spec: provider.Spec{Region: "testnet", Size: "lease", Tags: syntheticAkashTags()}}
	record, _, err := svc.Create(context.Background(), request, 60)
	if err == nil || record.State != intent.StateRefused || !strings.Contains(err.Error(), "effect gate") {
		t.Fatalf("record=%+v error=%v", record, err)
	}
	if len(order) != 1 || order[0] != "intent" || selected.CreateCalls != 0 {
		t.Fatalf("intent/reservation/provider order=%v provider calls=%d", order, selected.CreateCalls)
	}
	joined := strings.Join(db.queries, "\n")
	if !strings.Contains(joined, "INSERT INTO spend_quotes") || !strings.Contains(joined, "INSERT INTO spend_admissions") || strings.Contains(joined, "INSERT INTO reservations") {
		t.Fatalf("closed Akash request did not persist its quote/admission before refusing: %s", joined)
	}
	_, _, retryErr := svc.Create(context.Background(), request, 60)
	if retryErr == nil || !strings.Contains(retryErr.Error(), "requires reconciliation") || selected.CreateCalls != 0 || len(order) != 1 {
		t.Fatalf("retry error=%v provider calls=%d order=%v", retryErr, selected.CreateCalls, order)
	}
}

func TestCreateReleasesHoldWhenProviderFails(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order, fail: true}
	svc := fixtureService(st, cr, p, &ledgerDB{})
	_, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "two", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err == nil || !cr.released {
		t.Fatalf("err=%v released=%t", err, cr.released)
	}
}

func TestCreateReportsFailureToPersistRefusedIntent(t *testing.T) {
	order := []string{}
	writeErr := errors.New("intent write unavailable")
	st := &store{records: map[string]intent.Record{}, order: &order, transitionErr: writeErr}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order, fail: true}
	svc := fixtureService(st, cr, p, &ledgerDB{})
	_, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "write-fail", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if !errors.Is(err, writeErr) || !cr.released {
		t.Fatalf("error = %v, write error present = %t, reservation released = %t", err, errors.Is(err, writeErr), cr.released)
	}
}

func TestCreatePersistsHeldReservationAndLinksInstance(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	order := []string{}
	db := &ledgerDB{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := fixtureService(st, cr, p, db)
	_, created, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "ledger", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err != nil {
		t.Fatal(err)
	}
	reservationQueries := 0
	for _, query := range db.queries {
		if strings.Contains(query, "reservations") {
			reservationQueries++
		}
	}
	if created.ID == "" || reservationQueries != 2 {
		t.Fatalf("created=%+v queries=%v", created, db.queries)
	}
}

func TestCX23QuoteIsAdmittedAndProviderCalledOnce(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := fixtureService(st, cr, p, &ledgerDB{})
	_, created, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "cx23", RequestedBy: "owner", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "cx23"}}, 60)
	if err != nil || created.ID == "" || p.CreateCalls != 1 {
		t.Fatalf("created=%+v calls=%d err=%v", created, p.CreateCalls, err)
	}
}

func TestCX33QuoteIsStoredAndRefusedBeforeProviderCall(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := fixtureService(st, cr, p, &ledgerDB{})
	record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "cx33", RequestedBy: "owner", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "cx33"}}, 60)
	if err == nil || !strings.Contains(err.Error(), "exceeds cap") || record.State != intent.StateRefused || p.CreateCalls != 0 || len(order) != 1 {
		t.Fatalf("record=%+v calls=%d order=%v err=%v", record, p.CreateCalls, order, err)
	}
}

func TestMissingBillingFactsAreStoredAsRefusal(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	db := &ledgerDB{}
	svc := fixtureService(st, cr, p, db)
	svc.Prices = fixturePrices{missing: true}
	record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "missing", RequestedBy: "owner", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "unknown"}}, 60)
	if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 || len(db.queries) != 2 {
		t.Fatalf("state=%s calls=%d writes=%v err=%v", record.State, p.CreateCalls, db.queries, err)
	}
}

func TestStaleBillingFactsAreRefusedBeforeProviderCall(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	db := &ledgerDB{}
	svc := fixtureService(st, cr, p, db)
	prices := svc.Prices.(fixturePrices)
	fact := prices.prices["cx23"]
	fact.ExpiresAt = time.Date(2026, 11, 23, 11, 59, 0, 0, time.UTC)
	prices.prices["cx23"] = fact
	svc.Prices = prices
	record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "stale", RequestedBy: "owner", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "cx23"}}, 60)
	if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 || len(db.queries) != 2 {
		t.Fatalf("state=%s calls=%d writes=%v err=%v", record.State, p.CreateCalls, db.queries, err)
	}
}

func TestCatalogueCurrencyAndTaxBasisMustMatchTheCap(t *testing.T) {
	for _, tc := range []struct {
		name          string
		currency, tax string
	}{{"currency", "EUR", "pre-tax"}, {"tax basis", "USD", "tax-included"}} {
		t.Run(tc.name, func(t *testing.T) {
			order := []string{}
			st := &store{records: map[string]intent.Record{}, order: &order}
			cr := &credits{order: &order}
			p := &orderedProvider{order: &order}
			db := &ledgerDB{}
			svc := fixtureService(st, cr, p, db)
			prices := svc.Prices.(fixturePrices)
			fact := prices.prices["cx23"]
			fact.Currency = tc.currency
			fact.TaxBasis = tc.tax
			prices.prices["cx23"] = fact
			svc.Prices = prices
			record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "mismatch-" + tc.name, RequestedBy: "owner", Provider: "hetzner", Spec: provider.Spec{Region: "nbg1", Size: "cx23"}}, 60)
			if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 {
				t.Fatalf("state=%s creates=%d err=%v", record.State, p.CreateCalls, err)
			}
		})
	}
}

func TestCataloguePricingIsProviderNeutralAndGatedSourcesFailClosed(t *testing.T) {
	t.Run("DigitalOcean fixture admitted through same path", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "digitalocean"}
		svc := fixtureService(st, cr, p, &ledgerDB{})
		record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "do-quote", RequestedBy: "owner", Provider: "digitalocean", Spec: provider.Spec{Region: "nyc1", Size: "cx23"}}, 60)
		if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || p.CreateCalls != 1 {
			t.Fatalf("record=%+v instance=%+v creates=%d err=%v", record, instance, p.CreateCalls, err)
		}
	})
	t.Run("crypto gate with no catalogue offer refuses", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "crypto"}
		db := &ledgerDB{}
		svc := fixtureService(st, cr, p, db)
		svc.Prices = fixturePrices{missing: true}
		record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "crypto-gate", RequestedBy: "owner", Provider: "crypto", Spec: provider.Spec{Region: "testnet", Size: "small"}}, 60)
		if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 {
			t.Fatalf("state=%s creates=%d err=%v", record.State, p.CreateCalls, err)
		}
	})
	t.Run("crypto stub gate refuses even with a valid quote", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "crypto"}
		db := &ledgerDB{}
		svc := fixtureService(st, cr, p, db)
		gate, err := effectgate.New(effectgate.Config{Default: effectgate.Stub, Modes: map[string]effectgate.Mode{"crypto": effectgate.Stub}})
		if err != nil {
			t.Fatal(err)
		}
		svc.Effects = gate
		record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "crypto-stub-gate", RequestedBy: "owner", Provider: "crypto", Spec: provider.Spec{Region: "testnet", Size: "cx23"}}, 60)
		if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 {
			t.Fatalf("state=%s creates=%d err=%v", record.State, p.CreateCalls, err)
		}
		if !strings.Contains(err.Error(), "effect gate") {
			t.Fatalf("error = %v; want explicit closed source gate", err)
		}
	})
	t.Run("adopted zero-cost fixture admitted", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "adopted-vps"}
		svc := fixtureService(st, cr, p, &ledgerDB{})
		prices := svc.Prices.(fixturePrices)
		fact := prices.prices["cx23"]
		fact.ServerMinor = 0
		fact.IPv4Minor = 0
		prices.prices["cx23"] = fact
		svc.Prices = prices
		record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "adopt-zero", RequestedBy: "owner", Provider: "adopted-vps", Spec: provider.Spec{Region: "manual", Size: "declared-zero"}}, 60)
		if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || p.CreateCalls != 1 {
			t.Fatalf("record=%+v instance=%+v creates=%d err=%v", record, instance, p.CreateCalls, err)
		}
	})
	t.Run("adopted VPS without declared catalogue cost refuses", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "adopted-vps"}
		svc := fixtureService(st, cr, p, &ledgerDB{})
		svc.Prices = fixturePrices{missing: true}
		record, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "adopt-missing-cost", RequestedBy: "owner", Provider: "adopted-vps", Spec: provider.Spec{Region: "manual", Size: "declared-cost"}}, 60)
		if err == nil || record.State != intent.StateRefused || p.CreateCalls != 0 {
			t.Fatalf("state=%s creates=%d err=%v", record.State, p.CreateCalls, err)
		}
	})
	t.Run("adopted VPS declared cost is priced through shared cap", func(t *testing.T) {
		order := []string{}
		st := &store{records: map[string]intent.Record{}, order: &order}
		cr := &credits{order: &order}
		p := &orderedProvider{order: &order, name: "adopted-vps"}
		svc := fixtureService(st, cr, p, &ledgerDB{})
		prices := svc.Prices.(fixturePrices)
		fact := prices.prices["cx23"]
		fact.ServerMinor, fact.IPv4Minor = 500, 0
		prices.prices["declared-cost"] = fact
		svc.Prices = prices
		record, instance, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "adopt-declared-cost", RequestedBy: "owner", Provider: "adopted-vps", Spec: provider.Spec{Region: "manual", Size: "declared-cost"}}, 60)
		if err != nil || record.State != intent.StateFulfilled || instance.ID == "" || p.CreateCalls != 1 {
			t.Fatalf("record=%+v instance=%+v creates=%d err=%v", record, instance, p.CreateCalls, err)
		}
	})
}

type renewingCredits struct {
	order []string
	next  int
}

func (c *renewingCredits) ReserveCredits(context.Context, string, string, float64, time.Duration) (meter.Reservation, error) {
	c.next++
	id := fmt.Sprintf("next-%d", c.next)
	c.order = append(c.order, "reserve:"+id)
	return meter.Reservation{ID: id}, nil
}
func (c *renewingCredits) ReleaseReservation(_ context.Context, id string) error {
	c.order = append(c.order, "release:"+id)
	return nil
}
func (*renewingCredits) FinalizeReservation(context.Context, string, float64) error { return nil }

func TestRenewReservationsHoldsSuccessorBeforeReleasingPredecessor(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	db, err := sql.Open("sqlite", "file:renew-reservations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instances (id TEXT PRIMARY KEY,tenant TEXT,reservation_id TEXT,state TEXT); CREATE TABLE reservations (id TEXT PRIMARY KEY,instance_id TEXT,supersedes TEXT,meter_key TEXT,state TEXT,held_at TEXT,settled_at TEXT,quantity INTEGER); INSERT INTO instances VALUES ('i1','owner','old','running'); INSERT INTO reservations VALUES ('old','i1','','compute_minutes','held','2026-09-04T00:00:00Z','',60);`)
	if err != nil {
		t.Fatal(err)
	}
	credits := &renewingCredits{}
	if err := (Service{DB: db, Meter: meter.Service{Credits: credits}, Window: time.Hour}).RenewReservations(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(credits.order, ","), "reserve:next-1,release:old"; got != want {
		t.Fatalf("credit order = %q, want %q", got, want)
	}
	var supersedes, state string
	if err := db.QueryRow(`SELECT supersedes,state FROM reservations WHERE id='next-1'`).Scan(&supersedes, &state); err != nil {
		t.Fatal(err)
	}
	if supersedes != "old" || state != "held" {
		t.Fatalf("successor = %q/%q, want old/held", supersedes, state)
	}
	var current string
	if err := db.QueryRow(`SELECT reservation_id FROM instances WHERE id='i1'`).Scan(&current); err != nil || current != "next-1" {
		t.Fatalf("instance reservation = %q, %v", current, err)
	}
}
