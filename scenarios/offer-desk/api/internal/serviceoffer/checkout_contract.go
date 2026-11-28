package serviceoffer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CheckoutMode is deliberately a closed enum. A sandbox order cannot be
// promoted to a live order by changing a runtime flag; live checkout requires a
// separate contract and an operator decision.
type CheckoutMode string

const CheckoutModeSandbox CheckoutMode = "sandbox"

type OrderState string

const (
	DoorBrowser      = "browser"
	DoorBrowserAgent = "browsing_agent"
	DoorMCP          = "mcp"
	DoorAgentManager = "agent_manager_x402"
)

const (
	OrderCreated            OrderState = "created"
	OrderAuthorized         OrderState = "authorized"
	OrderVerificationFailed OrderState = "verification_failed"
	OrderVerified           OrderState = "verified"
	OrderCapturePending     OrderState = "capture_pending"
	OrderCaptureFailed      OrderState = "capture_failed"
	OrderCaptured           OrderState = "captured"
	OrderVoided             OrderState = "voided"
	OrderLedgerPending      OrderState = "ledger_pending"
	OrderReconciled         OrderState = "reconciled"
	OrderDeclined           OrderState = "declined"
)

type OrderEventKind string

const (
	EventAuthorized         OrderEventKind = "authorized"
	EventDeclined           OrderEventKind = "declined"
	EventVerified           OrderEventKind = "verified"
	EventVerificationFailed OrderEventKind = "verification_failed"
	EventCaptureRequested   OrderEventKind = "capture_requested"
	EventCaptureFailed      OrderEventKind = "capture_failed"
	EventCaptured           OrderEventKind = "captured"
	EventVoided             OrderEventKind = "voided"
	EventLedgerPending      OrderEventKind = "ledger_pending"
	EventReconciled         OrderEventKind = "reconciled"
)

// SandboxOrder is the durable identity and immutable commercial snapshot a
// provider-neutral checkout implementation must persist before contacting a
// provider. JobID is assigned before authorization so every channel can retry
// with the same order identity.
type SandboxOrder struct {
	OrderID         string       `json:"order_id"`
	JobID           string       `json:"job_id"`
	OfferID         string       `json:"offer_id"`
	OfferVersion    int          `json:"offer_version"`
	Door            string       `json:"door"`
	Mode            CheckoutMode `json:"mode"`
	AmountMinor     int64        `json:"amount_minor"`
	Currency        string       `json:"currency"`
	TermsSHA256     string       `json:"terms_sha256"`
	State           OrderState   `json:"state"`
	AuthorizationID string       `json:"authorization_id,omitempty"`
	VerificationID  string       `json:"verification_id,omitempty"`
	CaptureID       string       `json:"capture_id,omitempty"`
	LedgerPostingID string       `json:"ledger_posting_id,omitempty"`
	Events          []OrderEvent `json:"events,omitempty"`
}

// OrderEvent is the append-only transition receipt. EventID is unique within
// an order. Replaying the same event is harmless; reusing an ID with a changed
// payload is rejected. Provider callback IDs belong in EventID.
type OrderEvent struct {
	EventID   string         `json:"event_id"`
	Kind      OrderEventKind `json:"kind"`
	ReceiptID string         `json:"receipt_id,omitempty"`
}

type ProviderOperation string

const (
	ProviderAuthorize ProviderOperation = "authorize"
	ProviderCapture   ProviderOperation = "capture"
	ProviderVoid      ProviderOperation = "void"
)

// ProviderIdempotencyKey derives the key from the order identity and
// operation, so retries do not accidentally create a second authorization,
// capture, or void. Provider adapters must forward it unchanged.
func ProviderIdempotencyKey(orderID string, operation ProviderOperation) (string, error) {
	if strings.TrimSpace(orderID) == "" {
		return "", errors.New("order_id is required for provider idempotency")
	}
	switch operation {
	case ProviderAuthorize, ProviderCapture, ProviderVoid:
		return "service-order:" + orderID + ":" + string(operation), nil
	default:
		return "", fmt.Errorf("unsupported provider operation %q", operation)
	}
}

// LedgerExternalID keeps every replay of one captured order on the same
// Money Ledger `(adapter_id, external_id)` identity. Capture IDs are included
// so accidental provider receipt rebinding is visible as a conflict.
func LedgerExternalID(orderID, captureID string) (string, error) {
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(captureID) == "" {
		return "", errors.New("order_id and capture_id are required for ledger identity")
	}
	return "service-order:" + orderID + ":capture:" + captureID, nil
}

// NewSandboxOrder freezes the offer and channel identity before any provider
// call. It performs no persistence, payment, ledger write, or delivery.
func NewSandboxOrder(orderID, jobID string, offer Record, door string) (SandboxOrder, error) {
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(jobID) == "" {
		return SandboxOrder{}, errors.New("order_id and job_id are required before authorization")
	}
	if err := Validate(offer); err != nil {
		return SandboxOrder{}, fmt.Errorf("validate offer snapshot: %w", err)
	}
	if strings.TrimSpace(door) == "" {
		return SandboxOrder{}, errors.New("purchase door is required")
	}
	if !supportedSandboxDoor(door) {
		return SandboxOrder{}, fmt.Errorf("unsupported sandbox purchase door %q", door)
	}
	order := SandboxOrder{
		OrderID: orderID, JobID: jobID, OfferID: offer.ID, OfferVersion: offer.Version,
		Door: door, Mode: CheckoutModeSandbox, AmountMinor: offer.ProposedPrice.AmountMinor,
		Currency: offer.ProposedPrice.Currency, State: OrderCreated,
	}
	order.TermsSHA256 = termsDigest(order)
	return order, nil
}

