package reconcile

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

var (
	ErrInvoiceUnmatched     = errors.New("invoice line does not match an approved reservation")
	ErrInvoiceConfirmation  = errors.New("invoice line requires an explicit operator confirmation")
	ErrInvoiceIdentityReuse = errors.New("invoice source identity was reused with different accounting facts")
)

// InvoiceLine is a normalized provider fact. InvoiceID and LineID must come
// from the provider's stable source identity; callers must never synthesize a
// live source ID from a timestamp or local instance ID. Fixture provenance is
// explicit and is never inferred from an ID prefix.
type InvoiceLine struct {
	Provider, InvoiceID, LineID string
	ProviderInstanceID          string
	AmountMinor                 int64 // signed ledger amount; expense lines are negative
	Currency, Provenance        string
	OccurredAt                  time.Time
	RangeFrom, RangeTo          time.Time
}

// InvoiceSourceFacts is the strict boundary for turning a provider observation
// into a matchable invoice line. Values are deliberately supplied by the
// source adapter; this function never derives stable IDs or currency.
type InvoiceSourceFacts struct {
	Provider, InvoiceID, LineID, ProviderInstanceID string
	AmountMinor                                     int64
	Currency, Provenance                            string
	OccurredAt, RangeFrom, RangeTo                  time.Time
}

func NormalizeInvoiceSourceLine(f InvoiceSourceFacts) (InvoiceLine, error) {
	if strings.TrimSpace(f.Provider) == "" || strings.TrimSpace(f.InvoiceID) == "" || strings.TrimSpace(f.LineID) == "" || strings.TrimSpace(f.ProviderInstanceID) == "" {
		return InvoiceLine{}, fmt.Errorf("provider, stable invoice and line IDs, and provider instance ID are required")
	}
	if f.AmountMinor == 0 || strings.TrimSpace(f.Currency) == "" || strings.TrimSpace(f.Provenance) == "" {
		return InvoiceLine{}, fmt.Errorf("nonzero minor amount, currency, and provenance are required")
	}
	if f.OccurredAt.IsZero() || f.RangeFrom.IsZero() || f.RangeTo.IsZero() || !f.RangeFrom.Before(f.RangeTo) {
		return InvoiceLine{}, fmt.Errorf("occurrence and valid source date range are required")
	}
	return InvoiceLine{Provider: strings.TrimSpace(f.Provider), InvoiceID: strings.TrimSpace(f.InvoiceID), LineID: strings.TrimSpace(f.LineID), ProviderInstanceID: strings.TrimSpace(f.ProviderInstanceID), AmountMinor: f.AmountMinor, Currency: strings.ToUpper(strings.TrimSpace(f.Currency)), Provenance: strings.TrimSpace(f.Provenance), OccurredAt: f.OccurredAt.UTC(), RangeFrom: f.RangeFrom.UTC(), RangeTo: f.RangeTo.UTC()}, nil
}

func (l InvoiceLine) ExternalID() string {
	return strings.Join([]string{"compute-manager", l.Provider, "invoice", l.InvoiceID, "line", l.LineID}, ":")
}

