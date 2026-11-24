package reconcile

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"compute-manager/internal/store"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	_ "modernc.org/sqlite"
)

func TestNormalizeInvoiceSourceLineRequiresStableSourceFactsAndPreservesRange(t *testing.T) {
	from := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	facts := InvoiceSourceFacts{Provider: "digitalocean", InvoiceID: "fixture-invoice", LineID: "fixture-line", ProviderInstanceID: "fixture-droplet", AmountMinor: -1250, Currency: "usd", Provenance: "FIXTURE ONLY: synthetic source record", OccurredAt: to, RangeFrom: from, RangeTo: to}
	line, err := NormalizeInvoiceSourceLine(facts)
	if err != nil {
		t.Fatal(err)
	}
	if line.ExternalID() != "compute-manager:digitalocean:invoice:fixture-invoice:line:fixture-line" || line.AmountMinor != -1250 || line.Currency != "USD" || !line.RangeFrom.Equal(from) || !line.RangeTo.Equal(to) || line.Provenance != facts.Provenance {
		t.Fatalf("normalized line lost source facts: %+v", line)
	}
	facts.LineID = ""
	if _, err := NormalizeInvoiceSourceLine(facts); err == nil {
		t.Fatal("missing provider line identity was accepted")
	}
	facts.LineID = "fixture-line"
	facts.Currency = ""
	if _, err := NormalizeInvoiceSourceLine(facts); err == nil {
		t.Fatal("missing currency was accepted")
	}
	facts.Currency = "USD"
	facts.RangeTo = facts.RangeFrom
	if _, err := NormalizeInvoiceSourceLine(facts); err == nil {
		t.Fatal("empty date range was accepted")
	}
}

type fixtureInvoiceLedger struct {
	posts           int
	keys            map[string]string
	failAfterCreate bool
}

func (l *fixtureInvoiceLedger) PostInvoice(_ context.Context, posting InvoicePosting) (string, error) {
	externalID := posting.ExternalID
	if l.keys == nil {
		l.keys = make(map[string]string)
	}
	if id, ok := l.keys[externalID]; ok {
		return id, nil
	}
	l.posts++
	id := "fixture-posting-" + externalID
	l.keys[externalID] = id
	if l.failAfterCreate {
		l.failAfterCreate = false
		return "", errors.New("fixture lost response after ledger persisted posting")
	}
	return id, nil
}

func (l *fixtureInvoiceLedger) ReverseInvoice(context.Context, string, string) (string, error) {
	return "fixture-reversal", nil
}

func invoiceFixtureDB(t *testing.T) *sql.DB {
	return invoiceFixtureDBAt(t, ":memory:")
}

