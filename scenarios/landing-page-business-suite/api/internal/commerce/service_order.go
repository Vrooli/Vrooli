package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	serviceOrderSandboxMode      = "sandbox"
	ServiceOrderDoorBrowser      = "browser"
	ServiceOrderDoorBrowserAgent = "browsing_agent"
	ServiceOrderDoorMCP          = "mcp"
	ServiceOrderDoorAgentRun     = "agent_manager_x402"
	ServiceOrderCreated          = "created"
	ServiceOrderAuthorized       = "authorized"
	ServiceOrderVerifyFailed     = "verification_failed"
	ServiceOrderVerified         = "verified"
	ServiceOrderCapturePending   = "capture_pending"
	ServiceOrderCaptured         = "captured"
	ServiceOrderVoided           = "voided"
	ServiceOrderDeclined         = "declined"
	ServiceOrderLedgerPending    = "ledger_pending"
	ServiceOrderReconciled       = "reconciled"
)

var ErrServiceOrderIdempotencyConflict = errors.New("service order idempotency key was reused with changed offer terms")

type ServiceOrderQuote struct {
	OfferID      string
	OfferVersion int
	AmountMinor  int64
	Currency     string
	X402PriceID  string
}

type ServiceOrderOfferRequest struct {
	OfferID           string
	OfferVersion      int
	Door              string
	AgentManagerRunID string
	IdempotencyKey    string
	JobInput          json.RawMessage
	PaymentSignature  string
}

// ServiceOrderOfferSource resolves the authoritative staged offer record.
// Callers of BeginFromOffer select only the offer identity and cannot choose
// a different amount, currency, version or Treasury price ID.
type ServiceOrderOfferSource interface {
	ResolveServiceOrderQuote(context.Context, string, int) (ServiceOrderQuote, error)
}

type BeginServiceOrderRequest struct {
	Quote             ServiceOrderQuote
	Door              string
	AgentManagerRunID string
	IdempotencyKey    string
	JobInput          json.RawMessage
	PaymentSignature  string
}