func invoiceFactsDigest(line InvoiceLine, instanceID, reservationID, quoteID string, destination InvoiceDestination) string {
	facts := struct {
		Provider, InvoiceID, LineID, ProviderInstanceID                                                     string
		AmountMinor                                                                                         int64
		Currency, Provenance, OccurredAt, InstanceID, ReservationID, QuoteID, LedgerBookID, LedgerAccountID string
	}{line.Provider, line.InvoiceID, line.LineID, line.ProviderInstanceID, line.AmountMinor, strings.ToUpper(strings.TrimSpace(line.Currency)), line.Provenance, line.OccurredAt.UTC().Format(time.RFC3339Nano), instanceID, reservationID, quoteID, destination.BookID, destination.AccountID}
	encoded, _ := json.Marshal(facts)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// InvoiceDestination is the explicitly selected Money Ledger posting target.
// It is persisted and included in the approval digest so a confirmation cannot
// be replayed against a different book or account.
type InvoiceDestination struct{ BookID, AccountID string }

func (d InvoiceDestination) Validate() error {
	if strings.TrimSpace(d.BookID) == "" || strings.TrimSpace(d.AccountID) == "" {
		return fmt.Errorf("Money Ledger book and account must be explicitly configured")
	}
	return nil
}

type InvoicePosting struct {
	Line        InvoiceLine
	ExternalID  string
	Destination InvoiceDestination
}

// InvoiceLedger is the narrow supported event-ingest/reversal seam. A
// production implementation must call Money Ledger's generated Connect API;
// tests may use a fixture implementation but cannot claim a ledger post.
type InvoiceLedger interface {
	PostInvoice(context.Context, InvoicePosting) (postingID string, err error)
	ReverseInvoice(context.Context, string, string) (reversalPostingID string, err error)
}

type InvoiceReconciler struct {
	DB          *sql.DB
	Ledger      InvoiceLedger
	Destination InvoiceDestination
	Now         func() time.Time
}

const InvoiceApprovalEventType = "compute-manager.provider-invoice-post.decision_requested.v1"

// RequestInvoiceApproval emits a durable, deterministic Notification Hub ask.
// The invoice receipt itself is the pending record; its digest is carried in
// the Hub correlation so the verified answer cannot authorize changed facts.
func (r InvoiceReconciler) RequestInvoiceApproval(ctx context.Context, externalID string, publisher operatordecision.Publisher) (string, error) {
	if r.DB == nil || publisher == nil {
		return "", fmt.Errorf("invoice confirmation store and Notification Hub publisher are required")
	}
	var line InvoiceLine
	var instanceID, reservationID, quoteID, state, digest, createdAt, reverses, bookID, accountID string
	var occurred, storedEventID string
	err := r.DB.QueryRowContext(ctx, `SELECT provider,invoice_id,line_id,provider_instance_id,amount_minor,currency,provenance,occurred_at,instance_id,reservation_id,quote_id,state,invoice_digest,created_at,reverses_external_id,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).
		Scan(&line.Provider, &line.InvoiceID, &line.LineID, &line.ProviderInstanceID, &line.AmountMinor, &line.Currency, &line.Provenance, &occurred, &instanceID, &reservationID, &quoteID, &state, &digest, &createdAt, &reverses, &bookID, &accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceUnmatched
	}
	if err != nil {
		return "", err
	}
	if state != "awaiting_confirmation" && state != "correction_pending" {
		return "", ErrInvoiceConfirmation
	}
	line.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
	if err != nil {
		return "", err
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return "", err
	}
	facts := operatordecision.Facts{
		Subject:    "Provider invoice needs approval",
		Question:   fmt.Sprintf("Post %s %d minor units for %s invoice %s line %s?", strings.ToUpper(line.Currency), line.AmountMinor, line.Provider, line.InvoiceID, line.LineID),
		Options:    []operatordecision.Option{{Key: "approve", Label: "Approve this invoice post"}, {Key: "decline", Label: "Do not post"}},
		Reversible: false, Urgency: "high",
		BigPicture:       "This records one provider invoice line against the approved Compute Manager reservation. No provider is contacted and no infrastructure is changed.",
		TechnicalDetails: fmt.Sprintf("Source: %s\nInvoice: %s\nLine: %s\nProvider instance: %s\nCompute Manager instance: %s\nReservation: %s\nSpend quote: %s\nAmount: %d %s minor units\nLedger book: %s\nLedger account: %s\nInvoice digest: %s\nProvenance: %s\nCorrection of: %s", line.Provider, line.InvoiceID, line.LineID, line.ProviderInstanceID, instanceID, reservationID, quoteID, line.AmountMinor, strings.ToUpper(line.Currency), bookID, accountID, digest, line.Provenance, reverses),
		Correlation:      map[string]string{"invoice_external_id": externalID, "invoice_digest": digest, "reservation_id": reservationID, "quote_id": quoteID, "ledger_book_id": bookID, "ledger_account_id": accountID},
	}
	if err := facts.Validate(); err != nil {
		return "", fmt.Errorf("invoice confirmation facts invalid: %w", err)
	}
	event := operatordecision.BuildEventTyped("compute-manager", InvoiceApprovalEventType, created, facts)
	eventID := operatordecision.EventID(event)
	if err = r.DB.QueryRowContext(ctx, `SELECT confirmation_event_id FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).Scan(&storedEventID); err != nil {
		return "", err
	}
	if storedEventID != "" && storedEventID != eventID {
		return "", ErrInvoiceIdentityReuse
	}
	result, err := r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET confirmation_event_id=?,updated_at=? WHERE external_id=? AND invoice_digest=? AND state IN ('awaiting_confirmation','correction_pending') AND (confirmation_event_id='' OR confirmation_event_id=?)`, eventID, r.now(), externalID, digest, eventID)
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if changed == 0 {
		return "", ErrInvoiceConfirmation
	}
	if err := publisher.PublishDomainEvent(ctx, event); err != nil {
		return eventID, fmt.Errorf("publish invoice phone confirmation: %w", err)
	}
	return eventID, nil
}

// InvoiceApprovalEffect is attached only to Compute Manager's verified
// operatordecision webhook with an ask-reader configured. Apply consumes the
// Hub read-back (not the signed event claim) and persists a typed human receipt.
type InvoiceApprovalEffect struct {
	DB     *sql.DB
	Now    func() time.Time
	Reader operatordecision.AskReader
}

func (e InvoiceApprovalEffect) Handles(eventType string) bool {
	return eventType == InvoiceApprovalEventType
}

func (e InvoiceApprovalEffect) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	if !e.Handles(resolution.SourceEventType) || resolution.Source != "compute-manager" {
		return false, nil
	}
	if e.DB == nil || e.Reader == nil || resolution.AskID == "" {
		return true, fmt.Errorf("invoice approval requires durable receipts and Notification Hub read-back")
	}
	hubAsk, err := e.Reader.ReadAsk(ctx, resolution.AskID)
	if err != nil {
		return true, fmt.Errorf("read invoice decision from Notification Hub: %w", err)
	}
	verified, err := operatordecision.VerifiedResolution(resolution, hubAsk)
	if err != nil {
		return true, fmt.Errorf("verify invoice decision from Notification Hub: %w", err)
	}
	resolution = verified
	externalID := resolution.Correlation["invoice_external_id"]
	digest := resolution.Correlation["invoice_digest"]
	if externalID == "" || digest == "" {
		return true, fmt.Errorf("verified invoice decision is missing its correlation")
	}
	approved := resolution.State == operatordecision.StateAnswered && resolution.Answer == "approve" && !resolution.ByDefault && resolution.AnsweredByIdentityKind == "verified_human" && strings.TrimSpace(resolution.AnsweredBy) != "" && strings.TrimSpace(resolution.AnsweredByIdentitySource) != ""
	declined := resolution.State == operatordecision.StateAnswered && resolution.Answer == "decline" && !resolution.ByDefault && resolution.AnsweredByIdentityKind == "verified_human" && strings.TrimSpace(resolution.AnsweredBy) != "" && strings.TrimSpace(resolution.AnsweredByIdentitySource) != ""
	if !approved && !declined {
		return true, nil
	}
	var state, storedDigest, confirmationEventID, reservationID, quoteID, bookID, accountID string
	resolved := resolution.AnsweredAt
	if resolved == "" {
		resolved = resolution.ResolvedAt
	}
	parsed, err := time.Parse(time.RFC3339Nano, resolved)
	if err != nil {
		return true, fmt.Errorf("verified invoice decision has invalid answer time")
	}
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return true, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `SELECT state,invoice_digest,confirmation_event_id,reservation_id,quote_id,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).Scan(&state, &storedDigest, &confirmationEventID, &reservationID, &quoteID, &bookID, &accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return true, ErrInvoiceUnmatched
	}
	if err != nil {
		return true, err
	}
	if storedDigest != digest {
		return true, ErrInvoiceIdentityReuse
	}
	if confirmationEventID == "" || confirmationEventID != resolution.RequesterRef || resolution.Correlation["reservation_id"] != reservationID || resolution.Correlation["quote_id"] != quoteID || resolution.Correlation["ledger_book_id"] != bookID || resolution.Correlation["ledger_account_id"] != accountID {
		return true, ErrInvoiceIdentityReuse
	}
	// An ask and an invoice may each be confirmed only once. This immutable
	// binding survives process restart and makes webhook redelivery a no-op only
	// when every verified-human and invoice identity field is identical.
	var oldExternal, oldRequester, oldDigest, oldAnswer, oldActor, oldSource string
	err = tx.QueryRowContext(ctx, `SELECT external_id,requester_ref,invoice_digest,answer,answered_by,identity_source FROM invoice_confirmation_receipts WHERE ask_id=?`, resolution.AskID).Scan(&oldExternal, &oldRequester, &oldDigest, &oldAnswer, &oldActor, &oldSource)
	if err == nil {
		if oldExternal == externalID && oldRequester == resolution.RequesterRef && oldDigest == digest && oldAnswer == resolution.Answer && oldActor == resolution.AnsweredBy && oldSource == resolution.AnsweredByIdentitySource {
			return true, nil
		}
		return true, ErrInvoiceConfirmation
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return true, err
	}
	var anotherAsk string
	err = tx.QueryRowContext(ctx, `SELECT ask_id FROM invoice_confirmation_receipts WHERE external_id=?`, externalID).Scan(&anotherAsk)
	if err == nil {
		return true, ErrInvoiceConfirmation
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return true, err
	}
	newState := "declined"
	if approved {
		newState = "confirmed"
		if state == "correction_pending" {
			newState = "correction_confirmed"
		}
	}
	allowed := state == "awaiting_confirmation" || state == "correction_pending"
	if !allowed {
		return true, ErrInvoiceConfirmation
	}
	verifiedAt := parsed.UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_confirmation_receipts(ask_id,external_id,requester_ref,invoice_digest,answer,answered_by,identity_source,verified_at,recorded_at) VALUES(?,?,?,?,?,?,?,?,?)`, resolution.AskID, externalID, resolution.RequesterRef, digest, resolution.Answer, resolution.AnsweredBy, resolution.AnsweredByIdentitySource, verifiedAt, e.now()); err != nil {
		return true, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state=?,confirmation_ask_id=?,confirmation_actor=?,confirmation_identity_source=?,confirmation_digest=?,confirmed_at=?,updated_at=? WHERE external_id=? AND invoice_digest=? AND state=?`, newState, resolution.AskID, resolution.AnsweredBy, resolution.AnsweredByIdentitySource, digest, verifiedAt, e.now(), externalID, digest, state)
	if err != nil {
		return true, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return true, err
	}
	if changed != 1 {
		return true, ErrInvoiceConfirmation
	}
	if err = tx.Commit(); err != nil {
		return true, err
	}
	return true, nil
}