func supportedSandboxDoor(door string) bool {
	switch door {
	case DoorBrowser, DoorBrowserAgent, DoorMCP, DoorAgentManager:
		return true
	default:
		return false
	}
}

// ApplyOrderEvent validates one durable state transition. Provider and ledger
// receipts are references only; this function does not claim to contact either
// owner. A caller must persist the returned order atomically with the event.
func ApplyOrderEvent(order SandboxOrder, event OrderEvent) (SandboxOrder, error) {
	if order.Mode != CheckoutModeSandbox {
		return order, errors.New("only sandbox orders are accepted by this contract")
	}
	if order.TermsSHA256 == "" || order.TermsSHA256 != termsDigest(order) {
		return order, errors.New("sandbox order's immutable terms changed")
	}
	if strings.TrimSpace(order.OrderID) == "" || strings.TrimSpace(order.JobID) == "" {
		return order, errors.New("order and job identity must be durable before transitions")
	}
	if strings.TrimSpace(event.EventID) == "" {
		return order, errors.New("event_id is required for idempotent replay")
	}
	for _, prior := range order.Events {
		if prior.EventID == event.EventID {
			if prior == event {
				return order, nil
			}
			return order, fmt.Errorf("event_id %q was replayed with a different payload", event.EventID)
		}
	}

	next := order
	next.Events = append(append([]OrderEvent(nil), order.Events...), event)
	needReceipt := func() error {
		if strings.TrimSpace(event.ReceiptID) == "" {
			return errors.New("provider or ledger receipt_id is required")
		}
		return nil
	}
	switch event.Kind {
	case EventAuthorized:
		if order.State != OrderCreated {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.AuthorizationID = OrderAuthorized, event.ReceiptID
	case EventDeclined:
		if order.State != OrderCreated {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.AuthorizationID = OrderDeclined, event.ReceiptID
	case EventVerified:
		if order.State != OrderAuthorized {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.VerificationID = OrderVerified, event.ReceiptID
	case EventVerificationFailed:
		if order.State != OrderAuthorized {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.VerificationID = OrderVerificationFailed, event.ReceiptID
	case EventCaptureRequested:
		if order.State != OrderVerified && order.State != OrderCaptureFailed {
			return order, transitionError(order.State, event.Kind)
		}
		next.State = OrderCapturePending
	case EventCaptureFailed:
		if order.State != OrderCapturePending {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State = OrderCaptureFailed
	case EventCaptured:
		if order.State != OrderCapturePending {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.CaptureID = OrderCaptured, event.ReceiptID
	case EventVoided:
		if order.State != OrderAuthorized && order.State != OrderVerificationFailed && order.State != OrderCaptureFailed {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State = OrderVoided
	case EventLedgerPending:
		if order.State != OrderCaptured {
			return order, transitionError(order.State, event.Kind)
		}
		next.State = OrderLedgerPending
	case EventReconciled:
		if order.State != OrderCaptured && order.State != OrderLedgerPending {
			return order, transitionError(order.State, event.Kind)
		}
		if err := needReceipt(); err != nil {
			return order, err
		}
		next.State, next.LedgerPostingID = OrderReconciled, event.ReceiptID
	default:
		return order, fmt.Errorf("unsupported order event kind %q", event.Kind)
	}
	return next, nil
}

func transitionError(state OrderState, event OrderEventKind) error {
	return fmt.Errorf("cannot apply %s while order is %s", event, state)
}

type orderTerms struct {
	OrderID, JobID, OfferID string
	OfferVersion            int
	Door                    string
	Mode                    CheckoutMode
	AmountMinor             int64
	Currency                string
}

func termsDigest(order SandboxOrder) string {
	terms := orderTerms{order.OrderID, order.JobID, order.OfferID, order.OfferVersion, order.Door, order.Mode, order.AmountMinor, order.Currency}
	data, _ := json.Marshal(terms)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// Provider-neutral contract: implementations must use the given idempotency
// key unchanged across retries and return stable receipt IDs. A Stripe-specific
// adapter is intentionally outside this prototype.
type SandboxPaymentProvider interface {
	Authorize(context.Context, AuthorizationRequest) (AuthorizationReceipt, error)
	Capture(context.Context, CaptureRequest) (CaptureReceipt, error)
	Void(context.Context, VoidRequest) (VoidReceipt, error)
}

type AuthorizationRequest struct {
	OrderID, IdempotencyKey string
	AmountMinor             int64
	Currency                string
}
type AuthorizationReceipt struct {
	OrderID, AuthorizationID, ProviderEventID string
	Approved                                  bool
}
type CaptureRequest struct {
	OrderID, AuthorizationID, IdempotencyKey string
	AmountMinor                              int64
	Currency                                 string
}
type CaptureReceipt struct {
	OrderID, CaptureID, ProviderEventID string
	Captured                            bool
}
type VoidRequest struct{ OrderID, AuthorizationID, IdempotencyKey string }
type VoidReceipt struct {
	OrderID, VoidID, ProviderEventID string
	Voided                           bool
}

// SandboxLedger is separate from the live ledger contract and can only target a
// named isolated journey fixture. It is not a Money Ledger production API.
type SandboxLedger interface {
	ReconcileSandboxCapture(context.Context, LedgerReconciliationRequest) (LedgerReceipt, error)
}

type LedgerReconciliationRequest struct {
	OrderID, JobID, OfferID   string
	CaptureID, IdempotencyKey string
	AmountMinor               int64
	Currency                  string
	FixtureID                 string
}
type LedgerReceipt struct {
	OrderID, PostingID, ExternalID string
	Duplicate                      bool
}