func invoiceFixtureDBAt(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(store.Schema()); err != nil {
		t.Fatal(err)
	}
	// Fixture-only approved CX23 booking. These values are not a provider quote
	// and cannot be interpreted as live price evidence.
	_, err = db.Exec(`
	INSERT INTO instance_intents(id,idempotency_key,provider,state,instance_id,created_at) VALUES('intent-fixture','fixture-key','digitalocean','complete','instance-fixture','2026-11-23T00:00:00Z');
	INSERT INTO instances(id,provider,provider_instance_id,state,reservation_id,expires_at,created_at) VALUES('instance-fixture','digitalocean','provider-vm-fixture','running','reservation-fixture','2026-12-01T00:00:00Z','2026-11-23T00:00:00Z');
	INSERT INTO reservations(id,intent_id,instance_id,meter_key,state,held_at,quantity) VALUES('reservation-fixture','intent-fixture','instance-fixture','compute_minutes','held','2026-11-23T00:00:00Z',60);
	INSERT INTO spend_quotes(id,idempotency_key,provider,region,size,ipv4,term,currency,server_minor,ipv4_minor,monthly_minor,tax_basis,source_version,observed_at,expires_at,state,quote_json,created_at) VALUES('quote-fixture','fixture-key','digitalocean','fixture','fixture-cx23',0,'fixture-only','USD',709,0,709,'fixture-only','fixture-catalogue','2026-11-23T00:00:00Z','2026-11-24T00:00:00Z','approved','{"fixture_only":true}','2026-11-23T00:00:00Z');
	INSERT INTO spend_admissions(idempotency_key,quote_id,cap_currency,cap_monthly_minor,cap_tax_basis,decision,created_at) VALUES('fixture-key','quote-fixture','USD',709,'fixture-only','approved','2026-11-23T00:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func approveInvoiceFixture(t *testing.T, db *sql.DB, externalID, askID string) {
	t.Helper()
	var digest, reservationID, quoteID, state, bookID, accountID string
	if err := db.QueryRow(`SELECT invoice_digest,reservation_id,quote_id,state,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).Scan(&digest, &reservationID, &quoteID, &state, &bookID, &accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE invoice_reconciliation_receipts SET confirmation_event_id=? WHERE external_id=?`, "fixture-requester", externalID); err != nil {
		t.Fatal(err)
	}
	claim := operatordecision.DecisionResolution{AskID: askID, State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "fixture-human", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture-cloudflare-access", AnsweredAt: "2026-11-23T13:00:00Z", ResolvedAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: "fixture-requester", Correlation: map[string]string{"invoice_external_id": externalID, "invoice_digest": digest, "reservation_id": reservationID, "quote_id": quoteID, "ledger_book_id": bookID, "ledger_account_id": accountID}}
	reader := fixedInvoiceAskReader{ask: askreader.Ask{ID: askID, State: "answered", Answer: "approve", AnsweredBy: "fixture-human", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture-cloudflare-access", AnsweredAt: "2026-11-23T13:00:00Z", ResolvedAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: "fixture-requester", Correlation: claim.Correlation}}
	handled, err := (InvoiceApprovalEffect{DB: db, Reader: reader}).Apply(context.Background(), claim)
	if err != nil || !handled {
		t.Fatalf("apply fixture verified-human resolution handled=%v err=%v initialState=%s", handled, err, state)
	}
}

type fixedInvoiceAskReader struct{ ask askreader.Ask }

func (r fixedInvoiceAskReader) ReadAsk(_ context.Context, id string) (askreader.Ask, error) {
	if id != r.ask.ID {
		return askreader.Ask{}, askreader.ErrAskNotFound
	}
	return r.ask, nil
}

var fixtureInvoiceDestination = InvoiceDestination{BookID: "fixture-book", AccountID: "fixture-compute-expense"}

func TestFixtureInvoiceRequiresExplicitConfirmationAndRetriesExactlyOnce(t *testing.T) {
	ctx := context.Background()
	db := invoiceFixtureDB(t)
	ledger := &fixtureInvoiceLedger{failAfterCreate: true}
	r := InvoiceReconciler{DB: db, Ledger: ledger, Destination: fixtureInvoiceDestination, Now: func() time.Time { return time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC) }}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY: synthetic invoice line; no live provider call", OccurredAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)}
	id, err := r.RecordFixtureOrSource(ctx, line)
	if err != nil {
		t.Fatal(err)
	}
	if id != line.ExternalID() {
		t.Fatalf("external ID=%q", id)
	}
	if _, err := r.ConfirmAndPost(ctx, id); !errors.Is(err, ErrInvoiceConfirmation) {
		t.Fatalf("empty confirmation err=%v", err)
	}
	if ledger.posts != 0 {
		t.Fatalf("post calls before confirmation=%d", ledger.posts)
	}
	approveInvoiceFixture(t, db, id, "fixture-ask-01")
	if _, err := r.ConfirmAndPost(ctx, id); err == nil {
		t.Fatal("expected simulated lost response")
	}
	posting, err := r.ConfirmAndPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if posting == "" || ledger.posts != 1 {
		t.Fatalf("posting=%q creates=%d, want one stable create", posting, ledger.posts)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM invoice_reconciliation_receipts WHERE external_id=? AND state='posted' AND ledger_posting_id=? AND provenance LIKE 'FIXTURE ONLY:%'`, id, posting).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("persisted fixture receipt count=%d", count)
	}
}