// RecordFixtureOrSource persists a normalized line only when it resolves to
// the exact local instance, reservation, approved admission, and approved quote.
// It deliberately does not post: a separate explicit confirmation is required.
func (r InvoiceReconciler) RecordFixtureOrSource(ctx context.Context, line InvoiceLine) (string, error) {
	if r.DB == nil {
		return "", fmt.Errorf("invoice reconciliation database is unavailable")
	}
	if err := r.Destination.Validate(); err != nil {
		return "", err
	}
	if line.Provider == "" || line.InvoiceID == "" || line.LineID == "" || line.ProviderInstanceID == "" || line.Currency == "" || line.Provenance == "" || line.AmountMinor == 0 || line.OccurredAt.IsZero() {
		return "", fmt.Errorf("invoice line requires provider, stable invoice and line IDs, instance, amount, currency, provenance and occurrence time")
	}
	externalID := line.ExternalID()
	now := r.now()
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var instanceID, reservationID, quoteID, quoteCurrency, quoteState, decision string
	err = tx.QueryRowContext(ctx, `SELECT i.id,r.id,q.id,q.currency,q.state,a.decision
		FROM instances i
		JOIN reservations r ON r.id=i.reservation_id AND r.instance_id=i.id
		JOIN instance_intents n ON n.id=r.intent_id
		JOIN spend_quotes q ON q.idempotency_key=n.idempotency_key
		JOIN spend_admissions a ON a.quote_id=q.id
		WHERE i.provider=? AND q.provider=i.provider AND i.provider_instance_id=?`, line.Provider, line.ProviderInstanceID).
		Scan(&instanceID, &reservationID, &quoteID, &quoteCurrency, &quoteState, &decision)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceUnmatched
	}
	if err != nil {
		return "", err
	}
	if quoteState != "approved" || decision != "approved" || !strings.EqualFold(quoteCurrency, line.Currency) {
		return "", ErrInvoiceUnmatched
	}
	digest := invoiceFactsDigest(line, instanceID, reservationID, quoteID, r.Destination)
	_, err = tx.ExecContext(ctx, `INSERT INTO invoice_reconciliation_receipts
		(external_id,provider,invoice_id,line_id,provider_instance_id,instance_id,reservation_id,quote_id,ledger_book_id,ledger_account_id,amount_minor,currency,occurred_at,provenance,state,invoice_digest,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(external_id) DO NOTHING`,
		externalID, line.Provider, line.InvoiceID, line.LineID, line.ProviderInstanceID, instanceID, reservationID, quoteID, r.Destination.BookID, r.Destination.AccountID, line.AmountMinor, strings.ToUpper(line.Currency), line.OccurredAt.UTC().Format(time.RFC3339Nano), line.Provenance, "awaiting_confirmation", digest, now, now)
	if err != nil {
		return "", err
	}
	var provider, invoiceID, lineID, instance, reservation, quote, currency, provenance, storedDigest string
	var amount int64
	err = tx.QueryRowContext(ctx, `SELECT provider,invoice_id,line_id,instance_id,reservation_id,quote_id,amount_minor,currency,provenance,invoice_digest FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).
		Scan(&provider, &invoiceID, &lineID, &instance, &reservation, &quote, &amount, &currency, &provenance, &storedDigest)
	if err != nil {
		return "", err
	}
	if provider != line.Provider || invoiceID != line.InvoiceID || lineID != line.LineID || instance != instanceID || reservation != reservationID || quote != quoteID || amount != line.AmountMinor || !strings.EqualFold(currency, line.Currency) || provenance != line.Provenance || storedDigest != digest {
		return "", ErrInvoiceIdentityReuse
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return externalID, nil
}

// ConfirmAndPost records an explicit operator decision before calling the
// ledger. The ledger's idempotent stable external ID handles a lost response.
// Real lines must also carry a verified phone-confirmation actor; fixtures are
// only allowed through test ledger implementations.
func (r InvoiceReconciler) ConfirmAndPost(ctx context.Context, externalID string) (string, error) {
	if r.DB == nil || r.Ledger == nil {
		return "", fmt.Errorf("invoice ledger integration is unavailable")
	}
	var line InvoiceLine
	var state, provenance, confirmationActor, confirmationDigest, invoiceDigest, askID, confirmedAt, existingPosting, bookID, accountID string
	var occurred string
	err := r.DB.QueryRowContext(ctx, `SELECT provider,invoice_id,line_id,provider_instance_id,amount_minor,currency,provenance,occurred_at,state,confirmation_actor,confirmation_digest,invoice_digest,confirmation_ask_id,confirmed_at,ledger_posting_id,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, externalID).
		Scan(&line.Provider, &line.InvoiceID, &line.LineID, &line.ProviderInstanceID, &line.AmountMinor, &line.Currency, &provenance, &occurred, &state, &confirmationActor, &confirmationDigest, &invoiceDigest, &askID, &confirmedAt, &existingPosting, &bookID, &accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceUnmatched
	}
	if err != nil {
		return "", err
	}
	line.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
	if err != nil {
		return "", err
	}
	line.Provenance = provenance
	if state == "posted" && existingPosting != "" {
		return existingPosting, nil
	}
	if state != "confirmed" && state != "posting" {
		return "", ErrInvoiceConfirmation
	}
	if confirmationActor == "" || askID == "" || confirmedAt == "" || confirmationDigest != invoiceDigest {
		return "", ErrInvoiceConfirmation
	}
	if state == "confirmed" {
		res, err := r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='posting',updated_at=? WHERE external_id=? AND state='confirmed' AND confirmation_digest=invoice_digest`, r.now(), externalID)
		if err != nil {
			return "", err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return "", ErrInvoiceConfirmation
		}
	}
	postingID, err := r.Ledger.PostInvoice(ctx, InvoicePosting{Line: line, ExternalID: externalID, Destination: InvoiceDestination{BookID: bookID, AccountID: accountID}})
	if err != nil {
		return "", err
	}
	if postingID == "" {
		return "", fmt.Errorf("ledger returned an empty posting identity")
	}
	_, err = r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='posted',ledger_posting_id=?,updated_at=? WHERE external_id=? AND state='posting'`, postingID, r.now(), externalID)
	if err != nil {
		return "", err
	}
	return postingID, nil
}