type ServiceOrder struct {
	ID                 string          `json:"order_id"`
	JobID              string          `json:"job_id"`
	OfferID            string          `json:"offer_id"`
	OfferVersion       int             `json:"offer_version"`
	JobInput           json.RawMessage `json:"job_input"`
	AmountMinor        int64           `json:"amount_minor"`
	Currency           string          `json:"currency"`
	Door               string          `json:"door"`
	AgentManagerRunID  string          `json:"agent_manager_run_id,omitempty"`
	Mode               string          `json:"mode"`
	State              string          `json:"state"`
	AuthorizationID    string          `json:"authorization_id,omitempty"`
	VerificationID     string          `json:"verification_id,omitempty"`
	CaptureID          string          `json:"capture_id,omitempty"`
	LedgerPostingID    string          `json:"ledger_posting_id,omitempty"`
	LedgerReceiptID    string          `json:"ledger_receipt_id,omitempty"`
	TreasuryExternalID string          `json:"treasury_external_id,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type ServiceOrderAuthorizationRequest struct {
	OrderID, JobID, IdempotencyKey                       string
	Door, OfferID, ResourceID, PriceID, PaymentSignature string
	OfferVersion                                         int
	AmountMinor                                          int64
	Currency                                             string
}
type ServiceOrderAuthorizationReceipt struct {
	EventID, AuthorizationID, Payer, Network string
	Sandbox, Approved                        bool
}
type ServiceOrderCaptureRequest struct {
	OrderID, JobID, AuthorizationID, IdempotencyKey string
	Door, ResourceID, PriceID, Payer, Network       string
	AmountMinor                                     int64
	Currency                                        string
}
type ServiceOrderCaptureReceipt struct {
	EventID, CaptureID, TreasuryExternalID string
	Sandbox, Captured                      bool
}
type (
	ServiceOrderVoidRequest struct{ OrderID, JobID, ResourceID, PriceID, Payer, Network, AuthorizationID, IdempotencyKey, Door string }
	ServiceOrderVoidReceipt struct {
		EventID, VoidID string
		Sandbox, Voided bool
	}
)

type ServiceOrderVerification struct {
	ReceiptID string
	Passed    bool
}

// ServiceOrderSandboxProvider is a payment edge for the service-order runtime.
// Its receipts must explicitly identify test mode. Implementations must honor
// the supplied idempotency key; this package has no live provider adapter.
type ServiceOrderSandboxProvider interface {
	Authorize(context.Context, ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error)
	Capture(context.Context, ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error)
	Void(context.Context, ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error)
}

type ServiceOrderVerifier interface {
	Verify(context.Context, ServiceOrder) (ServiceOrderVerification, error)
}

type ServiceOrderLedgerReceipt struct {
	PostingID, ReceiptID, ExternalID string
	Duplicate                        bool
}
type ServiceOrderLedgerWriter interface {
	IngestSandboxCapture(context.Context, *sharedpb.MoneyEvent) (ServiceOrderLedgerReceipt, error)
}

// ServiceOrderDB is the narrow routed SQL surface used by durable orders.
// database.RoutedDB implements it so requests stay pinned to the same leased
// test database across create, verify, settle and receipt reconciliation.
type ServiceOrderDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type TreasuryX402ReceiptLookup interface {
	ReadAccepted(context.Context, string) (TreasuryX402Receipt, error)
}

// ServiceOrderLedgerTarget explicitly names an isolated test Money Ledger
// fixture or loopback test service. No other target can receive test captures.
type ServiceOrderLedgerTarget struct {
	FixtureID, AdapterID, BookID, RevenueAccountID string
}

type ServiceOrderSandboxOptions struct {
	DB               ServiceOrderDB
	Provider         ServiceOrderSandboxProvider
	Verifier         ServiceOrderVerifier
	Ledger           ServiceOrderLedgerWriter
	TreasuryReceipts TreasuryX402ReceiptLookup
	LedgerTarget     ServiceOrderLedgerTarget
	Now              func() time.Time
}

type ServiceOrderSandbox struct {
	db               ServiceOrderDB
	provider         ServiceOrderSandboxProvider
	verifier         ServiceOrderVerifier
	ledger           ServiceOrderLedgerWriter
	treasuryReceipts TreasuryX402ReceiptLookup
	target           ServiceOrderLedgerTarget
	now              func() time.Time
}

func NewServiceOrderSandbox(options ServiceOrderSandboxOptions) (*ServiceOrderSandbox, error) {
	if options.DB == nil || options.Provider == nil || options.Verifier == nil {
		return nil, errors.New("service order sandbox requires database, sandbox provider, and verifier")
	}
	target := options.LedgerTarget
	testTarget := strings.HasPrefix(target.FixtureID, "money-ledger-test-sqlite:") || strings.HasPrefix(target.FixtureID, "money-ledger-test-service:")
	ledgerConfigured := options.Ledger != nil && testTarget && target.AdapterID != "" && target.BookID != "" && target.RevenueAccountID != ""
	if !ledgerConfigured && options.TreasuryReceipts == nil {
		return nil, errors.New("service order requires either an isolated Money Ledger fixture writer or Treasury accepted-receipt lookup")
	}
	if options.Ledger != nil && !ledgerConfigured {
		return nil, errors.New("service order Money Ledger writer requires an isolated test target and complete routing")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &ServiceOrderSandbox{db: options.DB, provider: options.Provider, verifier: options.Verifier, ledger: options.Ledger, treasuryReceipts: options.TreasuryReceipts, target: target, now: options.Now}, nil
}

func (s *ServiceOrderSandbox) Begin(ctx context.Context, request BeginServiceOrderRequest) (*ServiceOrder, error) {
	if err := validateServiceOrderRequest(request); err != nil {
		return nil, err
	}
	if request.Door == ServiceOrderDoorAgentRun && s.treasuryReceipts == nil {
		return nil, errors.New("Agent Manager x402 orders require Treasury accepted-receipt readback")
	}
	if request.Door != ServiceOrderDoorAgentRun && s.ledger == nil {
		return nil, errors.New("non-x402 orders require an isolated Money Ledger writer")
	}
	digest := serviceOrderTermsDigest(request)
	jobInput := normalizedJobInput(request.JobInput)
	id, jobID := uuid.NewString(), uuid.NewString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO service_orders(id,job_id,idempotency_key,terms_sha256,offer_id,offer_version,job_payload,amount_minor,currency,door,agent_manager_run_id,x402_price_id,mode,state,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'sandbox','created',$13,$13) ON CONFLICT(idempotency_key) DO NOTHING`, id, jobID, request.IdempotencyKey, digest, request.Quote.OfferID, request.Quote.OfferVersion, string(jobInput), request.Quote.AmountMinor, request.Quote.Currency, request.Door, request.AgentManagerRunID, request.Quote.X402PriceID, s.now().UTC())
	if err != nil {
		return nil, fmt.Errorf("persist service order before authorization: %w", err)
	}
	order, err := s.Get(ctx, byOrderID("", request.IdempotencyKey))
	if err != nil {
		return nil, err
	}
	if order.TermsSHA256 != digest {
		return nil, ErrServiceOrderIdempotencyConflict
	}
	if order.State != ServiceOrderCreated {
		return publicOrder(order), nil
	}

	key := serviceOrderProviderKey(order.ID, "authorize")
	receipt, err := s.provider.Authorize(ctx, ServiceOrderAuthorizationRequest{OrderID: order.ID, JobID: order.JobID, IdempotencyKey: key, Door: request.Door, OfferID: request.Quote.OfferID, OfferVersion: request.Quote.OfferVersion, ResourceID: serviceOrderResourceID(request.Quote.OfferID, request.Quote.OfferVersion), PriceID: request.Quote.X402PriceID, PaymentSignature: request.PaymentSignature, AmountMinor: order.AmountMinor, Currency: order.Currency})
	if err != nil {
		return nil, fmt.Errorf("sandbox authorization outcome is pending; retry with the same order identity: %w", err)
	}
	if !receipt.Sandbox || strings.TrimSpace(receipt.EventID) == "" || strings.TrimSpace(receipt.AuthorizationID) == "" {
		return nil, errors.New("sandbox provider returned no explicitly sandbox authorization receipt")
	}
	state, authorizationID := ServiceOrderDeclined, ""
	if receipt.Approved {
		state, authorizationID = ServiceOrderAuthorized, receipt.AuthorizationID
	}
	if err := s.transition(ctx, order.ID, OrderTransition{EventID: receipt.EventID, Kind: "authorization", ReceiptID: receipt.AuthorizationID, State: state, AuthorizationID: authorizationID, Payer: receipt.Payer, Network: receipt.Network, Expected: ServiceOrderCreated}); err != nil {
		return nil, err
	}
	return s.getOrder(ctx, order.ID)
}

