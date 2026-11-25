package reconcile

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	_ "modernc.org/sqlite"
)

type invoiceDecisionPublisher struct {
	events []eventbus.DomainEvent
	err    error
}

func (p *invoiceDecisionPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return p.err
}

func signInvoiceDecision(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestInvoiceApprovalRequestAndVerifiedHubResolutionBindExactDigest(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "invoice-receipt.sqlite")
	db := invoiceFixtureDBAt(t, dbPath)
	r := InvoiceReconciler{DB: db, Destination: fixtureInvoiceDestination, Now: func() time.Time { return time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC) }}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY: synthetic invoice; no live provider call", OccurredAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), RangeFrom: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), RangeTo: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}
	id, err := r.RecordFixtureOrSource(ctx, line)
	require.NoError(t, err)
	publisher := &invoiceDecisionPublisher{}
	firstEventID, err := r.RequestInvoiceApproval(ctx, id, publisher)
	require.NoError(t, err)
	secondEventID, err := r.RequestInvoiceApproval(ctx, id, publisher)
	require.NoError(t, err)
	require.Equal(t, firstEventID, secondEventID)
	require.Len(t, publisher.events, 2)
	event := publisher.events[0]
	require.Equal(t, "compute-manager", event.Source)
	require.Equal(t, InvoiceApprovalEventType, event.EventType)
	payload := event.Payload
	corrRaw, ok := payload["correlation"].(map[string]any)
	require.True(t, ok)
	correlation := map[string]string{}
	for key, value := range corrRaw {
		correlation[key] = fmt.Sprint(value)
	}
	require.Equal(t, id, correlation["invoice_external_id"])
	require.NotEmpty(t, correlation["invoice_digest"])
	require.Contains(t, payload["technical_details"], "FIXTURE ONLY")
	require.Contains(t, payload["technical_details"], "Invoice period: 2026-11-01T00:00:00Z to 2026-12-01T00:00:00Z")
	requesterRef := operatordecision.EventID(event)
	task := askreader.Ask{ID: "fixture-ask", State: "answered", Answer: "approve", AnsweredBy: "fixture-human", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture-cloudflare-access", AnsweredAt: "2026-11-23T13:00:00Z", ResolvedAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: requesterRef, Correlation: correlation}
	reader := fixedInvoiceAskReader{ask: task}
	claim := map[string]any{"ask_id": task.ID, "state": task.State, "answer": task.Answer, "answered_by": task.AnsweredBy, "answered_by_identity_kind": task.AnsweredByIdentityKind, "answered_by_identity_source": task.AnsweredByIdentitySource, "answered_at": task.AnsweredAt, "resolved_at": task.ResolvedAt, "source": "compute-manager", "source_event_type": InvoiceApprovalEventType, "requester_ref": requesterRef, "correlation": correlation}
	envelope := map[string]any{"event_id": "fixture-resolution-event", "event_type": operatordecision.AskResolvedEventType, "source_scenario": "notification-hub", "payload": claim}
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	const secret = "fixture-webhook-secret"
	webhook := operatordecision.NewWebhook(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(secret), Source: "compute-manager", Reader: reader, Effects: []operatordecision.DomainEffect{InvoiceApprovalEffect{DB: db, Reader: reader}}, Logf: func(string, ...any) {}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/notification-hub", strings.NewReader(string(body)))
	req.Header.Set(operatordecision.SignatureHeader, signInvoiceDecision(body, secret))
	response := httptest.NewRecorder()
	webhook.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	// Redelivery before or after restart must only observe the original durable
	// human receipt; it cannot create a second authorization record.
	replay := httptest.NewRequest(http.MethodPost, "/api/v1/events/notification-hub", strings.NewReader(string(body)))
	replay.Header.Set(operatordecision.SignatureHeader, signInvoiceDecision(body, secret))
	replayed := httptest.NewRecorder()
	webhook.ServeHTTP(replayed, replay)
	require.Equal(t, http.StatusOK, replayed.Code, replayed.Body.String())
	var state, actor, identitySource, confirmedDigest, askID string
	err = db.QueryRow(`SELECT state,confirmation_actor,confirmation_identity_source,confirmation_digest,confirmation_ask_id FROM invoice_reconciliation_receipts WHERE external_id=?`, id).Scan(&state, &actor, &identitySource, &confirmedDigest, &askID)
	require.NoError(t, err)
	require.Equal(t, "confirmed", state)
	require.Equal(t, "fixture-human", actor)
	require.Equal(t, "fixture-cloudflare-access", identitySource)
	require.Equal(t, correlation["invoice_digest"], confirmedDigest)
	require.Equal(t, task.ID, askID)
	var receiptCount int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM invoice_confirmation_receipts WHERE ask_id=? AND external_id=? AND invoice_digest=? AND answer='approve' AND answered_by='fixture-human' AND identity_source='fixture-cloudflare-access'`, task.ID, id, confirmedDigest).Scan(&receiptCount))
	require.Equal(t, 1, receiptCount)
	require.NoError(t, db.Close())

	reopened, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer reopened.Close()
	restartedWebhook := operatordecision.NewWebhook(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(secret), Source: "compute-manager", Reader: reader, Effects: []operatordecision.DomainEffect{InvoiceApprovalEffect{DB: reopened, Reader: reader}}, Logf: func(string, ...any) {}})
	afterRestart := httptest.NewRequest(http.MethodPost, "/api/v1/events/notification-hub", strings.NewReader(string(body)))
	afterRestart.Header.Set(operatordecision.SignatureHeader, signInvoiceDecision(body, secret))
	restartedResponse := httptest.NewRecorder()
	restartedWebhook.ServeHTTP(restartedResponse, afterRestart)
	require.Equal(t, http.StatusOK, restartedResponse.Code, restartedResponse.Body.String())
	require.NoError(t, reopened.QueryRow(`SELECT count(*) FROM invoice_confirmation_receipts WHERE external_id=?`, id).Scan(&receiptCount))
	require.Equal(t, 1, receiptCount)
	var persistedFrom, persistedTo string
	require.NoError(t, reopened.QueryRow(`SELECT range_from,range_to FROM invoice_reconciliation_receipts WHERE external_id=?`, id).Scan(&persistedFrom, &persistedTo))
	require.Equal(t, fixtureInvoiceRangeFrom.Format(time.RFC3339Nano), persistedFrom)
	require.Equal(t, fixtureInvoiceRangeTo.Format(time.RFC3339Nano), persistedTo)
}

func TestInvoiceApprovalEffectRefusesHubMismatchAndCannotBeUsedWithoutReader(t *testing.T) {
	db := invoiceFixtureDB(t)
	r := InvoiceReconciler{DB: db, Destination: fixtureInvoiceDestination}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY", OccurredAt: time.Now(), RangeFrom: fixtureInvoiceRangeFrom, RangeTo: fixtureInvoiceRangeTo}
	id, err := r.RecordFixtureOrSource(context.Background(), line)
	require.NoError(t, err)
	var digest, reservation, quote string
	require.NoError(t, db.QueryRow(`SELECT invoice_digest,reservation_id,quote_id FROM invoice_reconciliation_receipts WHERE external_id=?`, id).Scan(&digest, &reservation, &quote))
	claim := operatordecision.DecisionResolution{AskID: "ask", State: "answered", Answer: "approve", AnsweredBy: "someone", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture", AnsweredAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: "event", Correlation: map[string]string{"invoice_external_id": id, "invoice_digest": digest, "reservation_id": reservation, "quote_id": quote}}
	handled, err := (InvoiceApprovalEffect{DB: db}).Apply(context.Background(), claim)
	require.True(t, handled)
	require.ErrorContains(t, err, "read-back")
	reader := fixedInvoiceAskReader{ask: askreader.Ask{ID: "ask", State: "answered", Answer: "approve", AnsweredBy: "someone", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture", AnsweredAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: "different-event", Correlation: claim.Correlation}}
	_, err = (InvoiceApprovalEffect{DB: db, Reader: reader}).Apply(context.Background(), claim)
	require.ErrorContains(t, err, "verify invoice decision")
	var state string
	require.NoError(t, db.QueryRow(`SELECT state FROM invoice_reconciliation_receipts WHERE external_id=?`, id).Scan(&state))
	require.Equal(t, "awaiting_confirmation", state)
}

func TestInvoiceReceiptRequiresVerifiedHumanAndRetriesUnavailableHubRead(t *testing.T) {
	db := invoiceFixtureDB(t)
	r := InvoiceReconciler{DB: db, Destination: fixtureInvoiceDestination}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-02", LineID: "fixture-line-02", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY", OccurredAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), RangeFrom: fixtureInvoiceRangeFrom, RangeTo: fixtureInvoiceRangeTo}
	id, err := r.RecordFixtureOrSource(context.Background(), line)
	require.NoError(t, err)
	var digest, reservation, quote, book, account string
	require.NoError(t, db.QueryRow(`SELECT invoice_digest,reservation_id,quote_id,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, id).Scan(&digest, &reservation, &quote, &book, &account))
	const requester = "fixture-requester-02"
	_, err = db.Exec(`UPDATE invoice_reconciliation_receipts SET confirmation_event_id=? WHERE external_id=?`, requester, id)
	require.NoError(t, err)
	corr := map[string]string{"invoice_external_id": id, "invoice_digest": digest, "reservation_id": reservation, "quote_id": quote, "ledger_book_id": book, "ledger_account_id": account}
	task := askreader.Ask{ID: "fixture-ask-02", State: "answered", Answer: "approve", AnsweredBy: "fixture-human", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture-human-provider", AnsweredAt: "2026-11-23T13:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: requester, Correlation: corr}
	claim := operatordecision.DecisionResolution{AskID: task.ID, State: task.State, Answer: task.Answer, AnsweredBy: task.AnsweredBy, AnsweredByIdentityKind: task.AnsweredByIdentityKind, AnsweredByIdentitySource: task.AnsweredByIdentitySource, AnsweredAt: task.AnsweredAt, Source: task.Source, SourceEventType: task.SourceEventType, RequesterRef: requester, Correlation: corr}
	effect := InvoiceApprovalEffect{DB: db, Reader: fixedInvoiceAskReader{ask: task}}
	nonHuman := task
	nonHuman.AnsweredByIdentityKind = "service"
	_, err = (InvoiceApprovalEffect{DB: db, Reader: fixedInvoiceAskReader{ask: nonHuman}}).Apply(context.Background(), claim)
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM invoice_confirmation_receipts WHERE external_id=?`, id).Scan(&count))
	require.Zero(t, count, "a non-human answer cannot become a durable receipt")

	_, err = (InvoiceApprovalEffect{DB: db, Reader: failingInvoiceAskReader{err: askreader.ErrHubUnavailable}}).Apply(context.Background(), claim)
	require.Error(t, err, "an unavailable read-back must leave the receipt pending for retry")
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM invoice_confirmation_receipts WHERE external_id=?`, id).Scan(&count))
	require.Zero(t, count)

	handled, err := effect.Apply(context.Background(), claim)
	require.NoError(t, err)
	require.True(t, handled)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM invoice_confirmation_receipts WHERE external_id=? AND answered_by='fixture-human'`, id).Scan(&count))
	require.Equal(t, 1, count)
}

type failingInvoiceAskReader struct{ err error }

func (r failingInvoiceAskReader) ReadAsk(context.Context, string) (askreader.Ask, error) {
	return askreader.Ask{}, r.err
}

func TestInvoiceApprovalPublisherFailureLeavesReceiptPendingAndStableEventForRetry(t *testing.T) {
	db := invoiceFixtureDB(t)
	r := InvoiceReconciler{DB: db, Destination: fixtureInvoiceDestination, Now: func() time.Time { return time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC) }}
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice-01", LineID: "fixture-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY", OccurredAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), RangeFrom: fixtureInvoiceRangeFrom, RangeTo: fixtureInvoiceRangeTo}
	id, err := r.RecordFixtureOrSource(context.Background(), line)
	require.NoError(t, err)
	failing := &invoiceDecisionPublisher{err: fmt.Errorf("fixture event transport unavailable")}
	first, err := r.RequestInvoiceApproval(context.Background(), id, failing)
	require.Error(t, err)
	require.NotEmpty(t, first)
	failing.err = nil
	second, err := r.RequestInvoiceApproval(context.Background(), id, failing)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, operatordecision.EventID(failing.events[0]), operatordecision.EventID(failing.events[1]))
}