// RecordCorrection links a provider-issued replacement/adjustment line to an
// already-posted original. The original accounting event stays immutable.
func (r InvoiceReconciler) RecordCorrection(ctx context.Context, originalExternalID string, corrected InvoiceLine) (string, error) {
	var originalState, originalInstance, originalPosting string
	err := r.DB.QueryRowContext(ctx, `SELECT state,instance_id,ledger_posting_id FROM invoice_reconciliation_receipts WHERE external_id=?`, originalExternalID).Scan(&originalState, &originalInstance, &originalPosting)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceUnmatched
	}
	if err != nil {
		return "", err
	}
	if originalState != "posted" || originalPosting == "" {
		return "", ErrInvoiceConfirmation
	}
	correctedExternalID, err := r.RecordFixtureOrSource(ctx, corrected)
	if err != nil {
		return "", err
	}
	var correctedInstance string
	if err := r.DB.QueryRowContext(ctx, `SELECT instance_id FROM invoice_reconciliation_receipts WHERE external_id=?`, correctedExternalID).Scan(&correctedInstance); err != nil {
		return "", err
	}
	if correctedInstance != originalInstance {
		return "", ErrInvoiceUnmatched
	}
	result, err := r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='correction_pending',reverses_external_id=?,updated_at=? WHERE external_id=? AND state='awaiting_confirmation'`, originalExternalID, r.now(), correctedExternalID)
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if changed == 0 {
		var state, linked string
		if err := r.DB.QueryRowContext(ctx, `SELECT state,reverses_external_id FROM invoice_reconciliation_receipts WHERE external_id=?`, correctedExternalID).Scan(&state, &linked); err != nil {
			return "", err
		}
		if linked != originalExternalID || (state != "correction_pending" && state != "reversal_posted" && state != "posted") {
			return "", ErrInvoiceIdentityReuse
		}
	}
	return correctedExternalID, nil
}

// ConfirmCorrectionAndPost applies one explicit decision to an append-only
// correction: reverse the original posting, then ingest the provider's new
// source line. Both ledger operations use stable identities and are safe to
// retry after a lost response.
func (r InvoiceReconciler) ConfirmCorrectionAndPost(ctx context.Context, correctionExternalID string) (string, error) {
	if r.DB == nil || r.Ledger == nil {
		return "", fmt.Errorf("invoice ledger integration is unavailable")
	}
	var originalExternalID, correctionState, originalPosting, reversalPosting, storedPosting string
	err := r.DB.QueryRowContext(ctx, `SELECT c.reverses_external_id,c.state,o.ledger_posting_id,c.reversal_posting_id,c.ledger_posting_id FROM invoice_reconciliation_receipts c JOIN invoice_reconciliation_receipts o ON o.external_id=c.reverses_external_id WHERE c.external_id=?`, correctionExternalID).Scan(&originalExternalID, &correctionState, &originalPosting, &reversalPosting, &storedPosting)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceUnmatched
	}
	if err != nil {
		return "", err
	}
	if correctionState == "posted" && storedPosting != "" {
		_, err := r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='reversed',updated_at=? WHERE external_id=? AND state='posted'`, r.now(), originalExternalID)
		if err != nil {
			return "", err
		}
		return storedPosting, nil
	}
	if correctionState != "correction_confirmed" && correctionState != "reversal_posted" {
		return "", ErrInvoiceConfirmation
	}
	var correctionDigest, approvalDigest, confirmationActor, askID, confirmedAt, bookID, accountID string
	if err := r.DB.QueryRowContext(ctx, `SELECT invoice_digest,confirmation_digest,confirmation_actor,confirmation_ask_id,confirmed_at,ledger_book_id,ledger_account_id FROM invoice_reconciliation_receipts WHERE external_id=?`, correctionExternalID).Scan(&correctionDigest, &approvalDigest, &confirmationActor, &askID, &confirmedAt, &bookID, &accountID); err != nil {
		return "", err
	}
	if correctionDigest == "" || correctionDigest != approvalDigest || confirmationActor == "" || askID == "" || confirmedAt == "" {
		return "", ErrInvoiceConfirmation
	}
	if reversalPosting == "" {
		if originalPosting == "" {
			return "", ErrInvoiceUnmatched
		}
		reversalPosting, err = r.Ledger.ReverseInvoice(ctx, originalPosting, "Compute Manager provider invoice correction: "+correctionExternalID)
		if err != nil {
			return "", err
		}
		if reversalPosting == "" {
			return "", fmt.Errorf("ledger returned an empty reversal identity")
		}
		if _, err = r.DB.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='reversal_posted',reversal_posting_id=?,updated_at=? WHERE external_id=? AND state='correction_confirmed'`, reversalPosting, r.now(), correctionExternalID); err != nil {
			return "", err
		}
	}
	var line InvoiceLine
	var occurred string
	err = r.DB.QueryRowContext(ctx, `SELECT provider,invoice_id,line_id,provider_instance_id,amount_minor,currency,provenance,occurred_at FROM invoice_reconciliation_receipts WHERE external_id=?`, correctionExternalID).Scan(&line.Provider, &line.InvoiceID, &line.LineID, &line.ProviderInstanceID, &line.AmountMinor, &line.Currency, &line.Provenance, &occurred)
	if err != nil {
		return "", err
	}
	line.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
	if err != nil {
		return "", err
	}
	newPosting, err := r.Ledger.PostInvoice(ctx, InvoicePosting{Line: line, ExternalID: correctionExternalID, Destination: InvoiceDestination{BookID: bookID, AccountID: accountID}})
	if err != nil {
		return "", err
	}
	if newPosting == "" {
		return "", fmt.Errorf("ledger returned an empty corrected posting identity")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='posted',ledger_posting_id=?,updated_at=? WHERE external_id=? AND state='reversal_posted'`, newPosting, r.now(), correctionExternalID); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE invoice_reconciliation_receipts SET state='reversed',updated_at=? WHERE external_id=? AND state='posted'`, r.now(), originalExternalID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return newPosting, nil
}

func (r InvoiceReconciler) now() string {
	if r.Now != nil {
		return r.Now().UTC().Format(time.RFC3339Nano)
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (e InvoiceApprovalEffect) now() string {
	if e.Now != nil {
		return e.Now().UTC().Format(time.RFC3339Nano)
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}