func (s *ServiceOrderSandbox) BeginFromOffer(ctx context.Context, source ServiceOrderOfferSource, request ServiceOrderOfferRequest) (*ServiceOrder, error) {
	if source == nil {
		return nil, errors.New("service offer source is required")
	}
	quote, err := source.ResolveServiceOrderQuote(ctx, request.OfferID, request.OfferVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve canonical service offer: %w", err)
	}
	if quote.OfferID != strings.TrimSpace(request.OfferID) || quote.OfferVersion != request.OfferVersion {
		return nil, errors.New("service offer source returned a different offer identity or version")
	}
	return s.Begin(ctx, BeginServiceOrderRequest{
		Quote: quote, Door: request.Door, AgentManagerRunID: request.AgentManagerRunID, IdempotencyKey: request.IdempotencyKey,
		JobInput: request.JobInput, PaymentSignature: request.PaymentSignature,
	})
}

func (s *ServiceOrderSandbox) VerifyAndCapture(ctx context.Context, orderID string) (*ServiceOrder, error) {
	order, err := s.Get(ctx, byOrderID(orderID, ""))
	if err != nil {
		return nil, err
	}
	if order.State == ServiceOrderAuthorized {
		verification, verifyErr := s.verifier.Verify(ctx, order.public())
		if verifyErr != nil {
			return nil, fmt.Errorf("delivery verification remains pending: %w", verifyErr)
		}
		if strings.TrimSpace(verification.ReceiptID) == "" {
			return nil, errors.New("delivery verifier must provide a durable receipt id")
		}
		state := ServiceOrderVerifyFailed
		if verification.Passed {
			state = ServiceOrderVerified
		}
		if err := s.transition(ctx, order.ID, OrderTransition{EventID: "verification:" + verification.ReceiptID, Kind: "verification", ReceiptID: verification.ReceiptID, State: state, VerificationID: verification.ReceiptID, Expected: ServiceOrderAuthorized}); err != nil {
			return nil, err
		}
		order, err = s.Get(ctx, byOrderID(order.ID, ""))
		if err != nil {
			return nil, err
		}
	}
	order, err = s.Get(ctx, byOrderID(order.ID, ""))
	if err != nil {
		return nil, err
	}
	if order.State == ServiceOrderVerifyFailed {
		void, voidErr := s.provider.Void(ctx, ServiceOrderVoidRequest{OrderID: order.ID, JobID: order.JobID, ResourceID: serviceOrderResourceID(order.OfferID, order.OfferVersion), PriceID: order.X402PriceID, Payer: order.X402Payer, Network: order.X402Network, AuthorizationID: order.AuthorizationID, IdempotencyKey: serviceOrderProviderKey(order.ID, "void"), Door: order.Door})
		if voidErr != nil {
			return nil, fmt.Errorf("failed verification; sandbox authorization void is pending: %w", voidErr)
		}
		if !void.Sandbox || !void.Voided || void.EventID == "" || void.VoidID == "" {
			return nil, errors.New("sandbox provider did not confirm authorization void")
		}
		if err := s.transition(ctx, order.ID, OrderTransition{EventID: void.EventID, Kind: "void", ReceiptID: void.VoidID, State: ServiceOrderVoided, Expected: ServiceOrderVerifyFailed}); err != nil {
			return nil, err
		}
		return s.getOrder(ctx, order.ID)
	}
	if order.State == ServiceOrderVerified {
		if err := s.transition(ctx, order.ID, OrderTransition{EventID: "capture-request:" + order.ID, Kind: "capture_requested", State: ServiceOrderCapturePending, Expected: ServiceOrderVerified}); err != nil {
			return nil, err
		}
		order, err = s.Get(ctx, byOrderID(order.ID, ""))
		if err != nil {
			return nil, err
		}
	}
	if order.State == ServiceOrderCaptured || order.State == ServiceOrderLedgerPending || order.State == ServiceOrderReconciled {
		return publicOrder(order), nil
	}
	if order.State != ServiceOrderCapturePending {
		return publicOrder(order), fmt.Errorf("capture refused while service order is %s", order.State)
	}

	if order.Door == ServiceOrderDoorAgentRun && s.treasuryReceipts == nil {
		return nil, errors.New("Treasury receipt reader is required before capturing an x402 service order")
	}
	key := serviceOrderProviderKey(order.ID, "capture")
	receipt, err := s.provider.Capture(ctx, ServiceOrderCaptureRequest{OrderID: order.ID, JobID: order.JobID, AuthorizationID: order.AuthorizationID, IdempotencyKey: key, Door: order.Door, ResourceID: serviceOrderResourceID(order.OfferID, order.OfferVersion), PriceID: order.X402PriceID, Payer: order.X402Payer, Network: order.X402Network, AmountMinor: order.AmountMinor, Currency: order.Currency})
	if err != nil {
		return nil, fmt.Errorf("sandbox capture outcome is pending; retry with the same order identity: %w", err)
	}
	if !receipt.Sandbox || !receipt.Captured || strings.TrimSpace(receipt.EventID) == "" || strings.TrimSpace(receipt.CaptureID) == "" {
		return nil, errors.New("sandbox provider returned no explicitly sandbox capture receipt")
	}
	if err := s.captureAndQueue(ctx, order, receipt); err != nil {
		return nil, err
	}
	return s.getOrder(ctx, order.ID)
}