func TestInvoiceLineRequiresApprovedReservationMatchAndRejectsIdentityMutation(t *testing.T) {
	ctx := context.Background()
	r := InvoiceReconciler{DB: invoiceFixtureDB(t), Destination: fixtureInvoiceDestination}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "not-a-managed-instance", AmountMinor: -100, Currency: "USD", Provenance: "FIXTURE ONLY", OccurredAt: time.Now()}
	if _, err := r.RecordFixtureOrSource(ctx, line); !errors.Is(err, ErrInvoiceUnmatched) {
		t.Fatalf("unmatched err=%v", err)
	}
	line.ProviderInstanceID = "provider-vm-fixture"
	if _, err := r.RecordFixtureOrSource(ctx, line); err != nil {
		t.Fatal(err)
	}
	changedTarget := r
	changedTarget.Destination.AccountID = "fixture-other-account"
	if _, err := changedTarget.RecordFixtureOrSource(ctx, line); !errors.Is(err, ErrInvoiceIdentityReuse) {
		t.Fatalf("changed ledger destination identity err=%v", err)
	}
	line.AmountMinor = -200
	if _, err := r.RecordFixtureOrSource(ctx, line); !errors.Is(err, ErrInvoiceIdentityReuse) {
		t.Fatalf("mutated identity err=%v", err)
	}
}

func TestFixtureCorrectionAppendsLinkedReversalAndReplacement(t *testing.T) {
	ctx := context.Background()
	db := invoiceFixtureDB(t)
	ledger := &fixtureInvoiceLedger{}
	r := InvoiceReconciler{DB: db, Ledger: ledger, Destination: fixtureInvoiceDestination}
	original := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY: original synthetic line", OccurredAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)}
	originalID, err := r.RecordFixtureOrSource(ctx, original)
	if err != nil {
		t.Fatal(err)
	}
	approveInvoiceFixture(t, db, originalID, "fixture-ask-original")
	originalPosting, err := r.ConfirmAndPost(ctx, originalID)
	if err != nil {
		t.Fatal(err)
	}
	corrected := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-02", LineID: "fixture-adjustment-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1100, Currency: "USD", Provenance: "FIXTURE ONLY: synthetic corrected provider line", OccurredAt: time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)}
	correctionID, err := r.RecordCorrection(ctx, originalID, corrected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ConfirmCorrectionAndPost(ctx, correctionID); !errors.Is(err, ErrInvoiceConfirmation) {
		t.Fatalf("missing correction confirmation err=%v", err)
	}
	approveInvoiceFixture(t, db, correctionID, "fixture-ask-correction")
	newPosting, err := r.ConfirmCorrectionAndPost(ctx, correctionID)
	if err != nil {
		t.Fatal(err)
	}
	if newPosting == "" {
		t.Fatal("missing replacement posting")
	}
	var state, reversalOf, reversalPosting string
	if err := db.QueryRow(`SELECT state,reverses_external_id,reversal_posting_id FROM invoice_reconciliation_receipts WHERE external_id=?`, correctionID).Scan(&state, &reversalOf, &reversalPosting); err != nil {
		t.Fatal(err)
	}
	if state != "posted" || reversalOf != originalID || reversalPosting == "" {
		t.Fatalf("correction state=%q reversalOf=%q reversalPosting=%q", state, reversalOf, reversalPosting)
	}
	var originalState, storedOriginal string
	if err := db.QueryRow(`SELECT state,ledger_posting_id FROM invoice_reconciliation_receipts WHERE external_id=?`, originalID).Scan(&originalState, &storedOriginal); err != nil {
		t.Fatal(err)
	}
	if originalState != "reversed" || storedOriginal != originalPosting {
		t.Fatalf("original state=%q posting=%q", originalState, storedOriginal)
	}
	if ledger.posts != 2 {
		t.Fatalf("ledger expense create count=%d, want original plus corrected line", ledger.posts)
	}
}