func (s *ServiceOrderSandbox) Reconcile(ctx context.Context, orderID string) (*ServiceOrder, error) {
	order, err := s.Get(ctx, byOrderID(orderID, ""))
	if err != nil {
		return nil, err
	}
	if order.State == ServiceOrderReconciled {
		return publicOrder(order), nil
	}
	if order.Door == ServiceOrderDoorAgentRun {
		return s.reconcileTreasuryReceipt(ctx, order)
	}
	if s.ledger == nil {
		return nil, errors.New("non-x402 order reconciliation requires a Money Ledger writer")
	}
	if order.State != ServiceOrderLedgerPending {
		return nil, fmt.Errorf("ledger reconciliation refused while service order is %s", order.State)
	}
	var externalID, fixtureID, adapterID, bookID, accountID, currency, captureID string
	var amount int64
	var occurred time.Time
	if err := s.db.QueryRowContext(ctx, `SELECT external_id,fixture_id,adapter_id,book_id,account_id,amount_minor,currency,capture_id,occurred_at FROM service_order_ledger_outbox WHERE order_id=$1 AND delivered_at IS NULL`, order.ID).Scan(&externalID, &fixtureID, &adapterID, &bookID, &accountID, &amount, &currency, &captureID, &occurred); err != nil {
		return nil, err
	}
	if fixtureID != s.target.FixtureID {
		return nil, errors.New("service order outbox fixture does not match the isolated ledger target")
	}
	event := &sharedpb.MoneyEvent{ExternalId: externalID, AdapterId: adapterID, BookId: bookID, AccountId: accountID, AmountMinor: amount, Currency: currency, OccurredAt: timestamppb.New(occurred.UTC()), FetchedAt: timestamppb.New(occurred.UTC()), Basis: sharedpb.Basis_BASIS_DERIVED, Description: "TEST ONLY sandbox capture simulation; no real funds moved", Category: "test.service_order.sandbox"}
	receipt, err := s.ledger.IngestSandboxCapture(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("Money Ledger sandbox fixture reconciliation pending: %w", err)
	}
	if receipt.PostingID == "" || receipt.ExternalID != externalID {
		return nil, errors.New("Money Ledger returned a mismatched or incomplete sandbox posting receipt")
	}
	if err := s.completeLedgerReconciliation(ctx, order.ID, externalID, receipt); err != nil {
		return nil, err
	}
	return s.getOrder(ctx, order.ID)
}

func (s *ServiceOrderSandbox) reconcileTreasuryReceipt(ctx context.Context, order *serviceOrderRecord) (*ServiceOrder, error) {
	if order.State != ServiceOrderLedgerPending {
		return nil, fmt.Errorf("Treasury receipt reconciliation refused while service order is %s", order.State)
	}
	if s.treasuryReceipts == nil || strings.TrimSpace(order.TreasuryExternalID) == "" {
		return nil, errors.New("Treasury receipt lookup is unavailable for this x402 service order")
	}
	receipt, err := s.treasuryReceipts.ReadAccepted(ctx, order.TreasuryExternalID)
	if err != nil {
		return nil, fmt.Errorf("Treasury seller receipt remains pending: %w", err)
	}
	if receipt.ExternalID != order.TreasuryExternalID || receipt.AmountMinor != order.AmountMinor || receipt.Currency != order.Currency {
		return nil, errors.New("Treasury accepted receipt does not match the service order identity and quote")
	}
	if err := s.completeTreasuryReceiptLink(ctx, order, receipt); err != nil {
		return nil, err
	}
	return s.getOrder(ctx, order.ID)
}

func (s *ServiceOrderSandbox) Get(ctx context.Context, identity serviceOrderIdentity) (*serviceOrderRecord, error) {
	query, arg := `SELECT id::text,job_id::text,idempotency_key,terms_sha256,offer_id,offer_version,job_payload,amount_minor,currency,door,COALESCE(agent_manager_run_id,''),COALESCE(x402_price_id,''),COALESCE(x402_payer,''),COALESCE(x402_network,''),mode,state,COALESCE(authorization_id,''),COALESCE(verification_id,''),COALESCE(capture_id,''),COALESCE(ledger_posting_id,''),COALESCE(ledger_receipt_id,''),COALESCE(treasury_external_id,''),created_at,updated_at FROM service_orders WHERE id=$1`, any(identity.orderID)
	if identity.orderID == "" {
		query, arg = strings.Replace(query, "WHERE id=$1", "WHERE idempotency_key=$1", 1), identity.idempotencyKey
	}
	var order serviceOrderRecord
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&order.ID, &order.JobID, &order.IdempotencyKey, &order.TermsSHA256, &order.OfferID, &order.OfferVersion, &order.JobInput, &order.AmountMinor, &order.Currency, &order.Door, &order.AgentManagerRunID, &order.X402PriceID, &order.X402Payer, &order.X402Network, &order.Mode, &order.State, &order.AuthorizationID, &order.VerificationID, &order.CaptureID, &order.LedgerPostingID, &order.LedgerReceiptID, &order.TreasuryExternalID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrder exposes the durable sandbox order record for status pages and
// acceptance tests. It does not reveal provider credentials or permit state
// changes.
func (s *ServiceOrderSandbox) GetOrder(ctx context.Context, orderID string) (*ServiceOrder, error) {
	return s.getOrder(ctx, orderID)
}

func (s *ServiceOrderSandbox) getOrder(ctx context.Context, orderID string) (*ServiceOrder, error) {
	order, err := s.Get(ctx, byOrderID(orderID, ""))
	if err != nil {
		return nil, err
	}
	return publicOrder(order), nil
}

type serviceOrderRecord struct {
	ID, JobID, IdempotencyKey, TermsSHA256, OfferID                                                                                                  string
	JobInput                                                                                                                                         json.RawMessage
	OfferVersion                                                                                                                                     int
	AmountMinor                                                                                                                                      int64
	Currency, Door, AgentManagerRunID, X402PriceID, X402Payer, X402Network, Mode, State, AuthorizationID, VerificationID, CaptureID, LedgerPostingID string
	LedgerReceiptID, TreasuryExternalID                                                                                                              string
	CreatedAt, UpdatedAt                                                                                                                             time.Time
}

func (r serviceOrderRecord) public() ServiceOrder {
	return ServiceOrder{ID: r.ID, JobID: r.JobID, OfferID: r.OfferID, OfferVersion: r.OfferVersion, JobInput: append(json.RawMessage(nil), r.JobInput...), AmountMinor: r.AmountMinor, Currency: r.Currency, Door: r.Door, AgentManagerRunID: r.AgentManagerRunID, Mode: r.Mode, State: r.State, AuthorizationID: r.AuthorizationID, VerificationID: r.VerificationID, CaptureID: r.CaptureID, LedgerPostingID: r.LedgerPostingID, LedgerReceiptID: r.LedgerReceiptID, TreasuryExternalID: r.TreasuryExternalID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func publicOrder(r *serviceOrderRecord) *ServiceOrder {
	if r == nil {
		return nil
	}
	public := r.public()
	return &public
}

type serviceOrderIdentity struct{ orderID, idempotencyKey string }

func byOrderID(orderID, key string) serviceOrderIdentity {
	return serviceOrderIdentity{orderID: orderID, idempotencyKey: key}
}

type OrderTransition struct {
	EventID, Kind, ReceiptID, State, AuthorizationID, VerificationID, LedgerPostingID, Payer, Network string
	Expected                                                                                          string
}

func (s *ServiceOrderSandbox) transition(ctx context.Context, orderID string, transition OrderTransition) error {
	payload, _ := json.Marshal(transition)
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if duplicate, err := recordServiceOrderEvent(ctx, tx, orderID, transition.EventID, transition.Kind, transition.ReceiptID, digest); err != nil {
		return err
	} else if duplicate {
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE service_orders SET state=$3,authorization_id=CASE WHEN $4='' THEN authorization_id ELSE $4 END,verification_id=CASE WHEN $5='' THEN verification_id ELSE $5 END,ledger_posting_id=CASE WHEN $6='' THEN ledger_posting_id ELSE $6 END,x402_payer=CASE WHEN $7='' THEN x402_payer ELSE $7 END,x402_network=CASE WHEN $8='' THEN x402_network ELSE $8 END,updated_at=$9 WHERE id=$1 AND state=$2`, orderID, transition.Expected, transition.State, transition.AuthorizationID, transition.VerificationID, transition.LedgerPostingID, transition.Payer, transition.Network, s.now().UTC())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("service order transition %s refused from current state", transition.Kind)
	}
	return tx.Commit()
}

func (s *ServiceOrderSandbox) completeLedgerReconciliation(ctx context.Context, orderID, externalID string, receipt ServiceOrderLedgerReceipt) error {
	payload, _ := json.Marshal(struct {
		ExternalID string
		PostingID  string
		Duplicate  bool
	}{externalID, receipt.PostingID, receipt.Duplicate})
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	eventID := "ledger-ack:" + externalID
	if duplicate, err := recordServiceOrderEvent(ctx, tx, orderID, eventID, "ledger_reconciled", receipt.PostingID, digest); err != nil {
		return err
	} else if duplicate {
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE service_order_ledger_outbox SET delivered_at=$2,posting_id=$3,duplicate=$4 WHERE order_id=$1 AND delivered_at IS NULL`, orderID, s.now().UTC(), receipt.PostingID, receipt.Duplicate)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return errors.New("ledger outbox was already delivered without a matching order receipt")
	}
	result, err = tx.ExecContext(ctx, `UPDATE service_orders SET state=$3,ledger_posting_id=$4,ledger_receipt_id=$5,updated_at=$6 WHERE id=$1 AND state=$2`, orderID, ServiceOrderLedgerPending, ServiceOrderReconciled, receipt.PostingID, receipt.ReceiptID, s.now().UTC())
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return errors.New("service order changed before the ledger receipt was committed")
	}
	return tx.Commit()
}

func (s *ServiceOrderSandbox) completeTreasuryReceiptLink(ctx context.Context, order *serviceOrderRecord, receipt TreasuryX402Receipt) error {
	payload, _ := json.Marshal(receipt)
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if duplicate, err := recordServiceOrderEvent(ctx, tx, order.ID, "treasury-receipt:"+receipt.ReceiptID, "treasury_ledger_receipt", receipt.PostingID, digest); err != nil {
		return err
	} else if duplicate {
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE service_orders SET state=$3,ledger_posting_id=$4,ledger_receipt_id=$5,updated_at=$6 WHERE id=$1 AND state=$2 AND treasury_external_id=$7`, order.ID, ServiceOrderLedgerPending, ServiceOrderReconciled, receipt.PostingID, receipt.ReceiptID, s.now().UTC(), receipt.ExternalID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("service order changed before the Treasury receipt link was committed")
	}
	return tx.Commit()
}

func (s *ServiceOrderSandbox) captureAndQueue(ctx context.Context, order *serviceOrderRecord, receipt ServiceOrderCaptureReceipt) error {
	externalID := serviceOrderLedgerExternalID(order.ID, receipt.CaptureID)
	if order.Door == ServiceOrderDoorAgentRun {
		externalID = strings.TrimSpace(receipt.TreasuryExternalID)
		if !strings.HasPrefix(externalID, "treasury-x402-inflow:") {
			return errors.New("x402 capture receipt must identify the Treasury seller-side ledger external id")
		}
	} else if strings.TrimSpace(receipt.TreasuryExternalID) != "" {
		return errors.New("Treasury seller receipt cannot be linked to a non-x402 order")
	}
	payload, _ := json.Marshal(receipt)
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if duplicate, err := recordServiceOrderEvent(ctx, tx, order.ID, receipt.EventID, "capture", receipt.CaptureID, digest); err != nil {
		return err
	} else if duplicate {
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE service_orders SET state=$3,capture_id=$4,treasury_external_id=NULLIF($5,''),updated_at=$6 WHERE id=$1 AND state=$2`, order.ID, ServiceOrderCapturePending, ServiceOrderLedgerPending, receipt.CaptureID, receipt.TreasuryExternalID, s.now().UTC())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("capture receipt refused because the order is not capture-pending")
	}
	if order.Door != ServiceOrderDoorAgentRun {
		if _, err := tx.ExecContext(ctx, `INSERT INTO service_order_ledger_outbox(order_id,fixture_id,adapter_id,book_id,account_id,external_id,amount_minor,currency,capture_id,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, order.ID, s.target.FixtureID, s.target.AdapterID, s.target.BookID, s.target.RevenueAccountID, externalID, order.AmountMinor, order.Currency, receipt.CaptureID, s.now().UTC()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func recordServiceOrderEvent(ctx context.Context, tx *sql.Tx, orderID, eventID, kind, receiptID, digest string) (bool, error) {
	if strings.TrimSpace(eventID) == "" {
		return false, errors.New("service order event requires a stable id")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO service_order_events(order_id,event_id,event_kind,payload_sha256,receipt_id) VALUES($1,$2,$3,$4,NULLIF($5,'')) ON CONFLICT(event_id) DO NOTHING`, orderID, eventID, kind, digest, receiptID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 1 {
		return false, nil
	}
	var oldOrder, oldKind, oldDigest string
	if err := tx.QueryRowContext(ctx, `SELECT order_id::text,event_kind,payload_sha256 FROM service_order_events WHERE event_id=$1`, eventID).Scan(&oldOrder, &oldKind, &oldDigest); err != nil {
		return false, err
	}
	if oldOrder != orderID || oldKind != kind || oldDigest != digest {
		return false, fmt.Errorf("service order event %q was replayed with a different payload", eventID)
	}
	return true, nil
}

func validateServiceOrderRequest(request BeginServiceOrderRequest) error {
	q := request.Quote
	if strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(q.OfferID) == "" || q.OfferVersion < 1 || q.AmountMinor <= 0 || len(q.Currency) != 3 || q.Currency != strings.ToUpper(q.Currency) || !validServiceOrderDoor(request.Door) {
		return errors.New("service order requires a stable idempotency key, offer snapshot, positive minor-unit price, currency, and door")
	}
	if request.Door == ServiceOrderDoorAgentRun {
		if _, err := uuid.Parse(strings.TrimSpace(request.AgentManagerRunID)); err != nil || strings.TrimSpace(q.X402PriceID) == "" || strings.TrimSpace(request.PaymentSignature) == "" {
			return errors.New("Agent Manager x402 orders require a valid run id, Offer Desk price id, and payment signature")
		}
	} else if strings.TrimSpace(request.PaymentSignature) != "" || strings.TrimSpace(request.AgentManagerRunID) != "" {
		return errors.New("payment signature and Agent Manager run id are accepted only on the Agent Manager x402 door")
	}
	input := normalizedJobInput(request.JobInput)
	var object map[string]json.RawMessage
	if !json.Valid(input) || json.Unmarshal(input, &object) != nil || object == nil {
		return errors.New("service order job input must be a JSON object")
	}
	if q.OfferID != "company-market-scan" || q.OfferVersion != 1 {
		return errors.New("service order requires a supported, versioned Company Market Scan offer")
	}
	var sector string
	if len(object) != 1 || json.Unmarshal(object["sector"], &sector) != nil || strings.TrimSpace(sector) == "" || utf8.RuneCountInString(sector) > 500 {
		return errors.New("Company Market Scan requires one market or vendor category of 1 to 500 characters")
	}
	return nil
}

func validServiceOrderDoor(door string) bool {
	switch door {
	case ServiceOrderDoorBrowser, ServiceOrderDoorBrowserAgent, ServiceOrderDoorMCP, ServiceOrderDoorAgentRun:
		return true
	default:
		return false
	}
}

func serviceOrderTermsDigest(request BeginServiceOrderRequest) string {
	data, _ := json.Marshal(struct {
		Quote                ServiceOrderQuote
		Door                 string
		AgentManagerRunID    string
		JobInput             json.RawMessage
		PaymentSignatureHash string
	}{request.Quote, request.Door, request.AgentManagerRunID, normalizedJobInput(request.JobInput), fmt.Sprintf("%x", sha256.Sum256([]byte(request.PaymentSignature)))})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func serviceOrderResourceID(offerID string, version int) string {
	return fmt.Sprintf("%s:v%d", strings.TrimSpace(offerID), version)
}

func normalizedJobInput(input json.RawMessage) json.RawMessage {
	if len(input) == 0 {
		return json.RawMessage(`{}`)
	}
	return input
}

func serviceOrderProviderKey(orderID, operation string) string {
	return "service-order:" + orderID + ":" + operation
}

func serviceOrderLedgerExternalID(orderID, captureID string) string {
	return "service-order:" + orderID + ":capture:" + captureID
}
